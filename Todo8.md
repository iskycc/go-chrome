# UI 优化验收修复任务书

本文档记录当前 UI 优化分支的验收结论和必须继续修复的问题。当前修改方向整体正确，功能也能通过基础编译测试，但还不能认为已经满足要求，因为存在明确的运行时崩溃风险、状态显示遗漏和布局挂载问题。

## 当前验收结论

当前修改只能算 **部分满足要求**。

已通过：

```bash
go test ./...
```

但 `go test` 只证明代码能编译和非 UI 测试通过，不能证明 Fyne 界面在真实交互中没有崩溃、错位或控件挂载问题。后续 Agent 必须先修复本文档中的 P0 问题，再做实际窗口验收。

## 已满足的部分

### 1. 字体方向已基本满足

当前字体资源：

```text
assets/fonts/MapleMono-CN-Regular.ttf
assets/fonts/MapleMono-CN-Medium.ttf
```

嵌入位置：

```text
assets/embed.go
internal/ui/theme.go
```

当前实现已经通过 `AppUIFontRegular()` / `AppUIFontMedium()` 接入 Fyne theme，中文和英文理论上使用同一套 MapleMono-CN 字体，满足“中英文统一字体风格”的方向。

### 2. 全局常用按钮已基本满足

位置：

```text
internal/ui/global_toolbar.go
```

当前全局工具栏已包含：

- 流程选择
- 环境选择
- 启动 Chrome
- 关闭托管
- 运行
- 单步
- 停止流程
- 进度文字
- 进度条

这满足用户提出的“流程切换、启动、停止等常用功能需要在任何页面都能看到”的核心要求。

### 3. 进度条旁边文字显示为 `...` 的问题已基本处理

位置：

```text
internal/ui/labels.go
internal/ui/global_toolbar.go
internal/ui/run_panel.go
```

当前已经使用 `newProgressLabel()` 分离固定宽度前缀和可截断步骤名，不再直接把裸 `newTruncatingLabel()` 放到不稳定布局中。方向正确。

### 4. Todo7 的视觉精修已部分落地

新增或修改内容包括：

- `internal/ui/style.go`
- `internal/ui/theme.go`
- `internal/ui/status_bar.go`
- `internal/ui/global_toolbar.go`
- `internal/ui/step_table.go`
- `internal/ui/flow_library.go`
- `internal/ui/env_panel.go`
- `internal/ui/history_panel.go`
- `internal/ui/run_panel.go`
- `internal/ui/settings_panel.go`
- `internal/ui/step_property.go`

已实现的方向：

- 统一颜色 token。
- 深浅色主题区分。
- 工具栏分组。
- 空状态组件。
- 步骤表表头。
- 流程库两行列表。
- 环境列表两行列表。
- 历史记录两行列表。
- 环境变量表删除空“操作”列。
- 设置页按语义分区。

## P0-1. 修复步骤属性校验崩溃

### 问题

当前 `internal/ui/step_property.go` 中错误提示字段定义为：

```go
nameErr fyne.CanvasObject
targetErr fyne.CanvasObject
inputErr fyne.CanvasObject
expectedErr fyne.CanvasObject
waitBeforeErr fyne.CanvasObject
waitAfterErr fyne.CanvasObject
timeoutErr fyne.CanvasObject
```

初始化时使用：

```go
p.nameErr = newInlineError("")
```

但 `newInlineError()` 当前返回的是 `canvas.Text`：

```go
func newInlineError(message string) fyne.CanvasObject {
    t := canvas.NewText(message, uiColorDanger())
    t.TextSize = theme.CaptionTextSize()
    return t
}
```

校验时却强转成 `*widget.Label`：

```go
p.nameErr.(*widget.Label).SetText("步骤名称不能为空")
```

因此，只要触发表单校验错误，就可能 panic。

### 影响场景

以下交互都可能导致程序崩溃：

- 步骤名称为空时点击“应用到当前步骤”。
- XPath 为空时点击应用。
- URL 不合法时点击应用。
- 等待时间输入非数字时点击应用。
- 模板语法错误时点击应用。

