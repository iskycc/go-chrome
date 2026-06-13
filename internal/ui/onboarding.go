package ui

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) firstRunCheck() {
	if !a.browserMgr.IsInstalled() {
		a.runPanel.log("未检测到 Chrome，请点击「启动浏览器」下载")
	} else {
		a.runPanel.log("已检测到本地 Chrome")
	}
	if _, err := os.Stat(a.dirs.DataDir); err != nil {
		a.runPanel.log("数据目录检查失败：" + err.Error())
	} else {
		a.runPanel.log("数据目录就绪：" + a.dirs.DataDir)
	}
	if _, err := os.Stat(a.dirs.ConfigPath); err != nil {
		a.runPanel.log("配置文件检查失败：" + err.Error())
	} else {
		a.runPanel.log("配置文件就绪：" + a.dirs.ConfigPath)
	}
}

func (a *App) buildEmptyState() fyne.CanvasObject {
	newBtn := widget.NewButtonWithIcon("新建空白流程", theme.ContentAddIcon(), func() {
		a.createNewFlow()
	})
	newBtn.Importance = widget.HighImportance
	templateBtn := widget.NewButtonWithIcon("从模板创建", theme.ListIcon(), func() {
		a.showTemplatePickerDialog()
	})

	return newEmptyState(
		"暂无流程",
		"点击「新建」创建空白流程，或从内置模板开始",
		container.NewHBox(newBtn, templateBtn),
	)
}
