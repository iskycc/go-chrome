package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/runner"
)

type stepTablePanel struct {
	app           *App
	currentFlow   *flow.Flow
	stepsData     []flow.Step
	widget        fyne.CanvasObject
	table         *widget.Table
	selected      int
	statuses      []runner.Status
	selectedBar   *fyne.Container
	onStepChanged func()
}

func newStepTablePanel(app *App, onStepChanged func()) *stepTablePanel {
	p := &stepTablePanel{app: app, selected: -1, onStepChanged: onStepChanged}
	p.initTable()

	addBtn := widget.NewButtonWithIcon("新增步骤", theme.ContentAddIcon(), func() { p.showAddStepDialog() })
	copyBtn := widget.NewButtonWithIcon("复制", theme.ContentCopyIcon(), func() { p.copyStep() })
	delBtn := widget.NewButtonWithIcon("删除", theme.DeleteIcon(), func() { p.deleteStep() })
	upBtn := widget.NewButtonWithIcon("上移", theme.MoveUpIcon(), func() { p.moveStep(-1) })
	downBtn := widget.NewButtonWithIcon("下移", theme.MoveDownIcon(), func() { p.moveStep(1) })
	p.selectedBar = container.NewHBox(widget.NewLabel("选中步骤："), copyBtn, delBtn, upBtn, downBtn)
	p.selectedBar.Hide()

	p.widget = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("2. 步骤编排", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(addBtn),
			p.selectedBar,
		),
		nil, nil, nil,
		p.table,
	)
	return p
}

func (p *stepTablePanel) initTable() {
	cols := 9
	p.table = widget.NewTable(
		func() (int, int) {
			if p.currentFlow == nil {
				return 0, cols
			}
			return len(p.stepsData), cols
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("cell")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			if id.Row < 0 || id.Row >= len(p.stepsData) {
				label.SetText("")
				return
			}
			s := p.stepsData[id.Row]
			switch id.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", id.Row+1))
			case 1:
				if id.Row < len(p.statuses) {
					label.SetText(statusIcon(p.statuses[id.Row]))
				} else {
					label.SetText("")
				}
			case 2:
				if s.Enabled {
					label.SetText("✓")
				} else {
					label.SetText("")
				}
			case 3:
				label.SetText(truncate(s.Name, 14))
			case 4:
				label.SetText(stepTypeLabel(s.Type))
			case 5:
				label.SetText(truncate(s.Target.Value, 20))
			case 6:
				if s.Input.MaskInLogs {
					label.SetText("***")
				} else {
					label.SetText(truncate(s.Input.Text, 20))
				}
			case 7:
				label.SetText(fmt.Sprintf("%dms", s.WaitAfterMs))
			case 8:
				label.SetText(errorPolicyLabel(s.OnError))
			}
			if !s.Enabled {
				label.TextStyle = fyne.TextStyle{Italic: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
		},
	)
	p.table.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(p.stepsData) {
			return
		}
		p.selected = id.Row
		p.updateSelectedActions()
		p.app.onStepSelected(&p.stepsData[id.Row], id.Row)
	}
	p.table.SetColumnWidth(0, 40)
	p.table.SetColumnWidth(1, 40)
	p.table.SetColumnWidth(2, 40)
	p.table.SetColumnWidth(3, 120)
	p.table.SetColumnWidth(4, 100)
	p.table.SetColumnWidth(5, 140)
	p.table.SetColumnWidth(6, 140)
	p.table.SetColumnWidth(7, 60)
	p.table.SetColumnWidth(8, 80)
}

func statusIcon(st runner.Status) string {
	switch st {
	case runner.StatusRunning:
		return "●"
	case runner.StatusSuccess:
		return "✓"
	case runner.StatusFailed:
		return "✗"
	case runner.StatusSkipped:
		return "−"
	default:
		return ""
	}
}

func (p *stepTablePanel) loadFlow(f *flow.Flow) {
	p.currentFlow = f
	if f == nil {
		p.stepsData = nil
	} else {
		p.stepsData = f.Steps
	}
	p.selected = -1
	p.statuses = nil
	p.updateSelectedActions()
	p.table.Refresh()
}

func (p *stepTablePanel) showAddStepDialog() {
	if p.currentFlow == nil {
		return
	}
	selector := widget.NewSelect(stepTypeOptions, nil)
	selector.SetSelected(stepTypeOptions[0])

	var d dialog.Dialog
	d = dialog.NewCustomConfirm("选择步骤类型", "确定", "取消",
		container.NewVBox(
			widget.NewLabel("请选择要新增的步骤类型"),
			selector,
		),
		func(ok bool) {
			if !ok {
				return
			}
			t := stepTypeFromLabel(selector.Selected)
			name := selector.Selected + "步骤"
			newStep := flow.NewStep(name, t)
			idx := p.selected + 1
			if idx < 0 {
				idx = len(p.stepsData)
			}
			p.stepsData = append(p.stepsData[:idx], append([]flow.Step{newStep}, p.stepsData[idx:]...)...)
			p.currentFlow.Steps = p.stepsData
			p.table.Refresh()
			p.table.Select(widget.TableCellID{Row: idx, Col: 0})
			p.fireChanged()
		},
		p.app.mainWin,
	)
	d.Resize(fyne.NewSize(420, 180))
	d.Show()
}

func (p *stepTablePanel) deleteStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	p.stepsData = append(p.stepsData[:p.selected], p.stepsData[p.selected+1:]...)
	p.currentFlow.Steps = p.stepsData
	p.selected = -1
	p.updateSelectedActions()
	p.table.UnselectAll()
	p.table.Refresh()
	p.app.onStepSelected(nil, -1)
	p.fireChanged()
}

func (p *stepTablePanel) moveStep(delta int) {
	idx := p.selected
	newIdx := idx + delta
	if idx < 0 || newIdx < 0 || newIdx >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	p.stepsData[idx], p.stepsData[newIdx] = p.stepsData[newIdx], p.stepsData[idx]
	p.currentFlow.Steps = p.stepsData
	p.selected = newIdx
	p.updateSelectedActions()
	p.table.Select(widget.TableCellID{Row: newIdx, Col: 0})
	p.table.Refresh()
	p.fireChanged()
}

func (p *stepTablePanel) updateSelectedActions() {
	if p.selectedBar == nil {
		return
	}
	if p.currentFlow != nil && p.selected >= 0 && p.selected < len(p.stepsData) {
		p.selectedBar.Show()
	} else {
		p.selectedBar.Hide()
	}
}

func (p *stepTablePanel) copyStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	copied := p.stepsData[p.selected]
	copied.ID = ""
	p.stepsData = append(p.stepsData[:p.selected+1], append([]flow.Step{copied}, p.stepsData[p.selected+1:]...)...)
	p.currentFlow.Steps = p.stepsData
	p.table.Refresh()
	p.fireChanged()
}

func (p *stepTablePanel) setStatuses(statuses []runner.Status) {
	p.statuses = statuses
	p.table.Refresh()
}

func (p *stepTablePanel) clearStatuses() {
	p.statuses = nil
	p.table.Refresh()
}

func (p *stepTablePanel) selectedIndex() int {
	return p.selected
}

func (p *stepTablePanel) fireChanged() {
	if p.onStepChanged != nil {
		p.onStepChanged()
	}
}
