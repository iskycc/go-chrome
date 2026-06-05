package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// runPanel shows execution logs at the bottom.
type runPanel struct {
	app    *App
	logs   *widget.Entry
	widget fyne.CanvasObject
}

func newRunPanel(app *App) *runPanel {
	p := &runPanel{app: app}
	p.logs = widget.NewMultiLineEntry()
	p.logs.Disable()
	p.logs.Wrapping = fyne.TextWrapWord
	p.widget = container.NewBorder(
		widget.NewLabelWithStyle("Run Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		container.NewScroll(p.logs),
	)
	return p
}

func (p *runPanel) log(msg string) {
	fyne.Do(func() {
		p.logs.SetText(p.logs.Text + msg + "\n")
	})
}
