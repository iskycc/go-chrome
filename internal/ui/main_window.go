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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/assets"
	appdirs "go-chrome/internal/app"
	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/db"
	"go-chrome/internal/flow"
	"go-chrome/internal/logx"
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

	statusBar     *statusBar
	flowLibrary   *flowLibraryPanel
	flowEditor    *flowEditorPanel
	stepTable     *stepTablePanel
	stepProperty  *stepPropertyPanel
	runPanel      *runPanel
	historyPanel  *historyPanel
	settingsPanel *settingsPanel
	envPanel      *envPanel
	infoPanel     *infoPanel
	moduleTabs    *container.AppTabs
	workspace     *fyne.Container
	editorArea    fyne.CanvasObject
	emptyState    fyne.CanvasObject
	currentFlow   *flow.Flow
	runStatuses   []runner.Status

	dirty        bool
	chromeTicker *time.Ticker
	chromeDone   chan struct{}

	globalToolbar *globalToolbar
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
		// Stop any in-flight runs before tearing down Chrome so CDP
		// sessions close cleanly.
		if a.stepRunner != nil {
			a.stepRunner.Stop()
		}
		if a.runner != nil && a.runner.IsRunning() {
			a.runner.Stop()
		}
		if a.cfg.App.CloseManagedChromeOnExit && a.browserMgr != nil {
			if err := a.browserMgr.Stop(); err != nil {
				logx.Warnf("close managed Chrome on exit: %v", err)
			}
		}
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
	a.globalToolbar = newGlobalToolbar(a)
	a.flowLibrary = newFlowLibraryPanel(a)
	a.flowEditor = newFlowEditorPanel(a, onDirty)
	a.stepTable = newStepTablePanel(a, onDirty)
	a.stepProperty = newStepPropertyPanel(a, onDirty)
	a.runPanel = newRunPanel(a)
	a.historyPanel = newHistoryPanel(a)
	a.settingsPanel = newSettingsPanel(a)
	a.envPanel = newEnvPanel(a)
	a.infoPanel = newInfoPanel(a)

	a.editorArea = a.flowEditor.widget
	a.emptyState = a.buildEmptyState()
	a.emptyState.Hide()
	flowDetail := container.NewStack(a.editorArea, a.emptyState)

	flowModule := container.NewHSplit(a.flowLibrary.widget, flowDetail)
	flowModule.SetOffset(0.28)

	stepModule := container.NewHSplit(a.stepTable.widget, a.stepProperty.widget)
	stepModule.SetOffset(0.62)

	a.workspace = container.NewStack(flowModule)
	a.moduleTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("流程", theme.DocumentIcon(), flowModule),
		container.NewTabItemWithIcon("步骤", theme.ListIcon(), stepModule),
		container.NewTabItemWithIcon("环境配置", theme.SettingsIcon(), a.envPanel.widget),
		container.NewTabItemWithIcon("历史", theme.HistoryIcon(), a.historyPanel.widget),
		container.NewTabItemWithIcon("设置", theme.SettingsIcon(), a.settingsPanel.widget),
		container.NewTabItemWithIcon("运行详情", theme.MediaPlayIcon(), a.runPanel.widget),
		container.NewTabItemWithIcon("信息", theme.InfoIcon(), a.infoPanel.widget),
	)
	a.moduleTabs.SetTabLocation(container.TabLocationTop)
	a.moduleTabs.OnChanged = func(item *container.TabItem) {
		if item.Content == a.infoPanel.widget {
			a.infoPanel.refresh()
		}
	}

	// 使用紧凑的垂直布局，减小状态栏与工具栏之间的空隙。
	topArea := container.New(
		layout.NewCustomPaddedVBoxLayout(4),
		a.statusBar.widget,
		a.globalToolbar.widget,
	)

	content := container.NewBorder(
		topArea,
		nil,
		nil,
		nil,
		a.moduleTabs,
	)
	a.mainWin.SetContent(content)
	a.refreshFlowList()
	a.refreshEnvironmentSelectors()
	a.historyPanel.refreshFilters()
	a.restoreLastFlowSelection()
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
					st := a.browserMgr.Status()
					a.statusBar.setChrome(st)
					managed := st == browser.ChromeRunning || st == browser.ChromeStarting
					if a.runPanel != nil {
						a.runPanel.setChromeManaged(managed)
					}
					if a.globalToolbar != nil {
						a.globalToolbar.setChromeManaged(managed)
					}
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

	form := container.NewVBox(
		widget.NewLabel("流程名称"),
		nameEntry,
	)

	d := dialog.NewCustomConfirm("新建流程", "创建", "取消", form, func(ok bool) {
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
	d.Resize(fyne.NewSize(440, 160))
	d.Show()
}

func (a *App) saveCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if err := flow.Validate(a.currentFlow); err != nil {
		a.statusBar.setSave(SaveFailed)
		dialog.ShowError(fmt.Errorf("保存前校验失败: %w", err), a.mainWin)
		return
	}
	a.statusBar.setSave(SaveSaving)
	if err := a.saveCurrentFlowInternal(); err != nil {
		a.statusBar.setSave(SaveFailed)
		dialog.ShowError(err, a.mainWin)
		return
	}
	a.statusBar.setSave(SaveSuccess)
	a.markClean()
}

