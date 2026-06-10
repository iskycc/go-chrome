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
	app          *App
	widget       fyne.CanvasObject
	list         *widget.List
	results      []*runner.RunResult
	envSelect    *widget.Select
	statusSelect *widget.Select
	envIDsByName map[string]string
}

func newHistoryPanel(app *App) *historyPanel {
	p := &historyPanel{app: app}
	p.envIDsByName = map[string]string{"全部环境": ""}
	p.envSelect = widget.NewSelect([]string{"全部环境"}, func(string) {
		p.app.refreshHistory()
	})
	p.envSelect.SetSelected("全部环境")
	p.statusSelect = widget.NewSelect([]string{"全部状态", "成功", "失败"}, func(string) {
		p.app.refreshHistory()
	})
	p.statusSelect.SetSelected("全部状态")
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
		container.NewVBox(
			widget.NewLabelWithStyle("执行历史", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(widget.NewLabel("环境"), p.envSelect, widget.NewLabel("状态"), p.statusSelect),
		),
		nil, nil, nil,
		container.NewScroll(p.list),
	)
	return p
}

func (p *historyPanel) setResults(results []*runner.RunResult) {
	fyne.Do(func() {
		p.results = results
		p.list.Refresh()
	})
}

func (p *historyPanel) refreshFilters() {
	if p.app.envRepo == nil {
		return
	}
	envs, err := p.app.envRepo.List()
	if err != nil {
		return
	}
	options := []string{"全部环境"}
	ids := map[string]string{"全部环境": ""}
	for _, env := range envs {
		options = append(options, env.Name)
		ids[env.Name] = env.ID
	}
	current := p.envSelect.Selected
	if current == "" {
		current = "全部环境"
	}
	if _, ok := ids[current]; !ok {
		current = "全部环境"
	}
	fyne.Do(func() {
		p.envIDsByName = ids
		p.envSelect.Options = options
		p.envSelect.SetSelected(current)
	})
}

func (p *historyPanel) environmentFilter() string {
	if p == nil || p.envSelect == nil {
		return ""
	}
	return p.envIDsByName[p.envSelect.Selected]
}

func (p *historyPanel) statusFilter() runner.Status {
	if p == nil || p.statusSelect == nil {
		return ""
	}
	switch p.statusSelect.Selected {
	case "成功":
		return runner.StatusSuccess
	case "失败":
		return runner.StatusFailed
	default:
		return ""
	}
}
