package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

func (a *App) firstRunCheck() {
	if !a.browserMgr.IsInstalled() {
		a.runPanel.log("未检测到 Chrome，请点击「启动浏览器」下载")
	}
}

func (a *App) buildEmptyState() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("暂无流程", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("点击左侧「新建」创建流程，或导入示例流程快速体验。")
	hint.Alignment = fyne.TextAlignCenter

	newBtn := widget.NewButtonWithIcon("新建流程", theme.ContentAddIcon(), func() {
		a.createNewFlow()
	})
	importBtn := widget.NewButtonWithIcon("导入示例", theme.DocumentIcon(), func() {
		example := flow.NewExampleLoginFlow()
		if err := a.flowStore.Save(example); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.refreshFlowList()
		a.flowLibrary.selectFlow(example.ID)
	})

	return container.NewCenter(container.NewVBox(
		theme.DocumentIcon(),
		title,
		hint,
		container.NewHBox(newBtn, importBtn),
	))
}