// saveCurrentFlowInternal writes the current flow to storage and returns
// the underlying error so callers can decide whether to continue (e.g. when
// used by the unsaved-changes prompt, the caller should NOT switch flows
// if this returns an error).
func (a *App) saveCurrentFlowInternal() error {
	return a.flowStore.Save(a.currentFlow)
}

func (a *App) importFlow() {
	if a.dirty && a.currentFlow != nil {
		a.promptSaveBefore(func() { a.doImportFlow() })
		return
	}
	a.doImportFlow()
}

func (a *App) doImportFlow() {
	d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
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
	d.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	d.Resize(fyne.NewSize(680, 480))
	d.Show()
}

func (a *App) exportFlow() {
	if a.currentFlow == nil {
		return
	}
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()
		if err := a.flowStore.Export(a.currentFlow.ID, writer.URI().Path()); err != nil {
			dialog.ShowError(err, a.mainWin)
		}
	}, a.mainWin)
	d.SetFileName(safeFileName(a.currentFlow.Name))
	resizeFileDialog(d)
	d.Show()
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

// closeManagedChrome terminates the Chrome instance that this app started.
// It only affects the process tracked by the browser Manager; user-launched
// Chrome windows are not touched. A confirmation dialog is shown to avoid
// accidentally killing a running browser session.
func (a *App) closeManagedChrome() {
	st := a.browserMgr.Status()
	if st != browser.ChromeRunning && st != browser.ChromeStarting {
		a.runPanel.log("本程序未启动任何 Chrome，无需关闭")
		return
	}
	dialog.ShowConfirm("关闭 Chrome", "确认关闭本程序启动的 Chrome？\n（不会影响用户手动打开的 Chrome）", func(ok bool) {
		if !ok {
			return
		}
		if a.runner != nil && a.runner.IsRunning() {
			a.runner.Stop()
		}
		if a.stepRunner != nil {
			a.stepRunner.Stop()
		}
		if err := a.browserMgr.Stop(); err != nil {
			a.runPanel.log("关闭 Chrome 失败：" + err.Error())
			fyne.Do(func() { dialog.ShowError(err, a.mainWin) })
			return
		}
		a.runPanel.log("已关闭本程序启动的 Chrome")
	}, a.mainWin)
}

// stopCurrentRun stops whatever is currently running: either the full
// Runner or the StepRunner. After stopping, the UI is reset to a clean
// state and the user can start a new run.
func (a *App) stopCurrentRun() {
	if a.runner != nil && a.runner.IsRunning() {
		a.runner.Stop()
		a.runPanel.log("已停止完整流程运行")
		return
	}
	if a.stepRunner != nil && !a.stepRunner.IsFinished() {
		a.stepRunner.Stop()
		a.runPanel.log("已停止单步执行")
		a.runPanel.setRunning(false)
		if a.globalToolbar != nil {
			a.globalToolbar.setStepButtonText("单步执行")
		}
		a.stepRunner = nil
		return
	}
	a.runPanel.log("当前没有正在运行的任务")
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
	startStep := a.selectedStartStep()
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
	a.runPanel.setRunning(true)
	if a.globalToolbar != nil {
		a.globalToolbar.setRunning(true)
	}
	a.runStatuses = make([]runner.Status, len(a.currentFlow.Steps))
	for i := range a.runStatuses {
		a.runStatuses[i] = runner.StatusPending
	}
	a.stepTable.setStatuses(a.runStatuses)
	go func() {
		res := a.runner.RunFlow(a.currentFlow, runner.RunOptions{
			StartStep:     startStep,
			EnvironmentID: envID,
			EnvProvider:   envProvider,
		})
		a.runPanel.setSummary(res)
	}()
}