### 修复方案 A：把错误字段改回 `*widget.Label`

推荐做法：

```go
func newInlineErrorLabel(message string) *widget.Label {
    l := widget.NewLabel(message)
    l.TextStyle = fyne.TextStyle{Bold: true}
    return l
}
```

然后字段定义改为：

```go
nameErr *widget.Label
targetErr *widget.Label
inputErr *widget.Label
expectedErr *widget.Label
waitBeforeErr *widget.Label
waitAfterErr *widget.Label
timeoutErr *widget.Label
```

校验时直接：

```go
p.nameErr.SetText("步骤名称不能为空")
p.nameErr.Show()
```

如果需要红色文本，优先用 theme error color 或封装一个自定义 error label，不要再使用错误类型断言。

### 修复方案 B：保留 `canvas.Text`，但提供统一 setter

如果坚持使用 `canvas.Text`，则必须新增 helper：

```go
func setInlineError(obj fyne.CanvasObject, message string) {
    if t, ok := obj.(*canvas.Text); ok {
        t.Text = message
        t.Refresh()
        obj.Show()
    }
}
```

然后替换所有 `.(*widget.Label).SetText(...)`。

不推荐方案 B，因为 `widget.Form` 内部使用 `canvas.Text` 可能在高度计算和主题刷新上不如 `widget.Label` 稳定。

### 验收标准

- 步骤名称为空时点击应用，不崩溃，显示错误信息。
- URL 不合法时点击应用，不崩溃，显示错误信息。
- 等待时间输入 `abc` 时点击应用，不崩溃，显示错误信息。
- 模板表达式错误时点击应用，不崩溃，显示错误信息。
- 修复后重新执行：

```bash
go test ./...
```

## P0-2. 恢复顶部“保存状态”的可见性

### 问题

当前 `internal/ui/status_bar.go` 中创建了保存状态项：

```go
sb.save = newStatusItem("保存", "未修改", statusSuccess, 110)
```

并且 `setSave()` 仍会更新：

```go
sb.save.setValue("有未保存修改")
sb.save.setKind(statusWarning)
```

但实际顶部布局只放了保存按钮：

```go
sb.widget = container.NewHBox(
    title,
    itemSpacer,
    sb.flow.row,
    itemSpacer,
    sb.saveBtn,
    itemSpacer,
    sb.chrome.row,
    itemSpacer,
    sb.run.row,
)
```

`sb.save.row` 没有被加入布局，因此“未修改 / 有未保存修改 / 已保存 / 保存失败”等状态不可见。

### 为什么这是问题

用户之前已经反馈过顶部“未修改 / 已安装 / 已完成”等状态缺少标注。当前版本虽然给状态项做了字段名，但保存状态被保存按钮替代后，用户看不到当前流程是否有未保存修改。

### 修复方案

建议顶部状态栏保留“保存状态”，保存按钮可以继续存在，但不要替代状态。

推荐布局：

```go
sb.widget = container.NewHBox(
    title,
    itemSpacer,
    sb.flow.row,
    itemSpacer,
    sb.save.row,
    sb.saveBtn,
    itemSpacer,
    sb.chrome.row,
    itemSpacer,
    sb.run.row,
)
```

或者将保存按钮移动到全局工具栏 / 流程组中，状态栏只显示状态。

注意：如果保存按钮已经从全局工具栏移除，要确认用户仍能在任何页面保存当前流程。

### 验收标准

- 修改流程后顶部能看到“保存：有未保存修改”或等价状态。
- 保存成功后能短暂看到“已保存”，随后回到“未修改”。
- 保存失败时能看到“保存失败”。
- 保存按钮仍然可见或有等价保存入口。

## P0-3. 修复运行日志空状态未挂载问题

### 问题

当前 `internal/ui/run_panel.go` 创建了空状态：

```go
p.logEmpty = newEmptyState("暂无运行日志", "运行开始后日志会显示在这里", nil)
```

也有显示/隐藏逻辑：

