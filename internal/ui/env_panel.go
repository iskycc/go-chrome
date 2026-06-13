package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
	"go-chrome/internal/db"
)

// envListItem is a two-line cell for the environment list.
type envListItem struct {
	widget.BaseWidget

	name *widget.Label
	meta *widget.Label
	box  *fyne.Container

	onSecondaryTap func(e *fyne.PointEvent)
}

func newEnvListItem() *envListItem {
	item := &envListItem{}
	item.ExtendBaseWidget(item)

	item.name = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	item.name.Truncation = fyne.TextTruncateEllipsis

	item.meta = widget.NewLabel("")
	item.meta.Truncation = fyne.TextTruncateEllipsis

	item.box = container.NewVBox(item.name, item.meta)
	return item
}

func (item *envListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(item.box)
}

func (item *envListItem) TappedSecondary(e *fyne.PointEvent) {
	if item.onSecondaryTap != nil {
		item.onSecondaryTap(e)
	}
}

func (item *envListItem) MinSize() fyne.Size {
	return item.box.MinSize().Add(fyne.NewSize(0, theme.Padding()))
}

func (item *envListItem) setEnv(e *db.Environment, varCount int) {
	name := e.Name
	if e.IsActive {
		name += " [当前]"
	}
	item.name.SetText(name)

	desc := strings.TrimSpace(e.Description)
	if desc == "" {
		desc = "无说明"
	}
	item.meta.SetText(fmt.Sprintf("%d 个变量 · %s", varCount, desc))
}

// envPanel is the main environment configuration tab.
type envPanel struct {
	app          *App
	widget       fyne.CanvasObject
	list         *widget.List
	varTable     *widget.Table
	search       *widget.Entry
	currentEnvID string
	currentVarID string
	currentVars  []*db.EnvironmentVariable
	emptyState   fyne.CanvasObject
}

var envVarHeaders = []string{"KEY", "VALUE", "类型", "说明"}
var envVarWidths = []float32{140, 180, 72, 200}

