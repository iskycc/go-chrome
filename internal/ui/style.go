package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// statusKind identifies the semantic intent of a status badge.
type statusKind int

const (
	statusInfo statusKind = iota
	statusSuccess
	statusWarning
	statusDanger
	statusMuted
)

// currentVariant returns the active theme variant, defaulting to light when no
// application is initialized (e.g. during early package tests).
func currentVariant() fyne.ThemeVariant {
	if a := fyne.CurrentApp(); a != nil {
		return a.Settings().ThemeVariant()
	}
	return theme.VariantLight
}

// -----------------------------------------------------------------------------
// Semantic color tokens
// -----------------------------------------------------------------------------

func uiColorSurface() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0x11, G: 0x13, B: 0x18, A: 0xff}
	}
	return color.NRGBA{R: 0xf6, G: 0xf7, B: 0xf9, A: 0xff}
}

func uiColorPanel() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0x18, G: 0x1b, B: 0x21, A: 0xff}
	}
	return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
}

func uiColorSecondarySurface() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0x20, G: 0x24, B: 0x2c, A: 0xff}
	}
	return color.NRGBA{R: 0xf0, G: 0xf3, B: 0xf7, A: 0xff}
}

func uiColorBorder() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0x30, G: 0x36, B: 0x42, A: 0xff}
	}
	return color.NRGBA{R: 0xd8, G: 0xde, B: 0xe8, A: 0xff}
}

func uiColorText() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0xe6, G: 0xe8, B: 0xec, A: 0xff}
	}
	return color.NRGBA{R: 0x20, G: 0x24, B: 0x2a, A: 0xff}
}

func uiColorMutedText() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0xa4, G: 0xab, B: 0xb6, A: 0xff}
	}
	return color.NRGBA{R: 0x69, G: 0x71, B: 0x7d, A: 0xff}
}

func uiColorPrimary() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff}
	}
	return color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}
}

func uiColorSuccess() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0x4a, G: 0xde, B: 0x80, A: 0xff}
	}
	return color.NRGBA{R: 0x16, G: 0xa3, B: 0x4a, A: 0xff}
}

func uiColorWarning() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0xfb, G: 0xbf, B: 0x24, A: 0xff}
	}
	return color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}
}

func uiColorDanger() color.Color {
	if currentVariant() == theme.VariantDark {
		return color.NRGBA{R: 0xf8, G: 0x71, B: 0x71, A: 0xff}
	}
	return color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff}
}

func uiColorInfo() color.Color {
	return uiColorPrimary()
}

func uiColorForStatus(kind statusKind) color.Color {
	switch kind {
	case statusSuccess:
		return uiColorSuccess()
	case statusWarning:
		return uiColorWarning()
	case statusDanger:
		return uiColorDanger()
	case statusMuted:
		return uiColorMutedText()
	default:
		return uiColorInfo()
	}
}

// -----------------------------------------------------------------------------
// Typography and label helpers
// -----------------------------------------------------------------------------

// newMutedText returns a non-wrapping canvas.Text in the muted color. Prefer
// widget.Label when wrapping is required.
func newMutedText(text string) *canvas.Text {
	t := canvas.NewText(text, uiColorMutedText())
	t.TextSize = theme.CaptionTextSize()
	return t
}

// newSectionHeader builds a bold section title with optional action widgets
// aligned to the right.
func newSectionHeader(title string, actions ...fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if len(actions) == 0 {
		return titleLabel
	}
	return container.NewBorder(nil, nil, titleLabel, container.NewHBox(actions...))
}

// newPageTitle builds a page heading with an optional subtitle below it.
func newPageTitle(title, subtitle string) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if subtitle == "" {
		return titleLabel
	}
	return container.NewVBox(titleLabel, newMutedText(subtitle))
}

// -----------------------------------------------------------------------------
// Toolbar group
// -----------------------------------------------------------------------------

// newToolbarGroup wraps a set of controls with a subtle background and border
// to visually group related actions. Suitable for page-level panels, not the
// top toolbar.
func newToolbarGroup(title string, controls ...fyne.CanvasObject) fyne.CanvasObject {
	header := newMutedText(title)
	body := container.NewHBox(controls...)
	content := container.NewVBox(container.NewPadded(header), container.NewPadded(body))

	bg := canvas.NewRectangle(uiColorPanel())
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = uiColorBorder()
	border.StrokeWidth = 1

	return container.NewStack(bg, border, content)
}

// newInlineToolbarGroup is a compact, single-line group for the top toolbar.
// It shows a muted title followed by controls, all vertically centered.
func newInlineToolbarGroup(title string, controls ...fyne.CanvasObject) fyne.CanvasObject {
	return container.NewCenter(container.NewHBox(
		newMutedText(title),
		container.NewHBox(controls...),
	))
}

// -----------------------------------------------------------------------------
// Status badge
// -----------------------------------------------------------------------------

// sizedDot returns a colored circle constrained to the given size.
func sizedDot(clr color.Color, size float32) fyne.CanvasObject {
	dot := canvas.NewCircle(clr)
	return container.NewGridWrap(fyne.NewSize(size, size), dot)
}

// newStatusBadge returns a compact badge (colored dot + text) for a status
// value. The result is fixed-height and avoids pushing neighbouring widgets
// around when the text changes.
func newStatusBadge(text string, kind statusKind) fyne.CanvasObject {
	label := canvas.NewText(text, uiColorText())
	label.TextSize = theme.CaptionTextSize()
	return container.NewHBox(
		sizedDot(uiColorForStatus(kind), 8),
		label,
	)
}

// newStatusDot returns a small colored circle for use where text is provided
// separately.
func newStatusDot(kind statusKind) fyne.CanvasObject {
	return sizedDot(uiColorForStatus(kind), 8)
}

// -----------------------------------------------------------------------------
// Empty and error states
// -----------------------------------------------------------------------------

// newEmptyState shows a centered placeholder with a title, a short message, and
// an optional action control. Used instead of blank panels.
func newEmptyState(title, message string, action fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	msgLabel := widget.NewLabel(message)
	msgLabel.Alignment = fyne.TextAlignCenter
	msgLabel.Wrapping = fyne.TextWrapWord

	items := []fyne.CanvasObject{titleLabel, msgLabel}
	if action != nil {
		items = append(items, container.NewCenter(action))
	}
	return container.NewCenter(container.NewVBox(items...))
}

// newInlineError returns a compact error label for forms and inline feedback.
func newInlineError(message string) *widget.Label {
	l := widget.NewLabel(message)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

// -----------------------------------------------------------------------------
// Decorative rectangles
// -----------------------------------------------------------------------------

// newDivider returns a subtle horizontal separator with consistent color.
func newDivider() fyne.CanvasObject {
	rect := canvas.NewRectangle(uiColorBorder())
	return container.NewGridWrap(fyne.NewSize(1, 1), rect)
}

// newPillBackground returns a rounded rectangle suitable for badges.
func newPillBackground() *canvas.Rectangle {
	r := canvas.NewRectangle(uiColorSecondarySurface())
	r.CornerRadius = 4
	return r
}
