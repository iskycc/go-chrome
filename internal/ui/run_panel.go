package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	envSelect    *widget.Select
	stopBtn      *widget.Button
}

func newRunPanel(app *App) *runPanel {
	p := &runPanel{app: app}

	p.progressBar = widget.NewProgressBar()
	p.progressBar.Min = 0
	p.progressBar.Max = 1
	p.progressText = widget.NewLabel("就绪")

	p.logBox = container.NewVBox()
	p.logScroll = container.NewScroll(p.logBox)

	p.summary = widget.NewLabel("成功：0  失败：0  跳过：0  总耗时：0.0s")
	p.currentStep = widget.NewLabel("")
	p.artifactBox = container.NewHBox()

	runBtn := widget.NewButtonWithIcon("运行整个流程", theme.MediaPlayIcon(), func() {
		go p.app.runCurrentFlow()
	})
	stepBtn := widget.NewButtonWithIcon("单步执行", theme.MediaReplayIcon(), func() {
		go p.app.onStepButton()
	})
	p.app.stepBtn = stepBtn
	p.stopBtn = widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		p.app.runner.Stop()
	})
	p.stopBtn.Hide()

	var moreBtn *widget.Button
	moreBtn = widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() {
		menu := fyne.NewMenu("运行操作",
			fyne.NewMenuItemWithIcon("启动浏览器", theme.ViewRefreshIcon(), func() { go p.app.startBrowser() }),
			fyne.NewMenuItemWithIcon("管理环境", theme.SettingsIcon(), func() { p.app.showEnvManager() }),
			fyne.NewMenuItemWithIcon("浏览器下载配置", theme.ComputerIcon(), func() {
				if p.app.moduleTabs != nil {
					p.app.moduleTabs.SelectIndex(4)
				}
			}),
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, moreBtn.Size().Height), moreBtn)
	})

	p.envSelect = widget.NewSelect([]string{"默认环境"}, func(name string) {
		if p.app.envRepo == nil || name == "" {
			return
		}
		env, err := p.app.envRepo.GetByName(name)
		if err != nil {
			p.log("切换环境失败：" + err.Error())
			return
		}
		if err := p.app.envRepo.SetActive(env.ID); err != nil {
			p.log("保存当前环境失败：" + err.Error())
			return
		}
	})
	p.envSelect.SetSelected("默认环境")

	controls := container.NewHBox(runBtn, stepBtn, p.stopBtn, moreBtn)

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("运行摘要", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.summary,
	)

	topBar := container.NewVBox(
		container.NewBorder(nil, nil, p.progressText, controls, p.progressBar),
		container.NewHBox(widget.NewLabel("当前环境："), p.envSelect),
	)

	p.widget = container.NewBorder(
		topBar,
		container.NewHBox(p.currentStep, p.artifactBox),
		rightPanel, nil,
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
		if screenshot != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, widget.NewLabel("截图："+screenshot))
		}
		if htmlSnap != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, widget.NewLabel("HTML："+htmlSnap))
		}
		p.artifactBox.Refresh()
	})
}

func (p *runPanel) clearArtifacts() {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
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
	fyne.Do(func() {
		if p.stopBtn == nil {
			return
		}
		if running {
			p.stopBtn.Show()
		} else {
			p.stopBtn.Hide()
		}
	})
}

func (p *runPanel) refreshEnvironments() {
	if p.app.envRepo == nil {
		return
	}
	envs, err := p.app.envRepo.List()
	if err != nil {
		return
	}
	var names []string
	var active string
	for _, e := range envs {
		names = append(names, e.Name)
		if e.IsActive {
			active = e.Name
		}
	}
	if len(names) == 0 {
		names = []string{"默认环境"}
		active = "默认环境"
	}
	fyne.Do(func() {
		p.envSelect.Options = names
		p.envSelect.SetSelected(active)
	})
}
