package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

// globalToolbar holds the always-visible flow/environment/run controls.
type globalToolbar struct {
	app            *App
	widget         fyne.CanvasObject
	flowSelect     *widget.Select
	envSelect      *widget.Select
	saveBtn        *widget.Button
	startChromeBtn *widget.Button
	runBtn         *widget.Button
	stepBtn        *widget.Button
	stopBtn        *widget.Button
	progress       *widget.ProgressBar
	progressText   *widget.Label

	flowByName map[string]*flow.Flow
}

func newGlobalToolbar(app *App) *globalToolbar {
	t := &globalToolbar{app: app, flowByName: map[string]*flow.Flow{}}

	t.flowSelect = widget.NewSelect([]string{}, func(name string) {
		f := t.flowByName[name]
		if f == nil {
			return
		}
		app.onFlowSelected(f)
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

	t.startChromeBtn = widget.NewButtonWithIcon("启动浏览器", theme.ComputerIcon(), func() {
		go app.startBrowser()
	})

	t.runBtn = widget.NewButtonWithIcon("运行", theme.MediaPlayIcon(), func() {
		go app.runCurrentFlow()
	})
	t.runBtn.Importance = widget.HighImportance

	t.stepBtn = widget.NewButtonWithIcon("单步执行", theme.MediaReplayIcon(), func() {
		go app.onStepButton()
	})

	t.stopBtn = widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		app.stopCurrentRun()
	})
	t.stopBtn.Hide()
	t.stopBtn.Importance = widget.DangerImportance

	t.progress = widget.NewProgressBar()
	t.progress.Min = 0
	t.progress.Max = 1
	t.progressText = widget.NewLabel("就绪")

	progressBox := container.NewBorder(nil, nil, t.progressText, nil, t.progress)

	left := container.NewHBox(
		widget.NewLabelWithStyle("流程", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.flowSelect,
		t.saveBtn,
	)
	center := container.NewHBox(
		t.startChromeBtn,
		t.runBtn,
		t.stepBtn,
		t.stopBtn,
	)
	right := container.NewHBox(
		widget.NewLabelWithStyle("环境", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.envSelect,
	)

	t.widget = container.NewBorder(nil, progressBox, left, right, center)
	return t
}

// refreshFlows rebuilds the flow dropdown from the current flow library.
func (t *globalToolbar) refreshFlows(flows []*flow.Flow) {
	t.flowByName = make(map[string]*flow.Flow, len(flows))
	names := make([]string, 0, len(flows))
	var selected string
	for _, f := range flows {
		names = append(names, f.Name)
		t.flowByName[f.Name] = f
		if t.app.currentFlow != nil && f.ID == t.app.currentFlow.ID {
			selected = f.Name
		}
	}
	fyne.Do(func() {
		t.flowSelect.Options = names
		if selected != "" {
			t.flowSelect.SetSelected(selected)
		} else if len(names) > 0 {
			t.flowSelect.SetSelected(names[0])
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
		if total > 0 {
			t.progress.Max = float64(total)
			t.progress.SetValue(float64(current))
			t.progressText.SetText(fmt.Sprintf("第 %d/%d 步 · %s", current, total, stepName))
		} else {
			t.progress.SetValue(0)
			t.progressText.SetText("就绪")
		}
	})
}

// setRunning updates button visibility when a run starts/stops.
func (t *globalToolbar) setRunning(running bool) {
	fyne.Do(func() {
		if running {
			t.runBtn.Disable()
			t.stepBtn.Disable()
			t.stopBtn.Show()
		} else {
			t.runBtn.Enable()
			t.stepBtn.Enable()
			t.stopBtn.Hide()
			t.stepBtn.SetText("单步执行")
		}
	})
}

// setStepButtonText updates the single-step button label.
func (t *globalToolbar) setStepButtonText(label string) {
	fyne.Do(func() {
		t.stepBtn.SetText(label)
	})
}

// setChromeManaged updates the start-browser button availability.
func (t *globalToolbar) setChromeManaged(managed bool) {
	fyne.Do(func() {
		if managed {
			t.startChromeBtn.Disable()
		} else {
			t.startChromeBtn.Enable()
		}
	})
}
