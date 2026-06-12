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
	addBtn.Importance = widget.HighImportance
	copyBtn := widget.NewButtonWithIcon("复制", theme.ContentCopyIcon(), func() { p.copyStep() })
	delBtn := widget.NewButtonWithIcon("删除", theme.DeleteIcon(), func() { p.deleteStep() })
	delBtn.Importance = widget.DangerImportance
	upBtn := widget.NewButtonWithIcon("上移", theme.MoveUpIcon(), func() { p.moveStep(-1) })
	downBtn := widget.NewButtonWithIcon("下移", theme.MoveDownIcon(), func() { p.moveStep(1) })
	p.selectedBar = container.NewHBox(widget.NewLabel("选中步骤："), copyBtn, delBtn, upBtn, downBtn)
	p.selectedBar.Hide()

	p.widget = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("步骤编排", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
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
			// Truncating label so long step names, XPaths, and
			// input text get visually clipped with "…" instead
			// of drawing past the column width.
			return newContextMenuLabel("cell", nil)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*contextMenuLabel)
			if id.Row < 0 || id.Row >= len(p.stepsData) {
				label.SetText("")
				label.onSecondaryTap = nil
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
				label.SetText(s.Name)
			case 4:
				label.SetText(stepTypeLabel(s.Type))
			case 5:
				label.SetText(s.Target.Value)
			case 6:
				if s.Input.MaskInLogs {
					label.SetText("***")
				} else {
					label.SetText(s.Input.Text)
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
			label.onSecondaryTap = func(e *fyne.PointEvent) {
				p.showStepContextMenu(id.Row, e)
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
	d.Resize(fyne.NewSize(480, 200))
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

func (p *stepTablePanel) showStepContextMenu(row int, e *fyne.PointEvent) {
	if row < 0 || row >= len(p.stepsData) || p.currentFlow == nil {
		return
	}
	p.table.Select(widget.TableCellID{Row: row, Col: 0})
	s := &p.stepsData[row]

	editItem := fyne.NewMenuItem("编辑步骤属性", func() {
		p.app.onStepSelected(s, row)
	})
	runFromItem := fyne.NewMenuItem("从此步骤运行", func() {
		p.table.Select(widget.TableCellID{Row: row, Col: 0})
		p.app.runCurrentFlow()
	})
	copyItem := fyne.NewMenuItem("复制步骤", func() {
		p.copyStep()
	})
	deleteItem := fyne.NewMenuItem("删除步骤", func() {
		p.confirmDeleteStep()
	})
	upItem := fyne.NewMenuItem("上移", func() {
		p.moveStep(-1)
	})
	upItem.Disabled = row <= 0
	downItem := fyne.NewMenuItem("下移", func() {
		p.moveStep(1)
	})
	downItem.Disabled = row >= len(p.stepsData)-1
	toggleItem := fyne.NewMenuItem("启用", nil)
	if s.Enabled {
		toggleItem.Label = "禁用"
	}
	toggleItem.Action = func() {
		s.Enabled = !s.Enabled
		p.table.Refresh()
		p.fireChanged()
	}
	copyNameItem := fyne.NewMenuItem("复制步骤名称", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(s.Name))
		p.app.runPanel.log("步骤名称已复制到剪贴板")
	})
	copyTargetItem := fyne.NewMenuItem("复制 XPath/目标", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(s.Target.Value))
		p.app.runPanel.log("XPath/目标已复制到剪贴板")
	})
	copyInputItem := fyne.NewMenuItem("复制输入内容", func() {
		if s.Input.MaskInLogs {
			showWrappedConfirm("复制敏感输入", "该输入被标记为敏感，复制将把明文写入剪贴板，是否继续？", "继续", "取消", fyne.NewSize(480, 180), func(ok bool) {
				if ok {
					p.app.fyneApp.Clipboard().SetContent(clipCopy(s.Input.Text))
					p.app.runPanel.log("输入内容已复制到剪贴板")
				}
			}, p.app.mainWin)
			return
		}
		p.app.fyneApp.Clipboard().SetContent(clipCopy(s.Input.Text))
		p.app.runPanel.log("输入内容已复制到剪贴板")
	})

	menu := fyne.NewMenu("步骤操作",
		editItem,
		runFromItem,
		fyne.NewMenuItemSeparator(),
		copyItem,
		copyNameItem,
		copyTargetItem,
		copyInputItem,
		fyne.NewMenuItemSeparator(),
		toggleItem,
		upItem,
		downItem,
		fyne.NewMenuItemSeparator(),
		deleteItem,
	)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}

func (p *stepTablePanel) confirmDeleteStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) {
		return
	}
	s := p.stepsData[p.selected]
	msg := fmt.Sprintf("确定删除步骤 [%s] 吗？", truncateForDialog(s.Name, 80))
	showWrappedConfirm("确认删除", msg, "删除", "取消", fyne.NewSize(520, 180), func(ok bool) {
		if ok {
			p.deleteStep()
		}
	}, p.app.mainWin)
}

func (p *stepTablePanel) fireChanged() {
	if p.onStepChanged != nil {
		p.onStepChanged()
	}
}
