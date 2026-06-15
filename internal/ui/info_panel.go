package ui

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
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
	sample *sysinfo.Sampler

	systemOS              *widget.Label
	systemVersion         *widget.Label
	systemBuild           *widget.Label
	systemKernel          *widget.Label
	systemArch            *widget.Label
	systemHostname        *widget.Label
	systemCPUModel        *widget.Label
	systemCPUVendor       *widget.Label
	systemCPUIdentifier   *widget.Label
	systemCPUMHz          *widget.Label
	systemCPUCores        *widget.Label
	systemCPUUsage        *widget.Label
	systemMemoryTotal     *widget.Label
	systemMemoryAvailable *widget.Label
	systemMemoryUsed      *widget.Label
	systemMemoryUsage     *widget.Label

	selfPID       *widget.Label
	selfName      *widget.Label
	selfCPU       *widget.Label
	selfMemory    *widget.Label
	selfHeapAlloc *widget.Label
	selfHeapSys   *widget.Label
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
	refreshing    atomic.Bool
	lastRefresh   atomic.Int64
	visible       bool

	// Cached values to avoid calling SetText when nothing changed.
	cache struct {
		systemOS              string
		systemVersion         string
		systemBuild           string
		systemKernel          string
		systemArch            string
		systemHostname        string
		systemCPUModel        string
		systemCPUVendor       string
		systemCPUIdentifier   string
		systemCPUMHz          string
		systemCPUCores        string
		systemCPUUsage        string
		systemMemoryTotal     string
		systemMemoryAvailable string
		systemMemoryUsed      string
		systemMemoryUsage     string

		selfPID       string
		selfName      string
		selfCPU       string
		selfMemory    string
		selfHeapAlloc string
		selfHeapSys   string
		selfStartTime string
		selfUptime    string
		chromeStatus  string
		chromePID     string
		chromeName    string
		chromeCPU     string
		chromeMemory  string
	}
}

const infoPanelMinRefreshInterval = time.Second

func newInfoPanel(app *App) *infoPanel {
	p := &infoPanel{app: app, sample: sysinfo.NewSampler()}

	p.systemOS = widget.NewLabel("")
	p.systemVersion = widget.NewLabel("")
	p.systemBuild = widget.NewLabel("")
	p.systemKernel = widget.NewLabel("")
	p.systemArch = widget.NewLabel("")
	p.systemHostname = widget.NewLabel("")
	p.systemCPUModel = widget.NewLabel("")
	p.systemCPUVendor = widget.NewLabel("")
	p.systemCPUIdentifier = widget.NewLabel("")
	p.systemCPUMHz = widget.NewLabel("")
	p.systemCPUCores = widget.NewLabel("")
	p.systemCPUUsage = widget.NewLabel("")
	p.systemMemoryTotal = widget.NewLabel("")
	p.systemMemoryAvailable = widget.NewLabel("")
	p.systemMemoryUsed = widget.NewLabel("")
	p.systemMemoryUsage = widget.NewLabel("")
	p.selfPID = widget.NewLabel("")
	p.selfName = widget.NewLabel("")
	p.selfCPU = widget.NewLabel("")
	p.selfMemory = widget.NewLabel("")
	p.selfHeapAlloc = widget.NewLabel("")
	p.selfHeapSys = widget.NewLabel("")
	p.selfStartTime = widget.NewLabel("")
	p.selfUptime = widget.NewLabel("")
	p.chromeStatus = widget.NewLabel("")
	p.chromePID = widget.NewLabel("")
	p.chromeName = widget.NewLabel("")
	p.chromeCPU = widget.NewLabel("")
	p.chromeMemory = widget.NewLabel("")

	systemForm := widget.NewForm(
		widget.NewFormItem("系统", p.systemOS),
		widget.NewFormItem("版本", p.systemVersion),
		widget.NewFormItem("构建", p.systemBuild),
		widget.NewFormItem("内核", p.systemKernel),
		widget.NewFormItem("架构", p.systemArch),
		widget.NewFormItem("主机名", p.systemHostname),
		widget.NewFormItem("CPU 型号", p.systemCPUModel),
		widget.NewFormItem("CPU 厂商", p.systemCPUVendor),
		widget.NewFormItem("CPU 标识", p.systemCPUIdentifier),
		widget.NewFormItem("CPU 频率", p.systemCPUMHz),
		widget.NewFormItem("CPU 核心", p.systemCPUCores),
		widget.NewFormItem("CPU 占用", p.systemCPUUsage),
		widget.NewFormItem("总内存", p.systemMemoryTotal),
		widget.NewFormItem("可用内存", p.systemMemoryAvailable),
		widget.NewFormItem("已用内存", p.systemMemoryUsed),
		widget.NewFormItem("内存占用", p.systemMemoryUsage),
	)

	selfForm := widget.NewForm(
		widget.NewFormItem("PID", p.selfPID),
		widget.NewFormItem("名称", p.selfName),
		widget.NewFormItem("CPU", p.selfCPU),
		widget.NewFormItem("内存 (RSS)", p.selfMemory),
		widget.NewFormItem("Go 堆已用", p.selfHeapAlloc),
		widget.NewFormItem("Go 堆保留", p.selfHeapSys),
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
		go p.refresh()
	})
	refreshBtn.Importance = widget.MediumImportance

	content := container.NewVBox(
		newSectionHeader("系统信息", refreshBtn),
		newMutedText(fmt.Sprintf("平台：%s/%s", runtime.GOOS, runtime.GOARCH)),
		systemForm,

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
		go p.refresh()
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
	if !p.beginRefresh() {
		return
	}

	snap := p.collectSnapshot()
	fyne.Do(func() {
		defer p.refreshing.Store(false)
		if p.visible {
			p.applySnapshot(snap)
			p.lastRefresh.Store(time.Now().UnixNano())
		}
	})
}