func newEnvPanel(app *App) *envPanel {
	p := &envPanel{app: app}

	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("搜索环境...")
	p.search.OnChanged = func(string) {
		p.list.Refresh()
		p.syncListSelection()
	}

	p.list = widget.NewList(
		func() int { return len(p.filteredEnvs()) },
		func() fyne.CanvasObject {
			return newEnvListItem()
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			envs := p.filteredEnvs()
			if id < 0 || id >= len(envs) {
				return
			}
			e := envs[id]
			cell := item.(*envListItem)
			cell.setEnv(e, p.envVarCount(e.ID))
			cell.onSecondaryTap = func(e *fyne.PointEvent) {
				p.showEnvContextMenu(int(id), e)
			}
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		envs := p.filteredEnvs()
		if id >= 0 && id < len(envs) {
			p.currentEnvID = envs[id].ID
			p.refreshVars()
		}
	}

	p.varTable = widget.NewTableWithHeaders(
		func() (int, int) { return len(p.currentVars), len(envVarHeaders) },
		func() fyne.CanvasObject { return newContextMenuLabel("cell", nil) },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*contextMenuLabel)
			if id.Row < 0 || id.Row >= len(p.currentVars) {
				label.SetText("")
				label.onSecondaryTap = nil
				return
			}
			v := p.currentVars[id.Row]
			switch id.Col {
			case 0:
				label.SetText(v.Key)
			case 1:
				if v.IsSecret {
					label.SetText("******")
				} else {
					label.SetText(v.Value)
				}
			case 2:
				if v.IsSecret {
					label.SetText("敏感")
				} else {
					label.SetText("普通")
				}
			case 3:
				label.SetText(v.Description)
			}
			label.onSecondaryTap = func(e *fyne.PointEvent) {
				p.showVarContextMenu(id.Row, e)
			}
		},
	)
	p.varTable.ShowHeaderColumn = false
	p.varTable.CreateHeader = func() fyne.CanvasObject {
		l := widget.NewLabel("HEADER")
		l.TextStyle = fyne.TextStyle{Bold: true}
		return l
	}
	p.varTable.UpdateHeader = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		if id.Row == -1 {
			if id.Col >= 0 && id.Col < len(envVarHeaders) {
				label.SetText(envVarHeaders[id.Col])
			} else {
				label.SetText("")
			}
		} else {
			label.SetText("")
		}
	}
	for i, w := range envVarWidths {
		p.varTable.SetColumnWidth(i, w)
	}
	p.varTable.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(p.currentVars) {
			return
		}
		p.currentVarID = p.currentVars[id.Row].ID
		p.showEditVarDialog()
	}

	newEnvBtn := widget.NewButtonWithIcon("新建环境", theme.ContentAddIcon(), func() {
		p.showNewEnvDialog()
	})
	newEnvBtn.Importance = widget.HighImportance
	var envMoreBtn *widget.Button
	envMoreBtn = widget.NewButtonWithIcon("环境操作", theme.MoreHorizontalIcon(), func() {
		hasEnv := p.currentEnvID != ""
		renameItem := fyne.NewMenuItemWithIcon("重命名 / 说明", theme.DocumentCreateIcon(), func() { p.showRenameEnvDialog() })
		copyItem := fyne.NewMenuItemWithIcon("复制环境", theme.ContentCopyIcon(), func() { p.showCopyEnvDialog() })
		deleteItem := fyne.NewMenuItemWithIcon("删除环境", theme.DeleteIcon(), func() { p.showDeleteEnvDialog() })
		activeItem := fyne.NewMenuItemWithIcon("设为当前", theme.ConfirmIcon(), func() { p.showSetActiveEnvDialog() })
		renameItem.Disabled = !hasEnv
		copyItem.Disabled = !hasEnv
		deleteItem.Disabled = !hasEnv
		activeItem.Disabled = !hasEnv
		menu := fyne.NewMenu("环境操作",
			renameItem,
			copyItem,
			deleteItem,
			activeItem,
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, envMoreBtn.Size().Height), envMoreBtn)
	})

	newVarBtn := widget.NewButtonWithIcon("新增变量", theme.ContentAddIcon(), func() {
		p.showNewVarDialog()
	})
	newVarBtn.Importance = widget.HighImportance
	var varMoreBtn *widget.Button
	varMoreBtn = widget.NewButtonWithIcon("变量操作", theme.MoreHorizontalIcon(), func() {
		hasVar := p.currentVarID != ""
		editItem := fyne.NewMenuItemWithIcon("编辑变量", theme.DocumentCreateIcon(), func() { p.showEditVarDialog() })
		deleteItem := fyne.NewMenuItemWithIcon("删除变量", theme.DeleteIcon(), func() { p.showDeleteVarDialog() })
		editItem.Disabled = !hasVar
		deleteItem.Disabled = !hasVar
		menu := fyne.NewMenu("变量操作", editItem, deleteItem)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, varMoreBtn.Size().Height), varMoreBtn)
	})

	importBtn := widget.NewButtonWithIcon("导入配置", theme.DownloadIcon(), func() {
		p.showImportEnvDialog()
	})
	exportBtn := widget.NewButtonWithIcon("导出配置", theme.UploadIcon(), func() {
		p.showExportEnvDialog()
	})

	newVarBtnEmpty := widget.NewButtonWithIcon("新增变量", theme.ContentAddIcon(), func() {
		p.showNewVarDialog()
	})
	newVarBtnEmpty.Importance = widget.HighImportance
	importBtnEmpty := widget.NewButtonWithIcon("导入配置", theme.DownloadIcon(), func() {
		p.showImportEnvDialog()
	})
	p.emptyState = newEmptyState(
		"当前环境暂无变量",
		"点击新增变量或导入配置文件",
		container.NewHBox(newVarBtnEmpty, importBtnEmpty),
	)

	leftTop := container.NewVBox(
		newSectionHeader("环境列表", newEnvBtn, envMoreBtn),
		p.search,
	)
	left := container.NewBorder(leftTop, nil, nil, nil, container.NewScroll(p.list))

	rightTop := container.NewVBox(
		newSectionHeader("环境变量", newVarBtn, varMoreBtn),
		newSectionHeader("配置文件", importBtn, exportBtn),
	)
	rightContent := container.NewStack(p.varTable, p.emptyState)
	right := container.NewBorder(rightTop, nil, nil, nil, rightContent)

	split := container.NewHSplit(left, right)
	split.SetOffset(0.4)
	p.widget = split

	p.refresh()
	return p
}

