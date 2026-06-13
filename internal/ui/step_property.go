package ui

import (
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

// stepSection groups related form fields under a header.
type stepSection struct {
	title   fyne.CanvasObject
	content fyne.CanvasObject
	box     *fyne.Container
}

func newStepSection(title string, fields ...fyne.CanvasObject) *stepSection {
	s := &stepSection{
		title:   newSectionHeader(title),
		content: container.NewVBox(fields...),
	}
	s.box = container.NewVBox(s.title, s.content)
	s.box.Hide()
	return s
}

func (s *stepSection) show() {
	s.box.Show()
}

func (s *stepSection) hide() {
	s.box.Hide()
}

type stepPropertyPanel struct {
	app       *App
	widget    fyne.CanvasObject
	step      *flow.Step
	stepIndex int
	onApplied func()

	scroll *container.Scroll
	form   *fyne.Container

	nameEntry       *widget.Entry
	nameErr         *widget.Label
	typeSelect      *widget.Select
	targetEntry     *widget.Entry
	targetErr       *widget.Label
	inputEntry      *widget.Entry
	inputErr        *widget.Label
	expectedEntry   *widget.Entry
	expectedErr     *widget.Label
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

	basicSection   *stepSection
	targetSection  *stepSection
	inputSection   *stepSection
	expectedSection *stepSection
	waitSection    *stepSection
	errorSection   *stepSection
	noteSection    *stepSection

	applyBtn        *widget.Button
	applyStatusLabel *widget.Label
}

func newStepPropertyPanel(app *App, onApplied func()) *stepPropertyPanel {
	p := &stepPropertyPanel{app: app, onApplied: onApplied, stepIndex: -1}
	p.initWidgets()
	p.initSections()

	p.applyBtn = widget.NewButton("应用到当前步骤", func() { p.apply() })
	p.applyBtn.Importance = widget.HighImportance
	p.applyStatusLabel = widget.NewLabel("")

	bottom := container.NewBorder(nil, nil, nil, p.applyBtn, p.applyStatusLabel)
	p.form = container.NewVBox()
	p.scroll = container.NewScroll(p.form)

	p.widget = container.NewBorder(
		newSectionHeader("步骤属性"),
		container.NewPadded(bottom),
		nil, nil,
		p.scroll,
	)
	return p
}

func (p *stepPropertyPanel) initWidgets() {
	p.nameEntry = widget.NewEntry()
	p.nameEntry.SetPlaceHolder("例如：点击登录按钮")
	p.nameErr = newInlineError("")
	p.nameErr.Hide()

	p.typeSelect = widget.NewSelect(stepTypeOptions, func(s string) { p.rebuildForm() })

	p.targetEntry = widget.NewEntry()
	p.targetEntry.SetPlaceHolder("XPath 或打开网址")
	p.targetErr = newInlineError("")
	p.targetErr.Hide()

	p.inputEntry = widget.NewEntry()
	p.inputEntry.SetPlaceHolder("输入内容或模板，例如 SP${11000-11099}")
	p.inputErr = newInlineError("")
	p.inputErr.Hide()

	p.expectedEntry = widget.NewEntry()
	p.expectedEntry.SetPlaceHolder("期望包含的文本")
	p.expectedErr = newInlineError("")
	p.expectedErr.Hide()

	p.waitBeforeEntry = widget.NewEntry()
	p.waitBeforeEntry.SetText("0")
	p.waitBeforeErr = newInlineError("")
	p.waitBeforeErr.Hide()

	p.waitAfterEntry = widget.NewEntry()
	p.waitAfterEntry.SetText("500")
	p.waitAfterErr = newInlineError("")
	p.waitAfterErr.Hide()

	p.timeoutEntry = widget.NewEntry()
	p.timeoutEntry.SetText("10000")
	p.timeoutErr = newInlineError("")
	p.timeoutErr.Hide()

	p.onErrorSelect = widget.NewSelect(errorPolicyOptions, nil)
	p.onErrorSelect.SetSelected(errorPolicyLabel(flow.ErrStop))

	p.enabledCheck = widget.NewCheck("启用此步骤", nil)
	p.enabledCheck.SetChecked(true)

	p.maskLogsCheck = widget.NewCheck("日志中隐藏输入值", nil)

	p.noteEntry = widget.NewEntry()
	p.noteEntry.SetPlaceHolder("备注")

	p.previewLabel = widget.NewLabel("")
	p.previewLabel.Wrapping = fyne.TextWrapWord
	p.previewLabel.Truncation = fyne.TextTruncateEllipsis
}

func (p *stepPropertyPanel) initSections() {
	// Basic info
	p.basicSection = newStepSection("基础信息",
		widget.NewForm(
			widget.NewFormItem("步骤名称", container.NewVBox(p.nameEntry, p.nameErr)),
			widget.NewFormItem("步骤类型", p.typeSelect),
			widget.NewFormItem("启用状态", p.enabledCheck),
		),
	)

	// Target
	p.targetSection = newStepSection("定位目标",
		widget.NewForm(
			widget.NewFormItem("目标", container.NewVBox(p.targetEntry, p.targetErr)),
		),
	)

	// Input and template
	templateSnippets := map[string]string{
		"数字 6 位":        "${number:6}",
		"字母 8 位":        "${alpha:8}",
		"字母数字 10 位":     "${alnum:10}",
		"UUID":          "${uuid}",
		"日期 yyyyMMdd":   "${date:yyyyMMdd}",
		"时间戳":           "${timestamp}",
		"门店号范围":         "SP${11000-11099}",
		"环境变量 BASE_URL": "${env:BASE_URL}",
		"环境变量 USERNAME": "${env:USERNAME}",
		"环境变量 PASSWORD": "${env:PASSWORD}",
	}
	insertTemplateSelect := widget.NewSelect([]string{
		"数字 6 位",
		"字母 8 位",
		"字母数字 10 位",
		"UUID",
		"日期 yyyyMMdd",
		"时间戳",
		"门店号范围",
		"环境变量 BASE_URL",
		"环境变量 USERNAME",
		"环境变量 PASSWORD",
	}, func(label string) {
		snippet := templateSnippets[label]
		if snippet == "" {
			return
		}
		p.inputEntry.SetText(p.inputEntry.Text + snippet)
	})
	insertTemplateSelect.PlaceHolder = "插入模板"

	previewBtn := widget.NewButton("预览", func() {
		if p.inputEntry.Text != "" {
			samples := template.Preview(p.inputEntry.Text, 3)
			p.previewLabel.SetText(strings.Join(samples, "\n"))
		}
	})
	validateBtn := widget.NewButton("校验", func() {
		if err := template.Validate(p.inputEntry.Text); err != nil {
			dialog.ShowError(err, p.app.mainWin)
		} else {
			dialog.ShowInformation("校验通过", "输入模板语法正确", p.app.mainWin)
		}
	})

	templateToolbar := container.NewHBox(insertTemplateSelect, previewBtn, validateBtn)
	previewBox := container.NewGridWrap(fyne.NewSize(280, 64), p.previewLabel)

	p.inputSection = newStepSection("输入与模板",
		widget.NewForm(
			widget.NewFormItem("输入内容", container.NewVBox(p.inputEntry, templateToolbar, previewBox, p.inputErr)),
			widget.NewFormItem("日志脱敏", p.maskLogsCheck),
		),
	)

	// Expected text
	p.expectedSection = newStepSection("期望文本",
		widget.NewForm(
			widget.NewFormItem("期望文本", container.NewVBox(p.expectedEntry, p.expectedErr)),
		),
	)

	// Wait and timeout
	p.waitSection = newStepSection("等待与超时",
		widget.NewForm(
			widget.NewFormItem("执行前等待", p.sizedMSBox(p.waitBeforeEntry, p.waitBeforeErr)),
			widget.NewFormItem("执行后等待", p.sizedMSBox(p.waitAfterEntry, p.waitAfterErr)),
			widget.NewFormItem("超时时间", p.sizedMSBox(p.timeoutEntry, p.timeoutErr)),
		),
	)

	// Error handling
	p.errorSection = newStepSection("失败处理",
		widget.NewForm(
			widget.NewFormItem("失败处理", p.onErrorSelect),
		),
	)

	// Note
	p.noteSection = newStepSection("备注",
		widget.NewForm(
			widget.NewFormItem("备注", p.noteEntry),
		),
	)
}

// sizedMSBox wraps a numeric entry and its error label with a fixed width and
// an "ms" suffix so unit stays outside the input.
func (p *stepPropertyPanel) sizedMSBox(entry *widget.Entry, err fyne.CanvasObject) fyne.CanvasObject {
	entryBox := container.NewGridWrap(fyne.NewSize(120, entry.MinSize().Height), entry)
	return container.NewVBox(container.NewHBox(entryBox, widget.NewLabel("ms")), err)
}

func (p *stepPropertyPanel) rebuildForm() {
	if p.step == nil {
		return
	}
	p.form.Objects = nil

	sections := []*stepSection{p.basicSection}

	t := stepTypeFromLabel(p.typeSelect.Selected)
	switch t {
	case flow.StepNavigate:
		sections = append(sections, p.targetSection, p.waitSection, p.errorSection, p.noteSection)
	case flow.StepClick:
		sections = append(sections, p.targetSection, p.waitSection, p.errorSection, p.noteSection)
	case flow.StepInput, flow.StepClearAndInput:
		sections = append(sections, p.targetSection, p.inputSection, p.waitSection, p.errorSection, p.noteSection)
	case flow.StepWaitPresent, flow.StepWaitVisible, flow.StepGetText, flow.StepAssertExists:
		sections = append(sections, p.targetSection, p.waitSection, p.errorSection, p.noteSection)
	case flow.StepAssertText:
		sections = append(sections, p.targetSection, p.expectedSection, p.waitSection, p.errorSection, p.noteSection)
	case flow.StepWaitFixed:
		sections = append(sections, p.waitSection, p.noteSection)
	case flow.StepScreenshot:
		sections = append(sections, p.errorSection, p.noteSection)
	}

	for _, s := range sections {
		s.show()
		p.form.Objects = append(p.form.Objects, s.box)
	}

	p.form.Refresh()
}

func (p *stepPropertyPanel) loadStep(s *flow.Step, idx int, total int) {
	p.step = s
	p.stepIndex = idx
	p.clearErrors()
	p.nameEntry.SetText(s.Name)
	p.typeSelect.SetSelected(stepTypeLabel(s.Type))
	p.targetEntry.SetText(s.Target.Value)
	p.inputEntry.SetText(s.Input.Text)
	if s.Input.Text != "" {
		p.expectedEntry.SetText(s.Input.Text)
	} else {
		p.expectedEntry.SetText(s.Note)
	}
	p.waitBeforeEntry.SetText(strconv.Itoa(s.WaitBeforeMs))
	p.waitAfterEntry.SetText(strconv.Itoa(s.WaitAfterMs))
	p.timeoutEntry.SetText(strconv.Itoa(s.TimeoutMs))
	p.onErrorSelect.SetSelected(errorPolicyLabel(s.OnError))
	p.enabledCheck.SetChecked(s.Enabled)
	p.maskLogsCheck.SetChecked(s.Input.MaskInLogs)
	p.noteEntry.SetText(s.Note)
	p.previewLabel.SetText("")
	p.applyStatusLabel.SetText("")
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
			if s.Name == p.nameEntry.Text && i != p.stepIndex {
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
		} else if !strings.Contains(v, "${") && !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
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
		p.applyStatusLabel.SetText("请修正标红字段后再应用")
		return
	}

	p.step.Name = p.nameEntry.Text
	p.step.Type = stepTypeFromLabel(p.typeSelect.Selected)
	p.step.Target = flow.Target{Strategy: flow.TargetXPath, Value: ""}
	if p.step.Type != flow.StepWaitFixed && p.step.Type != flow.StepScreenshot {
		p.step.Target.Value = p.targetEntry.Text
	}
	p.step.Input = flow.Input{Mode: flow.InputTemplate}
	if p.step.Type == flow.StepInput || p.step.Type == flow.StepClearAndInput {
		p.step.Input.Text = p.inputEntry.Text
		p.step.Input.MaskInLogs = p.maskLogsCheck.Checked
	}
	p.step.WaitBeforeMs, _ = strconv.Atoi(p.waitBeforeEntry.Text)
	p.step.WaitAfterMs, _ = strconv.Atoi(p.waitAfterEntry.Text)
	p.step.TimeoutMs, _ = strconv.Atoi(p.timeoutEntry.Text)
	p.step.OnError = errorPolicyFromLabel(p.onErrorSelect.Selected)
	p.step.Enabled = p.enabledCheck.Checked
	p.step.Note = p.noteEntry.Text

	if p.step.Type == flow.StepAssertText {
		p.step.Input.Text = p.expectedEntry.Text
		p.step.Note = p.noteEntry.Text
	}

	p.app.currentFlow.Steps = p.app.stepTable.stepsData
	p.app.stepTable.table.Refresh()
	if p.onApplied != nil {
		p.onApplied()
	}

	p.applyStatusLabel.SetText("已应用")
	go func() {
		time.Sleep(2 * time.Second)
		fyne.Do(func() { p.applyStatusLabel.SetText("") })
	}()
}

