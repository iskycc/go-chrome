# 进度条文字显示为 `...` 的 UI 修复任务书

本文档记录当前构建版本中“进度条旁边的文字显示为 `...`”的问题定位、根因分析和修复方案。该问题属于布局约束和文本截断控件使用不当，不是字体文件本身的问题。

## 用户反馈

进度条旁边的字体/文字显示成 `...`，应为 UI 布局问题，需要继续排查和修复。

## 已定位位置

当前至少有两个进度文字位置存在风险。

### 1. 全局工具栏进度文字

文件：

```text
internal/ui/global_toolbar.go
```

相关代码：

```go
t.progressText = newTruncatingLabel("就绪")
progressBox := container.NewBorder(nil, nil, t.progressText, nil, t.progress)
...
t.widget = container.NewBorder(nil, progressBox, nil, nil, top)
```

更新文字：

```go
t.progressText.SetText(fmt.Sprintf("第 %d/%d 步 · %s", current, total, stepName))
```

### 2. 运行详情页进度文字

文件：

```text
internal/ui/run_panel.go
```

相关代码：

```go
p.progressText = newTruncatingLabel("就绪")
progressArea := container.NewVBox(p.progressBar, p.progressText)
topBar := container.NewBorder(nil, nil, progressArea, actionBtns)
```

更新文字：

```go
p.progressText.SetText(fmt.Sprintf("第 %d 步 / 共 %d 步 · %s", current, total, stepName))
```

## 根因分析

### 1. `newTruncatingLabel` 不适合直接放进没有稳定宽度的容器

`newTruncatingLabel()` 当前实现：

```go
func newTruncatingLabel(initial string) *widget.Label {
    l := widget.NewLabel(initial)
    l.Truncation = fyne.TextTruncateEllipsis
    return l
}
```

`TextTruncateEllipsis` 的行为是：当 Label 可用宽度不足时显示省略号。如果父容器没有给它稳定宽度，或者它被放到 `Border` 的 left 区域且被其它控件挤压，就可能只剩极小宽度，最终显示为 `...`。

### 2. 全局工具栏的 `Border(left=t.progressText, center=t.progress)` 容易挤压文字

`container.NewBorder(nil, nil, t.progressText, nil, t.progress)` 中：

- `t.progressText` 位于 left。
- `t.progress` 位于 center。
- 如果整体宽度不足或 center 争夺空间，left 区域的截断 Label 可能拿不到足够宽度。

更关键的是，全局工具栏上方已经有很多按钮组：

```text
流程 / 浏览器 / 执行 / 环境
```

底部进度区又被放入主窗口顶部区域，实际可用宽度在小窗口下不稳定。

### 3. 运行详情页进度区域也没有给文字明确宽度

`progressArea := container.NewVBox(p.progressBar, p.progressText)` 中，`p.progressText` 的宽度取决于父容器布局和右侧按钮区。右侧 `清空日志 / 复制日志 / 打开产物目录 / 更多` 会压缩左侧区域，导致文字被截断成 `...`。

### 4. 这不是单纯字体问题

即使更换 JetBrains Code CJK 字体，如果 Label 可用宽度仍然过小，仍会显示 `...`。字体变更可能让宽度计算更明显，但根因是布局没有给进度文字稳定空间。

## P0-1. 修复全局工具栏进度文字布局

### 目标

全局工具栏进度文字不能再显示为 `...`。在窄窗口下可以合理截断长步骤名，但必须至少显示进度前缀，例如：

```text
第 2/6 步 · 输入...
```

而不是：

```text
...
```

### 推荐方案 A：进度文字和进度条分两行

将全局工具栏底部进度区改为：

```text
第 2/6 步 · 输入用户名
[======================        ]
```

代码方向：

```go
t.progressText = widget.NewLabel("就绪")
t.progressText.Wrapping = fyne.TextWrapOff

progressLabelBox := container.NewGridWrap(
    fyne.NewSize(520, t.progressText.MinSize().Height),
    t.progressText,
)

progressBox := container.NewVBox(
    progressLabelBox,
    t.progress,
)
```

注意：

- 这里可以不用 `TextTruncateEllipsis`，改用手动 `truncate(...)` 控制最大字符数。
- 如果使用 ellipsis，必须外包 `GridWrap` 给稳定宽度。

### 推荐方案 B：使用固定宽度 label + 进度条

如果仍想单行展示：

```text
第 2/6 步 · 输入用户名     [===========      ]
```

则必须给文字固定宽度：

```go
t.progressText = widget.NewLabel("就绪")
t.progressText.Truncation = fyne.TextTruncateEllipsis

progressTextBox := container.NewGridWrap(
    fyne.NewSize(360, t.progressText.MinSize().Height),
    t.progressText,
)

progressBox := container.NewBorder(nil, nil, progressTextBox, nil, t.progress)
```

不要把裸 `newTruncatingLabel` 直接作为 `Border` left。

### 推荐显示策略

不要把完整步骤名无限塞进进度文字。建议在设置文本时先手动截断：

```go
name := truncate(stepName, 40)
t.progressText.SetText(fmt.Sprintf("第 %d/%d 步 · %s", current, total, name))
```

空闲时：

```go
t.progressText.SetText("就绪")
```

### 验收标准

- 1400x900 下完整显示合理长度步骤名。
- 1280x720 下至少显示 `第 N/M 步`，不能只显示 `...`。
- 超长步骤名可截断，但进度数字不能丢。
- 切换运行、停止、重置后不会变成 `...`。

