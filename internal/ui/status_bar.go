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
// and a colored dot. The dot circle is reused and only recolored on status
// changes to avoid allocating new canvas objects every tick.
type statusItem struct {
	field fyne.CanvasObject
	value *widget.Label
	dot   *canvas.Circle
	row   fyne.CanvasObject
	kind  statusKind
}

// newStatusItem builds a compact status row: [muted field, value, dot]. The
// value label is wrapped in a fixed-width container so long values truncate
// with ellipsis instead of squeezing neighbouring items.
func newStatusItem(field, defaultValue string, kind statusKind, valueWidth float32) *statusItem {
	val := widget.NewLabel(defaultValue)
	val.Truncation = fyne.TextTruncateEllipsis
	val.Wrapping = fyne.TextWrapOff

	valueBox := container.NewGridWrap(fyne.NewSize(valueWidth, val.MinSize().Height), val)

	dot := canvas.NewCircle(uiColorForStatus(kind))
	dot.StrokeWidth = 0
	dotBox := container.NewGridWrap(fyne.NewSize(8, 8), dot)

	si := &statusItem{
		field: newMutedText(field),
		value: val,
		dot:   dot,
		kind:  kind,
	}
	si.row = container.NewCenter(container.NewHBox(si.field, valueBox, dotBox))
	return si
}

func (si *statusItem) setValue(text string) {
	if si.value.Text == text {
		return
	}
	si.value.SetText(truncate(text, 200))
}

func (si *statusItem) setKind(kind statusKind) {
	if si.kind == kind {
		return
	}
	si.kind = kind
	si.setColor(uiColorForStatus(kind))
}

// setColor recolors the existing dot circle instead of creating a new one.
func (si *statusItem) setColor(c color.Color) {
	if si.dot == nil {
		return
	}
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
	sb.flow = newStatusItem("当前流程", "未选择", statusMuted, 180)
	sb.save = newStatusItem("保存", "未修改", statusSuccess, 110)
	sb.chrome = newStatusItem("Chrome", "未安装", statusWarning, 110)
	sb.run = newStatusItem("运行", "空闲", statusMuted, 160)

	title := widget.NewLabelWithStyle("Go Chrome", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Wrap the title and status rows in a subtle card so the status bar reads as
	// a single, cohesive header strip.
	inner := container.NewHBox(
		title,
		newVerticalSeparator(),
		sb.save.row,
		newVerticalSeparator(),
		sb.chrome.row,
		newVerticalSeparator(),
		sb.run.row,
	)
	sb.widget = newStatusBarCard(inner)
	return sb
}

// newVerticalSeparator draws a subtle vertical line for separating status items.
func newVerticalSeparator() fyne.CanvasObject {
	sep := canvas.NewRectangle(color.Transparent)
	sep.StrokeColor = uiColorBorder()
	sep.StrokeWidth = 1
	sep.SetMinSize(fyne.NewSize(1, 20))
	return sep
}

// newStatusBarCard wraps the status bar content with a light background and
// subtle border, replacing the previous invisible spacers.
func newStatusBarCard(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(uiColorSecondarySurface())
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = uiColorBorder()
	border.StrokeWidth = 1
	return container.NewStack(bg, border, container.NewPadded(content))
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
