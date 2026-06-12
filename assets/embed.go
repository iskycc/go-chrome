package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png fonts/CascadiaNextSC.wght.ttf fonts/CascadiaCode-SemiLight.ttf
var assetFS embed.FS

// Icon returns the application icon resource.
func Icon() fyne.Resource {
	data, err := assetFS.ReadFile("icon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", data)
}

// AppUIFont returns the embedded Cascadia Next SC font resource for the UI.
func AppUIFont() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/CascadiaNextSC.wght.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaNextSC.wght.ttf", data)
}

// CodeFont returns the embedded Cascadia Code SemiLight font resource for
// code-specific scenes.
func CodeFont() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/CascadiaCode-SemiLight.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaCode-SemiLight.ttf", data)
}
