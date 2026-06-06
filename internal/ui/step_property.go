package ui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

type stepPropertyPanel struct {
	app       *App
	widget    fyne.CanvasObject
	step      *flow.Step
	onApplied func()

	form        *widget.Form
	nameEntry   *widget.Entry
	nameErr     *widget.Label
	typeSelect  *widget.Select
	targetEntry *widget.Entry
	targetErr   *widget.Label
	inputEntry  *widget.Entry
	inputErr    *widget.Label
	expectedEntry *widget.Entry
	expectedErr   *widget.Label
	waitBeforeEntry *widget.Entry
	waitBeforeErr   *widget.Label
	waitAfterEntry  *widget.Entry
	waitAfterErr    *widget.Label
	timeoutEntry    *widget.Entry
	timeoutErr      *widget.Label
	onErrorSelect   *widget.Select
	enabledCheck    *widget.Check
	maskLogsCheck   *widget.Check
	noteEntry       *widget.Entry
	previewLabel    *widget.Label

	nameItem       *widget.FormItem
	targetItem     *widget.FormItem
	inputItem      *widget.FormItem
	expectedItem   *widget.FormItem
	waitBeforeItem *widget.FormItem
	waitAfterItem  *widget.FormItem
	timeoutItem    *widget.FormItem
	onErrorItem    *widget.FormItem
	enabledItem    *widget.FormItem
	maskLogsItem   *widget.FormItem
	noteItem       *widget.FormItem
}

