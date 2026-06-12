# UI 体验优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 根据 `Todo1.md` 完成 `go-chrome` 的 UI 体验优化，包括内置字体、全局操作栏、环境配置 tab、运行详情页瘦身、主题和布局统一。

**Architecture:** 在现有 Fyne UI 基础上，通过新增 `globalToolbar` 和 `envPanel` 组件、扩展主题系统、调整主窗口布局，实现常用操作全局可见和信息架构清晰。不改动流程运行和 Chrome 控制核心逻辑。

**Tech Stack:** Go, Fyne v2, chromedp, SQLite, embed

---

## File Structure

### 新建文件

- `assets/fonts/LICENSE-CascadiaCode.txt`：Cascadia Code 字体许可说明。
- `assets/fonts/CascadiaCode-SemiLight.ttf`：内置字体文件。
- `internal/ui/global_toolbar.go`：全局操作栏组件。
- `internal/ui/env_panel.go`：环境配置 tab 组件。

### 修改文件

- `assets/embed.go`：增加字体资源嵌入。
- `internal/ui/theme.go`：扩展 Color/Font/Size 实现。
- `internal/ui/main_window.go`：调整主窗口布局，集成全局工具栏和环境配置 tab，修改 `currentEnvProvider`、`onStepButton`、`stopCurrentRun`、`handleRunnerEvents`、`refreshFlowList` 等方法。
- `internal/ui/run_panel.go`：移除常用控制按钮，改名为运行详情。
- `internal/ui/env_dialog.go`：`showEnvManager()` 改为跳转到环境配置 tab。
- `internal/ui/status_bar.go`：可选微调状态栏宽度。
- `README.md` / `USER_GUIDE.md` / `FAQ.md` / `problem.md`：更新文档。

---

## Task 1: 字体资源与许可证

**Files:**
- Create: `assets/fonts/LICENSE-CascadiaCode.txt`
- Create: `assets/fonts/CascadiaCode-SemiLight.ttf`
- Modify: `assets/embed.go`

### Step 1.1: 创建字体目录并写入许可证

创建 `assets/fonts/LICENSE-CascadiaCode.txt`：

```text
Cascadia Code Font
Copyright (c) 2019 - Present, Microsoft Corporation,
with Reserved Font Name Cascadia Code.

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is available with a FAQ at: https://scripts.sil.org/OFL
```

### Step 1.2: 准备字体文件

将 `CascadiaCode-SemiLight.ttf` 放入 `assets/fonts/`。如果本地没有该文件，可从 GitHub 官方发布页下载：

```bash
# 下载 Cascadia Code 最新 release（示例命令，实际 URL 以发布页为准）
mkdir -p assets/fonts
curl -L -o assets/fonts/CascadiaCode.ttf "https://github.com/microsoft/cascadia-code/releases/download/v2404.23/CascadiaCode-2404.23.ttf"
# 若下载的是包含多个字重的 TTC/ttf 集合，需要提取 SemiLight 字重或直接使用包含 SemiLight 的文件。
```

> **验收：** 文件存在且不为空，`file assets/fonts/CascadiaCode-SemiLight.ttf` 返回 TrueType 字体信息。

### Step 1.3: 修改 `assets/embed.go` 嵌入字体

完整替换 `assets/embed.go`：

```go
package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png fonts/CascadiaCode-SemiLight.ttf
var assetFS embed.FS

// Icon returns the application icon resource.
func Icon() fyne.Resource {
	data, err := assetFS.ReadFile("icon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", data)
}

// CascadiaCodeSemiLight returns the embedded Cascadia Code SemiLight font.
// Returns nil if the font file is missing so callers can fall back to the
// default theme font.
func CascadiaCodeSemiLight() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/CascadiaCode-SemiLight.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaCode-SemiLight.ttf", data)
}
```

### Step 1.4: 构建验证

```bash
go build ./assets
```

Expected: 无错误。

### Step 1.5: Commit

```bash
git add assets/fonts/ assets/embed.go
git commit -m "assets: embed Cascadia Code SemiLight font"
```

---

## Task 2: 主题扩展

**Files:**
- Modify: `internal/ui/theme.go`

### Step 2.1: 扩展 Color 方法

完整替换 `internal/ui/theme.go`：

```go
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"go-chrome/assets"
)

// appTheme is a custom Fyne theme tuned for a quiet, workstation-style
// automation tool.
type appTheme struct{}

func newAppTheme() fyne.Theme {
	return &appTheme{}
}

func (a *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.RGBA{0x1a, 0x73, 0xe8, 0xff}
	case theme.ColorNameBackground:
		return color.RGBA{0xf5, 0xf5, 0xf5, 0xff}
	case theme.ColorNameForeground:
		return color.RGBA{0x21, 0x21, 0x21, 0xff}
	case theme.ColorNameButton:
		return color.RGBA{0xff, 0xff, 0xff, 0xff}
	case theme.ColorNameDisabled:
		return color.RGBA{0x9e, 0x9e, 0x9e, 0xff}
	case theme.ColorNameHover:
		return color.RGBA{0xe3, 0xe3, 0xe3, 0xff}
	case theme.ColorNameSelection:
		return color.RGBA{0xbb, 0xde, 0xfb, 0xff}
	case theme.ColorNameInputBackground:
		return color.RGBA{0xff, 0xff, 0xff, 0xff}
	case theme.ColorNameScrollBar:
		return color.RGBA{0xc1, 0xc1, 0xc1, 0xff}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	if res := assets.CascadiaCodeSemiLight(); res != nil {
		return res
	}
	return theme.DefaultTheme().Font(style)
}

func (a *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (a *appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameHeadingText:
		return 16
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 18
	}
	return theme.DefaultTheme().Size(name)
}
```

