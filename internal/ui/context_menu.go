package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// contextMenuLabel is a label that can show a context menu on secondary tap
// (right-click). It keeps the normal tap behavior for selection, so it can be
// used inside widget.List and widget.Table cells.
type contextMenuLabel struct {
	widget.Label
	onSecondaryTap func(e *fyne.PointEvent)
}

// newContextMenuLabel creates a label that will invoke onSecondaryTap when the
// user right-clicks it.
func newContextMenuLabel(text string, onSecondaryTap func(e *fyne.PointEvent)) *contextMenuLabel {
	l := &contextMenuLabel{onSecondaryTap: onSecondaryTap}
	l.ExtendBaseWidget(l)
	l.Text = text
	l.Truncation = fyne.TextTruncateEllipsis
	l.Wrapping = fyne.TextWrapOff
	return l
}

// TappedSecondary implements fyne.SecondaryTappable and shows the context menu
// at the pointer position.
func (l *contextMenuLabel) TappedSecondary(e *fyne.PointEvent) {
	if l.onSecondaryTap != nil {
		l.onSecondaryTap(e)
	}
}

// showContextMenu shows a menu at the given absolute position. It is a thin
// wrapper around widget.ShowPopUpMenuAtPosition so callers do not need to
// remember the canvas API.
func showContextMenu(menu *fyne.Menu, c fyne.Canvas, pos fyne.Position) {
	widget.ShowPopUpMenuAtPosition(menu, c, pos)
}

// clipCopy returns a string safe to place on the system clipboard. It limits
// the length to avoid accidentally copying huge values and ensures the result
// is valid UTF-8 by replacing invalid sequences with the replacement character.
func clipCopy(s string) string {
	if len(s) > 32768 {
		return s[:32768]
	}
	return s
}
