//go:build !windows

package assets

import "fyne.io/fyne/v2"

// AppIcon returns the PNG resource on non-Windows platforms.
func AppIcon() fyne.Resource {
	return Icon()
}
