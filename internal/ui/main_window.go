package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/assets"
	appdirs "go-chrome/internal/app"
	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/db"
	"go-chrome/internal/flow"
	"go-chrome/internal/runner"
	"go-chrome/internal/template"
)

type App struct {
	fyneApp     fyne.App
	mainWin     fyne.Window
	cfg         *config.Config
	dirs        *appdirs.Directories
	flowStore   *db.FlowStore
	recentStore *flow.RecentStore
	browserMgr  *browser.Manager
	runner      *runner.Runner
	stepRunner  *runner.StepRunner
	envRepo     *db.EnvRepo
	recentRepo  *db.RecentRepo
	runRepo     *db.RunRepo

	statusBar    *statusBar
	flowLibrary  *flowLibraryPanel
	flowEditor   *flowEditorPanel
	stepTable    *stepTablePanel
	stepProperty *stepPropertyPanel
	runPanel     *runPanel
	historyPanel *historyPanel
	workspace    *fyne.Container
	editorArea   fyne.CanvasObject
	emptyState   fyne.CanvasObject
	currentFlow  *flow.Flow
	runStatuses  []runner.Status

	dirty        bool
	chromeTicker *time.Ticker
	chromeDone   chan struct{}

	stepBtn *widget.Button
}

// runHistoryAdapter adapts db.RunRepo to runner.HistorySaver.
type runHistoryAdapter struct {
	repo *db.RunRepo
}

func (a *runHistoryAdapter) Save(result *runner.RunResult) error {
	return a.repo.Save(result, result.EnvironmentID)
}

func New(cfg *config.Config, dirs *appdirs.Directories) *App {
	return &App{cfg: cfg, dirs: dirs}
}

func (a *App) Run() {
	a.fyneApp = app.NewWithID("com.go-chrome.app")
	a.fyneApp.Settings().SetTheme(newAppTheme())
	if ico := assets.Icon(); ico != nil {
		a.fyneApp.SetIcon(ico)
	}
	a.mainWin = a.fyneApp.NewWindow("Chrome 自动化编排工具")
	a.mainWin.Resize(fyne.NewSize(float32(a.cfg.App.WindowWidth), float32(a.cfg.App.WindowHeight)))
	a.mainWin.SetOnClosed(func() {
		size := a.mainWin.Canvas().Size()
		a.cfg.App.WindowWidth = int(size.Width)
		a.cfg.App.WindowHeight = int(size.Height)
		_ = config.Save(a.dirs.ConfigPath, a.cfg)
		if a.recentRepo != nil && a.recentStore != nil {
			_ = a.recentRepo.Save(a.recentStore.FlowIDs)
		}
		if a.chromeDone != nil {
			close(a.chromeDone)
		}
	})

	if err := a.initDeps(); err != nil {
		dialog.ShowError(fmt.Errorf("应用初始化失败: %w", err), a.mainWin)
		a.mainWin.ShowAndRun()
		return
	}
	a.buildUI()
	a.firstRunCheck()
	a.startChromeTicker()
	a.mainWin.ShowAndRun()
}

func (a *App) initDeps() error {
	// SQLite is the single source of truth
	sqliteDB, err := db.Open(filepath.Join(a.dirs.DataDir, "go-chrome.db"))
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}

	a.flowStore, err = db.NewFlowStore(sqliteDB)
	if err != nil {
		return fmt.Errorf("初始化流程存储失败: %w", err)
	}

	// Recent flows backed by SQLite
	r := db.NewRecentRepo(sqliteDB)
	ids, _ := r.Load()
	a.recentStore = &flow.RecentStore{FlowIDs: ids}
	a.recentRepo = r

	// Environment repository
	a.envRepo = db.NewEnvRepo(sqliteDB)
	if err := a.envRepo.CreateDefaultIfNone(); err != nil {
		return fmt.Errorf("初始化环境仓库失败: %w", err)
	}

	a.browserMgr = browser.NewManager(&a.cfg.Chrome)
	a.browserMgr.LoadManifest() // best effort

	// Run history backed by SQLite
	a.runRepo = db.NewRunRepo(sqliteDB)
	historySaver := &runHistoryAdapter{repo: a.runRepo}

	a.runner = runner.NewRunner(&a.cfg.Runner, a.browserMgr, historySaver)
	go a.handleRunnerEvents()
	return nil
}