// selectedStartStep returns the step index selected in the step table,
// or 0 if no step is selected. This lets the user run a flow from a
// specific step by selecting it before clicking Run.
func (a *App) selectedStartStep() int {
	if a.stepTable == nil {
		return 0
	}
	idx := a.stepTable.selectedIndex()
	if idx < 0 || a.currentFlow == nil || idx >= len(a.currentFlow.Steps) {
		return 0
	}
	return idx
}

func (a *App) checkEnvVars() []string {
	if a.currentFlow == nil {
		return nil
	}
	_, provider, err := a.currentEnvProvider()
	if err != nil {
		return []string{"无法获取运行环境: " + err.Error()}
	}
	return runner.MissingEnvVars(a.currentFlow, a.selectedStartStep(), provider)
}

func (a *App) currentEnvProvider() (string, template.EnvProvider, error) {
	if a.envRepo == nil {
		return "", nil, fmt.Errorf("环境仓库未初始化")
	}
	selectedName := ""
	if a.globalToolbar != nil && a.globalToolbar.envSelect != nil {
		selectedName = a.globalToolbar.envSelect.Selected
	}
	if selectedName == "" {
		selectedName = "默认环境"
	}
	env, err := a.envRepo.GetByName(selectedName)
	if err != nil {
		return "", nil, err
	}
	return env.ID, a.envRepo.EnvProvider(env.ID), nil
}

// refreshEnvironmentSelectors updates both the global toolbar and run panel
// environment dropdowns, if they exist.
func (a *App) refreshEnvironmentSelectors() {
	if a.globalToolbar != nil {
		a.globalToolbar.refreshEnvironments()
	}
	if a.runPanel != nil {
		a.runPanel.refreshEnvironments()
	}
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
	if a.globalToolbar != nil {
		a.globalToolbar.setRunning(true)
	}
	a.runStatuses = make([]runner.Status, len(a.currentFlow.Steps))
	for i := range a.runStatuses {
		a.runStatuses[i] = runner.StatusPending
	}
	a.stepTable.setStatuses(a.runStatuses)
	if a.globalToolbar != nil {
		a.globalToolbar.setStepButtonText("下一步")
	}
	a.nextStep()
}

