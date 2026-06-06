package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconFS embed.FS

// Icon returns the application icon resource.
func Icon() fyne.Resource {
	data, err := iconFS.ReadFile("icon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", data)
}
