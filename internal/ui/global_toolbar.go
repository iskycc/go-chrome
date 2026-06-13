package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

// flowSelectOption pairs a display label with the flow ID it represents.
// Labels are disambiguated when multiple flows share the same name.
type flowSelectOption struct {
	Label string
	ID    string
}

// globalToolbar holds the always-visible flow/environment/run controls.
type globalToolbar struct {
	app            *App
	widget         fyne.CanvasObject
	flowSelect     *widget.Select
	envSelect      *widget.Select
	saveBtn        *widget.Button
	startChromeBtn *widget.Button
	stopChromeBtn  *widget.Button
	runBtn         *widget.Button
	stepBtn        *widget.Button
	stopBtn        *widget.Button
	progress       *widget.ProgressBar
	progressText   *progressLabel
	progressLine   fyne.CanvasObject

	flowOptions []flowSelectOption
	flowByID    map[string]*flow.Flow
}

func newGlobalToolbar(app *App) *globalToolbar {
	t := &globalToolbar{
		app:      app,
		flowByID: map[string]*flow.Flow{},
	}

	t.flowSelect = widget.NewSelect([]string{}, func(label string) {
		for _, opt := range t.flowOptions {
			if opt.Label == label {
				f := t.flowByID[opt.ID]
				if f == nil {
					return
				}
				app.onFlowSelected(f)
				return
			}
		}
	})
	t.flowSelect.PlaceHolder = "选择流程"

	t.envSelect = widget.NewSelect([]string{"默认环境"}, func(name string) {
		if app.envRepo == nil || name == "" {
			return
		}
		env, err := app.envRepo.GetByName(name)
		if err != nil {
			if app.runPanel != nil {
				app.runPanel.log("切换环境失败：" + err.Error())
			}
			return
		}
		if err := app.envRepo.SetActive(env.ID); err != nil {
			if app.runPanel != nil {
				app.runPanel.log("保存当前环境失败：" + err.Error())
			}
			return
		}
	})
	t.envSelect.SetSelected("默认环境")

	t.saveBtn = widget.NewButtonWithIcon("保存", theme.DocumentSaveIcon(), func() {
		app.saveCurrentFlow()
	})
	t.saveBtn.Importance = widget.MediumImportance

	t.startChromeBtn = widget.NewButtonWithIcon("启动", theme.ComputerIcon(), func() {
		go app.startBrowser()
	})

	t.stopChromeBtn = widget.NewButtonWithIcon("关闭", theme.CancelIcon(), func() {
		go app.closeManagedChrome()
	})
	t.stopChromeBtn.Importance = widget.DangerImportance
	t.stopChromeBtn.Disable()

	t.runBtn = widget.NewButtonWithIcon("运行", theme.MediaPlayIcon(), func() {
		go app.runCurrentFlow()
	})
	t.runBtn.Importance = widget.HighImportance

	t.stepBtn = widget.NewButtonWithIcon("单步", theme.MediaReplayIcon(), func() {
		go app.onStepButton()
	})

	t.stopBtn = widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		app.stopCurrentRun()
	})
	t.stopBtn.Disable()
	t.stopBtn.Importance = widget.DangerImportance

	t.progress = widget.NewProgressBar()
	t.progress.Min = 0
	t.progress.Max = 1
	t.progressText = newProgressLabel(160, 360)

	// Operation band: single line, single-height controls only.
	flowBox := newInlineToolbarGroup("流程",
		container.NewGridWrap(fyne.NewSize(220, t.flowSelect.MinSize().Height), t.flowSelect),
		t.saveBtn,
	)
	browserBox := newInlineToolbarGroup("浏览器",
		t.startChromeBtn,
		t.stopChromeBtn,
	)
	runBox := newInlineToolbarGroup("执行",
		t.runBtn,
		t.stepBtn,
		t.stopBtn,
	)
	envBox := newInlineToolbarGroup("环境",
		container.NewGridWrap(fyne.NewSize(160, t.envSelect.MinSize().Height), t.envSelect),
	)

	operationBand := container.NewHBox(
		flowBox,
		browserBox,
		runBox,
		envBox,
	)

	// Progress line: a separate, lightweight status line below the buttons.
	progressBarBox := container.NewGridWrap(fyne.NewSize(360, t.progress.MinSize().Height), t.progress)
	t.progressLine = container.NewHBox(
		t.progressText.box,
		progressBarBox,
	)
	t.progressLine.Hide()

	t.widget = container.NewVBox(operationBand, t.progressLine)
	return t
}

