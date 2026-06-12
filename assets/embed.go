package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
//go:embed fonts/CascadiaCode-SemiLight.ttf
var assetsFS embed.FS

// Icon returns the application icon resource.
func Icon() fyne.Resource {
	data, err := assetsFS.ReadFile("icon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", data)
}

// CascadiaCodeSemiLight returns the embedded Cascadia Code SemiLight font resource.
func CascadiaCodeSemiLight() fyne.Resource {
	data, err := assetsFS.ReadFile("fonts/CascadiaCode-SemiLight.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaCode-SemiLight.ttf", data)
}
