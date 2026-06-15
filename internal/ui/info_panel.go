package ui

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/browser"
	"go-chrome/internal/sysinfo"
)

// infoPanel shows resource usage for the current process and the managed
// Chrome process. It refreshes automatically while the app is running and the
// panel is visible, and avoids unnecessary UI churn by only updating labels
// when values actually change.
type infoPanel struct {
	app    *App
	widget fyne.CanvasObject

	selfPID       *widget.Label
	selfName      *widget.Label
	selfCPU       *widget.Label
	selfMemory    *widget.Label
	selfGoHeap    *widget.Label
	selfStartTime *widget.Label
	selfUptime    *widget.Label
	chromeStatus  *widget.Label
	chromePID     *widget.Label
	chromeName    *widget.Label
	chromeCPU     *widget.Label
	chromeMemory  *widget.Label

	refreshTicker *time.Ticker
	stopRefresh   chan struct{}
	refreshWg     sync.WaitGroup
	visible       bool

	// Cached values to avoid calling SetText when nothing changed.
	cache struct {
		selfPID       string
		selfName      string
		selfCPU       string
		selfMemory    string
		selfGoHeap    string
		selfStartTime string
		selfUptime    string
		chromeStatus  string
		chromePID     string
		chromeName    string
		chromeCPU     string
		chromeMemory  string
	}
}

func newInfoPanel(app *App) *infoPanel {
	p := &infoPanel{app: app}

	p.selfPID = widget.NewLabel("")
	p.selfName = widget.NewLabel("")
	p.selfCPU = widget.NewLabel("")
	p.selfMemory = widget.NewLabel("")
	p.selfGoHeap = widget.NewLabel("")
	p.selfStartTime = widget.NewLabel("")
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
		widget.NewFormItem("内存 (RSS)", p.selfMemory),
		widget.NewFormItem("Go 堆内存", p.selfGoHeap),
		widget.NewFormItem("启动时间", p.selfStartTime),
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

	gcBtn := widget.NewButton("强制 GC", func() {
		runtime.GC()
		runtime.GC()
		p.refresh()
	})
	gcBtn.Importance = widget.LowImportance

	content := container.NewVBox(
		newSectionHeader("系统信息", refreshBtn, gcBtn),
		newMutedText(fmt.Sprintf("平台：%s/%s", runtime.GOOS, runtime.GOARCH)),

		newSectionHeader("当前程序"),
		selfForm,

		newSectionHeader("托管 Chrome"),
		chromeForm,
	)

	p.widget = container.NewScroll(content)
	return p
}

// SetVisible starts or stops automatic refreshing based on whether the info
// tab is currently shown. This avoids allocating UI objects while the user is
// on another tab.
func (p *infoPanel) SetVisible(visible bool) {
	p.visible = visible
	if visible {
		p.refresh()
		p.startAutoRefresh(5 * time.Second)
	} else {
		p.stopAutoRefresh()
	}
}

func (p *infoPanel) startAutoRefresh(interval time.Duration) {
	if p.refreshTicker != nil {
		return
	}
	p.refreshTicker = time.NewTicker(interval)
	p.stopRefresh = make(chan struct{})
	p.refreshWg.Add(1)
	go func() {
		defer p.refreshWg.Done()
		for {
			select {
			case <-p.refreshTicker.C:
				p.refresh()
			case <-p.stopRefresh:
				return
			}
		}
	}()
}

func (p *infoPanel) stopAutoRefresh() {
	if p.refreshTicker == nil {
		return
	}
	p.refreshTicker.Stop()
	close(p.stopRefresh)
	p.refreshWg.Wait()
	p.refreshTicker = nil
	p.stopRefresh = nil
}

func (p *infoPanel) refresh() {
	fyne.Do(func() {
		self, err := sysinfo.SelfInfo()
		if err != nil {
			p.setLabel(p.selfPID, &p.cache.selfPID, "读取失败")
			p.setLabel(p.selfName, &p.cache.selfName, "")
			p.setLabel(p.selfCPU, &p.cache.selfCPU, "")
			p.setLabel(p.selfMemory, &p.cache.selfMemory, "")
			p.setLabel(p.selfGoHeap, &p.cache.selfGoHeap, "")
			p.setLabel(p.selfStartTime, &p.cache.selfStartTime, "")
			p.setLabel(p.selfUptime, &p.cache.selfUptime, "")
		} else {
			p.setLabel(p.selfPID, &p.cache.selfPID, fmt.Sprintf("%d", self.PID))
			p.setLabel(p.selfName, &p.cache.selfName, self.Name)
			p.setLabel(p.selfCPU, &p.cache.selfCPU, sysinfo.FormatCPU(self.CPU))
			p.setLabel(p.selfMemory, &p.cache.selfMemory, sysinfo.FormatMemory(self.MemoryMB))
			heapAlloc, _ := sysinfo.GoMemStats()
			p.setLabel(p.selfGoHeap, &p.cache.selfGoHeap, sysinfo.FormatMemory(heapAlloc))
			if start, err := sysinfo.StartTime(); err == nil {
				p.setLabel(p.selfStartTime, &p.cache.selfStartTime, sysinfo.FormatStartTime(start))
			} else {
				p.setLabel(p.selfStartTime, &p.cache.selfStartTime, "-")
			}
			if uptime, err := sysinfo.Uptime(); err == nil {
				p.setLabel(p.selfUptime, &p.cache.selfUptime, uptime.String())
			} else {
				p.setLabel(p.selfUptime, &p.cache.selfUptime, "-")
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
			p.setLabel(p.chromeStatus, &p.cache.chromeStatus, status)
			p.setLabel(p.chromePID, &p.cache.chromePID, "-")
			p.setLabel(p.chromeName, &p.cache.chromeName, "-")
			p.setLabel(p.chromeCPU, &p.cache.chromeCPU, "-")
			p.setLabel(p.chromeMemory, &p.cache.chromeMemory, "-")
		} else {
			p.setLabel(p.chromeStatus, &p.cache.chromeStatus, "运行中")
			p.setLabel(p.chromePID, &p.cache.chromePID, fmt.Sprintf("%d", chrome.PID))
			p.setLabel(p.chromeName, &p.cache.chromeName, chrome.Name)
			p.setLabel(p.chromeCPU, &p.cache.chromeCPU, sysinfo.FormatCPU(chrome.CPU))
			p.setLabel(p.chromeMemory, &p.cache.chromeMemory, sysinfo.FormatMemory(chrome.MemoryMB))
		}
	})
}

// setLabel only calls SetText when the value changed, which avoids allocating
// new text objects and triggering re-renders on every tick.
func (p *infoPanel) setLabel(label *widget.Label, cache *string, value string) {
	if *cache == value {
		return
	}
	*cache = value
	label.SetText(value)
}
