package ui

import (
	"fmt"

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

	refreshVars := func(envID string) {
		if envID == "" {
			currentVars = nil
		} else {
			currentVars, _ = a.envRepo.ListVars(envID)
		}
		if varList != nil {
			varList.Refresh()
		}
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
			_ = a.envRepo.Save(e)
			list.Refresh()
		}, a.mainWin)
	})

	setActiveBtn := widget.NewButton("设为当前", func() {
		if currentEnvID != "" {
			_ = a.envRepo.SetActive(currentEnvID)
			list.Refresh()
		}
	})

	newVarBtn := widget.NewButton("新增变量", func() {
		if currentEnvID == "" {
			dialog.ShowInformation("提示", "请先选择一个环境", a.mainWin)
			return
		}
		keyEntry := widget.NewEntry()
		keyEntry.SetPlaceHolder("变量名")
		valEntry := widget.NewEntry()
		valEntry.SetPlaceHolder("变量值")
		secretCheck := widget.NewCheck("敏感变量", nil)
		dialog.ShowForm("新增变量", "添加", "取消", []*widget.FormItem{
			widget.NewFormItem("变量名", keyEntry),
			widget.NewFormItem("变量值", valEntry),
			widget.NewFormItem("", secretCheck),
		}, func(ok bool) {
			if !ok || keyEntry.Text == "" {
				return
			}
			v := &db.EnvironmentVariable{
				ID:            uuid.New().String(),
				EnvironmentID: currentEnvID,
				Key:           keyEntry.Text,
				Value:         valEntry.Text,
				IsSecret:      secretCheck.Checked,
			}
			_ = a.envRepo.SaveVar(v)
			refreshVars(currentEnvID)
		}, a.mainWin)
	})

	left := container.NewBorder(
		container.NewVBox(widget.NewLabel("环境列表"), newEnvBtn, setActiveBtn),
		nil, nil, nil,
		list,
	)
	right := container.NewBorder(
		container.NewVBox(widget.NewLabel("环境变量"), newVarBtn),
		nil, nil, nil,
		varList,
	)

	content := container.NewHSplit(left, right)
	content.SetOffset(0.4)

	d := dialog.NewCustom("环境管理", "关闭", content, a.mainWin)
	d.Resize(fyne.NewSize(640, 480))
	d.Show()
}
