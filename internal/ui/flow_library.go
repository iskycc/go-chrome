package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

// flowListItem is a two-line list cell for the flow library: name on top and
// metadata (description / tags / step count) below.
type flowListItem struct {
	widget.BaseWidget

	name           *widget.Label
	meta           *widget.Label
	box            *fyne.Container
	onSecondaryTap func(e *fyne.PointEvent)
	onDoubleTap    func()
}

func newFlowListItem() *flowListItem {
	item := &flowListItem{}
	item.ExtendBaseWidget(item)

	item.name = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	item.name.Truncation = fyne.TextTruncateEllipsis

	item.meta = widget.NewLabel("")
	item.meta.Truncation = fyne.TextTruncateEllipsis

	inner := container.NewVBox(item.name, item.meta)
	item.box = container.NewPadded(inner)
	return item
}

func (item *flowListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(item.box)
}

func (item *flowListItem) TappedSecondary(e *fyne.PointEvent) {
	if item.onSecondaryTap != nil {
		item.onSecondaryTap(e)
	}
}

func (item *flowListItem) DoubleTapped(_ *fyne.PointEvent) {
	if item.onDoubleTap != nil {
		item.onDoubleTap()
	}
}

func (item *flowListItem) MinSize() fyne.Size {
	return item.box.MinSize().Add(fyne.NewSize(0, theme.Padding()))
}

func (item *flowListItem) setFlow(f *flow.Flow) {
	item.name.SetText(f.Name)
	item.meta.SetText(flowMetaText(f))
}

func flowMetaText(f *flow.Flow) string {
	parts := []string{}

	desc := strings.TrimSpace(f.Description)
	if desc != "" {
		parts = append(parts, desc)
	} else {
		parts = append(parts, "无描述")
	}

	if len(f.Tags) > 0 {
		parts = append(parts, strings.Join(f.Tags, " · "))
	}

	parts = append(parts, fmt.Sprintf("%d 个步骤", len(f.Steps)))
	return strings.Join(parts, " · ")
}

type flowLibraryPanel struct {
	app           *App
	flows         []*flow.Flow
	list          *widget.List
	search        *widget.Entry
	tagFilter     *widget.Select
	selectedIndex int
	widget        fyne.CanvasObject
}

func newFlowLibraryPanel(app *App) *flowLibraryPanel {
	p := &flowLibraryPanel{app: app}
	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("搜索流程...")

	p.list = widget.NewList(
		func() int { return len(p.flows) },
		func() fyne.CanvasObject {
			return newFlowListItem()
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.flows) {
				return
			}
			f := p.flows[id]
			cell := item.(*flowListItem)
			cell.setFlow(f)
			cell.onSecondaryTap = func(e *fyne.PointEvent) {
				p.showFlowContextMenu(int(id), e)
			}
			cell.onDoubleTap = func() {
				p.openFlowAtIndex(int(id))
			}
		},
	)
	// Single click only selects; double-click opens the flow for editing.
	p.list.OnSelected = func(id widget.ListItemID) {
		p.selectedIndex = int(id)
	}
	p.list.OnUnselected = func(id widget.ListItemID) { p.selectedIndex = -1 }

	p.search.OnChanged = func(s string) { p.filter() }
	p.tagFilter = widget.NewSelect([]string{"全部标签"}, func(s string) { p.filter() })
	p.tagFilter.SetSelected("全部标签")

	newBtn := widget.NewButtonWithIcon("新建", theme.ContentAddIcon(), func() { p.app.createNewFlow() })
	newBtn.Importance = widget.HighImportance

	var moreBtn *widget.Button
	moreBtn = widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() {
		hasSelection := p.selectedIndex >= 0 && p.selectedIndex < len(p.flows)
		exportItem := fyne.NewMenuItemWithIcon("导出当前流程", theme.UploadIcon(), func() { p.app.exportFlow() })
		cloneItem := fyne.NewMenuItemWithIcon("复制当前流程", theme.ContentCopyIcon(), func() {
			if p.selectedIndex >= 0 && p.selectedIndex < len(p.flows) {
				p.app.onFlowClone(p.flows[p.selectedIndex])
			}
		})
		deleteItem := fyne.NewMenuItemWithIcon("删除当前流程", theme.DeleteIcon(), func() {
			if p.selectedIndex >= 0 && p.selectedIndex < len(p.flows) {
				p.app.onFlowDelete(p.flows[p.selectedIndex])
			}
		})
		exportItem.Disabled = !hasSelection
		cloneItem.Disabled = !hasSelection
		deleteItem.Disabled = !hasSelection
		menu := fyne.NewMenu("流程操作",
			fyne.NewMenuItemWithIcon("新建流程", theme.ContentAddIcon(), func() { p.app.createNewFlow() }),
			fyne.NewMenuItemWithIcon("从模板创建", theme.ListIcon(), func() { p.app.showTemplatePickerDialog() }),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItemWithIcon("导入流程", theme.DownloadIcon(), func() { p.app.importFlow() }),
			fyne.NewMenuItemSeparator(),
			exportItem,
			cloneItem,
			deleteItem,
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, moreBtn.Size().Height), moreBtn)
	})

	filters := container.NewVBox(
		p.search,
		container.NewGridWrap(fyne.NewSize(180, p.tagFilter.MinSize().Height), p.tagFilter),
	)

	top := container.NewVBox(
		newSectionHeader("流程库", newBtn, moreBtn),
		filters,
	)

	p.widget = container.NewBorder(
		top,
		nil, nil, nil,
		p.list,
	)
	return p
}

