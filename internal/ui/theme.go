package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// appTheme is a custom Fyne theme.
type appTheme struct{}

func newAppTheme() fyne.Theme {
	return &appTheme{}
}

func (a *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.RGBA{0x1a, 0x73, 0xe8, 0xff}
	}
	if name == theme.ColorNameBackground {
		return color.RGBA{0xf5, 0xf5, 0xf5, 0xff}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (a *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (a *appTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