func newStepPropertyPanel(app *App, onApplied func()) *stepPropertyPanel {
	p := &stepPropertyPanel{app: app, onApplied: onApplied}
	p.initWidgets()
	p.initForm()
	p.widget = container.NewBorder(
		widget.NewLabelWithStyle("3. 步骤属性", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewButton("应用到当前步骤", func() { p.apply() }),
		nil, nil,
		container.NewScroll(p.form),
	)
	return p
}

func (p *stepPropertyPanel) initWidgets() {
	p.nameEntry = widget.NewEntry()
	p.nameEntry.SetPlaceHolder("例如：点击登录按钮")
	p.nameErr = widget.NewLabel("")
	p.nameErr.TextStyle = fyne.TextStyle{Bold: true}
	p.nameErr.Hide()

	p.typeSelect = widget.NewSelect(stepTypeOptions, func(s string) { p.rebuildForm() })

	p.targetEntry = widget.NewEntry()
	p.targetEntry.SetPlaceHolder("XPath 或打开网址")
	p.targetErr = widget.NewLabel("")
	p.targetErr.TextStyle = fyne.TextStyle{Bold: true}
	p.targetErr.Hide()

	p.inputEntry = widget.NewEntry()
	p.inputEntry.SetPlaceHolder("输入内容或模板，例如 SP${11000-11099}")
	p.inputErr = widget.NewLabel("")
	p.inputErr.TextStyle = fyne.TextStyle{Bold: true}
	p.inputErr.Hide()

	p.expectedEntry = widget.NewEntry()
	p.expectedEntry.SetPlaceHolder("期望包含的文本")
	p.expectedErr = widget.NewLabel("")
	p.expectedErr.TextStyle = fyne.TextStyle{Bold: true}
	p.expectedErr.Hide()

	p.waitBeforeEntry = widget.NewEntry()
	p.waitBeforeEntry.SetText("0")
	p.waitBeforeErr = widget.NewLabel("")
	p.waitBeforeErr.TextStyle = fyne.TextStyle{Bold: true}
	p.waitBeforeErr.Hide()

	p.waitAfterEntry = widget.NewEntry()
	p.waitAfterEntry.SetText("500")
	p.waitAfterErr = widget.NewLabel("")
	p.waitAfterErr.TextStyle = fyne.TextStyle{Bold: true}
	p.waitAfterErr.Hide()

	p.timeoutEntry = widget.NewEntry()
	p.timeoutEntry.SetText("10000")
	p.timeoutErr = widget.NewLabel("")
	p.timeoutErr.TextStyle = fyne.TextStyle{Bold: true}
	p.timeoutErr.Hide()

	p.onErrorSelect = widget.NewSelect(errorPolicyOptions, nil)
	p.onErrorSelect.SetSelected(errorPolicyLabel(flow.ErrStop))

	p.enabledCheck = widget.NewCheck("启用此步骤", nil)
	p.enabledCheck.SetChecked(true)

	p.maskLogsCheck = widget.NewCheck("日志中隐藏输入值", nil)

	p.noteEntry = widget.NewEntry()
	p.noteEntry.SetPlaceHolder("备注")

	p.previewLabel = widget.NewLabel("模板预览：")
}

func (p *stepPropertyPanel) initForm() {
	previewBtn := widget.NewButton("预览", func() {
		if p.inputEntry.Text != "" {
			samples := template.Preview(p.inputEntry.Text, 3)
			p.previewLabel.SetText("模板预览：" + strings.Join(samples, "，"))
		}
	})
	validateBtn := widget.NewButton("校验", func() {
		if err := template.Validate(p.inputEntry.Text); err != nil {
			dialog.ShowError(err, p.app.mainWin)
		} else {
			dialog.ShowInformation("校验通过", "输入模板语法正确", p.app.mainWin)
		}
	})

	p.nameItem = widget.NewFormItem("步骤名称", container.NewVBox(p.nameEntry, p.nameErr))
	p.targetItem = widget.NewFormItem("目标", container.NewVBox(p.targetEntry, p.targetErr))
	p.inputItem = widget.NewFormItem("输入内容", container.NewVBox(p.inputEntry, container.NewHBox(previewBtn, validateBtn), p.previewLabel, p.inputErr))
	p.expectedItem = widget.NewFormItem("期望文本", container.NewVBox(p.expectedEntry, p.expectedErr))
	p.waitBeforeItem = widget.NewFormItem("执行前等待(ms)", container.NewVBox(p.waitBeforeEntry, p.waitBeforeErr))
	p.waitAfterItem = widget.NewFormItem("执行后等待(ms)", container.NewVBox(p.waitAfterEntry, p.waitAfterErr))
	p.timeoutItem = widget.NewFormItem("超时时间(ms)", container.NewVBox(p.timeoutEntry, p.timeoutErr))
	p.onErrorItem = widget.NewFormItem("失败处理", p.onErrorSelect)
	p.enabledItem = widget.NewFormItem("启用状态", p.enabledCheck)
	p.maskLogsItem = widget.NewFormItem("日志脱敏", p.maskLogsCheck)
	p.noteItem = widget.NewFormItem("备注", p.noteEntry)

	p.form = widget.NewForm()
}

func (p *stepPropertyPanel) rebuildForm() {
	if p.step == nil {
		return
	}
	p.form.Items = nil
	p.form.Items = append(p.form.Items, p.nameItem)

	t := stepTypeFromLabel(p.typeSelect.Selected)
	show := func(it *widget.FormItem) { p.form.Items = append(p.form.Items, it) }

	switch t {
	case flow.StepNavigate:
		show(p.targetItem)
		show(p.waitAfterItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepClick:
		show(p.targetItem)
		show(p.waitBeforeItem)
		show(p.waitAfterItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepInput, flow.StepClearAndInput:
		show(p.targetItem)
		show(p.inputItem)
		show(p.maskLogsItem)
		show(p.waitBeforeItem)
		show(p.waitAfterItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepWaitPresent, flow.StepWaitVisible, flow.StepGetText, flow.StepAssertExists:
		show(p.targetItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepAssertText:
		show(p.targetItem)
		show(p.expectedItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepWaitFixed:
		show(p.waitAfterItem)
		show(p.noteItem)
	case flow.StepScreenshot:
		show(p.noteItem)
		show(p.onErrorItem)
	}
	p.form.Refresh()
}

func (p *stepPropertyPanel) loadStep(s *flow.Step, idx int, total int) {
	p.step = s
	p.clearErrors()
	p.nameEntry.SetText(s.Name)
	p.typeSelect.SetSelected(stepTypeLabel(s.Type))
	p.targetEntry.SetText(s.Target.Value)
	p.inputEntry.SetText(s.Input.Text)
	p.expectedEntry.SetText(s.Note)
	p.waitBeforeEntry.SetText(strconv.Itoa(s.WaitBeforeMs))
	p.waitAfterEntry.SetText(strconv.Itoa(s.WaitAfterMs))
	p.timeoutEntry.SetText(strconv.Itoa(s.TimeoutMs))
	p.onErrorSelect.SetSelected(errorPolicyLabel(s.OnError))
	p.enabledCheck.SetChecked(s.Enabled)
	p.maskLogsCheck.SetChecked(s.Input.MaskInLogs)
	p.noteEntry.SetText(s.Note)
	p.previewLabel.SetText("模板预览：")
	p.rebuildForm()
}

func (p *stepPropertyPanel) clearErrors() {
	p.nameErr.Hide()
	p.targetErr.Hide()
	p.inputErr.Hide()
	p.expectedErr.Hide()
	p.waitBeforeErr.Hide()
	p.waitAfterErr.Hide()
	p.timeoutErr.Hide()
}

func (p *stepPropertyPanel) validate() bool {
	p.clearErrors()
	ok := true

	if strings.TrimSpace(p.nameEntry.Text) == "" {
		p.nameErr.SetText("步骤名称不能为空")
		p.nameErr.Show()
		ok = false
	}
	if p.app.currentFlow != nil {
		for i, s := range p.app.currentFlow.Steps {
			if s.Name == p.nameEntry.Text && i != p.app.stepTable.selectedIndex() {
				p.nameErr.SetText("步骤名称已存在")
				p.nameErr.Show()
				ok = false
				break
			}
		}
	}

	t := stepTypeFromLabel(p.typeSelect.Selected)
	if t == flow.StepNavigate {
		v := strings.TrimSpace(p.targetEntry.Text)
		if v == "" {
			p.targetErr.SetText("网址不能为空")
			p.targetErr.Show()
			ok = false
		} else if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
			p.targetErr.SetText("网址必须以 http:// 或 https:// 开头")
			p.targetErr.Show()
			ok = false
		}
	} else if flow.NeedsElement(t) && t != flow.StepWaitFixed && t != flow.StepScreenshot {
		if strings.TrimSpace(p.targetEntry.Text) == "" {
			p.targetErr.SetText("XPath 不能为空")
			p.targetErr.Show()
			ok = false
		}
	}

	if t == flow.StepAssertText {
		if strings.TrimSpace(p.expectedEntry.Text) == "" {
			p.expectedErr.SetText("期望文本不能为空")
			p.expectedErr.Show()
			ok = false
		}
	}

	if v, err := strconv.Atoi(p.waitBeforeEntry.Text); err != nil || v < 0 {
		p.waitBeforeErr.SetText("必须为非负整数")
		p.waitBeforeErr.Show()
		ok = false
	}
	if v, err := strconv.Atoi(p.waitAfterEntry.Text); err != nil || v < 0 {
		p.waitAfterErr.SetText("必须为非负整数")
		p.waitAfterErr.Show()
		ok = false
	}
	if v, err := strconv.Atoi(p.timeoutEntry.Text); err != nil || v < 0 {
		p.timeoutErr.SetText("必须为非负整数")
		p.timeoutErr.Show()
		ok = false
	}

	if p.inputEntry.Text != "" {
		if err := template.Validate(p.inputEntry.Text); err != nil {
			p.inputErr.SetText(err.Error())
			p.inputErr.Show()
			ok = false
		}
	}

	return ok
}

func (p *stepPropertyPanel) apply() {
	if p.step == nil || p.app.currentFlow == nil {
		return
	}
	if !p.validate() {
		return
	}

	p.step.Name = p.nameEntry.Text
	p.step.Type = stepTypeFromLabel(p.typeSelect.Selected)
	p.step.Target = flow.Target{Strategy: flow.TargetXPath, Value: p.targetEntry.Text}
	p.step.Input = flow.Input{
		Mode:       flow.InputTemplate,
		Text:       p.inputEntry.Text,
		MaskInLogs: p.maskLogsCheck.Checked,
	}
	p.step.WaitBeforeMs, _ = strconv.Atoi(p.waitBeforeEntry.Text)
	p.step.WaitAfterMs, _ = strconv.Atoi(p.waitAfterEntry.Text)
	p.step.TimeoutMs, _ = strconv.Atoi(p.timeoutEntry.Text)
	p.step.OnError = errorPolicyFromLabel(p.onErrorSelect.Selected)
	p.step.Enabled = p.enabledCheck.Checked
	p.step.Note = p.noteEntry.Text

	if p.step.Type == flow.StepAssertText {
		p.step.Note = p.expectedEntry.Text
	}

	p.app.currentFlow.Steps = p.app.stepTable.stepsData
	p.app.stepTable.table.Refresh()
	if p.onApplied != nil {
		p.onApplied()
	}
}

func (p *stepPropertyPanel) clear() {
	p.step = nil
	p.nameEntry.SetText("")
	p.typeSelect.SetSelected("")
	p.targetEntry.SetText("")
	p.inputEntry.SetText("")
	p.expectedEntry.SetText("")
	p.waitBeforeEntry.SetText("0")
	p.waitAfterEntry.SetText("500")
	p.timeoutEntry.SetText("10000")
	p.onErrorSelect.SetSelected(errorPolicyLabel(flow.ErrStop))
	p.enabledCheck.SetChecked(true)
	p.maskLogsCheck.SetChecked(false)
	p.noteEntry.SetText("")
	p.previewLabel.SetText("模板预览：")
	p.clearErrors()
	p.form.Items = nil
	p.form.Refresh()
}
