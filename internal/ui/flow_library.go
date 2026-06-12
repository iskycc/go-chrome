package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

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
			return widget.NewLabel("Flow Name")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.flows) {
				return
			}
			f := p.flows[id]
			name := f.Name
			if len(name) > 24 {
				name = name[:21] + "..."
			}
			tags := ""
			if len(f.Tags) > 0 {
				ts := strings.Join(f.Tags, ", ")
				if len(ts) > 20 {
					ts = ts[:17] + "..."
				}
				tags = " [" + ts + "]"
			}
			item.(*widget.Label).SetText(name + tags)
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		p.selectedIndex = int(id)
		if id >= 0 && id < len(p.flows) {
			f := p.flows[id]
			loaded, err := p.app.flowStore.Load(f.ID)
			if err != nil {
				p.app.runPanel.log("读取流程失败: " + err.Error())
				p.app.onFlowSelected(f)
				return
			}
			p.app.onFlowSelected(loaded)
		}
	}
	p.list.OnUnselected = func(id widget.ListItemID) { p.selectedIndex = -1 }

	p.search.OnChanged = func(s string) { p.filter() }
	p.tagFilter = widget.NewSelect([]string{"全部标签"}, func(s string) { p.filter() })
	p.tagFilter.SetSelected("全部标签")

	newBtn := widget.NewButtonWithIcon("新建", theme.ContentAddIcon(), func() { p.app.createNewFlow() })
	saveBtn := widget.NewButtonWithIcon("保存", theme.DocumentSaveIcon(), func() { p.app.saveCurrentFlow() })
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

	p.widget = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("流程库", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			p.search,
			p.tagFilter,
			container.NewHBox(newBtn, saveBtn, moreBtn),
		),
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
