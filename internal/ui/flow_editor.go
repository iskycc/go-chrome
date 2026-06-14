package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

type flowEditorPanel struct {
	app       *App
	widget    fyne.CanvasObject
	nameEntry *widget.Entry
	descEntry *widget.Entry
	tagsEntry *widget.Entry
	onChanged func() // notify App of dirty state
}

func newFlowEditorPanel(app *App, onChanged func()) *flowEditorPanel {
	p := &flowEditorPanel{app: app, onChanged: onChanged}
	p.nameEntry = widget.NewEntry()
	p.nameEntry.SetPlaceHolder("流程名称")
	p.descEntry = widget.NewEntry()
	p.descEntry.SetPlaceHolder("流程描述")
	p.tagsEntry = widget.NewEntry()
	p.tagsEntry.SetPlaceHolder("标签，用逗号分隔")

	p.nameEntry.OnChanged = func(s string) {
		if p.app.currentFlow != nil {
			p.app.currentFlow.Name = s
			p.fireChanged()
		}
	}
	p.descEntry.OnChanged = func(s string) {
		if p.app.currentFlow != nil {
			p.app.currentFlow.Description = s
			p.fireChanged()
		}
	}
	p.tagsEntry.OnChanged = func(s string) {
		if p.app.currentFlow != nil {
			p.app.currentFlow.Tags = parseTags(s)
			p.fireChanged()
		}
	}

	form := widget.NewForm(
		widget.NewFormItem("名称", p.nameEntry),
		widget.NewFormItem("描述", p.descEntry),
		widget.NewFormItem("标签", p.tagsEntry),
	)
	// 标题与表单之间增加轻量间距，让右侧属性区更有层级感。
	sectionSpacer := canvas.NewRectangle(color.Transparent)
	sectionSpacer.SetMinSize(fyne.NewSize(1, 12))

	p.widget = container.NewPadded(
		container.NewVBox(
			newSectionHeader("流程属性"),
			sectionSpacer,
			form,
		),
	)
	return p
}

func (p *flowEditorPanel) fireChanged() {
	if p.onChanged != nil {
		p.onChanged()
	}
}

func (p *flowEditorPanel) loadFlow(f *flow.Flow) {
	if f == nil {
		p.nameEntry.SetText("")
		p.descEntry.SetText("")
		p.tagsEntry.SetText("")
		return
	}
	p.nameEntry.SetText(f.Name)
	p.descEntry.SetText(f.Description)
	p.tagsEntry.SetText(strings.Join(f.Tags, ", "))
}

func parseTags(s string) []string {
	var tags []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