func (a *App) buildUI() {
	onDirty := func() { a.markDirty() }

	a.statusBar = newStatusBar(a)
	a.flowLibrary = newFlowLibraryPanel(a)
	a.flowEditor = newFlowEditorPanel(a, onDirty)
	a.stepTable = newStepTablePanel(a, onDirty)
	a.stepProperty = newStepPropertyPanel(a, onDirty)
	a.runPanel = newRunPanel(a)
	a.historyPanel = newHistoryPanel(a)

	centerTop := container.NewBorder(a.flowEditor.widget, nil, nil, nil, a.stepTable.widget)
	center := container.NewHSplit(centerTop, a.stepProperty.widget)
	center.SetOffset(0.55)
	a.editorArea = center
	a.emptyState = a.buildEmptyState()
	a.emptyState.Hide()
	a.workspace = container.NewStack(a.editorArea, a.emptyState)

	bottom := container.NewHSplit(a.runPanel.widget, a.historyPanel.widget)
	bottom.SetOffset(0.72)

	mainSplit := container.NewVSplit(
		container.NewBorder(a.statusBar.widget, nil, a.flowLibrary.widget, nil, a.workspace),
		bottom,
	)
	mainSplit.SetOffset(0.72)

	a.mainWin.SetContent(mainSplit)
	a.refreshFlowList()
	a.runPanel.refreshEnvironments()
	a.historyPanel.refreshFilters()
}

func (a *App) markDirty() {
	a.dirty = true
	a.statusBar.setSave(SaveDirty)
}

func (a *App) markClean() {
	a.dirty = false
	a.statusBar.setSave(SaveUnmodified)
}

func (a *App) startChromeTicker() {
	a.chromeTicker = time.NewTicker(1 * time.Second)
	a.chromeDone = make(chan struct{})
	go func() {
		for {
			select {
			case <-a.chromeTicker.C:
				fyne.Do(func() {
					a.statusBar.setChrome(a.browserMgr.Status())
				})
			case <-a.chromeDone:
				return
			}
		}
	}()
}

func (a *App) createNewFlow() {
	if a.dirty && a.currentFlow != nil {
		a.promptSaveBefore(func() { a.doCreateNewFlow() })
		return
	}
	a.doCreateNewFlow()
}

func (a *App) doCreateNewFlow() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("请输入流程名称")
	dialog.ShowForm("新建流程", "创建", "取消", []*widget.FormItem{
		widget.NewFormItem("流程名称", nameEntry),
	}, func(ok bool) {
		if !ok || nameEntry.Text == "" {
			return
		}
		f := flow.NewFlow(nameEntry.Text)
		if err := a.flowStore.Save(f); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.setCurrentFlow(f)
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) saveCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if err := flow.Validate(a.currentFlow); err != nil {
		dialog.ShowError(fmt.Errorf("保存前校验失败: %w", err), a.mainWin)
		return
	}
	a.statusBar.setSave(SaveSaving)
	if err := a.flowStore.Save(a.currentFlow); err != nil {
		a.statusBar.setSave(SaveFailed)
		dialog.ShowError(err, a.mainWin)
		return
	}
	a.statusBar.setSave(SaveSuccess)
	a.markClean()
}

func (a *App) importFlow() {
	if a.dirty && a.currentFlow != nil {
		a.promptSaveBefore(func() { a.doImportFlow() })
		return
	}
	a.doImportFlow()
}

func (a *App) doImportFlow() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		f, err := a.flowStore.Import(reader.URI().Path())
		if err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.setCurrentFlow(f)
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) exportFlow() {
	if a.currentFlow == nil {
		return
	}
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()
		if err := a.flowStore.Export(a.currentFlow.ID, writer.URI().Path()); err != nil {
			dialog.ShowError(err, a.mainWin)
		}
	}, a.mainWin)
}

