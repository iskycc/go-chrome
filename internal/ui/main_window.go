package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appdirs "go-chrome/internal/app"
	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/logx"
	"go-chrome/internal/runner"
)

// App is the main GUI application.
type App struct {
	fyneApp    fyne.App
	mainWin    fyne.Window
	cfg        *config.Config
	dirs       *appdirs.Directories
	flowStore  *flow.Store
	browserMgr *browser.Manager
	runner     *runner.Runner

	flowList    *flowListPanel
	stepEditor  *stepEditorPanel
	runPanel    *runPanel
	currentFlow *flow.Flow
}

// New creates the GUI application.
func New(cfg *config.Config, dirs *appdirs.Directories) *App {
	return &App{cfg: cfg, dirs: dirs}
}

// Run starts the GUI event loop.
func (a *App) Run() {
	a.fyneApp = app.NewWithID("com.go-chrome.app")
	a.fyneApp.Settings().SetTheme(newAppTheme())
	a.mainWin = a.fyneApp.NewWindow("Go Chrome CDP Automation")
	a.mainWin.Resize(fyne.NewSize(1400, 900))

	a.initDeps()
	a.buildUI()
	a.mainWin.ShowAndRun()
}

func (a *App) initDeps() {
	var err error
	a.flowStore, err = flow.NewStore(a.dirs.FlowsDir)
	if err != nil {
		logx.Errorf("flow store: %v", err)
	}
	a.browserMgr = browser.NewManager(&a.cfg.Chrome)
	a.browserMgr.LoadManifest() // best effort
	a.runner = runner.NewRunner(&a.cfg.Runner, a.browserMgr)
	go a.handleRunnerEvents()
}

func (a *App) buildUI() {
	a.flowList = newFlowListPanel(a)
	a.stepEditor = newStepEditorPanel(a)
	a.runPanel = newRunPanel(a)

	// Main content: left flows, center steps, right properties
	content := container.NewBorder(
		a.buildToolbar(),
		a.runPanel.widget,
		a.flowList.widget,
		a.stepEditor.propertiesWidget,
		a.stepEditor.stepsWidget,
	)

	a.mainWin.SetContent(content)
	a.refreshFlowList()
}

func (a *App) buildToolbar() fyne.CanvasObject {
	newBtn := widget.NewButton("New Flow", func() {
		a.createNewFlow()
	})
	saveBtn := widget.NewButton("Save", func() {
		a.saveCurrentFlow()
	})
	importBtn := widget.NewButton("Import", func() {
		a.importFlow()
	})
	exportBtn := widget.NewButton("Export", func() {
		a.exportFlow()
	})
	startBrowserBtn := widget.NewButton("Start Chrome", func() {
		go a.startBrowser()
	})
	runBtn := widget.NewButton("Run Flow", func() {
		a.runCurrentFlow()
	})
	stopBtn := widget.NewButton("Stop", func() {
		a.runner.Stop()
	})
	stepBtn := widget.NewButton("Step", func() {
		a.stepCurrentFlow()
	})
	return container.NewHBox(newBtn, saveBtn, importBtn, exportBtn, startBrowserBtn, runBtn, stopBtn, stepBtn)
}

func (a *App) createNewFlow() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Flow name")
	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}
	dialog.ShowForm("New Flow", "Create", "Cancel", items, func(ok bool) {
		if !ok || nameEntry.Text == "" {
			return
		}
		f := flow.NewFlow(nameEntry.Text)
		if err := a.flowStore.Save(f); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.currentFlow = f
		a.stepEditor.loadFlow(f)
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) saveCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("Info", "No flow selected", a.mainWin)
		return
	}
	if err := a.flowStore.Save(a.currentFlow); err != nil {
		dialog.ShowError(err, a.mainWin)
		return
	}
	dialog.ShowInformation("Saved", "Flow saved successfully", a.mainWin)
}

func (a *App) importFlow() {
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
		a.currentFlow = f
		a.stepEditor.loadFlow(f)
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
		path := writer.URI().Path()
		if err := a.flowStore.Export(a.currentFlow.ID, path); err != nil {
			dialog.ShowError(err, a.mainWin)
		}
	}, a.mainWin)
}

func (a *App) startBrowser() {
	if !a.browserMgr.IsInstalled() {
		a.runPanel.log("Chrome not installed, downloading...")
		if err := a.browserMgr.Install(func(d, t int64) {
			if t > 0 {
				a.runPanel.log(fmt.Sprintf("Download %d%%", d*100/t))
			}
		}); err != nil {
			a.runPanel.log("Download failed: " + err.Error())
			fyne.Do(func() {
				dialog.ShowError(err, a.mainWin)
			})
			return
		}
	}
	port, err := a.browserMgr.Start()
	if err != nil {
		a.runPanel.log("Start failed: " + err.Error())
		fyne.Do(func() {
			dialog.ShowError(err, a.mainWin)
		})
		return
	}
	a.runPanel.log(fmt.Sprintf("Chrome started on port %d", port))
}

func (a *App) runCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("Info", "No flow selected", a.mainWin)
		return
	}
	if a.runner.IsRunning() {
		dialog.ShowInformation("Info", "Already running", a.mainWin)
		return
	}
	go func() {
		res := a.runner.RunFlow(a.currentFlow, 0)
		a.runPanel.log(fmt.Sprintf("Run finished: %s (%d success, %d failed)", res.Status, res.SuccessCount, res.FailedCount))
	}()
}

func (a *App) stepCurrentFlow() {
	if a.currentFlow == nil {
		return
	}
	// Simplified: run from current selected step index
	idx := a.stepEditor.selectedIndex()
	if idx < 0 {
		idx = 0
	}
	go func() {
		res := a.runner.RunFlow(a.currentFlow, idx)
		a.runPanel.log(fmt.Sprintf("Step run finished: %s", res.Status))
	}()
}

func (a *App) handleRunnerEvents() {
	for ev := range a.runner.Events() {
		switch ev.Type {
		case runner.EventLog:
			a.runPanel.log(ev.LogMessage)
		case runner.EventStepDone:
			a.stepEditor.updateStepStatus(ev.StepIndex, ev.Result.Status)
		case runner.EventRunDone:
			a.stepEditor.clearStatuses()
		}
	}
}

func (a *App) refreshFlowList() {
	flows, _ := a.flowStore.ListSorted()
	a.flowList.setFlows(flows)
}

func (a *App) onFlowSelected(f *flow.Flow) {
	a.currentFlow = f
	a.stepEditor.loadFlow(f)
}

func (a *App) onFlowDelete(f *flow.Flow) {
	if err := a.flowStore.Delete(f.ID); err != nil {
		dialog.ShowError(err, a.mainWin)
		return
	}
	if a.currentFlow != nil && a.currentFlow.ID == f.ID {
		a.currentFlow = nil
		a.stepEditor.loadFlow(nil)
	}
	a.refreshFlowList()
}

func (a *App) onFlowClone(f *flow.Flow) {
	cf := f.Clone()
	if err := a.flowStore.Save(cf); err != nil {
		dialog.ShowError(err, a.mainWin)
		return
	}
	a.refreshFlowList()
}
