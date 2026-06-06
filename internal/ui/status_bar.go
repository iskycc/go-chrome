package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/browser"
)

// SaveStatus tracks dirty state.
type SaveStatus int

const (
	SaveUnmodified SaveStatus = iota
	SaveDirty
	SaveSaving
	SaveSuccess
	SaveFailed
)

// RunStatus tracks execution state.
type RunStatus int

const (
	RunIdle RunStatus = iota
	RunRunning
	RunCompleted
	RunFailed
)

type statusBar struct {
	app         *App
	widget      fyne.CanvasObject
	flowLabel   *widget.Label
	saveLabel   *widget.Label
	chromeLabel *widget.Label
	runLabel    *widget.Label
}

func newStatusBar(app *App) *statusBar {
	sb := &statusBar{app: app}
	sb.flowLabel = widget.NewLabel("未选择流程")
	sb.saveLabel = widget.NewLabel("未修改")
	sb.chromeLabel = widget.NewLabel("未安装")
	sb.runLabel = widget.NewLabel("空闲")

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(20, 1))

	sb.widget = container.NewHBox(
		widget.NewLabelWithStyle("Go Chrome", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		spacer,
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.flowLabel),
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.saveLabel),
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.chromeLabel),
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.runLabel),
	)
	return sb
}

func (sb *statusBar) setFlow(name string) {
	fyne.Do(func() {
		if name == "" {
			sb.flowLabel.SetText("未选择流程")
		} else {
			sb.flowLabel.SetText(name)
		}
	})
}

func (sb *statusBar) setSave(st SaveStatus) {
	fyne.Do(func() {
		switch st {
		case SaveUnmodified:
			sb.saveLabel.SetText("未修改")
		case SaveDirty:
			sb.saveLabel.SetText("有未保存修改")
		case SaveSaving:
			sb.saveLabel.SetText("保存中")
		case SaveSuccess:
			sb.saveLabel.SetText("已保存")
		case SaveFailed:
			sb.saveLabel.SetText("保存失败")
		}
	})
}

func (sb *statusBar) setChrome(st browser.ChromeStatus) {
	fyne.Do(func() {
		switch st {
		case browser.ChromeNotInstalled:
			sb.chromeLabel.SetText("未安装")
		case browser.ChromeInstalled:
			sb.chromeLabel.SetText("已安装")
		case browser.ChromeDownloading:
			sb.chromeLabel.SetText("下载中")
		case browser.ChromeStarting:
			sb.chromeLabel.SetText("启动中")
		case browser.ChromeRunning:
			sb.chromeLabel.SetText("已启动")
		case browser.ChromeStartFailed:
			sb.chromeLabel.SetText("启动失败")
		}
	})
}

func (sb *statusBar) setRun(st RunStatus, current, total int, msg string) {
	fyne.Do(func() {
		switch st {
		case RunIdle:
			sb.runLabel.SetText("空闲")
		case RunRunning:
			if total > 0 {
				sb.runLabel.SetText(fmt.Sprintf("运行中 %d/%d", current, total))
			} else {
				sb.runLabel.SetText("运行中")
			}
		case RunCompleted:
			sb.runLabel.SetText("已完成")
		case RunFailed:
			if msg != "" {
				sb.runLabel.SetText("失败：" + msg)
			} else {
				sb.runLabel.SetText("失败")
			}
		}
	})
}

func statusColorGray() color.Color   { return color.NRGBA{0x9e, 0x9e, 0x9e, 0xff} }
func statusColorBlue() color.Color   { return color.NRGBA{0x1a, 0x73, 0xe8, 0xff} }
func statusColorGreen() color.Color  { return color.NRGBA{0x4c, 0xaf, 0x50, 0xff} }
func statusColorYellow() color.Color { return color.NRGBA{0xf9, 0xa8, 0x25, 0xff} }
func statusColorRed() color.Color    { return color.NRGBA{0xe5, 0x39, 0x35, 0xff} }