func (a *App) startBrowser() {
	if !a.browserMgr.IsInstalled() {
		a.runPanel.log("未检测到本地 Chrome，开始下载...")
		if err := a.browserMgr.Install(func(d, t int64) {
			if t > 0 {
				a.runPanel.log(fmt.Sprintf("下载进度 %d%%", d*100/t))
			}
		}); err != nil {
			a.runPanel.log("Chrome 下载失败：" + err.Error())
			fyne.Do(func() { dialog.ShowError(err, a.mainWin) })
			return
		}
	}
	port, err := a.browserMgr.Start()
	if err != nil {
		a.runPanel.log("Chrome 启动失败：" + err.Error())
		fyne.Do(func() { dialog.ShowError(err, a.mainWin) })
		return
	}
	a.runPanel.log(fmt.Sprintf("Chrome 已启动，调试端口：%d", port))
}

func (a *App) runCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if a.runner.IsRunning() {
		dialog.ShowInformation("提示", "当前已有流程正在运行", a.mainWin)
		return
	}
	// Check environment variables
	if missing := a.checkEnvVars(); len(missing) > 0 {
		dialog.ShowError(fmt.Errorf("运行前检查失败，缺少环境变量: %v", missing), a.mainWin)
		return
	}
	envID, envProvider, err := a.currentEnvProvider()
	if err != nil {
		dialog.ShowError(fmt.Errorf("获取运行环境失败: %w", err), a.mainWin)
		return
	}
	a.runPanel.reset()
	a.runStatuses = make([]runner.Status, len(a.currentFlow.Steps))
	for i := range a.runStatuses {
		a.runStatuses[i] = runner.StatusPending
	}
	a.stepTable.setStatuses(a.runStatuses)
	go func() {
		res := a.runner.RunFlow(a.currentFlow, runner.RunOptions{
			StartStep:     0,
			EnvironmentID: envID,
			EnvProvider:   envProvider,
		})
		a.runPanel.setSummary(res)
	}()
}

func (a *App) checkEnvVars() []string {
	if a.currentFlow == nil {
		return nil
	}
	_, provider, err := a.currentEnvProvider()
	if err != nil {
		return []string{"无法获取运行环境: " + err.Error()}
	}
	return runner.MissingEnvVars(a.currentFlow, 0, provider)
}

func (a *App) currentEnvProvider() (string, template.EnvProvider, error) {
	if a.envRepo == nil {
		return "", nil, fmt.Errorf("环境仓库未初始化")
	}
	selectedName := a.runPanel.envSelect.Selected
	if selectedName == "" {
		selectedName = "默认环境"
	}
	env, err := a.envRepo.GetByName(selectedName)
	if err != nil {
		return "", nil, err
	}
	return env.ID, a.envRepo.EnvProvider(env.ID), nil
}

func (a *App) onStepButton() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if a.stepRunner != nil && !a.stepRunner.IsFinished() {
		a.nextStep()
		return
	}
	if a.stepRunner != nil {
		a.stepRunner.Close()
	}
	// Check environment variables
	if missing := a.checkEnvVars(); len(missing) > 0 {
		dialog.ShowError(fmt.Errorf("运行前检查失败，缺少环境变量: %v", missing), a.mainWin)
		return
	}
	envID, envProvider, err := a.currentEnvProvider()
	if err != nil {
		dialog.ShowError(fmt.Errorf("获取运行环境失败: %w", err), a.mainWin)
		return
	}
	historySaver := &runHistoryAdapter{repo: a.runRepo}
	a.stepRunner = runner.NewStepRunner(&a.cfg.Runner, a.browserMgr, historySaver)
	if err := a.stepRunner.Init(a.currentFlow, envProvider, envID); err != nil {
		dialog.ShowError(err, a.mainWin)
		a.stepRunner = nil
		return
	}
	a.runStatuses = make([]runner.Status, len(a.currentFlow.Steps))
	for i := range a.runStatuses {
		a.runStatuses[i] = runner.StatusPending
	}
	a.stepTable.setStatuses(a.runStatuses)
	a.stepBtn.SetText("下一步")
	a.nextStep()
}

