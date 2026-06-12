package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
	"go-chrome/internal/db"
)

// envPanel is the main environment configuration tab.
type envPanel struct {
	app          *App
	widget       fyne.CanvasObject
	list         *widget.List
	varList      *widget.List
	search       *widget.Entry
	currentEnvID string
	currentVarID string
	currentVars  []*db.EnvironmentVariable
}

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
			return widget.NewLabel("环境")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			envs := p.filteredEnvs()
			if id < 0 || id >= len(envs) {
				return
			}
			e := envs[id]
			label := item.(*widget.Label)
			if e.IsActive {
				label.SetText(e.Name + " [当前]")
			} else {
				label.SetText(e.Name)
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

	p.varList = widget.NewList(
		func() int { return len(p.currentVars) },
		func() fyne.CanvasObject {
			return container.NewGridWithColumns(4,
				widget.NewLabel("KEY"),
				widget.NewLabel("VALUE"),
				widget.NewLabel("敏感"),
				widget.NewLabel("说明"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.currentVars) {
				return
			}
			v := p.currentVars[id]
			grid := item.(*fyne.Container)
			keyLabel := grid.Objects[0].(*widget.Label)
			valLabel := grid.Objects[1].(*widget.Label)
			secretLabel := grid.Objects[2].(*widget.Label)
			descLabel := grid.Objects[3].(*widget.Label)
			keyLabel.SetText(v.Key)
			if v.IsSecret {
				valLabel.SetText("******")
				secretLabel.SetText("是")
			} else {
				valLabel.SetText(v.Value)
				secretLabel.SetText("")
			}
			descLabel.SetText(v.Description)
		},
	)
	p.varList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(p.currentVars) {
			p.currentVarID = p.currentVars[id].ID
		}
	}

	newEnvBtn := widget.NewButtonWithIcon("新建环境", theme.ContentAddIcon(), func() {
		p.showNewEnvDialog()
	})
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
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItemWithIcon("导入配置", theme.DownloadIcon(), func() { p.showImportEnvDialog() }),
			fyne.NewMenuItemWithIcon("导出配置", theme.UploadIcon(), func() { p.showExportEnvDialog() }),
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, envMoreBtn.Size().Height), envMoreBtn)
	})

	newVarBtn := widget.NewButtonWithIcon("新增变量", theme.ContentAddIcon(), func() {
		p.showNewVarDialog()
	})
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

	leftTop := container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("环境列表", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			newEnvBtn,
			envMoreBtn,
		),
		p.search,
	)
	left := container.NewBorder(leftTop, nil, nil, nil, container.NewScroll(p.list))

	rightTop := container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("环境变量", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			newVarBtn,
			varMoreBtn,
		),
		container.NewGridWithColumns(4,
			widget.NewLabelWithStyle("KEY", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("VALUE", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("敏感", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("说明", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
	)
	right := container.NewBorder(rightTop, nil, nil, nil, container.NewScroll(p.varList))

	split := container.NewHSplit(left, right)
	split.SetOffset(0.4)
	p.widget = split

	p.refresh()
	return p
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
	if p.varList != nil {
		p.varList.Refresh()
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
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("环境名称")
	dialog.ShowForm("新建环境", "创建", "取消", []*widget.FormItem{
		widget.NewFormItem("名称", nameEntry),
	}, func(ok bool) {
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
	nameEntry := widget.NewEntry()
	nameEntry.SetText(env.Name)
	descEntry := widget.NewEntry()
	descEntry.SetText(env.Description)
	dialog.ShowForm("编辑环境", "保存", "取消", []*widget.FormItem{
		widget.NewFormItem("名称", nameEntry),
		widget.NewFormItem("说明", descEntry),
	}, func(ok bool) {
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
	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除环境 [%s] 吗？", env.Name), func(ok bool) {
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
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
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
	}, p.app.mainWin)
}

func (p *envPanel) showExportEnvDialog() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()
		if err := p.app.envRepo.Export(writer.URI().Path()); err != nil {
			dialog.ShowError(err, p.app.mainWin)
		}
	}, p.app.mainWin)
}

func (p *envPanel) showNewVarDialog() {
	if p.currentEnvID == "" {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder("变量名")
	keyEntry.OnChanged = func(s string) {
		upper := strings.ToUpper(s)
		if s != upper {
			keyEntry.SetText(upper)
		}
	}
	valEntry := widget.NewEntry()
	valEntry.SetPlaceHolder("变量值")
	secretCheck := widget.NewCheck("敏感变量", nil)
	dialog.ShowForm("新增变量", "添加", "取消", []*widget.FormItem{
		widget.NewFormItem("变量名", keyEntry),
		widget.NewFormItem("变量值", valEntry),
		widget.NewFormItem("", secretCheck),
	}, func(ok bool) {
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
	keyEntry := widget.NewEntry()
	keyEntry.SetText(v.Key)
	keyEntry.OnChanged = func(s string) {
		upper := strings.ToUpper(s)
		if s != upper {
			keyEntry.SetText(upper)
		}
	}
	valEntry := widget.NewEntry()
	valEntry.SetText(v.Value)
	descEntry := widget.NewEntry()
	descEntry.SetText(v.Description)
	secretCheck := widget.NewCheck("敏感变量", nil)
	secretCheck.SetChecked(v.IsSecret)
	dialog.ShowForm("编辑变量", "保存", "取消", []*widget.FormItem{
		widget.NewFormItem("变量名", keyEntry),
		widget.NewFormItem("变量值", valEntry),
		widget.NewFormItem("说明", descEntry),
		widget.NewFormItem("", secretCheck),
	}, func(ok bool) {
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
	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除变量 [%s] 吗？", v.Key), func(ok bool) {
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