func (p *infoPanel) beginRefresh() bool {
	now := time.Now()
	last := p.lastRefresh.Load()
	if last > 0 && now.Sub(time.Unix(0, last)) < infoPanelMinRefreshInterval {
		return false
	}
	if !p.refreshing.CompareAndSwap(false, true) {
		return false
	}
	last = p.lastRefresh.Load()
	if last > 0 && now.Sub(time.Unix(0, last)) < infoPanelMinRefreshInterval {
		p.refreshing.Store(false)
		return false
	}
	return true
}

type infoSnapshot struct {
	systemOS              string
	systemVersion         string
	systemBuild           string
	systemKernel          string
	systemArch            string
	systemHostname        string
	systemCPUModel        string
	systemCPUVendor       string
	systemCPUIdentifier   string
	systemCPUMHz          string
	systemCPUCores        string
	systemCPUUsage        string
	systemMemoryTotal     string
	systemMemoryAvailable string
	systemMemoryUsed      string
	systemMemoryUsage     string

	selfPID       string
	selfName      string
	selfCPU       string
	selfMemory    string
	selfHeapAlloc string
	selfHeapSys   string
	selfStartTime string
	selfUptime    string
	chromeStatus  string
	chromePID     string
	chromeName    string
	chromeCPU     string
	chromeMemory  string
}

func (p *infoPanel) collectSnapshot() infoSnapshot {
	var snap infoSnapshot

	system, err := p.sample.SystemInfo()
	if err != nil {
		snap.systemOS = "读取失败"
	} else {
		snap.systemOS = fallbackText(system.OSName)
		snap.systemVersion = fallbackText(system.OSVersion)
		snap.systemBuild = fallbackText(system.OSBuild)
		snap.systemKernel = fallbackText(system.Kernel)
		snap.systemArch = fallbackText(system.Arch)
		snap.systemHostname = fallbackText(system.Hostname)
		snap.systemCPUModel = fallbackText(system.CPUModel)
		snap.systemCPUVendor = fallbackText(system.CPUVendor)
		snap.systemCPUIdentifier = fallbackText(system.CPUIdentifier)
		snap.systemCPUMHz = formatMHz(system.CPUMHz)
		snap.systemCPUCores = formatCores(system.LogicalCPUs, system.PhysicalCores)
		snap.systemCPUUsage = sysinfo.FormatCPU(system.CPUUsage)
		snap.systemMemoryTotal = formatOptionalMemory(system.MemoryTotalMB)
		snap.systemMemoryAvailable = formatOptionalMemory(system.MemoryAvailableMB)
		snap.systemMemoryUsed = formatOptionalMemory(system.MemoryUsedMB)
		snap.systemMemoryUsage = sysinfo.FormatCPU(system.MemoryUsagePercent)
	}

	self, start, uptime, err := p.sample.SelfInfoWithUptime()
	if err != nil && !self.Exists {
		snap.selfPID = "读取失败"
	} else {
		snap.selfPID = fmt.Sprintf("%d", self.PID)
		snap.selfName = self.Name
		snap.selfCPU = sysinfo.FormatCPU(self.CPU)
		snap.selfMemory = sysinfo.FormatMemory(self.MemoryMB)
		heapAlloc, heapSys := sysinfo.GoMemStats()
		snap.selfHeapAlloc = sysinfo.FormatMemory(heapAlloc)
		snap.selfHeapSys = sysinfo.FormatMemory(heapSys)
		if err == nil {
			snap.selfStartTime = sysinfo.FormatStartTime(start)
		} else {
			snap.selfStartTime = "-"
		}
		if err == nil {
			snap.selfUptime = sysinfo.FormatUptime(uptime)
		} else {
			snap.selfUptime = "-"
		}
	}

	chromePID := 0
	if p.app.browserMgr != nil {
		chromePID = p.app.browserMgr.ManagedPID()
	}
	chrome, err := p.sample.ChromeInfo(chromePID)
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
		snap.chromeStatus = status
		snap.chromePID = "-"
		snap.chromeName = "-"
		snap.chromeCPU = "-"
		snap.chromeMemory = "-"
	} else {
		snap.chromeStatus = "运行中"
		snap.chromePID = fmt.Sprintf("%d", chrome.PID)
		snap.chromeName = chrome.Name
		snap.chromeCPU = sysinfo.FormatCPU(chrome.CPU)
		snap.chromeMemory = sysinfo.FormatMemory(chrome.MemoryMB)
	}

	return snap
}

