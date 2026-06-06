package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/runner"
)

// historyPanel shows run history for the current flow.
type historyPanel struct {
	app     *App
	widget  fyne.CanvasObject
	list    *widget.List
	results []*runner.RunResult
}

func newHistoryPanel(app *App) *historyPanel {
	p := &historyPanel{app: app}
	p.list = widget.NewList(
		func() int { return len(p.results) },
		func() fyne.CanvasObject {
			return widget.NewLabel("历史记录")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.results) {
				return
			}
			r := p.results[id]
			label := fmt.Sprintf("%s | %s | 成功:%d 失败:%d",
				r.StartedAt.Format("01-02 15:04:05"),
				r.Status,
				r.SuccessCount, r.FailedCount)
			item.(*widget.Label).SetText(label)
		},
	)
	p.widget = container.NewBorder(
		widget.NewLabelWithStyle("执行历史", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		container.NewScroll(p.list),
	)
	return p
}

func (p *historyPanel) setResults(results []*runner.RunResult) {
	p.results = results
	p.list.Refresh()
}
