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
	isDark := variant == theme.VariantDark

	switch name {
	case theme.ColorNamePrimary:
		if isDark {
			return color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff}
		}
		return color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}
	case theme.ColorNameBackground:
		if isDark {
			return color.NRGBA{R: 0x11, G: 0x13, B: 0x18, A: 0xff}
		}
		return color.NRGBA{R: 0xf6, G: 0xf7, B: 0xf9, A: 0xff}
	case theme.ColorNameForeground:
		if isDark {
			return color.NRGBA{R: 0xe6, G: 0xe8, B: 0xec, A: 0xff}
		}
		return color.NRGBA{R: 0x20, G: 0x24, B: 0x2a, A: 0xff}
	case theme.ColorNameButton:
		if isDark {
			return color.NRGBA{R: 0x18, G: 0x1b, B: 0x21, A: 0xff}
		}
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNameDisabled:
		if isDark {
			return color.NRGBA{R: 0x4a, G: 0x51, B: 0x5c, A: 0xff}
		}
		return color.NRGBA{R: 0x9e, G: 0xa3, B: 0xad, A: 0xff}
	case theme.ColorNameHover:
		if isDark {
			return color.NRGBA{R: 0x20, G: 0x2c, B: 0x3d, A: 0xff}
		}
		return color.NRGBA{R: 0xe8, G: 0xf0, B: 0xfe, A: 0xff}
	case theme.ColorNameSelection:
		if isDark {
			return color.NRGBA{R: 0x1e, G: 0x3a, B: 0x5f, A: 0xff}
		}
		return color.NRGBA{R: 0xbb, G: 0xde, B: 0xfb, A: 0xff}
	case theme.ColorNameInputBackground:
		if isDark {
			return color.NRGBA{R: 0x20, G: 0x24, B: 0x2c, A: 0xff}
		}
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNameScrollBar:
		if isDark {
			return color.NRGBA{R: 0x3e, G: 0x47, B: 0x55, A: 0xff}
		}
		return color.NRGBA{R: 0xbd, G: 0xc3, B: 0xcc, A: 0xff}
	case theme.ColorNameSeparator:
		if isDark {
			return color.NRGBA{R: 0x30, G: 0x36, B: 0x42, A: 0xff}
		}
		return color.NRGBA{R: 0xd8, G: 0xde, B: 0xe8, A: 0xff}
	case theme.ColorNamePlaceHolder:
		if isDark {
			return color.NRGBA{R: 0x7d, G: 0x86, B: 0x94, A: 0xff}
		}
		return color.NRGBA{R: 0x69, G: 0x71, B: 0x7d, A: 0xff}
	case theme.ColorNameError:
		if isDark {
			return color.NRGBA{R: 0xf8, G: 0x71, B: 0x71, A: 0xff}
		}
		return color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Bold {
		if f := assets.AppUIFontMedium(); f != nil {
			return f
		}
	}
	if f := assets.AppUIFontRegular(); f != nil {
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
