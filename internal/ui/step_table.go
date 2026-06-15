package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/runner"
)

type stepTableCell struct {
	widget.BaseWidget

	label          *widget.Label
	labelBox       fyne.CanvasObject
	dot            *canvas.Circle
	dotBox         fyne.CanvasObject
	box            *fyne.Container
	onSecondaryTap func(e *fyne.PointEvent)
	row            int
	panel          *stepTablePanel
}

func newStepTableCell() *stepTableCell {
	c := &stepTableCell{}
	c.ExtendBaseWidget(c)

	c.label = widget.NewLabel("")
	c.label.Truncation = fyne.TextTruncateEllipsis
	c.label.Wrapping = fyne.TextWrapOff
	c.labelBox = container.NewStack(c.label)

	c.dot = canvas.NewCircle(color.Transparent)
	c.dotBox = container.NewGridWrap(fyne.NewSize(8, 8), c.dot)

	c.box = container.NewBorder(nil, nil, c.dotBox, nil, c.labelBox)
	return c
}

func (c *stepTableCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.box)
}

func (c *stepTableCell) MinSize() fyne.Size {
	return c.box.MinSize()
}

func (c *stepTableCell) Tapped(e *fyne.PointEvent) {
	if c.panel != nil {
		c.panel.onCellTapped(c.row)
	}
}

func (c *stepTableCell) TappedSecondary(e *fyne.PointEvent) {
	if c.onSecondaryTap != nil {
		c.onSecondaryTap(e)
	}
}

func (c *stepTableCell) setText(text string) {
	c.label.SetText(text)
	c.label.Refresh()
}

func (c *stepTableCell) setDotColor(clr color.Color) {
	if clr == nil {
		c.dot.FillColor = color.Transparent
		c.dot.Refresh()
		return
	}
	c.dot.FillColor = clr
	c.dot.Refresh()
}

func (c *stepTableCell) setItalic(italic bool) {
	c.label.TextStyle = fyne.TextStyle{Italic: italic}
	c.label.Refresh()
}

type stepTablePanel struct {
	app           *App
	currentFlow   *flow.Flow
	stepsData     []flow.Step
	widget        fyne.CanvasObject
	table         *widget.Table
	emptyState    fyne.CanvasObject
	tableArea     *fyne.Container
	selected      int
	statuses      []runner.Status
	selectedBar   *fyne.Container
	onStepChanged func()

	lastTapRow int
	lastTapAt  time.Time
}

var stepTableHeaders = []string{"#", "状态", "启用", "步骤名称", "类型", "目标", "输入/期望", "等待", "失败处理"}
var stepTableWidths = []float32{44, 72, 56, 180, 120, 260, 220, 80, 110}

func newStepTablePanel(app *App, onStepChanged func()) *stepTablePanel {
	p := &stepTablePanel{app: app, selected: -1, onStepChanged: onStepChanged, lastTapRow: -1}
	p.initTable()

	addBtn := widget.NewButtonWithIcon("新增步骤", theme.ContentAddIcon(), func() { p.showAddStepDialog() })
	addBtn.Importance = widget.HighImportance

	copyBtn := widget.NewButtonWithIcon("复制", theme.ContentCopyIcon(), func() { p.copyStep() })
	copyBtn.Importance = widget.MediumImportance
	upBtn := widget.NewButtonWithIcon("上移", theme.MoveUpIcon(), func() { p.moveStep(-1) })
	downBtn := widget.NewButtonWithIcon("下移", theme.MoveDownIcon(), func() { p.moveStep(1) })
	delBtn := widget.NewButtonWithIcon("删除", theme.DeleteIcon(), func() { p.confirmDeleteStep() })
	delBtn.Importance = widget.DangerImportance

	p.selectedBar = container.NewHBox(
		newMutedText("已选中第 -- 步"),
		copyBtn,
		upBtn,
		downBtn,
		delBtn,
	)
	p.selectedBar.Hide()
	p.updateSelectedLabel()

	addBtnEmpty := widget.NewButtonWithIcon("新增步骤", theme.ContentAddIcon(), func() { p.showAddStepDialog() })
	addBtnEmpty.Importance = widget.HighImportance
	p.emptyState = newEmptyState(
		"当前流程暂无步骤",
		"点击「新增步骤」开始编排",
		addBtnEmpty,
	)
	p.tableArea = container.NewStack(p.table, p.emptyState)
	p.updateTableVisibility()

	p.widget = container.NewBorder(
		container.NewVBox(
			newSectionHeader("步骤编排", addBtn),
			p.selectedBar,
		),
		nil, nil, nil,
		p.tableArea,
	)
	return p
}

const doubleClickWindow = 500 * time.Millisecond