func (p *stepPropertyPanel) clear() {
	p.step = nil
	p.stepIndex = -1
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
	p.previewLabel.SetText("")
	p.applyStatusLabel.SetText("")
	p.clearErrors()
	p.form.Objects = nil
	p.form.Refresh()
}

func (p *stepPropertyPanel) hasUnappliedChanges() bool {
	if p.step == nil || p.typeSelect.Selected == "" {
		return false
	}
	if p.nameEntry.Text != p.step.Name {
		return true
	}
	if stepTypeFromLabel(p.typeSelect.Selected) != p.step.Type {
		return true
	}
	t := p.step.Type
	if t != flow.StepWaitFixed && t != flow.StepScreenshot && p.targetEntry.Text != p.step.Target.Value {
		return true
	}
	if (t == flow.StepInput || t == flow.StepClearAndInput) && p.inputEntry.Text != p.step.Input.Text {
		return true
	}
	if t == flow.StepAssertText {
		expected := p.step.Input.Text
		if expected == "" {
			expected = p.step.Note
		}
		if p.expectedEntry.Text != expected {
			return true
		}
	}
	if p.waitBeforeEntry.Text != strconv.Itoa(p.step.WaitBeforeMs) {
		return true
	}
	if p.waitAfterEntry.Text != strconv.Itoa(p.step.WaitAfterMs) {
		return true
	}
	if p.timeoutEntry.Text != strconv.Itoa(p.step.TimeoutMs) {
		return true
	}
	if errorPolicyFromLabel(p.onErrorSelect.Selected) != p.step.OnError {
		return true
	}
	if p.enabledCheck.Checked != p.step.Enabled {
		return true
	}
	if (t == flow.StepInput || t == flow.StepClearAndInput) && p.maskLogsCheck.Checked != p.step.Input.MaskInLogs {
		return true
	}
	return p.noteEntry.Text != p.step.Note
}