func (p *infoPanel) applySnapshot(snap infoSnapshot) {
	p.setLabel(p.systemOS, &p.cache.systemOS, snap.systemOS)
	p.setLabel(p.systemVersion, &p.cache.systemVersion, snap.systemVersion)
	p.setLabel(p.systemBuild, &p.cache.systemBuild, snap.systemBuild)
	p.setLabel(p.systemKernel, &p.cache.systemKernel, snap.systemKernel)
	p.setLabel(p.systemArch, &p.cache.systemArch, snap.systemArch)
	p.setLabel(p.systemHostname, &p.cache.systemHostname, snap.systemHostname)
	p.setLabel(p.systemCPUModel, &p.cache.systemCPUModel, snap.systemCPUModel)
	p.setLabel(p.systemCPUVendor, &p.cache.systemCPUVendor, snap.systemCPUVendor)
	p.setLabel(p.systemCPUIdentifier, &p.cache.systemCPUIdentifier, snap.systemCPUIdentifier)
	p.setLabel(p.systemCPUMHz, &p.cache.systemCPUMHz, snap.systemCPUMHz)
	p.setLabel(p.systemCPUCores, &p.cache.systemCPUCores, snap.systemCPUCores)
	p.setLabel(p.systemCPUUsage, &p.cache.systemCPUUsage, snap.systemCPUUsage)
	p.setLabel(p.systemMemoryTotal, &p.cache.systemMemoryTotal, snap.systemMemoryTotal)
	p.setLabel(p.systemMemoryAvailable, &p.cache.systemMemoryAvailable, snap.systemMemoryAvailable)
	p.setLabel(p.systemMemoryUsed, &p.cache.systemMemoryUsed, snap.systemMemoryUsed)
	p.setLabel(p.systemMemoryUsage, &p.cache.systemMemoryUsage, snap.systemMemoryUsage)
	p.setLabel(p.selfPID, &p.cache.selfPID, snap.selfPID)
	p.setLabel(p.selfName, &p.cache.selfName, snap.selfName)
	p.setLabel(p.selfCPU, &p.cache.selfCPU, snap.selfCPU)
	p.setLabel(p.selfMemory, &p.cache.selfMemory, snap.selfMemory)
	p.setLabel(p.selfHeapAlloc, &p.cache.selfHeapAlloc, snap.selfHeapAlloc)
	p.setLabel(p.selfHeapSys, &p.cache.selfHeapSys, snap.selfHeapSys)
	p.setLabel(p.selfStartTime, &p.cache.selfStartTime, snap.selfStartTime)
	p.setLabel(p.selfUptime, &p.cache.selfUptime, snap.selfUptime)
	p.setLabel(p.chromeStatus, &p.cache.chromeStatus, snap.chromeStatus)
	p.setLabel(p.chromePID, &p.cache.chromePID, snap.chromePID)
	p.setLabel(p.chromeName, &p.cache.chromeName, snap.chromeName)
	p.setLabel(p.chromeCPU, &p.cache.chromeCPU, snap.chromeCPU)
	p.setLabel(p.chromeMemory, &p.cache.chromeMemory, snap.chromeMemory)
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

func fallbackText(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func formatOptionalMemory(mb float64) string {
	if mb <= 0 {
		return "-"
	}
	return sysinfo.FormatMemory(mb)
}

func formatMHz(mhz int) string {
	if mhz <= 0 {
		return "-"
	}
	if mhz >= 1000 {
		return fmt.Sprintf("%.2f GHz", float64(mhz)/1000)
	}
	return fmt.Sprintf("%d MHz", mhz)
}

func formatCores(logical, physical int) string {
	if logical <= 0 && physical <= 0 {
		return "-"
	}
	if physical > 0 {
		return fmt.Sprintf("%d 逻辑 / %d 物理", logical, physical)
	}
	return fmt.Sprintf("%d 逻辑", logical)
}
