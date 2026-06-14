package ui

import (
	"fmt"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/browser"
	"go-chrome/internal/sysinfo"
)

// infoPanel shows resource usage for the current process and the managed
// Chrome process.
type infoPanel struct {
	app    *App
	widget fyne.CanvasObject

	selfPID      *widget.Label
	selfName     *widget.Label
	selfCPU      *widget.Label
	selfMemory   *widget.Label
	selfUptime   *widget.Label
	chromeStatus *widget.Label
	chromePID    *widget.Label
	chromeName   *widget.Label
	chromeCPU    *widget.Label
	chromeMemory *widget.Label
}

func newInfoPanel(app *App) *infoPanel {
	p := &infoPanel{app: app}

	p.selfPID = widget.NewLabel("")
	p.selfName = widget.NewLabel("")
	p.selfCPU = widget.NewLabel("")
	p.selfMemory = widget.NewLabel("")
	p.selfUptime = widget.NewLabel("")
	p.chromeStatus = widget.NewLabel("")
	p.chromePID = widget.NewLabel("")
	p.chromeName = widget.NewLabel("")
	p.chromeCPU = widget.NewLabel("")
	p.chromeMemory = widget.NewLabel("")

	selfForm := widget.NewForm(
		widget.NewFormItem("PID", p.selfPID),
		widget.NewFormItem("名称", p.selfName),
		widget.NewFormItem("CPU", p.selfCPU),
		widget.NewFormItem("内存", p.selfMemory),
		widget.NewFormItem("运行时长", p.selfUptime),
	)

	chromeForm := widget.NewForm(
		widget.NewFormItem("状态", p.chromeStatus),
		widget.NewFormItem("PID", p.chromePID),
		widget.NewFormItem("名称", p.chromeName),
		widget.NewFormItem("CPU", p.chromeCPU),
		widget.NewFormItem("内存", p.chromeMemory),
	)

	refreshBtn := widget.NewButtonWithIcon("刷新", theme.ViewRefreshIcon(), func() {
		p.refresh()
	})
	refreshBtn.Importance = widget.MediumImportance

	content := container.NewVBox(
		newSectionHeader("系统信息", refreshBtn),
		newMutedText(fmt.Sprintf("平台：%s/%s", runtime.GOOS, runtime.GOARCH)),

		newSectionHeader("当前程序"),
		selfForm,

		newSectionHeader("托管 Chrome"),
		chromeForm,
	)

	p.widget = container.NewScroll(content)
	p.refresh()
	return p
}

func (p *infoPanel) refresh() {
	fyne.Do(func() {
		self, err := sysinfo.SelfInfo()
		if err != nil {
			p.selfPID.SetText("读取失败")
			p.selfName.SetText("")
			p.selfCPU.SetText("")
			p.selfMemory.SetText("")
			p.selfUptime.SetText("")
		} else {
			p.selfPID.SetText(fmt.Sprintf("%d", self.PID))
			p.selfName.SetText(self.Name)
			p.selfCPU.SetText(sysinfo.FormatCPU(self.CPU))
			p.selfMemory.SetText(sysinfo.FormatMemory(self.MemoryMB))
			if uptime, err := sysinfo.Uptime(); err == nil {
				p.selfUptime.SetText(uptime.String())
			} else {
				p.selfUptime.SetText("-")
			}
		}

		chromePID := 0
		if p.app.browserMgr != nil {
			chromePID = p.app.browserMgr.ManagedPID()
		}
		chrome, err := sysinfo.ChromeInfo(chromePID)
		if err != nil || !chrome.Exists {
			status := "未启动"
			if p.app.browserMgr != nil {
				switch p.app.browserMgr.Status() {
				case browser.ChromeInstalled:
					status = "已安装（未启动）"
				case browser.ChromeNotInstalled:
					status = "未安装"
				case browser.ChromeStarting:
					status = "启动中"
				case browser.ChromeDownloading:
					status = "下载中"
				case browser.ChromeStartFailed:
					status = "启动失败"
				}
			}
			p.chromeStatus.SetText(status)
			p.chromePID.SetText("-")
			p.chromeName.SetText("-")
			p.chromeCPU.SetText("-")
			p.chromeMemory.SetText("-")
		} else {
			p.chromeStatus.SetText("运行中")
			p.chromePID.SetText(fmt.Sprintf("%d", chrome.PID))
			p.chromeName.SetText(chrome.Name)
			p.chromeCPU.SetText(sysinfo.FormatCPU(chrome.CPU))
			p.chromeMemory.SetText(sysinfo.FormatMemory(chrome.MemoryMB))
		}
	})
}
