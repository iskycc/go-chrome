package ui

import (
	"fmt"
	"image/color"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/runner"
)

type runPanel struct {
	app    *App
	widget fyne.CanvasObject

	progressBar  *widget.ProgressBar
	progressText *progressLabel
	logBox       *fyne.Container
	logScroll    *container.Scroll
	summary      *widget.Label
	currentStep  *currentStepLabel
	artifactBox  *fyne.Container
	artifactDir  string
}

func newRunPanel(app *App) *runPanel {
	p := &runPanel{app: app}

	p.progressBar = widget.NewProgressBar()
	p.progressBar.Min = 0
	p.progressBar.Max = 1
	p.progressText = newProgressLabel(180, 520)

	p.logBox = container.NewVBox()
	p.logScroll = container.NewScroll(p.logBox)

	p.summary = widget.NewLabel("成功：0  失败：0  跳过：0  总耗时：0.0s")
	p.currentStep = newCurrentStepLabel()
	p.artifactBox = container.NewHBox()

	clearLogBtn := widget.NewButtonWithIcon("清空日志", theme.DeleteIcon(), func() {
		p.clearLog()
	})
	copyLogBtn := widget.NewButtonWithIcon("复制日志", theme.ContentCopyIcon(), func() {
		p.copyLog()
	})
	openArtifactBtn := widget.NewButtonWithIcon("打开产物目录", theme.FolderOpenIcon(), func() {
		p.openArtifactDir()
	})

	var moreBtn *widget.Button
	moreBtn = widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() {
		menu := fyne.NewMenu("运行操作",
			fyne.NewMenuItemWithIcon("浏览器下载配置", theme.ComputerIcon(), func() {
				if p.app.moduleTabs != nil {
					p.app.moduleTabs.SelectIndex(4)
				}
			}),
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, moreBtn.Size().Height), moreBtn)
	})

	progressArea := container.NewVBox(p.progressText.box, p.progressBar)
	actionBtns := container.NewHBox(clearLogBtn, copyLogBtn, openArtifactBtn, moreBtn)
	topBar := container.NewBorder(nil, nil, nil, actionBtns, progressArea)

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("运行摘要", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.summary,
		p.currentStep.box,
		widget.NewLabelWithStyle("产物", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.artifactBox,
	)

	p.widget = container.NewBorder(
		topBar,
		nil,
		nil,
		rightPanel,
		p.logScroll,
	)
	return p
}

func (p *runPanel) log(msg string) {
	fyne.Do(func() {
		line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		item := newLogLine(line, logColor(msg), p)
		p.logBox.Add(item)
		if len(p.logBox.Objects) > 300 {
			p.logBox.Objects = p.logBox.Objects[len(p.logBox.Objects)-300:]
		}
		p.logBox.Refresh()
		p.logScroll.ScrollToBottom()
	})
}

func logColor(msg string) color.Color {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(msg, "失败") || strings.Contains(msg, "错误") || strings.Contains(lower, "error") || strings.Contains(lower, "failed"):
		return color.NRGBA{R: 220, G: 38, B: 38, A: 255}
	case strings.Contains(msg, "未检测") || strings.Contains(msg, "缺少") || strings.Contains(lower, "warn"):
		return color.NRGBA{R: 180, G: 83, B: 9, A: 255}
	case strings.Contains(msg, "成功") || strings.Contains(msg, "完成") || strings.Contains(msg, "就绪") || strings.Contains(msg, "已检测") || strings.Contains(lower, "success"):
		return color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	case strings.Contains(msg, "下载") || strings.Contains(msg, "启动") || strings.Contains(msg, "运行") || strings.Contains(msg, "进度"):
		return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	default:
		return color.NRGBA{R: 55, G: 65, B: 81, A: 255}
	}
}

func (p *runPanel) setProgress(current, total int, stepName string) {
	fyne.Do(func() {
		if total > 0 {
			p.progressBar.Max = float64(total)
			p.progressBar.SetValue(float64(current))
		}
		p.progressText.set(current, total, stepName)
	})
}

func (p *runPanel) setSummary(res *runner.RunResult) {
	fyne.Do(func() {
		elapsed := res.FinishedAt.Sub(res.StartedAt).Seconds()
		p.summary.SetText(fmt.Sprintf("成功：%d  失败：%d  跳过：%d  总耗时：%.1fs", res.SuccessCount, res.FailedCount, res.SkippedCount, elapsed))
	})
}

func (p *runPanel) setCurrentStep(name string) {
	fyne.Do(func() {
		p.currentStep.SetText(name)
	})
}

func (p *runPanel) setArtifacts(screenshot, htmlSnap string) {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
		p.artifactDir = ""
		if screenshot != "" {
			label := newContextMenuLabel("截图："+screenshot, func(e *fyne.PointEvent) {
				p.showArtifactContextMenu(screenshot, "截图路径", e)
			})
			p.artifactBox.Objects = append(p.artifactBox.Objects, label)
			p.artifactDir = filepath.Dir(screenshot)
		}
		if htmlSnap != "" {
			label := newContextMenuLabel("HTML："+htmlSnap, func(e *fyne.PointEvent) {
				p.showArtifactContextMenu(htmlSnap, "HTML 路径", e)
			})
			p.artifactBox.Objects = append(p.artifactBox.Objects, label)
			if p.artifactDir == "" {
				p.artifactDir = filepath.Dir(htmlSnap)
			}
		}
		p.artifactBox.Refresh()
	})
}

