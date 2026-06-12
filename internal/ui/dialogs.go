package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// sizedEntry returns a single-line entry with a comfortable minimum width.
func sizedEntry(placeholder string) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	return e
}

// sizedMultiLineEntry returns a multi-line entry with the requested number of
// rows and a comfortable minimum width. Use this for long values, URLs, JSON,
// or descriptions so the dialog does not blow out horizontally.
func sizedMultiLineEntry(placeholder string, rows int) *widget.Entry {
	e := widget.NewMultiLineEntry()
	e.SetPlaceHolder(placeholder)
	if rows < 2 {
		rows = 2
	}
	// Approximate a row height and add some padding for borders/scrollbars.
	height := float32(rows*18 + 12)
	e.Wrapping = fyne.TextWrapWord
	_ = height
	return e
}

// showSizedFormDialog shows a form dialog with an explicit size. It is used
// instead of dialog.ShowForm because the default Fyne form dialog is too
// narrow and its input fields are not wide enough for Chinese text or long
// values.
func showSizedFormDialog(title, confirm, cancel string, items []*widget.FormItem, size fyne.Size, cb func(bool), win fyne.Window) {
	rows := make([]fyne.CanvasObject, 0, len(items))
	for _, item := range items {
		if item.Text == "" {
			// No label: place the widget directly (e.g. a checkbox).
			rows = append(rows, item.Widget)
		} else {
			rows = append(rows, container.NewVBox(
				widget.NewLabel(item.Text),
				item.Widget,
			))
		}
	}
	content := container.NewVBox(rows...)
	d := dialog.NewCustomConfirm(title, confirm, cancel, content, cb, win)
	d.Resize(size)
	d.Show()
}

// showWrappedConfirm shows a confirmation dialog with a wrapping message.
// Long names are truncated before being displayed so the dialog does not grow
// beyond the requested size.
func showWrappedConfirm(title, message, confirm, cancel string, size fyne.Size, cb func(bool), win fyne.Window) {
	body := widget.NewLabel(message)
	body.Alignment = fyne.TextAlignLeading
	body.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm(title, confirm, cancel, body, cb, win)
	d.Resize(size)
	d.Show()
}

// resizeFileDialog sets a fixed, comfortable size on a file open/save dialog.
func resizeFileDialog(d dialog.Dialog) {
	d.Resize(fyne.NewSize(720, 520))
}

// truncateForDialog truncates a string to a safe length for dialog messages.
// It preserves valid UTF-8 and avoids breaking in the middle of a rune.
func truncateForDialog(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}

// safeFileName returns a sanitized version of a flow name suitable for use as
// a file name. It removes characters that are illegal on Windows and most
// filesystems, and limits the length so the default export file name does not
// overflow the save dialog.
func safeFileName(name string) string {
	replacer := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	name = truncateForDialog(name, 64)
	if name == "" {
		name = "flow"
	}
	return fmt.Sprintf("%s.json", name)
}
