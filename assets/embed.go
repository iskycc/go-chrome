package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png fonts/MapleMono-CN-Regular.ttf fonts/MapleMono-CN-Medium.ttf
var assetFS embed.FS

// Icon returns the application icon resource.
func Icon() fyne.Resource {
	data, err := assetFS.ReadFile("icon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", data)
}

// AppUIFontRegular returns the embedded Maple Mono CN Regular font resource
// for normal UI text. Maple Mono CN is a JetBrains Mono style font with full
// CJK coverage, giving a unified Chinese/English look.
func AppUIFontRegular() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/MapleMono-CN-Regular.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("MapleMono-CN-Regular.ttf", data)
}

// AppUIFontMedium returns the embedded Maple Mono CN Medium font resource for
// bold/important UI text such as tab labels, headings and primary buttons.
func AppUIFontMedium() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/MapleMono-CN-Medium.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("MapleMono-CN-Medium.ttf", data)
}

// CodeFont returns the embedded Maple Mono CN Regular font resource. Since the
// UI font already covers CJK, the code font uses the same family to avoid
// Chinese text falling back to a different typeface in logs or input fields.
func CodeFont() fyne.Resource {
	return AppUIFontRegular()
}