func (a *App) nextStep() {
	if a.stepRunner == nil {
		return
	}
	res, finished, err := a.stepRunner.Next()
	if err != nil {
		a.runPanel.log("单步执行错误：" + err.Error())
		a.stepBtn.SetText("单步执行")
		a.stepRunner.Close()
		a.stepRunner = nil
		return
	}
	if res != nil {
		idx := a.stepRunner.CurrentIndex() - 1
		a.setStepStatus(idx, res.Status)
		logMsg := fmt.Sprintf("步骤 %d %s: %s", a.stepRunner.CurrentIndex(), res.StepName, res.Status)
		if res.Error != "" {
			logMsg += " - " + res.Error
		}
		a.runPanel.log(logMsg)
		a.logArtifacts(res)
	}
	if finished {
		result := a.stepRunner.Result()
		a.runPanel.log(fmt.Sprintf("单步执行完成：%s（成功 %d，失败 %d）", result.Status, result.SuccessCount, result.FailedCount))
		a.stepBtn.SetText("单步执行")
		a.stepRunner.Close()
		a.stepRunner = nil
		a.refreshHistory()
	}
}

func (a *App) handleRunnerEvents() {
	var totalSteps int
	for ev := range a.runner.Events() {
		switch ev.Type {
		case runner.EventLog:
			a.runPanel.log(ev.LogMessage)
		case runner.EventStepStart:
			totalSteps = 0
			if a.currentFlow != nil {
				for _, s := range a.currentFlow.Steps {
					if s.Enabled {
						totalSteps++
					}
				}
			}
			stepName := ""
			if a.currentFlow != nil && ev.StepIndex >= 0 && ev.StepIndex < len(a.currentFlow.Steps) {
				stepName = a.currentFlow.Steps[ev.StepIndex].Name
			}
			a.runPanel.setProgress(ev.StepIndex+1, totalSteps, stepName)
			a.runPanel.setCurrentStep(stepName)
			a.setStepStatus(ev.StepIndex, runner.StatusRunning)
			a.statusBar.setRun(RunRunning, ev.StepIndex+1, totalSteps, "")
		case runner.EventStepDone:
			a.setStepStatus(ev.StepIndex, ev.Result.Status)
			if ev.Result.Status == runner.StatusFailed {
				a.runPanel.setArtifacts(ev.Result.Screenshot, ev.Result.HTMLSnapshot)
				a.logArtifacts(ev.Result)
				a.stepTable.table.Select(widget.TableCellID{Row: ev.StepIndex, Col: 0})
			}
		case runner.EventRunDone:
			if ev.RunResult != nil {
				if ev.RunResult.FailedCount > 0 {
					a.statusBar.setRun(RunFailed, 0, 0, fmt.Sprintf("第 %d 步失败", ev.RunResult.FailedCount))
				} else {
					a.statusBar.setRun(RunCompleted, 0, 0, "")
				}
				a.runPanel.setSummary(ev.RunResult)
			}
			a.refreshHistory()
		}
	}
}

func (a *App) logArtifacts(res *runner.StepResult) {
	if res == nil {
		return
	}
	if res.Screenshot != "" {
		a.runPanel.log("已保存截图：" + res.Screenshot)
	}
	if res.HTMLSnapshot != "" {
		a.runPanel.log("已保存页面 HTML：" + res.HTMLSnapshot)
	}
}

func (a *App) setStepStatus(index int, status runner.Status) {
	if index < 0 || index >= len(a.stepTable.stepsData) {
		return
	}
	if len(a.runStatuses) != len(a.stepTable.stepsData) {
		a.runStatuses = make([]runner.Status, len(a.stepTable.stepsData))
		for i := range a.runStatuses {
			a.runStatuses[i] = runner.StatusPending
		}
	}
	a.runStatuses[index] = status
	statuses := append([]runner.Status(nil), a.runStatuses...)
	fyne.Do(func() {
		a.stepTable.setStatuses(statuses)
	})
}

