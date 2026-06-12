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

// statusItem bundles the static field-name label with the dynamic value
// label and the dynamic color dot, so callers can update them as a unit.
type statusItem struct {
	field  *widget.Label
	value  *widget.Label
	dot    *canvas.Circle
	row    fyne.CanvasObject
	dotClr color.Color
}

// newStatusItem builds a row of [dot, field, value]. The value label is
// sized to valueWidth pixels wide and configured to ellipsize text that
// does not fit. Using TextTruncateEllipsis directly in an HBox is unsafe:
// Fyne computes the label's MinSize as a single character when
// Truncation is set (see widget/richtext.go textRenderer.MinSize), so
// the label can be squished to ~0 width by neighbouring widgets and
// render as "..." for any content. Wrapping it in a fixed-size
// container gives the label a stable width budget.
//
// We rely on textutil.Truncate (rune-aware) as a coarse safety net for
// pathological inputs (multi-kilobyte values), then let Fyne's
// per-pixel ellipsis handle the visual fit.
func newStatusItem(field, defaultValue string, defaultColor color.Color, valueWidth float32) *statusItem {
	val := widget.NewLabel(defaultValue)
	val.Truncation = fyne.TextTruncateEllipsis
	val.Wrapping = fyne.TextWrapOff

	// Fixed-width container that respects valueWidth as both a min and
	// max width. GridWrap is the simplest way to force a child to a
	// specific size inside an HBox.
	valueBox := container.NewGridWrap(fyne.NewSize(valueWidth, val.MinSize().Height), val)

	si := &statusItem{
		field:  widget.NewLabelWithStyle(field, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		value:  val,
		dot:    canvas.NewCircle(defaultColor),
		dotClr: defaultColor,
	}
	si.row = container.NewHBox(si.dot, si.field, valueBox)
	return si
}

func (si *statusItem) setValue(text string) {
	// Rune-aware truncate as a hard ceiling in case valueWidth would
	// otherwise show hundreds of glyphs before ellipsizing. This is a
	// safety net, not the primary mechanism.
	si.value.SetText(truncate(text, 200))
}

func (si *statusItem) setColor(c color.Color) {
	si.dotClr = c
	si.dot.FillColor = c
	si.dot.Refresh()
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
	sb.flow = newStatusItem("当前流程：", "未选择", statusColorGray(), 180)
	sb.save = newStatusItem("保存状态：", "未修改", statusColorGreen(), 110)
	sb.chrome = newStatusItem("Chrome：", "未安装", statusColorYellow(), 110)
	sb.run = newStatusItem("运行状态：", "空闲", statusColorGray(), 160)

	title := widget.NewLabelWithStyle("Go Chrome", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleSpacer := canvas.NewRectangle(color.Transparent)
	titleSpacer.SetMinSize(fyne.NewSize(20, 1))

	itemSpacer := canvas.NewRectangle(color.Transparent)
	itemSpacer.SetMinSize(fyne.NewSize(12, 1))

	sb.widget = container.NewHBox(
		title,
		titleSpacer,
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
		// When transitioning to a non-success state, cancel any pending
		// success -> clean timer so the indicator does not get clobbered.
		if sb.saveResetTimer != nil {
			sb.saveResetTimer.Stop()
			sb.saveResetTimer = nil
		}
		switch st {
		case SaveUnmodified:
			sb.save.setValue("未修改")
			sb.save.setColor(statusColorGreen())
		case SaveDirty:
			sb.save.setValue("有未保存修改")
			sb.save.setColor(statusColorYellow())
		case SaveSaving:
			sb.save.setValue("保存中")
			sb.save.setColor(statusColorBlue())
		case SaveSuccess:
			sb.save.setValue("已保存")
			sb.save.setColor(statusColorGreen())
			// Hold the "saved" state for 2 seconds, then settle to "未修改"
			// so the user actually sees the success indicator.
			sb.saveResetTimer = time.AfterFunc(2*time.Second, func() {
				sb.setSave(SaveUnmodified)
			})
		case SaveFailed:
			sb.save.setValue("保存失败")
			sb.save.setColor(statusColorRed())
		}
	})
}

func (sb *statusBar) setChrome(st browser.ChromeStatus) {
	fyne.Do(func() {
		switch st {
		case browser.ChromeNotInstalled:
			sb.chrome.setValue("未安装")
			sb.chrome.setColor(statusColorYellow())
		case browser.ChromeInstalled:
			sb.chrome.setValue("已安装")
			sb.chrome.setColor(statusColorGreen())
		case browser.ChromeDownloading:
			sb.chrome.setValue("下载中")
			sb.chrome.setColor(statusColorBlue())
		case browser.ChromeStarting:
			sb.chrome.setValue("启动中")
			sb.chrome.setColor(statusColorBlue())
		case browser.ChromeRunning:
			sb.chrome.setValue("已启动")
			sb.chrome.setColor(statusColorGreen())
		case browser.ChromeStartFailed:
			sb.chrome.setValue("启动失败")
			sb.chrome.setColor(statusColorRed())
		}
	})
}

func (sb *statusBar) setRun(st RunStatus, current, total int, msg string) {
	fyne.Do(func() {
		switch st {
		case RunIdle:
			sb.run.setValue("空闲")
			sb.run.setColor(statusColorGray())
		case RunRunning:
			if total > 0 {
				sb.run.setValue(fmt.Sprintf("运行中 %d/%d", current, total))
			} else {
				sb.run.setValue("运行中")
			}
			sb.run.setColor(statusColorBlue())
		case RunCompleted:
			sb.run.setValue("已完成")
			sb.run.setColor(statusColorGreen())
		case RunFailed:
			if msg != "" {
				sb.run.setValue("失败：" + msg)
			} else {
				sb.run.setValue("失败")
			}
			sb.run.setColor(statusColorRed())
		}
	})
}

func statusColorGray() color.Color   { return color.NRGBA{0x9e, 0x9e, 0x9e, 0xff} }
func statusColorBlue() color.Color   { return color.NRGBA{0x1a, 0x73, 0xe8, 0xff} }
func statusColorGreen() color.Color  { return color.NRGBA{0x4c, 0xaf, 0x50, 0xff} }
func statusColorYellow() color.Color { return color.NRGBA{0xf9, 0xa8, 0x25, 0xff} }
func statusColorRed() color.Color    { return color.NRGBA{0xe5, 0x39, 0x35, 0xff} }