```go
func (p *runPanel) updateLogVisibility() {
    hasLogs := len(p.logBox.Objects) > 0
    if hasLogs {
        p.logEmpty.Hide()
        p.logScroll.Show()
    } else {
        p.logEmpty.Show()
        p.logScroll.Hide()
    }
}
```

但是主布局中只放了：

```go
p.logScroll
```

没有把 `p.logEmpty` 加入任何容器。因此空状态永远不会显示。

### 修复方案

将日志区域改成 Stack：

```go
logArea := container.NewStack(p.logScroll, p.logEmpty)
p.updateLogVisibility()

p.widget = container.NewBorder(
    topBar,
    nil,
    nil,
    rightCard,
    logArea,
)
```

初始化时必须调用一次：

```go
p.updateLogVisibility()
```

否则默认状态可能是日志空白。

### 验收标准

- 刚打开运行详情页时显示“暂无运行日志”。
- 第一次写入日志后空状态隐藏，日志显示。
- 清空日志后空状态重新显示。

## P0-4. 修复设置页重复挂载同一输入控件

### 问题

当前 `internal/ui/settings_panel.go` 中 `customSHA` 和 `customLabel` 被放到了两个不同区域。

第一次在自定义 ZIP 区域：

```go
p.customBox = container.NewVBox(
    widget.NewForm(
        widget.NewFormItem("自定义下载 URL", p.customURL),
        widget.NewFormItem("SHA256", p.customSHA),
        widget.NewFormItem("版本标签", p.customLabel),
    ),
)
```

第二次在“下载校验”区域：

```go
widget.NewForm(
    widget.NewFormItem("SHA256", p.customSHA),
    widget.NewFormItem("版本标签", p.customLabel),
)
```

Fyne 中同一个 `CanvasObject` 不应同时挂到多个父容器。这样可能造成显示异常、布局错乱或某个区域控件消失。

### 修复方案

二选一。

#### 方案 A：SHA256 和版本标签只属于自定义 ZIP

如果 SHA256 / 版本标签只对自定义 ZIP 有意义，则删除“下载校验”区域中的重复表单，把它们保留在 `customBox` 内。

结构：

```text
Chrome 来源
  下载来源
  通道标记
  自定义下载 URL
  SHA256
  版本标签

安装与数据目录
  安装目录
  用户数据目录

缓存策略
  fallback
  keep cache
```

#### 方案 B：下载校验区域保留，customBox 只放 URL

如果想保留“下载校验”分区，则 `customBox` 只放：

```go
widget.NewFormItem("自定义下载 URL", p.customURL)
```

`customSHA` 和 `customLabel` 只出现在“下载校验”区域。

推荐方案 B，因为它符合当前 Todo7 的分区方向。

### 验收标准

- `customSHA` 和 `customLabel` 在代码中只挂载到一个容器。
- 切换“官方 Stable / 自定义 ZIP”时布局不跳动、不丢控件。
- 保存配置后 SHA256 和版本标签仍能正确写入配置。

## P1-1. 检查运行详情页右侧信息面板高度

### 问题

当前代码：

```go
rightCard := container.NewGridWrap(fyne.NewSize(240, rightPanel.MinSize().Height), rightPanel)
```

`rightPanel.MinSize().Height` 在初始化时可能偏小，后续摘要变成多行、产物路径增加后，右侧面板可能高度不足或裁切。

### 修复方案

建议不要用初始化时的 `MinSize().Height` 固定高度。可以：

1. 只固定宽度，不固定高度。
2. 或者给右侧面板使用 `container.NewScroll(rightPanel)`。
3. 或者使用一个足够稳定的固定高度，例如 360，但需要窗口验收确认。

推荐方案：

```go
rightCard := container.NewGridWrap(fyne.NewSize(260, 360), container.NewScroll(rightPanel))
```

或直接：

```go
rightCard := container.NewGridWrap(fyne.NewSize(260, 1), rightPanel)
```

具体以 Fyne 实际布局效果为准。

### 验收标准