func (p *envPanel) envVarCount(envID string) int {
	if p.app.envRepo == nil || envID == "" {
		return 0
	}
	vars, _ := p.app.envRepo.ListVars(envID)
	return len(vars)
}

func (p *envPanel) refresh() {
	if p.currentEnvID == "" {
		for _, e := range p.allEnvs() {
			if e.IsActive {
				p.currentEnvID = e.ID
				break
			}
		}
	}
	p.list.Refresh()
	p.syncListSelection()
	p.refreshVars()
	p.app.refreshEnvironmentSelectors()
	if p.app.historyPanel != nil {
		p.app.historyPanel.refreshFilters()
	}
}

func (p *envPanel) refreshVars() {
	if p.app.envRepo == nil {
		p.currentVars = nil
	} else if p.currentEnvID == "" {
		p.currentVars = nil
	} else {
		p.currentVars, _ = p.app.envRepo.ListVars(p.currentEnvID)
	}
	p.currentVarID = ""
	if p.varTable != nil {
		p.varTable.Refresh()
		p.varTable.UnselectAll()
	}
	if len(p.currentVars) == 0 {
		p.emptyState.Show()
		p.varTable.Hide()
	} else {
		p.emptyState.Hide()
		p.varTable.Show()
	}
}

func (p *envPanel) allEnvs() []*db.Environment {
	if p.app.envRepo == nil {
		return nil
	}
	envs, _ := p.app.envRepo.List()
	return envs
}

func (p *envPanel) filteredEnvs() []*db.Environment {
	envs := p.allEnvs()
	term := strings.TrimSpace(strings.ToLower(p.search.Text))
	if term == "" {
		return envs
	}
	filtered := make([]*db.Environment, 0, len(envs))
	for _, e := range envs {
		if strings.Contains(strings.ToLower(e.Name), term) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (p *envPanel) syncListSelection() {
	for i, e := range p.filteredEnvs() {
		if e.ID == p.currentEnvID {
			p.list.Select(i)
			return
		}
	}
	p.list.UnselectAll()
}

func (p *envPanel) showEnvContextMenu(idx int, e *fyne.PointEvent) {
	envs := p.filteredEnvs()
	if idx < 0 || idx >= len(envs) {
		return
	}
	p.list.Select(idx)
	env := envs[idx]
	p.currentEnvID = env.ID

	setActiveItem := fyne.NewMenuItem("设为当前环境", func() {
		p.showSetActiveEnvDialog()
	})
	renameItem := fyne.NewMenuItem("重命名 / 说明", func() {
		p.showRenameEnvDialog()
	})
	copyItem := fyne.NewMenuItem("复制环境", func() {
		p.showCopyEnvDialog()
	})
	exportItem := fyne.NewMenuItem("导出全部环境配置", func() {
		p.showExportEnvDialog()
	})
	copyNameItem := fyne.NewMenuItem("复制环境名称", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(env.Name))
		p.app.runPanel.log("环境名称已复制到剪贴板")
	})
	copyIDItem := fyne.NewMenuItem("复制环境 ID", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(env.ID))
		p.app.runPanel.log("环境 ID 已复制到剪贴板")
	})
	deleteItem := fyne.NewMenuItem("删除环境", func() {
		p.showDeleteEnvDialog()
	})
	deleteItem.IsQuit = true

	menu := fyne.NewMenu("环境操作",
		setActiveItem,
		renameItem,
		fyne.NewMenuItemSeparator(),
		copyItem,
		exportItem,
		copyNameItem,
		copyIDItem,
		fyne.NewMenuItemSeparator(),
		deleteItem,
	)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}