func (a *App) refreshFlowList() {
	flows, _ := a.flowStore.ListSorted()
	a.flowLibrary.setFlows(flows)
	a.updateEmptyState(len(flows) == 0)
}

func (a *App) updateEmptyState(empty bool) {
	if a.editorArea == nil || a.emptyState == nil {
		return
	}
	fyne.Do(func() {
		if empty {
			a.editorArea.Hide()
			a.emptyState.Show()
		} else {
			a.emptyState.Hide()
			a.editorArea.Show()
		}
	})
}

func (a *App) refreshHistory() {
	if a.historyPanel == nil {
		return
	}
	if a.currentFlow == nil || a.runRepo == nil {
		a.historyPanel.setResults(nil)
		return
	}
	results, err := a.runRepo.ListFiltered(
		a.currentFlow.ID,
		a.historyPanel.environmentFilter(),
		a.historyPanel.statusFilter(),
		20,
	)
	if err != nil {
		a.runPanel.log("读取运行历史失败：" + err.Error())
		return
	}
	a.historyPanel.setResults(results)
}

func (a *App) setCurrentFlow(f *flow.Flow) {
	a.currentFlow = f
	if a.recentStore != nil && f != nil {
		a.recentStore.Touch(f.ID)
	}
	a.flowEditor.loadFlow(f)
	a.stepTable.loadFlow(f)
	a.runStatuses = nil
	a.stepProperty.clear()
	if f != nil {
		a.statusBar.setFlow(f.Name)
	} else {
		a.statusBar.setFlow("")
	}
	a.markClean()
	a.refreshHistory()
}

func (a *App) onFlowSelected(f *flow.Flow) {
	if a.dirty && a.currentFlow != nil && a.currentFlow.ID != f.ID {
		a.promptSaveBefore(func() { a.setCurrentFlow(f) })
		return
	}
	a.setCurrentFlow(f)
}

func (a *App) promptSaveBefore(next func()) {
	dialog.ShowConfirm("未保存的修改",
		fmt.Sprintf("当前流程 [%s] 有未保存的修改，是否保存？", a.currentFlow.Name),
		func(save bool) {
			if save {
				a.saveCurrentFlow()
				next()
			} else {
				a.markClean()
				next()
			}
		}, a.mainWin)
}

func (a *App) onFlowDelete(f *flow.Flow) {
	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除流程 [%s] 吗？", f.Name), func(ok bool) {
		if !ok {
			return
		}
		if err := a.flowStore.Delete(f.ID); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		if a.currentFlow != nil && a.currentFlow.ID == f.ID {
			a.setCurrentFlow(nil)
		}
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) onFlowClone(f *flow.Flow) {
	cf := f.Clone()
	if err := a.flowStore.Save(cf); err != nil {
		dialog.ShowError(err, a.mainWin)
		return
	}
	a.refreshFlowList()
}

func (a *App) onTagFilter(tag string) {
	if strings.TrimSpace(tag) == "" {
		a.refreshFlowList()
		return
	}
	flows, _ := a.flowStore.ListSorted()
	var filtered []*flow.Flow
	for _, f := range flows {
		for _, t := range f.Tags {
			if t == tag {
				filtered = append(filtered, f)
				break
			}
		}
	}
	a.flowLibrary.setFlows(filtered)
}

func (a *App) onStepSelected(s *flow.Step, idx int) {
	if s == nil {
		a.stepProperty.clear()
		return
	}
	if a.stepProperty.stepIndex != idx && a.stepProperty.hasUnappliedChanges() {
		dialog.ShowConfirm("未应用的步骤修改",
			"当前步骤属性有未应用修改，是否先应用到当前步骤？",
			func(apply bool) {
				if apply {
					a.stepProperty.apply()
				}
				a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
			}, a.mainWin)
		return
	}
	a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
}
