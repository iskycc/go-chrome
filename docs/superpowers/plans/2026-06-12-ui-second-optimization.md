# UI 二次优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 根据 `Todo2.md` 完成 `go-chrome` 的 UI 二次优化，重点是统一中英文字体、强化全局工具栏、修复流程下拉同名问题、优化环境配置页。

**Architecture:** 用微软 Cascadia Next SC 替换不含中文的 Cascadia Code SemiLight 作为全局 UI 字体；重组全局工具栏为流程/浏览器/执行/环境分组；修复流程下拉同名覆盖；环境配置变量列表改为表格并强化导入导出。

**Tech Stack:** Go, Fyne v2, embed

---

## File Structure

### 新建/替换字体资源

- `assets/fonts/CascadiaNextSC.wght.ttf`：已下载的官方 CJK 可变字体。
- `assets/fonts/LICENSE-CascadiaNext.txt`：Cascadia Next 的 OFL 1.1 许可。

### 修改文件

- `assets/embed.go`：暴露 `AppUIFont()` 和 `CodeFont()`。
- `internal/ui/theme.go`：`Font()` 返回 `AppUIFont()`。
- `internal/ui/global_toolbar.go`：新增关闭托管 Chrome 按钮、停止当前流程常驻禁用、流程下拉同名处理、分组布局。
- `internal/ui/run_panel.go`：移除关闭 Chrome 入口。
- `internal/ui/env_panel.go`：变量列表改为表格，强化导入导出。
- `README.md` / `USER_GUIDE.md` / `FAQ.md` / `problem.md`：文档同步。

---

## Task 1: 字体资源与入口改造

**Files:**
- Create: `assets/fonts/LICENSE-CascadiaNext.txt`
- Modify: `assets/embed.go`
- Modify: `internal/ui/theme.go`

### Step 1.1: 创建 Cascadia Next 许可证

`assets/fonts/LICENSE-CascadiaNext.txt`：

```text
Cascadia Next Font
Copyright (c) Microsoft Corporation.

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is available with a FAQ at: https://scripts.sil.org/OFL
```

### Step 1.2: 替换字体嵌入入口

完整替换 `assets/embed.go`：

```go
package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png fonts/CascadiaNextSC.wght.ttf fonts/CascadiaCode-SemiLight.ttf
var assetFS embed.FS

// Icon returns the application icon resource.
func Icon() fyne.Resource {
	data, err := assetFS.ReadFile("icon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", data)
}

// AppUIFont returns the global UI font that covers CJK characters.
// Currently Cascadia Next SC.
func AppUIFont() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/CascadiaNextSC.wght.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaNextSC.wght.ttf", data)
}

// CodeFont returns the monospace font for logs/code inputs.
// Kept as original Cascadia Code SemiLight.
func CodeFont() fyne.Resource {
	data, err := assetFS.ReadFile("fonts/CascadiaCode-SemiLight.ttf")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("CascadiaCode-SemiLight.ttf", data)
}
```

### Step 1.3: 更新主题字体

修改 `internal/ui/theme.go` 的 `Font()`：

```go
func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	if res := assets.AppUIFont(); res != nil {
		return res
	}
	return theme.DefaultTheme().Font(style)
}
```

### Step 1.4: 验证字体覆盖 CJK

```bash
fc-scan --format '%{charset}\n' assets/fonts/CascadiaNextSC.wght.ttf | grep -q '4e00' && echo "CJK OK" || echo "CJK MISSING"
```

Expected: `CJK OK`

### Step 1.5: 构建验证

```bash
go build -mod=readonly ./...
```

Expected: OK

### Step 1.6: Commit

```bash
git add assets/fonts/CascadiaNextSC.wght.ttf assets/fonts/LICENSE-CascadiaNext.txt assets/embed.go internal/ui/theme.go
git commit -m "assets,ui: use Cascadia Next SC as global UI font for CJK coverage"
```

---

## Task 2: 全局工具栏重组

**Files:**
- Modify: `internal/ui/global_toolbar.go`
- Modify: `internal/ui/main_window.go`（若需调整 App 方法）

### Step 2.1: 流程下拉同名处理

