package ui

import (
	"fmt"
	"image/color"
	"time"

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

// statusItem bundles the static field-name label with the dynamic value label
// and a colored dot. Callers update them as a unit.
type statusItem struct {
	field fyne.CanvasObject
	value *widget.Label
	dot   fyne.CanvasObject
	row   fyne.CanvasObject
}

// newStatusItem builds a compact status row: [dot, muted field, value]. The
// value label is wrapped in a fixed-width container so long values truncate
// with ellipsis instead of squeezing neighbouring items.
func newStatusItem(field, defaultValue string, kind statusKind, valueWidth float32) *statusItem {
	val := widget.NewLabel(defaultValue)
	val.Truncation = fyne.TextTruncateEllipsis
	val.Wrapping = fyne.TextWrapOff

	valueBox := container.NewGridWrap(fyne.NewSize(valueWidth, val.MinSize().Height), val)

	si := &statusItem{
		field: newMutedText(field),
		value: val,
		dot:   newStatusDot(kind),
	}
	si.row = container.NewHBox(si.dot, si.field, valueBox)
	return si
}

func (si *statusItem) setValue(text string) {
	si.value.SetText(truncate(text, 200))
}

func (si *statusItem) setKind(kind statusKind) {
	si.dot = newStatusDot(kind)
	// Rebuild the row so the new dot is visible.
	objects := si.row.(*fyne.Container).Objects
	if len(objects) >= 3 {
		objects[0] = si.dot
	}
	si.row.Refresh()
}

// setColor is kept for callers that already have a raw color. It recolors the
// dot without changing the badge helper.
func (si *statusItem) setColor(c color.Color) {
	if dot, ok := si.dot.(*fyne.Container); ok {
		if circle, ok := dot.Objects[0].(*canvas.Circle); ok {
			circle.FillColor = c
			circle.Refresh()
		}
	}
}

type statusBar struct {
	app    *App
	widget fyne.CanvasObject

	flow   *statusItem
	save   *statusItem
	chrome *statusItem
	run    *statusItem

	saveResetTimer *time.Timer
}

func newStatusBar(app *App) *statusBar {
	sb := &statusBar{app: app}
	sb.flow = newStatusItem("当前流程", "未选择", statusMuted, 220)
	sb.save = newStatusItem("保存", "未修改", statusSuccess, 110)
	sb.chrome = newStatusItem("Chrome", "未安装", statusWarning, 110)
	sb.run = newStatusItem("运行", "空闲", statusMuted, 160)

	title := widget.NewLabelWithStyle("Go Chrome", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	itemSpacer := canvas.NewRectangle(color.Transparent)
	itemSpacer.SetMinSize(fyne.NewSize(16, 1))

	sb.widget = container.NewHBox(
		title,
		itemSpacer,
		sb.flow.row,
		itemSpacer,
		sb.save.row,
		itemSpacer,
		sb.chrome.row,
		itemSpacer,
		sb.run.row,
	)
	return sb
}

func (sb *statusBar) setFlow(name string) {
	fyne.Do(func() {
		if name == "" {
			sb.flow.setValue("未选择")
		} else {
			sb.flow.setValue(name)
		}
	})
}

func (sb *statusBar) setSave(st SaveStatus) {
	fyne.Do(func() {
		if sb.saveResetTimer != nil {
			sb.saveResetTimer.Stop()
			sb.saveResetTimer = nil
		}
		switch st {
		case SaveUnmodified:
			sb.save.setValue("未修改")
			sb.save.setKind(statusSuccess)
		case SaveDirty:
			sb.save.setValue("有未保存修改")
			sb.save.setKind(statusWarning)
		case SaveSaving:
			sb.save.setValue("保存中")
			sb.save.setKind(statusInfo)
		case SaveSuccess:
			sb.save.setValue("已保存")
			sb.save.setKind(statusSuccess)
			sb.saveResetTimer = time.AfterFunc(2*time.Second, func() {
				sb.setSave(SaveUnmodified)
			})
		case SaveFailed:
			sb.save.setValue("保存失败")
			sb.save.setKind(statusDanger)
		}
	})
}

func (sb *statusBar) setChrome(st browser.ChromeStatus) {
	fyne.Do(func() {
		switch st {
		case browser.ChromeNotInstalled:
			sb.chrome.setValue("未安装")
			sb.chrome.setKind(statusWarning)
		case browser.ChromeInstalled:
			sb.chrome.setValue("已安装")
			sb.chrome.setKind(statusSuccess)
		case browser.ChromeDownloading:
			sb.chrome.setValue("下载中")
			sb.chrome.setKind(statusInfo)
		case browser.ChromeStarting:
			sb.chrome.setValue("启动中")
			sb.chrome.setKind(statusInfo)
		case browser.ChromeRunning:
			sb.chrome.setValue("已启动")
			sb.chrome.setKind(statusSuccess)
		case browser.ChromeStartFailed:
			sb.chrome.setValue("启动失败")
			sb.chrome.setKind(statusDanger)
		}
	})
}

func (sb *statusBar) setRun(st RunStatus, current, total int, msg string) {
	fyne.Do(func() {
		switch st {
		case RunIdle:
			sb.run.setValue("空闲")
			sb.run.setKind(statusMuted)
		case RunRunning:
			if total > 0 {
				sb.run.setValue(fmt.Sprintf("运行中 %d/%d", current, total))
			} else {
				sb.run.setValue("运行中")
			}
			sb.run.setKind(statusInfo)
		case RunCompleted:
			sb.run.setValue("已完成")
			sb.run.setKind(statusSuccess)
		case RunFailed:
			if msg != "" {
				sb.run.setValue("失败：" + msg)
			} else {
				sb.run.setValue("失败")
			}
			sb.run.setKind(statusDanger)
		}
	})
}