- 摘要多行显示完整。
- 截图和 HTML 产物同时存在时不被裁切。
- 1280x720 下右侧区域仍可读。

## P1-2. 检查全局工具栏窄窗口拥挤问题

### 当前状态

`internal/ui/global_toolbar.go` 已经把浏览器和执行按钮放入 `newToolbarGroup()`，方向正确。但顶部元素仍较多：

```text
流程选择 | 浏览器组 | 执行组 | 环境选择 | 进度条
```

### 需要实际验收

必须在以下窗口尺寸检查：

```text
1280x720
1366x768
1440x900
1920x1080
```

重点看：

- `启动 Chrome` 是否被挤压。
- `关闭托管` 是否完整。
- `停止流程` 是否完整。
- 环境选择是否过窄。
- 进度文字是否重新显示为 `...`。
- Tab 栏是否被顶部工具栏挤压。

### 可能修复方向

如果 1280 宽仍拥挤：

1. 将按钮文案进一步压缩：

```text
启动
关闭托管
运行
单步
停止
```

2. 用图标按钮 + tooltip 替代部分文字按钮。
3. 把进度带固定在工具栏第二行。
4. 将低频按钮放入更多菜单，但不能隐藏“停止流程”和“关闭托管”。

## P1-3. 检查状态色在深色模式下是否真实可读

### 当前状态

`internal/ui/theme.go` 已按 `variant` 分浅色/深色返回颜色，方向正确。

但 `internal/ui/style.go` 中部分 helper 通过：

```go
fyne.CurrentApp().Settings().ThemeVariant()
```

动态取当前 variant。需要确认主题切换或初始化时颜色是否刷新正常。

### 验收标准

- 浅色模式下文字、输入框、表格、状态点对比度正常。
- 深色模式下背景、输入框、按钮、状态点不混色。
- `canvas.Text` 创建后如果系统主题切换，颜色不会长期停留在旧 variant。若无法动态刷新，至少保证启动时当前 variant 正确。

## P1-4. 检查两行 List 项的行高

### 当前状态

以下列表已改成两行 cell：

- `flowListItem`
- `envListItem`
- `historyListItem`

但 Fyne `widget.List` 默认 item 高度可能不足，需要实际确认两行是否被裁切。

### 需要检查的位置

```text
internal/ui/flow_library.go
internal/ui/env_panel.go
internal/ui/history_panel.go
```

### 可能修复方向

如果两行显示被裁切，需要设置 item height：

```go
p.list.SetItemHeight(id, height)
```

或在模板 cell 中提供稳定 `MinSize()`，自定义 widget 的 `MinSize()` 必须能覆盖两行高度。

### 验收标准

- 流程库每行两行文本完整显示。
- 环境列表每行两行文本完整显示。
- 历史记录每行两行文本完整显示。
- 选中项不会裁切第二行。

## P2-1. 文档和工作区状态整理

### 当前状态

当前工作区有大量未提交文件：

```text
Todo1.md
Todo2.md
Todo3.md
Todo4.md
Todo5.md
Todo6.md
Todo7.md
Todo8.md
internal/ui/style.go
```

以及多处 UI 修改。

### 建议

在修复 P0 问题并完成验收前，不建议提交。提交前应确认：

- 是否需要把 `Todo1.md` 到 `Todo8.md` 全部提交。
- 是否有旧 Todo 内容已经过时，需要保留还是归档。
- `git diff` 中没有误改无关逻辑。
- 没有提交 `data/`、`logs/`、`chrome/`、构建产物。

## 最低通过标准

后续 Agent 至少完成以下事项，才能认为当前 UI 优化满足要求：

1. 修复步骤属性校验 panic。
2. 顶部保存状态重新可见。
3. 运行日志空状态实际显示。
4. 设置页不再重复挂载同一个输入控件。
5. `go test ./...` 通过。
6. 在 1280x720 和 1440x900 下实际打开 UI 检查，无明显挤压、裁切、`...` 异常。
7. 确认字体仍为 MapleMono-CN，中英文没有回退割裂。