func (p *flowLibraryPanel) setFlows(flows []*flow.Flow) {
	p.flows = flows
	p.refreshTags()
	p.list.Refresh()
}

func (p *flowLibraryPanel) refreshTags() {
	tagSet := map[string]bool{"全部标签": true}
	for _, f := range p.flows {
		for _, t := range f.Tags {
			tagSet[t] = true
		}
	}
	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	selected := p.tagFilter.Selected
	p.tagFilter.Options = tags
	if selected != "" {
		p.tagFilter.SetSelected(selected)
	} else {
		p.tagFilter.SetSelected("全部标签")
	}
}

func (p *flowLibraryPanel) filter() {
	query := strings.ToLower(strings.TrimSpace(p.search.Text))
	selectedTag := p.tagFilter.Selected
	if selectedTag == "全部标签" {
		selectedTag = ""
	}

	allFlows, _ := p.app.flowStore.ListSorted()
	var results []*flow.Flow
	for _, f := range allFlows {
		if selectedTag != "" {
			hasTag := false
			for _, t := range f.Tags {
				if t == selectedTag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		if query == "" {
			results = append(results, f)
			continue
		}
		if strings.Contains(strings.ToLower(f.Name), query) || strings.Contains(strings.ToLower(f.Description), query) {
			results = append(results, f)
			continue
		}
		for _, t := range f.Tags {
			if strings.Contains(strings.ToLower(t), query) {
				results = append(results, f)
				break
			}
		}
	}
	p.flows = results
	p.list.Refresh()
}

func (p *flowLibraryPanel) refresh() {
	p.filter()
}

func (p *flowLibraryPanel) selectFlow(id string) bool {
	for i, f := range p.flows {
		if f.ID == id {
			p.list.Select(i)
			return true
		}
	}
	return false
}

func (p *flowLibraryPanel) openFlowAtIndex(idx int) {
	if idx < 0 || idx >= len(p.flows) {
		return
	}
	p.list.Select(idx)
	f := p.flows[idx]
	loaded, err := p.app.flowStore.Load(f.ID)
	if err != nil {
		p.app.runPanel.log("读取流程失败: " + err.Error())
		p.app.onFlowSelected(f)
		return
	}
	p.app.onFlowSelected(loaded)
}

func (p *flowLibraryPanel) showFlowContextMenu(idx int, e *fyne.PointEvent) {
	if idx < 0 || idx >= len(p.flows) {
		return
	}
	p.list.Select(idx)
	f := p.flows[idx]

	openItem := fyne.NewMenuItem("编辑流程", func() {
		p.openFlowAtIndex(idx)
	})
	runItem := fyne.NewMenuItem("运行此流程", func() {
		p.list.Select(idx)
		p.app.runCurrentFlow()
	})
	shortcutItem := fyne.NewMenuItem("生成桌面快捷方式", func() {
		p.app.showCreateShortcutDialogForFlow(f)
	})
	cloneItem := fyne.NewMenuItem("复制流程", func() {
		p.app.onFlowClone(f)
	})
	exportItem := fyne.NewMenuItem("导出流程", func() {
		p.list.Select(idx)
		p.app.exportFlow()
	})
	copyNameItem := fyne.NewMenuItem("复制流程名称", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(f.Name))
		p.app.runPanel.log("流程名称已复制到剪贴板")
	})
	copyIDItem := fyne.NewMenuItem("复制流程 ID", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(f.ID))
		p.app.runPanel.log("流程 ID 已复制到剪贴板")
	})
	deleteItem := fyne.NewMenuItem("删除流程", func() {
		p.app.onFlowDelete(f)
	})
	deleteItem.IsQuit = true

	menu := fyne.NewMenu("流程操作",
		openItem,
		runItem,
		shortcutItem,
		fyne.NewMenuItemSeparator(),
		cloneItem,
		exportItem,
		copyNameItem,
		copyIDItem,
		fyne.NewMenuItemSeparator(),
		deleteItem,
	)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}