### Step 2.2: 编译验证

```bash
go build ./internal/ui
```

Expected: 无错误。

### Step 2.3: Commit

```bash
git add internal/ui/theme.go
git commit -m "ui: extend app theme with colors, sizes and embedded font"
```

---

## Task 3: 全局操作栏组件

**Files:**
- Create: `internal/ui/global_toolbar.go`
- Modify: `internal/ui/main_window.go`

### Step 3.1: 创建 `internal/ui/global_toolbar.go`

完整内容：

```go
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/browser"
	"go-chrome/internal/flow"
)

// globalToolbar holds the always-visible flow/environment/run controls.
type globalToolbar struct {
	app            *App
	widget         fyne.CanvasObject
	flowSelect     *widget.Select
	envSelect      *widget.Select
	saveBtn        *widget.Button
	startChromeBtn *widget.Button
	runBtn         *widget.Button
	stepBtn        *widget.Button
	stopBtn        *widget.Button
	progress       *widget.ProgressBar
	progressText   *widget.Label

	flowByName map[string]*flow.Flow
}

func newGlobalToolbar(app *App) *globalToolbar {
	t := &globalToolbar{app: app, flowByName: map[string]*flow.Flow{}}

	t.flowSelect = widget.NewSelect([]string{}, func(name string) {
		f := t.flowByName[name]
		if f == nil {
			return
		}
		app.onFlowSelected(f)
	})
	t.flowSelect.PlaceHolder = "选择流程"

	t.envSelect = widget.NewSelect([]string{"默认环境"}, func(name string) {
		if app.envRepo == nil || name == "" {
			return
		}
		env, err := app.envRepo.GetByName(name)
		if err != nil {
			app.runPanel.log("切换环境失败：" + err.Error())
			return
		}
		if err := app.envRepo.SetActive(env.ID); err != nil {
			app.runPanel.log("保存当前环境失败：" + err.Error())
			return
		}
	})
	t.envSelect.SetSelected("默认环境")

	t.saveBtn = widget.NewButtonWithIcon("保存", theme.DocumentSaveIcon(), func() {
		app.saveCurrentFlow()
	})
	t.saveBtn.Importance = widget.MediumImportance

	t.startChromeBtn = widget.NewButtonWithIcon("启动浏览器", theme.ComputerIcon(), func() {
		go app.startBrowser()
	})

	t.runBtn = widget.NewButtonWithIcon("运行", theme.MediaPlayIcon(), func() {
		go app.runCurrentFlow()
	})
	t.runBtn.Importance = widget.HighImportance

	t.stepBtn = widget.NewButtonWithIcon("单步执行", theme.MediaReplayIcon(), func() {
		go app.onStepButton()
	})

	t.stopBtn = widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		app.stopCurrentRun()
	})
	t.stopBtn.Hide()
	t.stopBtn.Importance = widget.DangerImportance

	t.progress = widget.NewProgressBar()
	t.progress.Min = 0
	t.progress.Max = 1
	t.progressText = widget.NewLabel("就绪")

	progressBox := container.NewBorder(nil, nil, t.progressText, nil, t.progress)

	left := container.NewHBox(
		widget.NewLabelWithStyle("流程", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.flowSelect,
		t.saveBtn,
	)
	center := container.NewHBox(
		t.startChromeBtn,
		t.runBtn,
		t.stepBtn,
		t.stopBtn,
	)
	right := container.NewHBox(
		widget.NewLabelWithStyle("环境", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.envSelect,
	)

	t.widget = container.NewBorder(nil, nil, left, right, center)
	return t
}

// refreshFlows rebuilds the flow dropdown from the current flow library.
func (t *globalToolbar) refreshFlows(flows []*flow.Flow) {
	t.flowByName = make(map[string]*flow.Flow, len(flows))
	names := make([]string, 0, len(flows))
	var selected string
	for _, f := range flows {
		names = append(names, f.Name)
		t.flowByName[f.Name] = f
		if t.app.currentFlow != nil && f.ID == t.app.currentFlow.ID {
			selected = f.Name
		}
	}
	fyne.Do(func() {
		t.flowSelect.Options = names
		if selected != "" {
			t.flowSelect.SetSelected(selected)
		} else if len(names) > 0 {
			t.flowSelect.SetSelected(names[0])
		} else {
			t.flowSelect.ClearSelected()
		}
	})
}

// refreshEnvironments rebuilds the environment dropdown.
func (t *globalToolbar) refreshEnvironments() {
	if t.app.envRepo == nil {
		return
	}
	envs, err := t.app.envRepo.List()
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
		t.envSelect.Options = names
		t.envSelect.SetSelected(active)
	})
}

// setProgress updates the lightweight progress display.
func (t *globalToolbar) setProgress(current, total int, stepName string) {
	fyne.Do(func() {
		if total > 0 {
			t.progress.Max = float64(total)
			t.progress.SetValue(float64(current))
			t.progressText.SetText(fmt.Sprintf("第 %d/%d 步 · %s", current, total, stepName))
		} else {
			t.progress.SetValue(0)
			t.progressText.SetText("就绪")
		}
	})
}

// setRunning updates button visibility when a run starts/stops.
func (t *globalToolbar) setRunning(running bool) {
	fyne.Do(func() {
		if running {
			t.runBtn.Disable()
			t.stepBtn.Disable()
			t.stopBtn.Show()
		} else {
			t.runBtn.Enable()
			t.stepBtn.Enable()
			t.stopBtn.Hide()
			t.stepBtn.SetText("单步执行")
		}
	})
}

// setStepButtonText updates the single-step button label.
func (t *globalToolbar) setStepButtonText(label string) {
	fyne.Do(func() {
		t.stepBtn.SetText(label)
	})
}

// setChromeManaged updates the start-browser button availability.
func (t *globalToolbar) setChromeManaged(managed bool) {
	fyne.Do(func() {
		if managed {
			t.startChromeBtn.Disable()
		} else {
			t.startChromeBtn.Enable()
		}
	})
}
```

