package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/config"
)

type settingsPanel struct {
	app    *App
	widget fyne.CanvasObject

	sourceSelect  *widget.Select
	channelSelect *widget.Select
	customURL     *widget.Entry
	customSHA     *widget.Entry
	customLabel   *widget.Entry
	installDir    *widget.Entry
	userDataDir   *widget.Entry
	fallback      *widget.Check
	keepCache     *widget.Check
	statusLabel   *widget.Label
	customBox     *fyne.Container
}

func newSettingsPanel(app *App) *settingsPanel {
	p := &settingsPanel{app: app}

	p.sourceSelect = widget.NewSelect([]string{"官方 Stable", "自定义 ZIP"}, func(string) {
		p.updateCustomVisibility()
	})
	p.channelSelect = widget.NewSelect([]string{"Stable", "Beta", "Dev", "Canary"}, nil)

	p.customURL = widget.NewEntry()
	p.customURL.SetPlaceHolder("https://example.com/chrome-win64.zip")
	p.customSHA = widget.NewEntry()
	p.customSHA.SetPlaceHolder("可选，用于校验 ZIP SHA256")
	p.customLabel = widget.NewEntry()
	p.customLabel.SetPlaceHolder("可选，例如 company-mirror-126")

	p.installDir = widget.NewEntry()
	p.installDir.SetPlaceHolder("./chrome")
	p.userDataDir = widget.NewEntry()
	p.userDataDir.SetPlaceHolder("./data/chrome-profile")

	p.fallback = widget.NewCheck("自定义下载失败后回退官方 Stable", nil)
	p.keepCache = widget.NewCheck("保留下载缓存 ZIP", nil)
	p.statusLabel = widget.NewLabel("")

	p.customBox = container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("自定义下载 URL", p.customURL),
		),
	)
	p.customBox.Hide()

	saveBtn := widget.NewButtonWithIcon("保存配置", theme.DocumentSaveIcon(), func() {
		p.save()
	})
	saveBtn.Importance = widget.HighImportance
	resetBtn := widget.NewButtonWithIcon("恢复默认", theme.ViewRefreshIcon(), func() {
		p.load(config.Default().Chrome)
		p.statusLabel.SetText("已恢复默认值，点击保存后生效")
	})

	installDirBox := p.pathEntryBox(p.installDir)
	userDataDirBox := p.pathEntryBox(p.userDataDir)

	content := container.NewVBox(
		newSectionHeader("Chrome 来源", saveBtn, resetBtn),
		newMutedText("已有本地 Chrome 时不会重新下载。"),
		widget.NewForm(
			widget.NewFormItem("下载来源", p.sourceSelect),
			widget.NewFormItem("通道标记", p.channelSelect),
		),
		p.customBox,

		newSectionHeader("安装与数据目录"),
		newMutedText("每次流程重放仍会使用隔离 profile 启动新的托管 Chrome。"),
		widget.NewForm(
			widget.NewFormItem("安装目录", installDirBox),
			widget.NewFormItem("用户数据目录", userDataDirBox),
		),

		newSectionHeader("下载校验"),
		newMutedText("填写 SHA256 可在自定义下载后进行完整性校验。"),
		widget.NewForm(
			widget.NewFormItem("SHA256", p.customSHA),
			widget.NewFormItem("版本标签", p.customLabel),
		),

		newSectionHeader("缓存策略"),
		widget.NewForm(
			widget.NewFormItem("", p.fallback),
			widget.NewFormItem("", p.keepCache),
		),

		container.NewHBox(p.statusLabel),
	)

	p.widget = container.NewScroll(container.NewPadded(content))
	p.load(app.cfg.Chrome)
	return p
}

// pathEntryBox wraps a directory entry with a wider budget and a choose button.
func (p *settingsPanel) pathEntryBox(entry *widget.Entry) fyne.CanvasObject {
	entryBox := container.NewGridWrap(fyne.NewSize(360, entry.MinSize().Height), entry)
	chooseBtn := widget.NewButton("选择目录", func() {
		fd := dialog.NewFolderOpen(func(reader fyne.ListableURI, err error) {
			if err != nil || reader == nil {
				return
			}
			entry.SetText(reader.Path())
		}, p.app.mainWin)
		resizeFileDialog(fd)
		fd.Show()
	})
	return container.NewHBox(entryBox, chooseBtn)
}

func (p *settingsPanel) load(cfg config.ChromeConfig) {
	p.sourceSelect.SetSelected(sourceLabel(cfg.DownloadSource))
	if cfg.Channel == "" {
		cfg.Channel = "Stable"
	}
	p.channelSelect.SetSelected(cfg.Channel)
	p.customURL.SetText(cfg.CustomDownloadURL)
	p.customSHA.SetText(cfg.CustomDownloadSHA256)
	p.customLabel.SetText(cfg.CustomVersionLabel)
	p.installDir.SetText(cfg.InstallDir)
	p.userDataDir.SetText(cfg.UserDataDir)
	p.fallback.SetChecked(cfg.FallbackToOfficial)
	p.keepCache.SetChecked(cfg.KeepDownloadCache)
	p.updateCustomVisibility()
}

func (p *settingsPanel) save() {
	source := sourceValue(p.sourceSelect.Selected)
	if source == "custom_url" && p.customURL.Text == "" {
		dialog.ShowInformation("配置不完整", "自定义下载来源需要填写 ZIP 下载 URL", p.app.mainWin)
		return
	}
	if p.installDir.Text == "" || p.userDataDir.Text == "" {
		dialog.ShowInformation("配置不完整", "安装目录和用户数据目录不能为空", p.app.mainWin)
		return
	}

	p.app.cfg.Chrome.DownloadSource = source
	p.app.cfg.Chrome.Channel = p.channelSelect.Selected
	p.app.cfg.Chrome.CustomDownloadURL = p.customURL.Text
	p.app.cfg.Chrome.CustomDownloadSHA256 = p.customSHA.Text
	p.app.cfg.Chrome.CustomVersionLabel = p.customLabel.Text
	p.app.cfg.Chrome.InstallDir = p.installDir.Text
	p.app.cfg.Chrome.UserDataDir = p.userDataDir.Text
	p.app.cfg.Chrome.FallbackToOfficial = p.fallback.Checked
	p.app.cfg.Chrome.KeepDownloadCache = p.keepCache.Checked

	if err := config.Save(p.app.dirs.ConfigPath, p.app.cfg); err != nil {
		dialog.ShowError(fmt.Errorf("保存浏览器配置失败: %w", err), p.app.mainWin)
		return
	}
	_ = p.app.browserMgr.LoadManifest()
	p.statusLabel.SetText("浏览器配置已保存")
	if p.app.runPanel != nil {
		p.app.runPanel.log("浏览器下载配置已保存")
	}
}

func (p *settingsPanel) updateCustomVisibility() {
	if p.customBox == nil {
		return
	}
	if sourceValue(p.sourceSelect.Selected) == "custom_url" {
		p.customBox.Show()
	} else {
		p.customBox.Hide()
	}
	p.widget.Refresh()
}

func sourceLabel(source string) string {
	if source == "custom_url" {
		return "自定义 ZIP"
	}
	return "官方 Stable"
}

func sourceValue(label string) string {
	if label == "自定义 ZIP" {
		return "custom_url"
	}
	return "official_stable"
}
