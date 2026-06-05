package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

// stepEditorPanel shows the step table and property editor.
type stepEditorPanel struct {
	app              *App
	currentFlow      *flow.Flow
	stepsData        []flow.Step
	stepsWidget      fyne.CanvasObject
	propertiesWidget fyne.CanvasObject

	// Table
	stepList *widget.List
	selected int

	// Property editors
	propName       *widget.Entry
	propType       *widget.Select
	propTarget     *widget.Entry
	propInput      *widget.Entry
	propWaitBefore *widget.Entry
	propWaitAfter  *widget.Entry
	propTimeout    *widget.Entry
	propOnError    *widget.Select
	propEnabled    *widget.Check
	propNote       *widget.Entry
	propPreview    *widget.Label
	propMaskLogs   *widget.Check
}

func newStepEditorPanel(app *App) *stepEditorPanel {
	p := &stepEditorPanel{app: app, selected: -1}
	p.initStepsWidget()
	p.initPropertiesWidget()
	return p
}

func (p *stepEditorPanel) initStepsWidget() {
	p.stepList = widget.NewList(
		func() int { return len(p.stepsData) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(themeIcon(theme.IconNameDocument)),
				widget.NewLabel("Step"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.stepsData) {
				return
			}
			s := p.stepsData[id]
			box := item.(*fyne.Container)
			label := box.Objects[1].(*widget.Label)
			label.SetText(fmt.Sprintf("%d: %s (%s)", id+1, s.Name, s.Type))
			if !s.Enabled {
				label.TextStyle = fyne.TextStyle{Italic: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
		},
	)
	p.stepList.OnSelected = func(id widget.ListItemID) {
		p.selected = id
		if id >= 0 && id < len(p.stepsData) {
			p.loadStepProperties(&p.stepsData[id])
		}
	}

	addBtn := widget.NewButton("Add", func() { p.addStep() })
	delBtn := widget.NewButton("Del", func() { p.deleteStep() })
	upBtn := widget.NewButton("Up", func() { p.moveStep(-1) })
	downBtn := widget.NewButton("Down", func() { p.moveStep(1) })
	copyBtn := widget.NewButton("Copy", func() { p.copyStep() })

	p.stepsWidget = container.NewBorder(
		container.NewHBox(addBtn, delBtn, upBtn, downBtn, copyBtn),
		nil, nil, nil,
		p.stepList,
	)
}

func (p *stepEditorPanel) initPropertiesWidget() {
	p.propName = widget.NewEntry()
	p.propName.SetPlaceHolder("Step name")
	p.propType = widget.NewSelect([]string{
		string(flow.StepNavigate), string(flow.StepClick), string(flow.StepInput),
		string(flow.StepClearAndInput), string(flow.StepWaitPresent), string(flow.StepWaitVisible),
		string(flow.StepWaitFixed), string(flow.StepGetText), string(flow.StepAssertExists),
		string(flow.StepAssertText), string(flow.StepScreenshot),
	}, nil)
	p.propTarget = widget.NewEntry()
	p.propTarget.SetPlaceHolder("XPath")
	p.propInput = widget.NewEntry()
	p.propInput.SetPlaceHolder("Input text or template")
	p.propWaitBefore = widget.NewEntry()
	p.propWaitBefore.SetText("0")
	p.propWaitAfter = widget.NewEntry()
	p.propWaitAfter.SetText("500")
	p.propTimeout = widget.NewEntry()
	p.propTimeout.SetText("10000")
	p.propOnError = widget.NewSelect([]string{string(flow.ErrStop), string(flow.ErrContinue), string(flow.ErrRetry)}, nil)
	p.propOnError.SetSelected(string(flow.ErrStop))
	p.propEnabled = widget.NewCheck("Enabled", nil)
	p.propEnabled.SetChecked(true)
	p.propNote = widget.NewEntry()
	p.propNote.SetPlaceHolder("Note")
	p.propPreview = widget.NewLabel("Preview: ")
	p.propMaskLogs = widget.NewCheck("Mask in logs", nil)

	previewBtn := widget.NewButton("Preview", func() {
		if p.propInput.Text != "" {
			samples := template.Preview(p.propInput.Text, 3)
			p.propPreview.SetText("Preview: " + strings.Join(samples, ", "))
		}
	})
	validateBtn := widget.NewButton("Validate", func() {
		if err := template.Validate(p.propInput.Text); err != nil {
			dialog.ShowError(err, p.app.mainWin)
		} else {
			dialog.ShowInformation("Valid", "Template syntax is valid", p.app.mainWin)
		}
	})

	applyBtn := widget.NewButton("Apply", func() { p.applyStepProperties() })

	form := widget.NewForm(
		widget.NewFormItem("Name", p.propName),
		widget.NewFormItem("Type", p.propType),
		widget.NewFormItem("Target (XPath)", p.propTarget),
		widget.NewFormItem("Input", container.NewBorder(nil, nil, nil, container.NewHBox(previewBtn, validateBtn), p.propInput)),
		widget.NewFormItem("", p.propPreview),
		widget.NewFormItem("Wait Before (ms)", p.propWaitBefore),
		widget.NewFormItem("Wait After (ms)", p.propWaitAfter),
		widget.NewFormItem("Timeout (ms)", p.propTimeout),
		widget.NewFormItem("On Error", p.propOnError),
		widget.NewFormItem("Enabled", p.propEnabled),
		widget.NewFormItem("Mask in logs", p.propMaskLogs),
		widget.NewFormItem("Note", p.propNote),
	)

	p.propertiesWidget = container.NewBorder(
		widget.NewLabelWithStyle("Step Properties", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		applyBtn, nil, nil,
		container.NewScroll(form),
	)
}

func (p *stepEditorPanel) loadFlow(f *flow.Flow) {
	p.currentFlow = f
	if f == nil {
		p.stepsData = nil
	} else {
		p.stepsData = f.Steps
	}
	p.selected = -1
	p.stepList.UnselectAll()
	p.stepList.Refresh()
	p.clearProperties()
}

func (p *stepEditorPanel) clearProperties() {
	p.propName.SetText("")
	p.propType.SetSelected("")
	p.propTarget.SetText("")
	p.propInput.SetText("")
	p.propWaitBefore.SetText("0")
	p.propWaitAfter.SetText("500")
	p.propTimeout.SetText("10000")
	p.propOnError.SetSelected(string(flow.ErrStop))
	p.propEnabled.SetChecked(true)
	p.propNote.SetText("")
	p.propPreview.SetText("Preview: ")
	p.propMaskLogs.SetChecked(false)
}

func (p *stepEditorPanel) loadStepProperties(s *flow.Step) {
	p.propName.SetText(s.Name)
	p.propType.SetSelected(string(s.Type))
	p.propTarget.SetText(s.Target.Value)
	p.propInput.SetText(s.Input.Text)
	p.propWaitBefore.SetText(strconv.Itoa(s.WaitBeforeMs))
	p.propWaitAfter.SetText(strconv.Itoa(s.WaitAfterMs))
	p.propTimeout.SetText(strconv.Itoa(s.TimeoutMs))
	p.propOnError.SetSelected(string(s.OnError))
	p.propEnabled.SetChecked(s.Enabled)
	p.propNote.SetText(s.Note)
	p.propMaskLogs.SetChecked(s.Input.MaskInLogs)
}

func (p *stepEditorPanel) applyStepProperties() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	s := &p.stepsData[p.selected]
	s.Name = p.propName.Text
	s.Type = flow.StepType(p.propType.Selected)
	s.Target = flow.Target{Strategy: flow.TargetXPath, Value: p.propTarget.Text}
	s.Input = flow.Input{
		Mode:       flow.InputTemplate,
		Text:       p.propInput.Text,
		MaskInLogs: p.propMaskLogs.Checked,
	}
	if v, err := strconv.Atoi(p.propWaitBefore.Text); err == nil {
		s.WaitBeforeMs = v
	}
	if v, err := strconv.Atoi(p.propWaitAfter.Text); err == nil {
		s.WaitAfterMs = v
	}
	if v, err := strconv.Atoi(p.propTimeout.Text); err == nil {
		s.TimeoutMs = v
	}
	s.OnError = flow.ErrorPolicy(p.propOnError.Selected)
	s.Enabled = p.propEnabled.Checked
	s.Note = p.propNote.Text
	p.currentFlow.Steps = p.stepsData
	p.stepList.Refresh()
}

func (p *stepEditorPanel) addStep() {
	if p.currentFlow == nil {
		return
	}
	newStep := flow.NewStep("New Step", flow.StepClick)
	p.stepsData = append(p.stepsData, newStep)
	p.currentFlow.Steps = p.stepsData
	p.stepList.Refresh()
}

func (p *stepEditorPanel) deleteStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	p.stepsData = append(p.stepsData[:p.selected], p.stepsData[p.selected+1:]...)
	p.currentFlow.Steps = p.stepsData
	p.selected = -1
	p.stepList.UnselectAll()
	p.stepList.Refresh()
	p.clearProperties()
}

func (p *stepEditorPanel) moveStep(delta int) {
	idx := p.selected
	newIdx := idx + delta
	if idx < 0 || newIdx < 0 || newIdx >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	p.stepsData[idx], p.stepsData[newIdx] = p.stepsData[newIdx], p.stepsData[idx]
	p.currentFlow.Steps = p.stepsData
	p.selected = newIdx
	p.stepList.Select(newIdx)
	p.stepList.Refresh()
}

func (p *stepEditorPanel) copyStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	copied := p.stepsData[p.selected]
	copied.ID = "" // Will be regenerated on save if needed, but flow model doesn't auto-regen
	p.stepsData = append(p.stepsData[:p.selected+1], append([]flow.Step{copied}, p.stepsData[p.selected+1:]...)...)
	p.currentFlow.Steps = p.stepsData
	p.stepList.Refresh()
}

func (p *stepEditorPanel) selectedIndex() int {
	return p.selected
}

func (p *stepEditorPanel) updateStepStatus(idx int, status interface{}) {
	// Refresh list to show status visually (simplified)
	fyne.Do(func() {
		p.stepList.Refresh()
	})
}

func (p *stepEditorPanel) clearStatuses() {
	fyne.Do(func() {
		p.stepList.Refresh()
	})
}

func themeIcon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}