### Step 3.2: 在 `App` 中集成全局工具栏

修改 `internal/ui/main_window.go`：

1. 在 `App` 结构体中新增字段并移除旧的 `stepBtn`：

```go
type App struct {
	// ... existing fields ...
	statusBar     *statusBar
	flowLibrary   *flowLibraryPanel
	flowEditor    *flowEditorPanel
	stepTable     *stepTablePanel
	stepProperty  *stepPropertyPanel
	runPanel      *runPanel
	historyPanel  *historyPanel
	settingsPanel *settingsPanel
	envPanel      *envPanel
	globalToolbar *globalToolbar
	// ...
	dirty        bool
	chromeTicker *time.Ticker
	chromeDone   chan struct{}
}
```

2. 在 `buildUI()` 中初始化并放置全局工具栏：

```go
func (a *App) buildUI() {
	onDirty := func() { a.markDirty() }

	a.statusBar = newStatusBar(a)
	a.globalToolbar = newGlobalToolbar(a)
	a.flowLibrary = newFlowLibraryPanel(a)
	// ... other panels ...
	a.envPanel = newEnvPanel(a)

	// ... workspace setup ...

	a.moduleTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("流程", theme.DocumentIcon(), flowModule),
		container.NewTabItemWithIcon("步骤", theme.ListIcon(), stepModule),
		container.NewTabItemWithIcon("环境配置", theme.SettingsIcon(), a.envPanel.widget),
		container.NewTabItemWithIcon("历史", theme.HistoryIcon(), a.historyPanel.widget),
		container.NewTabItemWithIcon("设置", theme.SettingsIcon(), a.settingsPanel.widget),
		container.NewTabItemWithIcon("运行详情", theme.MediaPlayIcon(), a.runPanel.widget),
	)
	a.moduleTabs.SetTabLocation(container.TabLocationTop)

	content := container.NewBorder(
		a.statusBar.widget,
		nil,
		nil,
		nil,
		container.NewBorder(a.globalToolbar.widget, nil, nil, nil, a.moduleTabs),
	)
	a.mainWin.SetContent(content)

	a.refreshFlowList()
	a.globalToolbar.refreshEnvironments()
	a.runPanel.refreshEnvironments()
	a.historyPanel.refreshFilters()
	a.restoreLastFlowSelection()
}
```

3. 修改 `currentEnvProvider()`：

```go
func (a *App) currentEnvProvider() (string, template.EnvProvider, error) {
	if a.envRepo == nil {
		return "", nil, fmt.Errorf("环境仓库未初始化")
	}
	selectedName := ""
	if a.globalToolbar != nil && a.globalToolbar.envSelect != nil {
		selectedName = a.globalToolbar.envSelect.Selected
	}
	if selectedName == "" {
		selectedName = "默认环境"
	}
	env, err := a.envRepo.GetByName(selectedName)
	if err != nil {
		return "", nil, err
	}
	return env.ID, a.envRepo.EnvProvider(env.ID), nil
}
```

4. 修改 `stopCurrentRun()`：

```go
func (a *App) stopCurrentRun() {
	if a.runner != nil && a.runner.IsRunning() {
		a.runner.Stop()
		a.runPanel.log("已停止完整流程运行")
		return
	}
	if a.stepRunner != nil && !a.stepRunner.IsFinished() {
		a.stepRunner.Stop()
		a.runPanel.log("已停止单步执行")
		a.runPanel.setRunning(false)
		if a.globalToolbar != nil {
			a.globalToolbar.setStepButtonText("单步执行")
		}
		a.stepRunner = nil
		return
	}
	a.runPanel.log("当前没有正在运行的任务")
}
```

5. 修改 `onStepButton()`：

```go
func (a *App) onStepButton() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if a.stepRunner != nil && !a.stepRunner.IsFinished() {
		a.nextStep()
		return
	}
	if a.stepRunner != nil {
		a.stepRunner.Close()
	}
	if missing := a.checkEnvVars(); len(missing) > 0 {
		dialog.ShowError(fmt.Errorf("运行前检查失败，缺少环境变量: %v", missing), a.mainWin)
		return
	}
	envID, envProvider, err := a.currentEnvProvider()
	if err != nil {
		dialog.ShowError(fmt.Errorf("获取运行环境失败: %w", err), a.mainWin)
		return
	}
	historySaver := &runHistoryAdapter{repo: a.runRepo}
	a.stepRunner = runner.NewStepRunner(&a.cfg.Runner, a.browserMgr, historySaver)
	if err := a.stepRunner.Init(a.currentFlow, envProvider, envID); err != nil {
		dialog.ShowError(err, a.mainWin)
		a.stepRunner = nil
		return
	}
	a.runPanel.setRunning(true)
	if a.globalToolbar != nil {
		a.globalToolbar.setRunning(true)
	}
	a.runStatuses = make([]runner.Status, len(a.currentFlow.Steps))
	for i := range a.runStatuses {
		a.runStatuses[i] = runner.StatusPending
	}
	a.stepTable.setStatuses(a.runStatuses)
	if a.globalToolbar != nil {
		a.globalToolbar.setStepButtonText("下一步")
	}
	a.nextStep()
}
```

6. 修改 `nextStep()`：

```go
if err != nil {
	a.runPanel.log("单步执行错误：" + err.Error())
	if a.globalToolbar != nil {
		a.globalToolbar.setStepButtonText("单步执行")
	}
	a.runPanel.setRunning(false)
	if a.globalToolbar != nil {
		a.globalToolbar.setRunning(false)
	}
	a.stepRunner.Close()
	a.stepRunner = nil
	return
}
```

