package assets

import (
	"embed"
	"sync"

	"fyne.io/fyne/v2"
)

//go:embed icon.png fonts/MapleMono-CN-Regular.ttf
var assetFS embed.FS

var (
	iconOnce sync.Once
	iconRes  fyne.Resource

	fontOnce sync.Once
	fontRes  fyne.Resource
)

func Icon() fyne.Resource {
	iconOnce.Do(func() {
		data, err := assetFS.ReadFile("icon.png")
		if err != nil {
			return
		}
		iconRes = fyne.NewStaticResource("icon.png", data)
	})
	return iconRes
}

func AppUIFontRegular() fyne.Resource {
	fontOnce.Do(func() {
		data, err := assetFS.ReadFile("fonts/MapleMono-CN-Regular.ttf")
		if err != nil {
			return
		}
		fontRes = fyne.NewStaticResource("MapleMono-CN-Regular.ttf", data)
	})
	return fontRes
}
