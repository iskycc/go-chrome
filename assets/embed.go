package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png fonts/CascadiaNextSC-450.ttf fonts/CascadiaCode-SemiLight.ttf
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
// The font is pre-baked at wght=450 (between Regular 400 and Medium 500)
// so CJK glyphs render with a clearly visible stroke weight; the source
// variable font's default wght=200 (ExtraLight) made every Chinese
// character look too thin. See cmd/bake-font for the patching logic.
func AppUIFont() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/CascadiaNextSC-450.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaNextSC-450.ttf", data)
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
