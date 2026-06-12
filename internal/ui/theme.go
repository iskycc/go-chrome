package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"go-chrome/assets"
)

// appTheme is a custom Fyne theme with a consistent palette, font, and size system.
type appTheme struct{}

func newAppTheme() fyne.Theme {
	return &appTheme{}
}

func (a *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.RGBA{0x1a, 0x73, 0xe8, 0xff}
	case theme.ColorNameBackground:
		return color.RGBA{0xf5, 0xf5, 0xf5, 0xff}
	case theme.ColorNameForeground:
		return color.RGBA{0x21, 0x21, 0x21, 0xff}
	case theme.ColorNameButton:
		return color.RGBA{0xff, 0xff, 0xff, 0xff}
	case theme.ColorNameDisabled:
		return color.RGBA{0x9e, 0x9e, 0x9e, 0xff}
	case theme.ColorNameHover:
		return color.RGBA{0xe8, 0xf0, 0xfe, 0xff}
	case theme.ColorNameSelection:
		return color.RGBA{0xbb, 0xde, 0xfb, 0xff}
	case theme.ColorNameInputBackground:
		return color.RGBA{0xff, 0xff, 0xff, 0xff}
	case theme.ColorNameScrollBar:
		return color.RGBA{0xbd, 0xbd, 0xbd, 0xff}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	if f := assets.CascadiaCodeSemiLight(); f != nil {
		return f
	}
	return theme.DefaultTheme().Font(style)
}

func (a *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (a *appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameHeadingText:
		return 16
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 18
	default:
		return theme.DefaultTheme().Size(name)
	}
}