func (a *App) nextStep() {
	if a.stepRunner == nil {
		return
	}
	res, finished, err := a.stepRunner.Next()
	if err != nil {
		a.runPanel.log("单步执行错误：" + err.Error())
		if a.globalToolbar != nil {
			a.globalToolbar.setStepButtonText("单步执行")
			a.globalToolbar.setRunning(false)
		}
		a.runPanel.setRunning(false)
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
		if a.globalToolbar != nil {
			a.globalToolbar.setStepButtonText("单步执行")
			a.globalToolbar.setRunning(false)
		}
		a.runPanel.setRunning(false)
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
			if a.globalToolbar != nil {
				a.globalToolbar.setProgress(ev.StepIndex+1, totalSteps, stepName)
			}
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
			a.runPanel.setRunning(false)
			if a.globalToolbar != nil {
				a.globalToolbar.setRunning(false)
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
	if a.globalToolbar != nil {
		a.globalToolbar.refreshFlows(flows)
	}
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

// restoreLastFlowSelection picks the flow to open on startup. Order:
//  1. Most recent flow ID from the recent store, if it still exists.
//  2. The first flow in the library, if any.
//  3. Empty selection (shows the empty state).
//
// The flow is selected via flowLibrary.selectFlow, which goes through the
// widget.List.OnSelected handler, so the full flow (including steps) is
// reloaded from storage.
func (a *App) restoreLastFlowSelection() {
	if a.flowLibrary == nil {
		return
	}
	if len(a.flowLibrary.flows) == 0 {
		return
	}
	if a.recentStore != nil {
		for _, id := range a.recentStore.FlowIDs {
			if a.flowLibrary.selectFlow(id) {
				return
			}
		}
	}
	a.flowLibrary.selectFlow(a.flowLibrary.flows[0].ID)
}

func (a *App) onFlowSelected(f *flow.Flow) {
	if a.dirty && a.currentFlow != nil && a.currentFlow.ID != f.ID {
		a.promptSaveBefore(func() { a.setCurrentFlow(f) })
		return
	}
	a.setCurrentFlow(f)
}

// promptSaveBefore shows a 3-way dialog asking the user what to do with
// unsaved changes. The 3 options are:
//   - 保存并继续: save the current flow; only call next() if save succeeds
//   - 放弃修改:    mark the flow as clean and call next()
//   - 取消:        do nothing
func (a *App) promptSaveBefore(next func()) {
	name := ""
	if a.currentFlow != nil {
		name = truncateForDialog(a.currentFlow.Name, 80)
	}

	body := widget.NewLabel(fmt.Sprintf(
		"当前流程 [%s] 有未保存的修改，要如何处理？",
		name,
	))
	body.Alignment = fyne.TextAlignLeading
	body.Wrapping = fyne.TextWrapWord

	saveBtn := widget.NewButtonWithIcon("保存并继续", theme.DocumentSaveIcon(), nil)
	saveBtn.Importance = widget.HighImportance
	discardBtn := widget.NewButtonWithIcon("放弃修改", theme.DeleteIcon(), nil)
	discardBtn.Importance = widget.DangerImportance
	cancelBtn := widget.NewButton("取消", nil)

	btnRow := container.NewGridWithColumns(3, saveBtn, discardBtn, cancelBtn)
	content := container.NewVBox(body, btnRow)

	d := dialog.NewCustomWithoutButtons("未保存的修改", content, a.mainWin)
	d.Resize(fyne.NewSize(560, 220))
	d.SetButtons([]fyne.CanvasObject{saveBtn, discardBtn, cancelBtn})

	saveBtn.OnTapped = func() {
		d.Hide()
		if a.currentFlow == nil {
			next()
			return
		}
		if err := flow.Validate(a.currentFlow); err != nil {
			dialog.ShowError(fmt.Errorf("保存前校验失败: %w", err), a.mainWin)
			return
		}
		if err := a.saveCurrentFlowInternal(); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.statusBar.setSave(SaveSuccess)
		a.markClean()
		next()
	}
	discardBtn.OnTapped = func() {
		d.Hide()
		a.markClean()
		next()
	}
	cancelBtn.OnTapped = func() {
		d.Hide()
	}

	d.Show()
}

func (a *App) onFlowDelete(f *flow.Flow) {
	msg := fmt.Sprintf("确定删除流程 [%s] 吗？", truncateForDialog(f.Name, 80))
	showWrappedConfirm("确认删除", msg, "删除", "取消", fyne.NewSize(520, 180), func(ok bool) {
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
		a.promptStepChanges(s, idx)
		return
	}
	a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
}

// promptStepChanges shows a 3-way dialog for unapplied step property changes.
func (a *App) promptStepChanges(s *flow.Step, idx int) {
	body := widget.NewLabel("当前步骤属性有未应用修改，要如何处理？")
	body.Alignment = fyne.TextAlignLeading
	body.Wrapping = fyne.TextWrapWord

	applyBtn := widget.NewButtonWithIcon("应用并切换", theme.ConfirmIcon(), nil)
	applyBtn.Importance = widget.HighImportance
	discardBtn := widget.NewButtonWithIcon("放弃修改", theme.DeleteIcon(), nil)
	discardBtn.Importance = widget.DangerImportance
	cancelBtn := widget.NewButton("取消", nil)

	content := container.NewVBox(body, container.NewHBox(applyBtn, discardBtn, cancelBtn))
	d := dialog.NewCustomWithoutButtons("未应用的步骤修改", content, a.mainWin)
	d.Resize(fyne.NewSize(520, 200))
	d.SetButtons([]fyne.CanvasObject{applyBtn, discardBtn, cancelBtn})

	applyBtn.OnTapped = func() {
		d.Hide()
		a.stepProperty.apply()
		a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
	}
	discardBtn.OnTapped = func() {
		d.Hide()
		a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
	}
	cancelBtn.OnTapped = func() {
		d.Hide()
	}

	d.Show()
}