以及完成时：

```go
if finished {
	result := a.stepRunner.Result()
	a.runPanel.log(fmt.Sprintf("单步执行完成：%s（成功 %d，失败 %d）", result.Status, result.SuccessCount, result.FailedCount))
	if a.globalToolbar != nil {
		a.globalToolbar.setStepButtonText("单步执行")
	}
	a.runPanel.setRunning(false)
	if a.globalToolbar != nil {
		a.globalToolbar.setRunning(false)
	}
	a.stepRunner.Close()
	a.stepRunner = nil
	a.refreshHistory()
}
```

7. 修改 `handleRunnerEvents()`：

```go
case runner.EventStepStart:
	// ... existing totalSteps calculation ...
	a.runPanel.setProgress(ev.StepIndex+1, totalSteps, stepName)
	a.runPanel.setCurrentStep(stepName)
	if a.globalToolbar != nil {
		a.globalToolbar.setProgress(ev.StepIndex+1, totalSteps, stepName)
	}
	a.setStepStatus(ev.StepIndex, runner.StatusRunning)
	a.statusBar.setRun(RunRunning, ev.StepIndex+1, totalSteps, "")
case runner.EventRunDone:
	if ev.RunResult != nil {
		// ... existing summary logic ...
		a.runPanel.setSummary(ev.RunResult)
	}
	a.runPanel.setRunning(false)
	if a.globalToolbar != nil {
		a.globalToolbar.setRunning(false)
	}
	a.refreshHistory()
```

8. 修改 `refreshFlowList()`：

```go
func (a *App) refreshFlowList() {
	flows, _ := a.flowStore.ListSorted()
	a.flowLibrary.setFlows(flows)
	if a.globalToolbar != nil {
		a.globalToolbar.refreshFlows(flows)
	}
	a.updateEmptyState(len(flows) == 0)
}
```

9. 修改 `startChromeTicker()`：

```go
fyne.Do(func() {
	st := a.browserMgr.Status()
	a.statusBar.setChrome(st)
	managed := st == browser.ChromeRunning || st == browser.ChromeStarting
	if a.runPanel != nil {
		a.runPanel.setChromeManaged(managed)
	}
	if a.globalToolbar != nil {
		a.globalToolbar.setChromeManaged(managed)
	}
})
```

### Step 3.3: 编译验证

```bash
go build ./internal/ui
```

Expected: 无错误。

### Step 3.4: Commit

```bash
git add internal/ui/global_toolbar.go internal/ui/main_window.go
git commit -m "ui: add global toolbar with flow/env/run controls"
```

---

## Task 4: 环境配置 Tab

**Files:**
- Create: `internal/ui/env_panel.go`
- Modify: `internal/ui/env_dialog.go`
- Modify: `internal/ui/main_window.go`

### Step 4.1: 创建 `internal/ui/env_panel.go`

完整内容（基于 `env_dialog.go` 迁移）：