func (p *envPanel) showVarContextMenu(row int, e *fyne.PointEvent) {
	if row < 0 || row >= len(p.currentVars) {
		return
	}
	p.varTable.Select(widget.TableCellID{Row: row, Col: 0})
	p.currentVarID = p.currentVars[row].ID
	v := p.currentVars[row]

	editItem := fyne.NewMenuItem("编辑变量", func() {
		p.showEditVarDialog()
	})
	copyKeyItem := fyne.NewMenuItem("复制变量名", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(v.Key))
		p.app.runPanel.log("变量名已复制到剪贴板")
	})
	copyValueItem := fyne.NewMenuItem("复制变量值", func() {
		if v.IsSecret {
			showWrappedConfirm("复制敏感变量值", "该变量为敏感变量，复制将把明文写入剪贴板，是否继续？", "继续", "取消", fyne.NewSize(480, 180), func(ok bool) {
				if ok {
					p.app.fyneApp.Clipboard().SetContent(clipCopy(v.Value))
					p.app.runPanel.log("变量值已复制到剪贴板")
				}
			}, p.app.mainWin)
			return
		}
		p.app.fyneApp.Clipboard().SetContent(clipCopy(v.Value))
		p.app.runPanel.log("变量值已复制到剪贴板")
	})
	copyRefItem := fyne.NewMenuItem("复制 env 引用", func() {
		ref := fmt.Sprintf("${env:%s}", v.Key)
		p.app.fyneApp.Clipboard().SetContent(clipCopy(ref))
		p.app.runPanel.log("环境变量引用已复制到剪贴板：" + ref)
	})
	toggleSecretItem := fyne.NewMenuItem("标记为敏感", nil)
	if v.IsSecret {
		toggleSecretItem.Label = "取消敏感标记"
	}
	toggleSecretItem.Action = func() {
		v.IsSecret = !v.IsSecret
		if err := p.app.envRepo.SaveVar(v); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}
	deleteItem := fyne.NewMenuItem("删除变量", func() {
		p.showDeleteVarDialog()
	})
	deleteItem.IsQuit = true

	menu := fyne.NewMenu("变量操作",
		editItem,
		copyKeyItem,
		copyValueItem,
		copyRefItem,
		fyne.NewMenuItemSeparator(),
		toggleSecretItem,
		fyne.NewMenuItemSeparator(),
		deleteItem,
	)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}

func (p *envPanel) envByID(id string) (*db.Environment, bool) {
	for _, e := range p.allEnvs() {
		if e.ID == id {
			return e, true
		}
	}
	return nil, false
}

func (p *envPanel) varByID(id string) (*db.EnvironmentVariable, bool) {
	for _, v := range p.currentVars {
		if v.ID == id {
			return v, true
		}
	}
	return nil, false
}

