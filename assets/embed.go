package assets

import (
	"embed"
	"sync"

	"fyne.io/fyne/v2"
)

//go:embed icon.png icon.ico fonts/MapleMono-CN-Regular.ttf
var assetFS embed.FS

var (
	iconOnce sync.Once
	iconRes  fyne.Resource

	icoOnce sync.Once
	icoRes  fyne.Resource

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

func IconICO() fyne.Resource {
	icoOnce.Do(func() {
		data, err := assetFS.ReadFile("icon.ico")
		if err != nil {
			return
		}
		icoRes = fyne.NewStaticResource("icon.ico", data)
	})
	return icoRes
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