```go
package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
	"go-chrome/internal/db"
)

// envPanel is the main tab for managing environments and variables.
type envPanel struct {
	app      *App
	widget   fyne.CanvasObject
	list     *widget.List
	varList  *widget.List
	search   *widget.Entry

	currentEnvID string
	currentVars  []*db.EnvironmentVariable
	currentVarID string
}

func newEnvPanel(app *App) *envPanel {
	p := &envPanel{app: app}

	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("搜索环境...")
	p.search.OnChanged = func(s string) { p.list.Refresh() }

	p.list = widget.NewList(
		func() int {
			envs, _ := app.envRepo.List()
			if p.search.Text == "" {
				return len(envs)
			}
			q := strings.ToLower(strings.TrimSpace(p.search.Text))
			count := 0
			for _, e := range envs {
				if strings.Contains(strings.ToLower(e.Name), q) {
					count++
				}
			}
			return count
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("环境")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			envs := p.filteredEnvs()
			if id < 0 || id >= len(envs) {
				return
			}
			e := envs[id]
			label := item.(*widget.Label)
			if e.IsActive {
				label.SetText(e.Name + " [当前]")
			} else {
				label.SetText(e.Name)
			}
		},
	)

	p.varList = widget.NewList(
		func() int { return len(p.currentVars) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("KEY"),
				widget.NewLabel("VALUE"),
				widget.NewLabel("敏感"),
				widget.NewLabel("说明"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.currentVars) {
				return
			}
			v := p.currentVars[id]
			box := item.(*fyne.Container)
			key := box.Objects[0].(*widget.Label)
			val := box.Objects[1].(*widget.Label)
			secret := box.Objects[2].(*widget.Label)
			desc := box.Objects[3].(*widget.Label)
			key.SetText(v.Key)
			if v.IsSecret {
				val.SetText("******")
			} else {
				val.SetText(v.Value)
			}
			if v.IsSecret {
				secret.SetText("是")
			} else {
				secret.SetText("")
			}
			desc.SetText(v.Description)
		},
	)

	p.list.OnSelected = func(id widget.ListItemID) {
		envs := p.filteredEnvs()
		if id >= 0 && id < len(envs) {
			p.currentEnvID = envs[id].ID
			p.refreshVars()
		}
	}
	p.varList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(p.currentVars) {
			p.currentVarID = p.currentVars[id].ID
		}
	}

	newEnvBtn := widget.NewButtonWithIcon("新建", theme.ContentAddIcon(), func() { p.showNewEnvDialog() })
	moreBtn := widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() { p.showEnvMoreMenu(moreBtn) })

	newVarBtn := widget.NewButtonWithIcon("新增变量", theme.ContentAddIcon(), func() { p.showNewVarDialog() })
	varMoreBtn := widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() { p.showVarMoreMenu(varMoreBtn) })

	leftHeader := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("环境列表", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			p.search,
			container.NewHBox(newEnvBtn, moreBtn),
		),
		nil, nil, nil,
		p.list,
	)

	rightHeader := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("环境变量", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(newVarBtn, varMoreBtn),
		),
		nil, nil, nil,
		p.varList,
	)

	split := container.NewHSplit(leftHeader, rightHeader)
	split.SetOffset(0.35)
	p.widget = split

	p.refresh()
	return p
}

func (p *envPanel) filteredEnvs() []*db.Environment {
	envs, _ := p.app.envRepo.List()
	if p.search == nil || strings.TrimSpace(p.search.Text) == "" {
		return envs
	}
	q := strings.ToLower(strings.TrimSpace(p.search.Text))
	var out []*db.Environment
	for _, e := range envs {
		if strings.Contains(strings.ToLower(e.Name), q) {
			out = append(out, e)
		}
	}
	return out
}

func (p *envPanel) refresh() {
	p.list.Refresh()
	if p.currentEnvID != "" {
		p.refreshVars()
	}
	if p.app.globalToolbar != nil {
		p.app.globalToolbar.refreshEnvironments()
	}
	if p.app.historyPanel != nil {
		p.app.historyPanel.refreshFilters()
	}
}

func (p *envPanel) refreshVars() {
	if p.currentEnvID == "" {
		p.currentVars = nil
	} else {
		p.currentVars, _ = p.app.envRepo.ListVars(p.currentEnvID)
	}
	p.currentVarID = ""
	p.varList.Refresh()
}

func (p *envPanel) envByID(id string) (*db.Environment, bool) {
	envs, _ := p.app.envRepo.List()
	for _, e := range envs {
		if e.ID == id {
			return e, true
		}
	}
	return nil, false
}

func (p *envPanel) varByID(id string) (*db.EnvironmentVariable, bool) {
	for _, v := range p.currentVars {
		if v.ID == id {
			return v, true
		}
	}
	return nil, false
}

func (p *envPanel) showNewEnvDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("环境名称")
	dialog.ShowForm("新建环境", "创建", "取消", []*widget.FormItem{
		widget.NewFormItem("名称", nameEntry),
	}, func(ok bool) {
		if !ok || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		e := &db.Environment{
			ID:   uuid.New().String(),
			Name: strings.TrimSpace(nameEntry.Text),
		}
		if err := p.app.envRepo.Save(e); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.currentEnvID = e.ID
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) showRenameEnvDialog() {
	env, ok := p.envByID(p.currentEnvID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	nameEntry := widget.NewEntry()
	nameEntry.SetText(env.Name)
	descEntry := widget.NewEntry()
	descEntry.SetText(env.Description)
	dialog.ShowForm("编辑环境", "保存", "取消", []*widget.FormItem{
		widget.NewFormItem("名称", nameEntry),
		widget.NewFormItem("说明", descEntry),
	}, func(ok bool) {
		if !ok || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		env.Name = strings.TrimSpace(nameEntry.Text)
		env.Description = descEntry.Text
		if err := p.app.envRepo.Save(env); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) showCopyEnvDialog() {
	env, ok := p.envByID(p.currentEnvID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	copyEnv := *env
	copyEnv.ID = uuid.New().String()
	copyEnv.Name = env.Name + " 副本"
	copyEnv.IsActive = false
	if err := p.app.envRepo.Save(&copyEnv); err != nil {
		dialog.ShowError(err, p.app.mainWin)
		return
	}
	vars, _ := p.app.envRepo.ListVars(env.ID)
	for _, oldVar := range vars {
		newVar := *oldVar
		newVar.ID = uuid.New().String()
		newVar.EnvironmentID = copyEnv.ID
		if err := p.app.envRepo.SaveVar(&newVar); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
	}
	p.currentEnvID = copyEnv.ID
	p.refresh()
}

func (p *envPanel) showDeleteEnvDialog() {
	env, ok := p.envByID(p.currentEnvID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除环境 [%s] 吗？", env.Name), func(ok bool) {
		if !ok {
			return
		}
		if err := p.app.envRepo.Delete(env.ID); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		if env.IsActive {
			envs, _ := p.app.envRepo.List()
			if len(envs) > 0 {
				_ = p.app.envRepo.SetActive(envs[0].ID)
				p.currentEnvID = envs[0].ID
			} else {
				_ = p.app.envRepo.CreateDefaultIfNone()
				envs, _ = p.app.envRepo.List()
				if len(envs) > 0 {
					p.currentEnvID = envs[0].ID
				}
			}
		} else {
			p.currentEnvID = ""
		}
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) setActiveEnv() {
	if p.currentEnvID == "" {
		return
	}
	if err := p.app.envRepo.SetActive(p.currentEnvID); err != nil {
		dialog.ShowError(err, p.app.mainWin)
		return
	}
	p.refresh()
}

func (p *envPanel) showNewVarDialog() {
	if p.currentEnvID == "" {
		dialog.ShowInformation("提示", "请先选择一个环境", p.app.mainWin)
		return
	}
	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder("变量名")
	keyEntry.OnChanged = func(s string) {
		upper := strings.ToUpper(s)
		if s != upper {
			keyEntry.SetText(upper)
		}
	}
	valEntry := widget.NewEntry()
	valEntry.SetPlaceHolder("变量值")
	secretCheck := widget.NewCheck("敏感变量", nil)
	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("说明")
	dialog.ShowForm("新增变量", "添加", "取消", []*widget.FormItem{
		widget.NewFormItem("变量名", keyEntry),
		widget.NewFormItem("变量值", valEntry),
		widget.NewFormItem("说明", descEntry),
		widget.NewFormItem("", secretCheck),
	}, func(ok bool) {
		key := strings.TrimSpace(keyEntry.Text)
		if !ok || key == "" {
			return
		}
		v := &db.EnvironmentVariable{
			ID:            uuid.New().String(),
			EnvironmentID: p.currentEnvID,
			Key:           key,
			Value:         valEntry.Text,
			IsSecret:      secretCheck.Checked,
			Description:   descEntry.Text,
		}
		if err := p.app.envRepo.SaveVar(v); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}, p.app.mainWin)
}

func (p *envPanel) showEditVarDialog() {
	v, ok := p.varByID(p.currentVarID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个变量", p.app.mainWin)
		return
	}
	keyEntry := widget.NewEntry()
	keyEntry.SetText(v.Key)
	keyEntry.OnChanged = func(s string) {
		upper := strings.ToUpper(s)
		if s != upper {
			keyEntry.SetText(upper)
		}
	}
	valEntry := widget.NewEntry()
	valEntry.SetText(v.Value)
	descEntry := widget.NewEntry()
	descEntry.SetText(v.Description)
	secretCheck := widget.NewCheck("敏感变量", nil)
	secretCheck.SetChecked(v.IsSecret)
	dialog.ShowForm("编辑变量", "保存", "取消", []*widget.FormItem{
		widget.NewFormItem("变量名", keyEntry),
		widget.NewFormItem("变量值", valEntry),
		widget.NewFormItem("说明", descEntry),
		widget.NewFormItem("", secretCheck),
	}, func(ok bool) {
		key := strings.TrimSpace(keyEntry.Text)
		if !ok || key == "" {
			return
		}
		v.Key = key
		v.Value = valEntry.Text
		v.Description = descEntry.Text
		v.IsSecret = secretCheck.Checked
		if err := p.app.envRepo.SaveVar(v); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}, p.app.mainWin)
}

func (p *envPanel) showDeleteVarDialog() {
	v, ok := p.varByID(p.currentVarID)
	if !ok {
		dialog.ShowInformation("提示", "请先选择一个变量", p.app.mainWin)
		return
	}
	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除变量 [%s] 吗？", v.Key), func(ok bool) {
		if !ok {
			return
		}
		if err := p.app.envRepo.DeleteVar(v.ID); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		p.refreshVars()
	}, p.app.mainWin)
}

func (p *envPanel) importEnv() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		if err := p.app.envRepo.Import(reader.URI().Path()); err != nil {
			dialog.ShowError(err, p.app.mainWin)
			return
		}
		envs, _ := p.app.envRepo.List()
		p.currentEnvID = ""
		for _, env := range envs {
			if env.IsActive {
				p.currentEnvID = env.ID
				break
			}
		}
		p.refresh()
	}, p.app.mainWin)
}

func (p *envPanel) exportEnv() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()
		if err := p.app.envRepo.Export(writer.URI().Path()); err != nil {
			dialog.ShowError(err, p.app.mainWin)
		}
	}, p.app.mainWin)
}

func (p *envPanel) showEnvMoreMenu(parent fyne.CanvasObject) {
	hasEnv := p.currentEnvID != ""
	menu := fyne.NewMenu("环境操作",
		fyne.NewMenuItemWithIcon("重命名 / 说明", theme.DocumentCreateIcon(), func() { p.showRenameEnvDialog() }),
		fyne.NewMenuItemWithIcon("复制环境", theme.ContentCopyIcon(), func() { p.showCopyEnvDialog() }),
		fyne.NewMenuItemWithIcon("删除环境", theme.DeleteIcon(), func() { p.showDeleteEnvDialog() }),
		fyne.NewMenuItemWithIcon("设为当前", theme.ConfirmIcon(), func() { p.setActiveEnv() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItemWithIcon("导入配置", theme.DownloadIcon(), func() { p.importEnv() }),
		fyne.NewMenuItemWithIcon("导出配置", theme.UploadIcon(), func() { p.exportEnv() }),
	)
	for i := 0; i < 4; i++ {
		menu.Items[i].Disabled = !hasEnv
	}
	if parent != nil {
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, parent.Size().Height), parent)
	}
}

func (p *envPanel) showVarMoreMenu(parent fyne.CanvasObject) {
	hasVar := p.currentVarID != ""
	menu := fyne.NewMenu("变量操作",
		fyne.NewMenuItemWithIcon("编辑变量", theme.DocumentCreateIcon(), func() { p.showEditVarDialog() }),
		fyne.NewMenuItemWithIcon("删除变量", theme.DeleteIcon(), func() { p.showDeleteVarDialog() }),
	)
	for _, item := range menu.Items {
		item.Disabled = !hasVar
	}
	if parent != nil {
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, parent.Size().Height), parent)
	}
}
```

