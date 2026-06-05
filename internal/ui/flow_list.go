package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

// flowListPanel shows the list of flows on the left.
type flowListPanel struct {
	app      *App
	flows    []*flow.Flow
	list     *widget.List
	search   *widget.Entry
	selected int
	widget   fyne.CanvasObject
}

func newFlowListPanel(app *App) *flowListPanel {
	p := &flowListPanel{app: app, selected: -1}
	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("Search flows...")
	p.search.OnChanged = func(s string) {
		p.filter(s)
	}

	p.list = widget.NewList(
		func() int { return len(p.flows) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Flow Name")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.flows) {
				return
			}
			item.(*widget.Label).SetText(p.flows[id].Name)
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		p.selected = id
		if id >= 0 && id < len(p.flows) {
			p.app.onFlowSelected(p.flows[id])
		}
	}
	p.list.OnUnselected = func(id widget.ListItemID) {
		if p.selected == id {
			p.selected = -1
		}
	}

	newBtn := widget.NewButton("New", func() { p.app.createNewFlow() })
	delBtn := widget.NewButton("Delete", func() {
		id := p.selected
		if id < 0 || id >= len(p.flows) {
			return
		}
		dialog.ShowConfirm("Delete", "Delete this flow?", func(ok bool) {
			if ok {
				p.app.onFlowDelete(p.flows[id])
			}
		}, p.app.mainWin)
	})
	cloneBtn := widget.NewButton("Clone", func() {
		id := p.selected
		if id < 0 || id >= len(p.flows) {
			return
		}
		p.app.onFlowClone(p.flows[id])
	})

	p.widget = container.NewBorder(
		container.NewVBox(p.search, container.NewHBox(newBtn, delBtn, cloneBtn)),
		nil, nil, nil,
		p.list,
	)
	return p
}

func (p *flowListPanel) setFlows(flows []*flow.Flow) {
	p.flows = flows
	if p.selected >= len(p.flows) {
		p.selected = -1
	}
	p.list.Refresh()
}

func (p *flowListPanel) filter(query string) {
	if strings.TrimSpace(query) == "" {
		flows, _ := p.app.flowStore.ListSorted()
		p.setFlows(flows)
		return
	}
	results, _ := p.app.flowStore.Search(query)
	p.setFlows(results)
}