func (p *runPanel) clearArtifacts() {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
		p.artifactDir = ""
		p.artifactBox.Refresh()
	})
}

func (p *runPanel) reset() {
	fyne.Do(func() {
		p.progressBar.SetValue(0)
		p.progressText.set(0, 0, "")
		p.currentStep.SetText("")
		p.clearArtifacts()
	})
}

func (p *runPanel) setRunning(running bool) {
	// Running state is now handled by the global toolbar; kept for caller compatibility.
	_ = running
}

func (p *runPanel) setChromeManaged(managed bool) {
	// Managed Chrome state is now handled by the global toolbar; kept for caller compatibility.
	_ = managed
}

func (p *runPanel) refreshEnvironments() {
	// Environment selection moved to the global toolbar; kept for caller compatibility.
}

func (p *runPanel) clearLog() {
	fyne.Do(func() {
		p.logBox.Objects = nil
		p.logBox.Refresh()
	})
}

func (p *runPanel) copyLog() {
	fyne.Do(func() {
		var lines []string
		for _, obj := range p.logBox.Objects {
			if item, ok := obj.(*logLine); ok {
				lines = append(lines, item.text)
			}
		}
		p.app.fyneApp.Clipboard().SetContent(strings.Join(lines, "\n"))
		p.log("日志已复制到剪贴板")
	})
}

func (p *runPanel) copyLogLine(text string) {
	fyne.Do(func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(text))
		p.log("当前日志行已复制到剪贴板")
	})
}

func (p *runPanel) openArtifactDir() {
	fyne.Do(func() {
		if p.artifactDir == "" {
			p.log("暂无产物目录")
			return
		}
		uri, err := url.Parse(storage.NewFileURI(p.artifactDir).String())
		if err != nil {
			p.log("打开产物目录失败：" + err.Error())
			return
		}
		if err := p.app.fyneApp.OpenURL(uri); err != nil {
			p.log("打开产物目录失败：" + err.Error())
		}
	})
}

func (p *runPanel) copyArtifactDir() {
	fyne.Do(func() {
		if p.artifactDir == "" {
			p.log("暂无产物目录")
			return
		}
		p.app.fyneApp.Clipboard().SetContent(clipCopy(p.artifactDir))
		p.log("产物目录路径已复制到剪贴板")
	})
}

func (p *runPanel) showArtifactContextMenu(path, kind string, e *fyne.PointEvent) {
	copyItem := fyne.NewMenuItem("复制"+kind, func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(path))
		p.log(kind + "已复制到剪贴板")
	})
	openDirItem := fyne.NewMenuItem("打开产物目录", func() {
		p.openArtifactDir()
	})
	openDirItem.Disabled = p.artifactDir == ""
	menu := fyne.NewMenu("产物操作", copyItem, openDirItem)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}

func (p *runPanel) showLogContextMenu(lineText string, e *fyne.PointEvent) {
	copyLineItem := fyne.NewMenuItem("复制当前日志行", func() {
		p.copyLogLine(lineText)
	})
	copyAllItem := fyne.NewMenuItem("复制全部日志", func() {
		p.copyLog()
	})
	clearItem := fyne.NewMenuItem("清空日志", func() {
		p.clearLog()
	})
	openDirItem := fyne.NewMenuItem("打开产物目录", func() {
		p.openArtifactDir()
	})
	openDirItem.Disabled = p.artifactDir == ""
	copyDirItem := fyne.NewMenuItem("复制产物目录路径", func() {
		p.copyArtifactDir()
	})
	copyDirItem.Disabled = p.artifactDir == ""

	menu := fyne.NewMenu("日志操作",
		copyLineItem,
		copyAllItem,
		fyne.NewMenuItemSeparator(),
		clearItem,
		fyne.NewMenuItemSeparator(),
		openDirItem,
		copyDirItem,
	)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}

// logLine is a single log entry that supports right-click context menu.
type logLine struct {
	widget.BaseWidget
	text    string
	p       *runPanel
	textObj *canvas.Text
}

func newLogLine(text string, color color.Color, p *runPanel) *logLine {
	item := &logLine{text: text, p: p}
	item.ExtendBaseWidget(item)
	item.textObj = canvas.NewText(text, color)
	item.textObj.TextSize = 13
	item.textObj.TextStyle = fyne.TextStyle{Monospace: true}
	return item
}

func (l *logLine) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout(l.textObj))
}

func (l *logLine) MinSize() fyne.Size {
	return l.textObj.MinSize().Add(fyne.NewSize(0, 2))
}

func (l *logLine) TappedSecondary(e *fyne.PointEvent) {
	l.p.showLogContextMenu(l.text, e)
}