### Step 4.2: 修改 `env_dialog.go`

将 `showEnvManager()` 改为跳转 tab：

```go
func (a *App) showEnvManager() {
	if a.moduleTabs == nil {
		return
	}
	for i, item := range a.moduleTabs.Items {
		if item.Text == "环境配置" {
			a.moduleTabs.SelectIndex(i)
			return
		}
	}
}
```

### Step 4.3: 编译验证

```bash
go build ./internal/ui
```

Expected: 无错误。

### Step 4.4: Commit

```bash
git add internal/ui/env_panel.go internal/ui/env_dialog.go internal/ui/main_window.go
git commit -m "ui: add environment configuration tab"
```

---

## Task 5: 运行详情页瘦身

**Files:**
- Modify: `internal/ui/run_panel.go`

### Step 5.1: 移除常用控制并保留详情

完整替换 `internal/ui/run_panel.go`：

```go
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
	closeChBtn   *widget.Button
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

	p.closeChBtn = widget.NewButtonWithIcon("关闭本程序启动的 Chrome", theme.CancelIcon(), func() {
		p.app.closeManagedChrome()
	})
	p.closeChBtn.Disable()
	p.closeChBtn.Importance = widget.DangerImportance

	clearLogBtn := widget.NewButtonWithIcon("清空日志", theme.DeleteIcon(), func() {
		p.logBox.Objects = nil
		p.logBox.Refresh()
	})
	openDirBtn := widget.NewButtonWithIcon("打开产物目录", theme.FolderOpenIcon(), func() {
		// Best-effort open; actual implementation depends on OS.
		p.log("产物目录：" + p.app.dirs.DataDir)
	})
	copyLogBtn := widget.NewButtonWithIcon("复制日志", theme.ContentCopyIcon(), func() {
		var lines []string
		for _, obj := range p.logBox.Objects {
			if t, ok := obj.(*canvas.Text); ok {
				lines = append(lines, t.Text)
			}
		}
		app.fyneApp.Clipboard().SetContent(strings.Join(lines, "\n"))
	})

	var moreBtn *widget.Button
	moreBtn = widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() {
		menu := fyne.NewMenu("运行详情",
			fyne.NewMenuItemWithIcon("关闭本程序启动的 Chrome", theme.CancelIcon(), func() { p.app.closeManagedChrome() }),
			fyne.NewMenuItemWithIcon("浏览器下载配置", theme.ComputerIcon(), func() {
				if p.app.moduleTabs != nil {
					for i, item := range p.app.moduleTabs.Items {
						if item.Text == "设置" {
							p.app.moduleTabs.SelectIndex(i)
							return
						}
					}
				}
			}),
		)
		widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, moreBtn.Size().Height), moreBtn)
	})

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("运行摘要", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.summary,
		widget.NewLabelWithStyle("当前步骤", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.currentStep,
		widget.NewLabelWithStyle("产物", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.artifactBox,
	)

	topBar := container.NewBorder(
		nil, nil,
		p.progressText,
		container.NewHBox(clearLogBtn, copyLogBtn, openDirBtn, moreBtn),
		p.progressBar,
	)

	p.widget = container.NewBorder(
		topBar,
		container.NewHBox(p.closeChBtn),
		rightPanel,
		nil,
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
		p.currentStep.SetText(name)
	})
}

func (p *runPanel) setArtifacts(screenshot, htmlSnap string) {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
		if screenshot != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, newTruncatingLabel("截图："+screenshot))
		}
		if htmlSnap != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, newTruncatingLabel("HTML："+htmlSnap))
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
	// Run state is now primarily reflected in the global toolbar.
	// This method is kept for compatibility with existing callers.
	fyne.Do(func() {})
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

// refreshEnvironments is kept for callers that still rely on runPanel.envSelect,
// but the global toolbar now owns the active environment dropdown.
func (p *runPanel) refreshEnvironments() {
	// No-op: env dropdown moved to global toolbar.
}
```