在 `internal/ui/global_toolbar.go` 中：

```go
type flowSelectOption struct {
	Label string
	ID    string
}
```

替换 `flowByName map[string]*flow.Flow` 为：

```go
flowOptions []flowSelectOption
flowByID    map[string]*flow.Flow
```

`refreshFlows` 生成 Label 时处理同名：

```go
func (t *globalToolbar) refreshFlows(flows []*flow.Flow) {
	t.flowByID = make(map[string]*flow.Flow, len(flows))
	nameCount := map[string]int{}
	for _, f := range flows {
		nameCount[f.Name]++
	}
	seen := map[string]int{}
	var options []flowSelectOption
	var names []string
	var selected string
	for _, f := range flows {
		label := f.Name
		if nameCount[f.Name] > 1 {
			seen[f.Name]++
			label = fmt.Sprintf("%s · %s", f.Name, f.ID[:6])
		}
		options = append(options, flowSelectOption{Label: label, ID: f.ID})
		names = append(names, label)
		t.flowByID[f.ID] = f
		if t.app.currentFlow != nil && f.ID == t.app.currentFlow.ID {
			selected = label
		}
	}
	fyne.Do(func() {
		t.flowOptions = options
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
```

选择回调按 ID 加载：

```go
t.flowSelect = widget.NewSelect([]string{}, func(name string) {
	for _, opt := range t.flowOptions {
		if opt.Label == name {
			if f := t.flowByID[opt.ID]; f != nil {
				app.onFlowSelected(f)
			}
			break
		}
	}
})
```

### Step 2.2: 新增关闭托管 Chrome 按钮和停止当前流程常驻

在 `globalToolbar` 结构体中新增：

```go
stopChromeBtn *widget.Button
```

初始化：

```go
t.stopChromeBtn = widget.NewButtonWithIcon("关闭托管", theme.CancelIcon(), func() {
	app.closeManagedChrome()
})
t.stopChromeBtn.Importance = widget.DangerImportance
t.stopChromeBtn.Disable()
```

修改 `setRunning`：

```go
func (t *globalToolbar) setRunning(running bool) {
	fyne.Do(func() {
		if running {
			t.runBtn.Disable()
			t.stepBtn.Disable()
			t.stopBtn.Enable()
		} else {
			t.runBtn.Enable()
			t.stepBtn.Enable()
			t.stopBtn.Disable()
			t.stepBtn.SetText("单步执行")
		}
	})
}
```

初始状态调用 `t.stopBtn.Disable()` 而不是 `t.stopBtn.Hide()`。

### Step 2.3: 分组布局

重组 `t.widget`：

```go
flowBox := container.NewHBox(
	widget.NewLabelWithStyle("流程", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	t.flowSelect,
	t.saveBtn,
)
browserBox := container.NewHBox(
	widget.NewLabelWithStyle("浏览器", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	t.startChromeBtn,
	t.stopChromeBtn,
)
execBox := container.NewHBox(
	widget.NewLabelWithStyle("执行", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	t.runBtn,
	t.stepBtn,
	t.stopBtn,
)
envBox := container.NewHBox(
	widget.NewLabelWithStyle("环境", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	t.envSelect,
)

progressBox := container.NewBorder(nil, nil, t.progressText, nil, t.progress)

row1 := container.NewHBox(flowBox, widget.NewSeparator(), browserBox, widget.NewSeparator(), envBox)
row2 := container.NewHBox(execBox, widget.NewSeparator(), progressBox)

t.widget = container.NewVBox(row1, row2)
```

注意：Fyne 的 `widget.NewSeparator()` 是垂直分隔线；如果希望水平分组，用 `widget.NewSeparator()` 默认是垂直的。若 Fyne 版本不同，可用 `canvas.NewLine` 或 spacer。

### Step 2.4: 同步 Chrome 状态

在 `setChromeManaged` 中同时控制启动/关闭按钮：

```go
func (t *globalToolbar) setChromeManaged(managed bool) {
	fyne.Do(func() {
		if managed {
			t.startChromeBtn.Disable()
			t.stopChromeBtn.Enable()
		} else {
			t.startChromeBtn.Enable()
			t.stopChromeBtn.Disable()
		}
	})
}
```