// refreshFlows rebuilds the flow dropdown from the current flow library.
func (t *globalToolbar) refreshFlows(flows []*flow.Flow) {
	t.flowByID = make(map[string]*flow.Flow, len(flows))
	nameCount := make(map[string]int, len(flows))
	for _, f := range flows {
		nameCount[f.Name]++
	}

	t.flowOptions = make([]flowSelectOption, 0, len(flows))
	labels := make([]string, 0, len(flows))
	var selected string
	for _, f := range flows {
		t.flowByID[f.ID] = f

		label := f.Name
		if nameCount[f.Name] > 1 {
			shortID := f.ID
			if len(shortID) > 6 {
				shortID = shortID[:6]
			}
			label = fmt.Sprintf("%s · %s", f.Name, shortID)
		}

		opt := flowSelectOption{Label: label, ID: f.ID}
		t.flowOptions = append(t.flowOptions, opt)
		labels = append(labels, label)

		if t.app.currentFlow != nil && f.ID == t.app.currentFlow.ID {
			selected = label
		}
	}

	fyne.Do(func() {
		t.flowSelect.Options = labels
		if selected != "" {
			t.flowSelect.SetSelected(selected)
		} else if len(labels) > 0 {
			t.flowSelect.SetSelected(labels[0])
		} else {
			t.flowSelect.ClearSelected()
		}
	})
}

// refreshEnvironments rebuilds the environment dropdown.
func (t *globalToolbar) refreshEnvironments() {
	if t.app.envRepo == nil {
		return
	}
	envs, err := t.app.envRepo.List()
	if err != nil {
		return
	}
	var names []string
	var active string
	for _, e := range envs {
		names = append(names, e.Name)
		if e.IsActive {
			active = e.Name
		}
	}
	if len(names) == 0 {
		names = []string{"默认环境"}
		active = "默认环境"
	}
	fyne.Do(func() {
		t.envSelect.Options = names
		t.envSelect.SetSelected(active)
	})
}

// setProgress updates the lightweight progress display.
func (t *globalToolbar) setProgress(current, total int, stepName string) {
	fyne.Do(func() {
		if total <= 0 {
			t.progress.SetValue(0)
			t.progressText.set(0, 0, "")
			if t.progressLine != nil {
				t.progressLine.Hide()
			}
			return
		}

		if t.progressLine != nil {
			t.progressLine.Show()
		}
		t.progress.Max = float64(total)
		t.progress.SetValue(float64(current))
		t.progressText.set(current, total, stepName)
	})
}

// setRunning updates button availability when a run starts/stops.
func (t *globalToolbar) setRunning(running bool) {
	fyne.Do(func() {
		if running {
			t.runBtn.Disable()
			t.stepBtn.Disable()
			t.stopBtn.Enable()
		} else {
			t.runBtn.Enable()
			t.stepBtn.Enable()
			t.stopBtn.Disable()
			t.stepBtn.SetText("单步")
		}
	})
}

// setStepButtonText updates the single-step button label.
func (t *globalToolbar) setStepButtonText(label string) {
	fyne.Do(func() {
		t.stepBtn.SetText(label)
	})
}

// setChromeManaged updates the browser button availability.
func (t *globalToolbar) setChromeManaged(managed bool) {
	fyne.Do(func() {
		if managed {
			t.startChromeBtn.Disable()
			t.stopChromeBtn.Enable()
		} else {
			t.startChromeBtn.Enable()
			t.stopChromeBtn.Disable()
		}
	})
}