func (p *stepTablePanel) initTable() {
	cols := len(stepTableHeaders)
	p.table = widget.NewTableWithHeaders(
		func() (int, int) {
			if p.currentFlow == nil {
				return 0, cols
			}
			return len(p.stepsData), cols
		},
		func() fyne.CanvasObject {
			return newStepTableCell()
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			c := cell.(*stepTableCell)
			c.row = id.Row
			c.panel = p
			if id.Row < 0 || id.Row >= len(p.stepsData) {
				c.setText("")
				c.setDotColor(nil)
				c.onSecondaryTap = nil
				return
			}
			s := p.stepsData[id.Row]
			disabled := !s.Enabled

			switch id.Col {
			case 0:
				c.setText(fmt.Sprintf("%d", id.Row+1))
				c.setDotColor(nil)
			case 1:
				c.setText(statusText(p.statusForRow(id.Row)))
				c.setDotColor(statusColor(p.statusForRow(id.Row)))
			case 2:
				if s.Enabled {
					c.setText("启用")
				} else {
					c.setText("停用")
				}
				c.setDotColor(nil)
			case 3:
				c.setText(s.Name)
				c.setDotColor(nil)
			case 4:
				c.setText(stepTypeLabel(s.Type))
				c.setDotColor(nil)
			case 5:
				c.setText(s.Target.Value)
				c.setDotColor(nil)
			case 6:
				if s.Input.MaskInLogs {
					c.setText("***")
				} else {
					c.setText(s.Input.Text)
				}
				c.setDotColor(nil)
			case 7:
				c.setText(fmt.Sprintf("%dms", s.WaitAfterMs))
				c.setDotColor(nil)
			case 8:
				c.setText(errorPolicyLabel(s.OnError))
				c.setDotColor(nil)
			}
			c.setItalic(disabled)
			c.onSecondaryTap = func(e *fyne.PointEvent) {
				p.showStepContextMenu(id.Row, e)
			}
		},
	)
	p.table.CreateHeader = func() fyne.CanvasObject {
		l := widget.NewLabel("header")
		l.TextStyle = fyne.TextStyle{Bold: true}
		l.Wrapping = fyne.TextWrapOff
		return l
	}
	p.table.UpdateHeader = func(id widget.TableCellID, cell fyne.CanvasObject) {
		l := cell.(*widget.Label)
		if id.Row == -1 && id.Col >= 0 && id.Col < len(stepTableHeaders) {
			l.SetText(stepTableHeaders[id.Col])
		} else {
			l.SetText("")
		}
	}
	for i, w := range stepTableWidths {
		p.table.SetColumnWidth(i, w)
	}
	p.table.ShowHeaderColumn = false
	p.table.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(p.stepsData) {
			return
		}
		p.selected = id.Row
		p.updateSelectedActions()
	}
}

func (p *stepTablePanel) onCellTapped(row int) {
	if row < 0 || row >= len(p.stepsData) {
		return
	}
	now := time.Now()
	if row == p.lastTapRow && now.Sub(p.lastTapAt) < doubleClickWindow {
		p.lastTapRow = -1
		p.app.onStepSelected(&p.stepsData[row], row)
		return
	}
	p.lastTapRow = row
	p.lastTapAt = now
	p.table.UnselectAll()
	p.table.Select(widget.TableCellID{Row: row, Col: 0})
}

func (p *stepTablePanel) statusForRow(row int) runner.Status {
	if row < 0 || row >= len(p.statuses) {
		return runner.StatusPending
	}
	return p.statuses[row]
}

func statusText(st runner.Status) string {
	switch st {
	case runner.StatusRunning:
		return "运行中"
	case runner.StatusSuccess:
		return "成功"
	case runner.StatusFailed:
		return "失败"
	case runner.StatusSkipped:
		return "跳过"
	default:
		return ""
	}
}

func statusColor(st runner.Status) color.Color {
	switch st {
	case runner.StatusRunning:
		return uiColorInfo()
	case runner.StatusSuccess:
		return uiColorSuccess()
	case runner.StatusFailed:
		return uiColorDanger()
	case runner.StatusSkipped:
		return uiColorMutedText()
	default:
		return nil
	}
}

func (p *stepTablePanel) updateTableVisibility() {
	if p.tableArea == nil || p.emptyState == nil {
		return
	}
	if len(p.stepsData) == 0 {
		p.table.Hide()
		p.emptyState.Show()
	} else {
		p.emptyState.Hide()
		p.table.Show()
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
	p.lastTapRow = -1
	p.statuses = nil
	p.updateSelectedActions()
	p.updateTableVisibility()
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
			p.lastTapRow = -1
			p.updateTableVisibility()
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
	p.lastTapRow = -1
	p.updateSelectedActions()
	p.updateTableVisibility()
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
	p.lastTapRow = -1
	p.updateSelectedActions()
	p.table.Select(widget.TableCellID{Row: newIdx, Col: 0})
	p.table.Refresh()
	p.fireChanged()
}

func (p *stepTablePanel) updateSelectedActions() {
	if p.selectedBar == nil {
		return
	}
	p.updateSelectedLabel()
	if p.currentFlow != nil && p.selected >= 0 && p.selected < len(p.stepsData) {
		p.selectedBar.Show()
	} else {
		p.selectedBar.Hide()
	}
}

func (p *stepTablePanel) updateSelectedLabel() {
	if p.selectedBar == nil {
		return
	}
	labelObj := p.selectedBar.Objects[0]
	if text, ok := labelObj.(*canvas.Text); ok {
		if p.selected >= 0 {
			text.Text = fmt.Sprintf("已选中第 %d 步", p.selected+1)
		} else {
			text.Text = "已选中第 -- 步"
		}
		text.Refresh()
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
	p.updateTableVisibility()
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
	p.lastTapRow = -1
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