### Step 2.5: 构建验证

```bash
go build -mod=readonly ./...
```

Expected: OK

### Step 2.6: Commit

```bash
git add internal/ui/global_toolbar.go
git commit -m "ui: reorganize global toolbar with browser/exec groups and unique flow labels"
```

---

## Task 3: 运行详情页移除关闭 Chrome 入口

**Files:**
- Modify: `internal/ui/run_panel.go`

### Step 3.1: 移除底部关闭 Chrome 按钮

在 `runPanel` 结构体中移除 `closeChBtn` 字段（或保留但不在主界面显示）。

为简化，改为保留字段但不在 widget 中放置，或者直接从结构体和构造中移除。

推荐移除：

```go
// 删除 closeChBtn 字段
```

### Step 3.2: 更新"更多"菜单

移除菜单中的关闭 Chrome 项，只保留浏览器下载配置跳转：

```go
moreBtn = widget.NewButtonWithIcon("更多", theme.MoreHorizontalIcon(), func() {
	menu := fyne.NewMenu("运行详情",
		fyne.NewMenuItemWithIcon("浏览器下载配置", theme.ComputerIcon(), func() {
			// 跳转设置 tab
		}),
	)
	widget.ShowPopUpMenuAtRelativePosition(menu, p.app.mainWin.Canvas(), fyne.NewPos(0, moreBtn.Size().Height), moreBtn)
})
```

### Step 3.3: 构建验证

```bash
go build -mod=readonly ./...
```

Expected: OK

### Step 3.4: Commit

```bash
git add internal/ui/run_panel.go
git commit -m "ui: remove close-chrome entry from run details panel"
```

---

## Task 4: 环境配置页变量表格与导入导出

**Files:**
- Modify: `internal/ui/env_panel.go`

### Step 4.1: 变量列表改为表格

将 `varList *widget.List` 改为 `varTable *widget.Table`，列：KEY、VALUE、敏感、说明、操作。

表格创建示例：

```go
p.varTable = widget.NewTable(
	func() (int, int) { return len(p.currentVars), 5 },
	func() fyne.CanvasObject {
		return newTruncatingLabel("cell")
	},
	func(id widget.TableCellID, cell fyne.CanvasObject) {
		if id.Row < 0 || id.Row >= len(p.currentVars) {
			return
		}
		v := p.currentVars[id.Row]
		label := cell.(*widget.Label)
		switch id.Col {
		case 0:
			label.SetText(v.Key)
		case 1:
			if v.IsSecret {
				label.SetText("******")
			} else {
				label.SetText(v.Value)
			}
		case 2:
			if v.IsSecret {
				label.SetText("是")
			} else {
				label.SetText("")
			}
		case 3:
			label.SetText(v.Description)
		case 4:
			label.SetText("编辑")
		}
	},
)
p.varTable.SetColumnWidth(0, 120)
p.varTable.SetColumnWidth(1, 160)
p.varTable.SetColumnWidth(2, 50)
p.varTable.SetColumnWidth(3, 160)
p.varTable.SetColumnWidth(4, 50)
p.varTable.OnSelected = func(id widget.TableCellID) {
	if id.Row >= 0 && id.Row < len(p.currentVars) {
		p.currentVarID = p.currentVars[id.Row].ID
		p.showEditVarDialog()
	}
}
```

### Step 4.2: 顶部增加导入/导出按钮

在环境配置页顶部右侧添加导入/导出按钮：

```go
importBtn := widget.NewButtonWithIcon("导入配置", theme.DownloadIcon(), func() { p.importEnv() })
exportBtn := widget.NewButtonWithIcon("导出配置", theme.UploadIcon(), func() { p.exportEnv() })

rightHeader := container.NewBorder(
	container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("环境变量", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			importBtn,
			exportBtn,
		),
		container.NewHBox(newVarBtn, varMoreBtn),
	),
	nil, nil, nil,
	p.varTable,
)
```

需要导入 `fyne.io/fyne/v2/layout`。

