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
	progressText *widget.Label
	logBox       *fyne.Container
	logScroll    *container.Scroll
	summary      *widget.Label
	currentStep  *widget.Label
	artifactBox  *fyne.Container
	closeChBtn   *widget.Button
	artifactDir  string
}

func newRunPanel(app *App) *runPanel {
	p := &runPanel{app: app}

	p.progressBar = widget.NewProgressBar()
	p.progressBar.Min = 0
	p.progressBar.Max = 1
	p.progressText = newTruncatingLabel("就绪")

	p.logBox = container.NewVBox()
	p.logScroll = container.NewScroll(p.logBox)

	p.summary = widget.NewLabel("成功：0  失败：0  跳过：0  总耗时：0.0s")
	p.currentStep = newTruncatingLabel("")
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

	p.closeChBtn = widget.NewButtonWithIcon("关闭本程序启动的 Chrome", theme.CancelIcon(), func() {
		p.app.closeManagedChrome()
	})
	p.closeChBtn.Disable()
	p.closeChBtn.Importance = widget.MediumImportance

	var moreBtn *widget.Button
	moreBtn = widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() {
		menu := fyne.NewMenu("运行操作",
			fyne.NewMenuItemWithIcon("关闭本程序启动的 Chrome", theme.CancelIcon(), func() { p.app.closeManagedChrome() }),
			fyne.NewMenuItemWithIcon("浏览器下载配置", theme.ComputerIcon(), func() {
				if p.app.moduleTabs != nil {
					p.app.moduleTabs.SelectIndex(4)
				}
			}),
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, moreBtn.Size().Height), moreBtn)
	})

	progressArea := container.NewVBox(p.progressBar, p.progressText)
	actionBtns := container.NewHBox(clearLogBtn, copyLogBtn, openArtifactBtn, moreBtn)
	topBar := container.NewBorder(nil, nil, progressArea, actionBtns)

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("运行摘要", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.summary,
		widget.NewLabelWithStyle("当前步骤", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.currentStep,
		widget.NewLabelWithStyle("产物", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.artifactBox,
	)

	bottomBar := container.NewHBox(p.closeChBtn)

	p.widget = container.NewBorder(
		topBar,
		bottomBar,
		nil,
		rightPanel,
		p.logScroll,
	)
	return p
}

func (p *runPanel) log(msg string) {
	fyne.Do(func() {
		line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		text := canvas.NewText(line, logColor(msg))
		text.TextSize = 13
		text.TextStyle = fyne.TextStyle{Monospace: true}
		p.logBox.Add(text)
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
		p.progressText.SetText(fmt.Sprintf("第 %d 步 / 共 %d 步 · %s", current, total, stepName))
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
		p.currentStep.SetText("当前步骤：" + name)
	})
}

func (p *runPanel) setArtifacts(screenshot, htmlSnap string) {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
		p.artifactDir = ""
		if screenshot != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, newTruncatingLabel("截图："+screenshot))
			p.artifactDir = filepath.Dir(screenshot)
		}
		if htmlSnap != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, newTruncatingLabel("HTML："+htmlSnap))
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
		p.progressText.SetText("就绪")
		p.currentStep.SetText("")
		p.clearArtifacts()
	})
}

func (p *runPanel) setRunning(running bool) {
	// Running state is now handled by the global toolbar; kept for caller compatibility.
	_ = running
}

func (p *runPanel) setChromeManaged(managed bool) {
	fyne.Do(func() {
		if p.closeChBtn == nil {
			return
		}
		if managed {
			p.closeChBtn.Enable()
		} else {
			p.closeChBtn.Disable()
		}
	})
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
			if t, ok := obj.(*canvas.Text); ok {
				lines = append(lines, t.Text)
			}
		}
		p.app.fyneApp.Clipboard().SetContent(strings.Join(lines, "\n"))
		p.log("日志已复制到剪贴板")
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