func (p *envPanel) showNewEnvDialog() {
	if p.app.envRepo == nil {
		return
	}
	nameEntry := sizedEntry("环境名称")
	showSizedFormDialog("新建环境", "创建", "取消", []*widget.FormItem{
		widget.NewFormItem("名称", nameEntry),
	}, fyne.NewSize(480, 180), func(ok bool) {
		name := strings.TrimSpace(nameEntry.Text)
		if !ok || name == "" {
			return
		}
		e := &db.Environment{
			ID:   uuid.New().String(),
			Name: name,
		}
		if err := p.app.envRepo.Save(e); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.currentEnvID = e.ID
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) showRenameEnvDialog() {
	env, ok := p.envByID(p.currentEnvID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	nameEntry := sizedEntry("")
	nameEntry.SetText(env.Name)
	descEntry := sizedMultiLineEntry("环境说明", 3)
	descEntry.SetText(env.Description)
	showSizedFormDialog("编辑环境", "保存", "取消", []*widget.FormItem{
		widget.NewFormItem("名称", nameEntry),
		widget.NewFormItem("说明", descEntry),
	}, fyne.NewSize(560, 260), func(ok bool) {
		if !ok || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		env.Name = strings.TrimSpace(nameEntry.Text)
		env.Description = descEntry.Text
		if err := p.app.envRepo.Save(env); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) showCopyEnvDialog() {
	env, ok := p.envByID(p.currentEnvID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	copyEnv := *env
	copyEnv.ID = uuid.New().String()
	copyEnv.Name = env.Name + " 副本"
	copyEnv.IsActive = false
	if err := p.app.envRepo.Save(&copyEnv); err != nil {
		dialog.ShowError(err, p.app.mainWin)
		return
	}
	vars, _ := p.app.envRepo.ListVars(env.ID)
	for _, oldVar := range vars {
		newVar := *oldVar
		newVar.ID = uuid.New().String()
		newVar.EnvironmentID = copyEnv.ID
		if err := p.app.envRepo.SaveVar(&newVar); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
	}
	p.currentEnvID = copyEnv.ID
	p.refresh()
}

func (p *envPanel) showDeleteEnvDialog() {
	env, ok := p.envByID(p.currentEnvID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	msg := fmt.Sprintf("确定删除环境 [%s] 吗？", truncateForDialog(env.Name, 80))
	showWrappedConfirm("确认删除", msg, "删除", "取消", fyne.NewSize(520, 180), func(ok bool) {
		if !ok {
			return
		}
		if err := p.app.envRepo.Delete(env.ID); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		if env.IsActive {
			envs, _ := p.app.envRepo.List()
			if len(envs) > 0 {
				_ = p.app.envRepo.SetActive(envs[0].ID)
				p.currentEnvID = envs[0].ID
			} else {
				_ = p.app.envRepo.CreateDefaultIfNone()
				envs, _ = p.app.envRepo.List()
				if len(envs) > 0 {
					p.currentEnvID = envs[0].ID
				}
			}
		} else {
			p.currentEnvID = ""
		}
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) showSetActiveEnvDialog() {
	if p.currentEnvID == "" {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	if err := p.app.envRepo.SetActive(p.currentEnvID); err != nil {
		dialog.ShowError(err, p.app.mainWin)
		return
	}
	p.refresh()
}

func (p *envPanel) showImportEnvDialog() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			if err != nil {
				dialog.ShowError(err, p.app.mainWin)
			}
			return
		}
		defer reader.Close()
		if err := p.app.envRepo.Import(reader.URI().Path()); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		envs, _ := p.app.envRepo.List()
		p.currentEnvID = ""
		for _, env := range envs {
			if env.IsActive {
				p.currentEnvID = env.ID
				break
			}
		}
		p.refresh()
		dialog.ShowInformation("导入成功", "环境配置已导入。", p.app.mainWin)
	}, p.app.mainWin)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	resizeFileDialog(fd)
	fd.Show()
}

func (p *envPanel) showExportEnvDialog() {
	p.confirmExportIfNeeded(func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				if err != nil {
					dialog.ShowError(err, p.app.mainWin)
				}
				return
			}
			defer writer.Close()
			if err := p.app.envRepo.Export(writer.URI().Path()); err != nil {
				dialog.ShowError(err, p.app.mainWin)
			}
		}, p.app.mainWin)
		fd.SetFileName("go-chrome-env-config.json")
		resizeFileDialog(fd)
		fd.Show()
	})
}

func (p *envPanel) confirmExportIfNeeded(cont func()) {
	hasSecret := false
	if p.app.envRepo != nil {
		envs, _ := p.app.envRepo.List()
		for _, env := range envs {
			vars, _ := p.app.envRepo.ListVars(env.ID)
			for _, v := range vars {
				if v.IsSecret {
					hasSecret = true
					break
				}
			}
			if hasSecret {
				break
			}
		}
	}
	if !hasSecret {
		cont()
		return
	}
	msg := "导出文件将包含敏感变量的明文值。\n请确认该文件只保存在可信位置。"
	showWrappedConfirm("确认导出", msg, "继续导出", "取消", fyne.NewSize(520, 200), func(ok bool) {
		if ok {
			cont()
		}
	}, p.app.mainWin)
}

func forceUppercaseEntry(e *widget.Entry) {
	e.OnChanged = func(s string) {
		if up := strings.ToUpper(s); s != up {
			e.SetText(up)
		}
	}
}

func (p *envPanel) showNewVarDialog() {
	if p.currentEnvID == "" {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	keyEntry := sizedEntry("变量名")
	forceUppercaseEntry(keyEntry)
	valEntry := sizedMultiLineEntry("变量值（支持 URL、Token、JSON 等长文本）", 3)
	secretCheck := widget.NewCheck("敏感变量", nil)
	showSizedFormDialog("新增变量", "添加", "取消", []*widget.FormItem{
		widget.NewFormItem("变量名", keyEntry),
		widget.NewFormItem("变量值", valEntry),
		widget.NewFormItem("", secretCheck),
	}, fyne.NewSize(640, 300), func(ok bool) {
		key := strings.TrimSpace(keyEntry.Text)
		if !ok || key == "" {
			return
		}
		v := &db.EnvironmentVariable{
			ID:            uuid.New().String(),
			EnvironmentID: p.currentEnvID,
			Key:           key,
			Value:         valEntry.Text,
			IsSecret:      secretCheck.Checked,
		}
		if err := p.app.envRepo.SaveVar(v); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}, p.app.mainWin)
}

func (p *envPanel) showEditVarDialog() {
	v, ok := p.varByID(p.currentVarID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个变量", p.app.mainWin)
		return
	}
	keyEntry := sizedEntry("")
	keyEntry.SetText(v.Key)
	forceUppercaseEntry(keyEntry)
	valEntry := sizedMultiLineEntry("变量值", 3)
	valEntry.SetText(v.Value)
	descEntry := sizedMultiLineEntry("变量说明", 2)
	descEntry.SetText(v.Description)
	secretCheck := widget.NewCheck("敏感变量", nil)
	secretCheck.SetChecked(v.IsSecret)
	showSizedFormDialog("编辑变量", "保存", "取消", []*widget.FormItem{
		widget.NewFormItem("变量名", keyEntry),
		widget.NewFormItem("变量值", valEntry),
		widget.NewFormItem("说明", descEntry),
		widget.NewFormItem("", secretCheck),
	}, fyne.NewSize(680, 360), func(ok bool) {
		key := strings.TrimSpace(keyEntry.Text)
		if !ok || key == "" {
			return
		}
		v.Key = key
		v.Value = valEntry.Text
		v.Description = descEntry.Text
		v.IsSecret = secretCheck.Checked
		if err := p.app.envRepo.SaveVar(v); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}, p.app.mainWin)
}

func (p *envPanel) showDeleteVarDialog() {
	v, ok := p.varByID(p.currentVarID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个变量", p.app.mainWin)
		return
	}
	msg := fmt.Sprintf("确定删除变量 [%s] 吗？", truncateForDialog(v.Key, 80))
	showWrappedConfirm("确认删除", msg, "删除", "取消", fyne.NewSize(520, 180), func(ok bool) {
		if !ok {
			return
		}
		if err := p.app.envRepo.DeleteVar(v.ID); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}, p.app.mainWin)
}