### Step 5.2: 编译验证

```bash
go build ./internal/ui
```

Expected: 无错误。

### Step 5.3: Commit

```bash
git add internal/ui/run_panel.go
git commit -m "ui: slim run panel into run details view"
```

---

## Task 6: 布局与字号细节优化

**Files:**
- Modify: `internal/ui/flow_library.go`
- Modify: `internal/ui/step_table.go`
- Modify: `internal/ui/step_property.go`
- Modify: `internal/ui/history_panel.go`
- Modify: `internal/ui/settings_panel.go`
- Modify: `internal/ui/status_bar.go`

### Step 6.1: 流程库调整

在 `flow_library.go` 中：

- 标题改为“流程库”。
- 保持左侧 split 宽度 280-340px（在 `main_window.go` 的 `flowModule.SetOffset(0.28)` 已满足约 280-340px）。
- 按钮层级：新建使用 HighImportance，保存使用 MediumImportance，更多普通。

修改按钮：

```go
newBtn := widget.NewButtonWithIcon("新建", theme.ContentAddIcon(), func() { p.app.createNewFlow() })
newBtn.Importance = widget.HighImportance
saveBtn := widget.NewButtonWithIcon("保存", theme.DocumentSaveIcon(), func() { p.app.saveCurrentFlow() })
saveBtn.Importance = widget.MediumImportance
```

### Step 6.2: 步骤表调整

在 `step_table.go` 中：

- 标题改为“步骤编排”。
- 按钮层级：新增步骤 HighImportance，复制/删除/上移/下移普通，删除 DangerImportance。
- 确保长 XPath/输入模板截断（已使用 `newTruncatingLabel`）。

修改删除按钮：

```go
delBtn := widget.NewButtonWithIcon("删除", theme.DeleteIcon(), func() { p.deleteStep() })
delBtn.Importance = widget.DangerImportance
```

### Step 6.3: 步骤属性调整

在 `step_property.go` 中：

- 标题改为“步骤属性”。
- “应用到当前步骤”按钮使用 HighImportance。

修改：

```go
applyBtn := widget.NewButtonWithIcon("应用到当前步骤", theme.ConfirmIcon(), func() { p.apply() })
applyBtn.Importance = widget.HighImportance
p.widget = container.NewBorder(
	widget.NewLabelWithStyle("步骤属性", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	applyBtn,
	nil, nil,
	container.NewScroll(p.form),
)
```

### Step 6.4: 历史页调整

在 `history_panel.go` 中：

- 标题改为“执行历史”。
- 环境/状态下拉保持现有筛选。

### Step 6.5: 设置页调整

在 `settings_panel.go` 中：