## P0-2. 修复运行详情页进度文字布局

### 目标

运行详情页顶部进度文字不再被右侧按钮挤压成 `...`。

### 推荐方案

当前：

```go
progressArea := container.NewVBox(p.progressBar, p.progressText)
actionBtns := container.NewHBox(clearLogBtn, copyLogBtn, openArtifactBtn, moreBtn)
topBar := container.NewBorder(nil, nil, progressArea, actionBtns)
```

建议改为：

```go
p.progressText = widget.NewLabel("就绪")
p.progressText.Wrapping = fyne.TextWrapOff

progressTextBox := container.NewGridWrap(
    fyne.NewSize(520, p.progressText.MinSize().Height),
    p.progressText,
)

progressArea := container.NewVBox(progressTextBox, p.progressBar)
topBar := container.NewBorder(nil, nil, nil, actionBtns, progressArea)
```

关键点：

- `progressArea` 放在 center。
- 右侧 action buttons 放在 right。
- 进度文字用固定宽度 box 或手动截断。
- 不要让文字 label 作为裸 truncating label 参与不稳定布局。

### 可选优化

运行详情页空间比全局工具栏更大，可以显示更完整的进度：

```text
第 2 步 / 共 6 步 · 输入用户名
```

如果步骤名很长，手动截断：

```go
name := truncate(stepName, 60)
```

### 验收标准

- 运行详情页顶部不再显示 `...`。
- 右侧按钮存在时不会挤压进度文字到不可读。
- 长步骤名截断合理，进度数字保留。

## P0-3. 给进度文字建立专用组件，不要复用 `newTruncatingLabel`

### 问题

`newTruncatingLabel` 适合表格、列表这种由父容器明确限制宽度的单元格，不适合全局进度、当前步骤、状态等需要保证关键前缀可见的文本。

### 建议新增 helper

建议在 `internal/ui/labels.go` 或新文件中增加：

```go
func newProgressLabel(initial string, width float32) (*widget.Label, fyne.CanvasObject) {
    l := widget.NewLabel(initial)
    l.Wrapping = fyne.TextWrapOff
    l.Truncation = fyne.TextTruncateEllipsis
    box := container.NewGridWrap(fyne.NewSize(width, l.MinSize().Height), l)
    return l, box
}
```

或者不用 ellipsis，由 setter 手动截断：

```go
func setProgressText(label *widget.Label, current, total int, stepName string, maxStepName int) {
    if total <= 0 {
        label.SetText("就绪")
        return
    }
    label.SetText(fmt.Sprintf("第 %d/%d 步 · %s", current, total, truncate(stepName, maxStepName)))
}
```

### 更稳妥方案

为了保证 `第 N/M 步` 永远可见，将进度数字和步骤名拆成两个 label：

```text
[第 2/6 步] [输入用户名...]
```

实现：

- `progressPrefixLabel` 固定文本，不截断。
- `progressStepLabel` 可截断。

这样即使宽度不足，用户也能看到当前进度数字。

### 验收标准

- 任何进度文字都不直接使用裸 `newTruncatingLabel`。
- 进度数字永远可见。
- 只有步骤名部分可以被截断。

## P1-1. 检查同类 `newTruncatingLabel` 使用点

以下位置也使用 `newTruncatingLabel`，需要确认是否存在类似 `...` 问题：

```text
internal/ui/run_panel.go:47      currentStep
internal/ui/run_panel.go:152     截图路径
internal/ui/run_panel.go:156     HTML 路径
internal/ui/template_dialog.go   模板列表和预览列表
internal/ui/step_table.go        表格 cell
internal/ui/flow_library.go      流程列表
internal/ui/context_menu.go      右键菜单 label
```

### 判断标准

- 表格 cell / 列表 row：可以继续用 `newTruncatingLabel`，因为父容器宽度明确。
- 顶部状态、进度、当前步骤、产物路径：需要稳定宽度、复制能力或手动截断，不能只依赖裸 truncating label。

### 建议

- `currentStep` 可拆成 `当前步骤：` + 可截断步骤名。
- 产物路径 label 最好固定宽度，并提供复制路径右键菜单。
- 模板列表如果仍有 `...`，也使用 `GridWrap` 或手动截断。

## 推荐实施顺序

1. 修改 `global_toolbar.go`：修复全局进度文字布局。
2. 修改 `run_panel.go`：修复运行详情页进度文字布局。
3. 新增专用 progress label helper。
4. 将进度数字和步骤名拆分，确保数字永远可见。
5. 检查 `currentStep` 和产物路径是否也出现 `...`，必要时一并修复。

## 手工验收清单

- 启动后全局工具栏进度显示 `就绪`，不是 `...`。
- 运行流程时全局工具栏显示 `第 N/M 步`，不是 `...`。
- 长步骤名时，进度数字保留，只有步骤名截断。
- 运行详情页顶部进度文字不显示 `...`。
- 1280x720 下验证进度文字仍可读。
- 1400x900 下验证进度文字显示完整或合理截断。
- 切换 Tab 不影响进度文字显示。
- 停止流程后进度文字回到 `就绪` 或合理完成状态，不变成 `...`。

## 注意事项

- 不要通过移除截断来解决问题，否则长步骤名会撑爆布局。
- 不要让进度条抢占全部宽度，必须给文字稳定宽度。
- 不要将 `newTruncatingLabel` 作为万能 label 使用；它只适合宽度明确的列表/表格单元格。
- 不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制或运行产物。
