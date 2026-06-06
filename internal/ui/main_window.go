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
	"go-chrome/internal/flow"
	"go-chrome/internal/logx"
	"go-chrome/internal/runner"
)

type App struct {
	fyneApp     fyne.App
	mainWin     fyne.Window
	cfg         *config.Config
	dirs        *appdirs.Directories
	flowStore   *flow.Store
	recentStore *flow.RecentStore
	browserMgr  *browser.Manager
	runner      *runner.Runner
	stepRunner  *runner.StepRunner
	history     *runner.HistoryStore

	statusBar    *statusBar
	flowLibrary  *flowLibraryPanel
	flowEditor   *flowEditorPanel
	stepTable    *stepTablePanel
	stepProperty *stepPropertyPanel
	runPanel     *runPanel
	currentFlow  *flow.Flow

	dirty        bool
	chromeTicker *time.Ticker
	chromeDone   chan struct{}

	stepBtn *widget.Button
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
		if a.recentStore != nil {
			_ = a.recentStore.Save()
		}
		if a.chromeDone != nil {
			close(a.chromeDone)
		}
	})

	a.initDeps()
	a.buildUI()
	a.firstRunCheck()
	a.startChromeTicker()
	a.mainWin.ShowAndRun()
}

func (a *App) initDeps() {
	var err error
	a.flowStore, err = flow.NewStore(a.dirs.FlowsDir)
	if err != nil {
		logx.Errorf("flow store: %v", err)
	}
	a.recentStore, _ = flow.NewRecentStore(filepath.Join(a.dirs.DataDir, "recent-flows.json"))
	a.browserMgr = browser.NewManager(&a.cfg.Chrome)
	a.browserMgr.LoadManifest() // best effort

	historyDir := filepath.Join(a.dirs.DataDir, "run-history")
	a.history, _ = runner.NewHistoryStore(historyDir)
	if a.history != nil {
		_ = a.history.Cleanup(a.cfg.App.LogRetentionDays)
	}

	a.runner = runner.NewRunner(&a.cfg.Runner, a.browserMgr, a.history)
	go a.handleRunnerEvents()
}

func (a *App) buildUI() {
	onDirty := func() { a.markDirty() }

	a.statusBar = newStatusBar(a)
	a.flowLibrary = newFlowLibraryPanel(a)
	a.flowEditor = newFlowEditorPanel(a, onDirty)
	a.stepTable = newStepTablePanel(a, onDirty)
	a.stepProperty = newStepPropertyPanel(a, onDirty)
	a.runPanel = newRunPanel(a)

	centerTop := container.NewBorder(a.flowEditor.widget, nil, nil, nil, a.stepTable.widget)
	center := container.NewHSplit(centerTop, a.stepProperty.widget)
	center.SetOffset(0.55)

	mainSplit := container.NewVSplit(
		container.NewBorder(a.statusBar.widget, nil, a.flowLibrary.widget, nil, center),
		a.runPanel.widget,
	)
	mainSplit.SetOffset(0.72)

	a.mainWin.SetContent(mainSplit)
	a.refreshFlowList()
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
	a.runPanel.reset()
	a.stepTable.clearStatuses()
	go func() {
		res := a.runner.RunFlow(a.currentFlow, 0)
		a.runPanel.setSummary(res)
	}()
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
	a.stepRunner = runner.NewStepRunner(&a.cfg.Runner, a.browserMgr, a.history)
	if err := a.stepRunner.Init(a.currentFlow); err != nil {
		dialog.ShowError(err, a.mainWin)
		a.stepRunner = nil
		return
	}
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
		logMsg := fmt.Sprintf("步骤 %d %s: %s", a.stepRunner.CurrentIndex(), res.StepName, res.Status)
		if res.Error != "" {
			logMsg += " - " + res.Error
		}
		a.runPanel.log(logMsg)
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
			a.statusBar.setRun(RunRunning, ev.StepIndex+1, totalSteps, "")
		case runner.EventStepDone:
			statuses := make([]runner.Status, len(a.stepTable.stepsData))
			for i := range statuses {
				statuses[i] = runner.StatusPending
			}
			if ev.StepIndex >= 0 && ev.StepIndex < len(statuses) {
				statuses[ev.StepIndex] = ev.Result.Status
			}
			a.stepTable.setStatuses(statuses)
			if ev.Result.Status == runner.StatusFailed {
				a.runPanel.setArtifacts(ev.Result.Screenshot, ev.Result.HTMLSnapshot)
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
			a.stepTable.clearStatuses()
			a.refreshHistory()
		}
	}
}

func (a *App) refreshFlowList() {
	flows, _ := a.flowStore.ListSorted()
	a.flowLibrary.setFlows(flows)
}

func (a *App) refreshHistory() {
	if a.currentFlow == nil {
		return
	}
}

func (a *App) setCurrentFlow(f *flow.Flow) {
	a.currentFlow = f
	if a.recentStore != nil && f != nil {
		a.recentStore.Touch(f.ID)
	}
	a.flowEditor.loadFlow(f)
	a.stepTable.loadFlow(f)
	a.stepProperty.clear()
	if f != nil {
		a.statusBar.setFlow(f.Name)
	} else {
		a.statusBar.setFlow("")
	}
	a.markClean()
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
	a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
}