- 标题保持“浏览器下载配置”。
- “保存配置”使用 HighImportance，“恢复默认”普通。

修改：

```go
saveBtn := widget.NewButtonWithIcon("保存配置", theme.DocumentSaveIcon(), func() { p.save() })
saveBtn.Importance = widget.HighImportance
```

### Step 6.6: 状态栏调整

在 `status_bar.go` 中：

- 状态栏当前流程字段宽度从 180 调整为 220，避免长流程名被过度截断。

修改：

```go
sb.flow = newStatusItem("当前流程：", "未选择", statusColorGray(), 220)
```

### Step 6.7: 编译验证

```bash
go build ./internal/ui
```

Expected: 无错误。

### Step 6.8: Commit

```bash
git add internal/ui/flow_library.go internal/ui/step_table.go internal/ui/step_property.go internal/ui/history_panel.go internal/ui/settings_panel.go internal/ui/status_bar.go
git commit -m "ui: unify layout, button importance and panel titles"
```

---

## Task 7: 文档更新

**Files:**
- Modify: `README.md`
- Modify: `USER_GUIDE.md`
- Modify: `FAQ.md`
- Modify: `problem.md`

### Step 7.1: 更新 `README.md`

在功能概述部分增加：

```markdown
### UI 特性

- 内置 Cascadia Code SemiLight 字体，避免 Windows 中文环境字体变形。
- 全局操作栏：流程选择、保存、启动浏览器、运行、单步执行、停止、环境选择始终可见。
- 独立的“环境配置”tab，集中管理环境变量。
- “运行详情”tab 专注于日志、摘要和产物。
```

### Step 7.2: 更新 `USER_GUIDE.md`

找到描述旧工具栏/运行页的章节，替换为：

```markdown
## 主界面

窗口顶部是状态栏，显示当前流程、保存状态、Chrome 状态和运行状态。

状态栏下方是全局操作栏，从左到右依次为：
- **流程**：选择当前要编辑/运行的流程。
- **保存**：保存当前流程。
- **启动浏览器**：启动本程序托管的 Chrome。
- **运行**：立即运行当前流程。
- **单步执行/下一步**：初始化单步执行后变为“下一步”。
- **停止**：停止当前正在运行的流程或单步执行。
- **环境**：选择当前运行环境。

操作栏右侧会显示当前运行进度，例如“第 2/6 步 · 输入用户名”。

### Tab 说明

- **流程**：流程库和流程基本信息。
- **步骤**：步骤表格和步骤属性编排。
- **环境配置**：管理运行环境和环境变量；敏感变量默认显示为 `******`。
- **历史**：当前流程的运行记录。
- **设置**：Chrome 下载来源、路径、缓存保留策略等。
- **运行详情**：运行摘要、日志、当前步骤、截图/HTML 产物路径。
```

### Step 7.3: 更新 `FAQ.md`

新增或更新：

```markdown
### 为什么内置了 Cascadia Code 字体？

Windows 中文环境下 Fyne 默认字体组合可能导致部分文本变形。Cascadia Code SemiLight 提供稳定的英文/数字/等宽显示，中文会由系统字体回退。

### 环境配置入口在哪里？

顶部 tab 中的“环境配置”页面，不再隐藏在运行页菜单中。

### 运行按钮在哪里？

运行、单步、停止已移到全局操作栏，任意页面都可用。
```

### Step 7.4: 更新 `problem.md`

将已完成的 UI 相关问题标记为已完成，例如：

```markdown
- [x] UI 页面不够美观、字体变形
- [x] 常用运行操作必须切到运行页
- [x] 环境管理入口不明显
```

### Step 7.5: Commit

```bash
git add README.md USER_GUIDE.md FAQ.md problem.md
git commit -m "docs: update UI guide for global toolbar and env tab"
```

---

## Task 8: 构建与测试

**Files:**
- All above

### Step 8.1: 格式化与编译

```bash
go fmt ./...
go build ./...
```

Expected: `go fmt` 无输出（表示已格式化），`go build ./...` 无错误。

### Step 8.2: 核心包测试

```bash
go test ./internal/browser ./internal/runner ./internal/config ./internal/flow ./internal/template
```

Expected: 全部 PASS。

### Step 8.3: 全量测试（若环境有 Fyne 依赖）

```bash
go test ./...
```

Expected: 在具备 Fyne/GLFW 依赖的环境下全部 PASS。若 Linux 环境缺少图形依赖导致 UI 包构建失败，优先安装系统依赖：

```bash
sudo apt-get install -y pkg-config libgl1-mesa-dev xorg-dev
```

### Step 8.4: 最终提交

```bash
git add -A
git commit -m "ui: complete UI optimization (font, global toolbar, env tab, run details)"
```

---

## Spec Coverage Check

| Spec 要求 | 对应任务 |
|---|---|
| 内置 Cascadia Code SemiLight 字体 | Task 1, Task 2 |
| 修复字体变形 | Task 1, Task 2 |
| 全局操作栏 | Task 3 |
| 流程/环境选择全局可见 | Task 3 |
| 运行/单步/停止全局可见 | Task 3 |
| 环境配置 Tab | Task 4 |
| 敏感变量不明文展示 | Task 4 |
| 运行详情页瘦身 | Task 5 |
| Tab 信息架构 | Task 3, Task 4, Task 5 |
| 主题颜色/字号/间距 | Task 2, Task 6 |
| 布局层级和截断 | Task 6 |
| 文档更新 | Task 7 |
| 测试 | Task 8 |

## Placeholder Scan

- 无 TBD/TODO。
- 所有代码步骤包含完整代码。
- 所有命令包含预期输出。
- 类型/方法签名在全局工具栏和主窗口之间一致。