### Step 4.3: 导出敏感变量提示

修改 `exportEnv()`：

```go
func (p *envPanel) exportEnv() {
	envs, _ := p.app.envRepo.List()
	hasSecret := false
	for _, env := range envs {
		vars, _ := p.app.envRepo.ListVars(env.ID)
		for _, v := range vars {
			if v.IsSecret {
				hasSecret = true
				break
			}
		}
		if hasSecret {
			break
		}
	}
	doExport := func() {
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
	if hasSecret {
		dialog.ShowConfirm("导出敏感变量",
			"当前环境包含敏感变量，导出文件将包含这些变量的明文值。是否继续？",
			func(ok bool) {
				if ok {
					doExport()
				}
			}, p.app.mainWin)
	} else {
		doExport()
	}
}
```

### Step 4.4: 导入反馈

修改 `importEnv()` 导入成功后明确反馈并刷新：

```go
func (p *envPanel) importEnv() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		if err := p.app.envRepo.Import(reader.URI().Path()); err != nil {
			dialog.ShowError(fmt.Errorf("导入失败: %w", err), p.app.mainWin)
			return
		}
		p.refresh()
		dialog.ShowInformation("导入成功", "环境配置已导入并刷新", p.app.mainWin)
	}, p.app.mainWin)
}
```

### Step 4.5: 构建验证

```bash
go build -mod=readonly ./...
```

Expected: OK

### Step 4.6: Commit

```bash
git add internal/ui/env_panel.go
git commit -m "ui: turn env variable list into table and elevate import/export"
```

---

## Task 5: 文档更新

**Files:**
- Modify: `README.md`
- Modify: `USER_GUIDE.md`
- Modify: `FAQ.md`
- Modify: `problem.md`

### Step 5.1: 更新 README.md

在 UI 特性部分更新字体说明：

```markdown
- 内置 Cascadia Next SC 字体，中英文统一渲染，避免 fallback 割裂。
- 全局操作栏固定流程、浏览器、执行、环境分组，常用操作任意页面可见。
```

### Step 5.2: 更新 USER_GUIDE.md

更新全局工具栏说明，新增关闭托管 Chrome 和停止当前流程位置描述。

### Step 5.3: 更新 FAQ.md

新增/更新 Q&A：
- 为什么中文和英文使用同一个字体？
- 关闭托管 Chrome 会关闭系统 Chrome 吗？
- 同名流程如何区分？

### Step 5.4: 更新 problem.md

标记 Todo2.md 中已完成问题。

### Step 5.5: Commit

```bash
git add README.md USER_GUIDE.md FAQ.md problem.md
git commit -m "docs: update guide for second UI optimization"
```

---

## Task 6: 构建与测试

**Files:** 所有上述

### Step 6.1: 格式化与构建

```bash
go fmt ./...
go build -mod=readonly ./...
```

Expected: build OK

### Step 6.2: 核心测试

```bash
go test ./internal/browser ./internal/runner ./internal/flow ./internal/template
```

Expected: PASS

### Step 6.3: 字体检查

```bash
fc-scan --format '%{charset}\n' assets/fonts/CascadiaNextSC.wght.ttf | grep -q '4e00' && echo "CJK OK"
```

Expected: `CJK OK`

### Step 6.4: 最终提交

```bash
git add -A
git reset -- Todo1.md Todo2.md
git commit -m "ui: complete second UI optimization (CJK font, toolbar, env panel)"
```

---

## Spec Coverage Check

| Spec 要求 | 任务 |
|---|---|
| 中英文字体统一 | Task 1 |
| 全局工具栏关闭托管 Chrome | Task 2 |
| 停止当前流程常驻禁用 | Task 2 |
| 工具栏分组布局 | Task 2 |
| 流程下拉同名修复 | Task 2 |
| 运行详情页移除关闭 Chrome | Task 3 |
| 环境配置变量表格 | Task 4 |
| 导入导出强化 | Task 4 |
| 文档更新 | Task 5 |
| 测试 | Task 6 |

## Placeholder Scan

- 无 TBD/TODO。
- 所有代码步骤包含代码片段。
- 命令包含预期输出。
