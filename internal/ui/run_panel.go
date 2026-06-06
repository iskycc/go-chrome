package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
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
	logsEntry    *widget.Entry
	summary      *widget.Label
	currentStep  *widget.Label
	artifactBox  *fyne.Container
}

func newRunPanel(app *App) *runPanel {
	p := &runPanel{app: app}

	p.progressBar = widget.NewProgressBar()
	p.progressBar.Min = 0
	p.progressBar.Max = 1
	p.progressText = widget.NewLabel("就绪")

	p.logsEntry = widget.NewMultiLineEntry()
	p.logsEntry.Disable()
	p.logsEntry.Wrapping = fyne.TextWrapWord

	p.summary = widget.NewLabel("成功：0  失败：0  跳过：0  总耗时：0.0s")
	p.currentStep = widget.NewLabel("")
	p.artifactBox = container.NewHBox()

	startBtn := widget.NewButtonWithIcon("启动浏览器", theme.ViewRefreshIcon(), func() {
		go p.app.startBrowser()
	})
	runBtn := widget.NewButtonWithIcon("运行整个流程", theme.MediaPlayIcon(), func() {
		go p.app.runCurrentFlow()
	})
	stepBtn := widget.NewButtonWithIcon("单步执行", theme.MediaReplayIcon(), func() {
		go p.app.onStepButton()
	})
	p.app.stepBtn = stepBtn
	stopBtn := widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		p.app.runner.Stop()
	})

	controls := container.NewHBox(startBtn, runBtn, stepBtn, stopBtn)

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("运行摘要", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.summary,
	)

	p.widget = container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, p.progressText, controls, p.progressBar),
		),
		container.NewHBox(p.currentStep, p.artifactBox),
		rightPanel, nil,
		container.NewScroll(p.logsEntry),
	)
	return p
}

func (p *runPanel) log(msg string) {
	fyne.Do(func() {
		p.logsEntry.SetText(p.logsEntry.Text + msg + "\n")
	})
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
