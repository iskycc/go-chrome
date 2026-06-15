//go:build windows

package assets

import "fyne.io/fyne/v2"

// AppIcon returns the ICO resource on Windows for best taskbar rendering.
func AppIcon() fyne.Resource {
	return IconICO()
}
