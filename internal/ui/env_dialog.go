package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
	"go-chrome/internal/db"
)

// showEnvManager opens the environment management dialog.
func (a *App) showEnvManager() {
	if a.envRepo == nil {
		dialog.ShowError(fmt.Errorf("环境管理不可用"), a.mainWin)
		return
	}

	list := widget.NewList(
		func() int {
			envs, _ := a.envRepo.List()
			return len(envs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("环境")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			envs, _ := a.envRepo.List()
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

	var currentEnvID string
	envs, _ := a.envRepo.List()
	for _, e := range envs {
		if e.IsActive {
			currentEnvID = e.ID
			break
		}
	}

	var varList *widget.List
	var currentVars []*db.EnvironmentVariable
	var currentVarID string

	refreshVars := func(envID string) {
		if envID == "" {
			currentVars = nil
		} else {
			currentVars, _ = a.envRepo.ListVars(envID)
		}
		currentVarID = ""
		if varList != nil {
			varList.Refresh()
		}
	}
	refreshEnvs := func() {
		list.Refresh()
		a.runPanel.refreshEnvironments()
		if a.historyPanel != nil {
			a.historyPanel.refreshFilters()
		}
	}
	envByID := func(id string) (*db.Environment, bool) {
		envs, _ := a.envRepo.List()
		for _, e := range envs {
			if e.ID == id {
				return e, true
			}
		}
		return nil, false
	}
	varByID := func(id string) (*db.EnvironmentVariable, bool) {
		for _, v := range currentVars {
			if v.ID == id {
				return v, true
			}
		}
		return nil, false
	}

	varList = widget.NewList(
		func() int { return len(currentVars) },
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewLabel("KEY"), widget.NewLabel("VALUE"))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(currentVars) {
				return
			}
			v := currentVars[id]
			box := item.(*fyne.Container)
			keyLabel := box.Objects[0].(*widget.Label)
			valLabel := box.Objects[1].(*widget.Label)
			keyLabel.SetText(v.Key)
			if v.IsSecret {
				valLabel.SetText("******")
			} else {
				valLabel.SetText(v.Value)
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		envs, _ := a.envRepo.List()
		if id >= 0 && id < len(envs) {
			currentEnvID = envs[id].ID
			refreshVars(currentEnvID)
		}
	}
	varList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(currentVars) {
			currentVarID = currentVars[id].ID
		}
	}

	newEnvBtn := widget.NewButton("新建环境", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("环境名称")
		dialog.ShowForm("新建环境", "创建", "取消", []*widget.FormItem{
			widget.NewFormItem("名称", nameEntry),
		}, func(ok bool) {
			if !ok || nameEntry.Text == "" {
				return
			}
			e := &db.Environment{
				ID:   uuid.New().String(),
				Name: nameEntry.Text,
			}
			if err := a.envRepo.Save(e); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			currentEnvID = e.ID
			refreshEnvs()
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	renameEnvBtn := widget.NewButton("重命名", func() {
		env, ok := envByID(currentEnvID)
		if !ok {
			dialog.ShowInformation("提示", "请先选择一个环境", a.mainWin)
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
			if err := a.envRepo.Save(env); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			refreshEnvs()
		}, a.mainWin)
	})

	copyEnvBtn := widget.NewButton("复制环境", func() {
		env, ok := envByID(currentEnvID)
		if !ok {
			dialog.ShowInformation("提示", "请先选择一个环境", a.mainWin)
			return
		}
		copyEnv := *env
		copyEnv.ID = uuid.New().String()
		copyEnv.Name = env.Name + " 副本"
		copyEnv.IsActive = false
		if err := a.envRepo.Save(&copyEnv); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		vars, _ := a.envRepo.ListVars(env.ID)
		for _, oldVar := range vars {
			newVar := *oldVar
			newVar.ID = uuid.New().String()
			newVar.EnvironmentID = copyEnv.ID
			if err := a.envRepo.SaveVar(&newVar); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
		}
		currentEnvID = copyEnv.ID
		refreshEnvs()
		refreshVars(currentEnvID)
	})

	deleteEnvBtn := widget.NewButton("删除环境", func() {
		env, ok := envByID(currentEnvID)
		if !ok {
			dialog.ShowInformation("提示", "请先选择一个环境", a.mainWin)
			return
		}
		dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除环境 [%s] 吗？", env.Name), func(ok bool) {
			if !ok {
				return
			}
			if err := a.envRepo.Delete(env.ID); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			if env.IsActive {
				envs, _ := a.envRepo.List()
				if len(envs) > 0 {
					_ = a.envRepo.SetActive(envs[0].ID)
					currentEnvID = envs[0].ID
				} else {
					_ = a.envRepo.CreateDefaultIfNone()
					envs, _ = a.envRepo.List()
					if len(envs) > 0 {
						currentEnvID = envs[0].ID
					}
				}
			} else {
				currentEnvID = ""
			}
			refreshEnvs()
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	importEnvBtn := widget.NewButton("导入配置", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			if err := a.envRepo.Import(reader.URI().Path()); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			envs, _ := a.envRepo.List()
			currentEnvID = ""
			for _, env := range envs {
				if env.IsActive {
					currentEnvID = env.ID
					break
				}
			}
			refreshEnvs()
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	exportEnvBtn := widget.NewButton("导出配置", func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()
			if err := a.envRepo.Export(writer.URI().Path()); err != nil {
				dialog.ShowError(err, a.mainWin)
			}
		}, a.mainWin)
	})

	setActiveBtn := widget.NewButton("设为当前", func() {
		if currentEnvID != "" {
			if err := a.envRepo.SetActive(currentEnvID); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			refreshEnvs()
		}
	})

	newVarBtn := widget.NewButton("新增变量", func() {
		if currentEnvID == "" {
			dialog.ShowInformation("提示", "请先选择一个环境", a.mainWin)
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
				EnvironmentID: currentEnvID,
				Key:           key,
				Value:         valEntry.Text,
				IsSecret:      secretCheck.Checked,
			}
			if err := a.envRepo.SaveVar(v); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	editVarBtn := widget.NewButton("编辑变量", func() {
		v, ok := varByID(currentVarID)
		if !ok {
			dialog.ShowInformation("提示", "请先选择一个变量", a.mainWin)
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
			if err := a.envRepo.SaveVar(v); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	deleteVarBtn := widget.NewButton("删除变量", func() {
		v, ok := varByID(currentVarID)
		if !ok {
			dialog.ShowInformation("提示", "请先选择一个变量", a.mainWin)
			return
		}
		dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除变量 [%s] 吗？", v.Key), func(ok bool) {
			if !ok {
				return
			}
			if err := a.envRepo.DeleteVar(v.ID); err != nil {
				dialog.ShowError(err, a.mainWin)
				return
			}
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	left := container.NewBorder(
		container.NewVBox(widget.NewLabel("环境列表"), newEnvBtn, renameEnvBtn, copyEnvBtn, deleteEnvBtn, setActiveBtn, importEnvBtn, exportEnvBtn),
		nil, nil, nil,
		list,
	)
	right := container.NewBorder(
		container.NewVBox(widget.NewLabel("环境变量"), newVarBtn, editVarBtn, deleteVarBtn),
		nil, nil, nil,
		varList,
	)

	content := container.NewHSplit(left, right)
	content.SetOffset(0.4)

	d := dialog.NewCustom("环境管理", "关闭", content, a.mainWin)
	d.Resize(fyne.NewSize(640, 480))
	d.Show()
}
