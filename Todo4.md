# UI 弹框尺寸与变形修复任务书

本文档记录当前构建版本中仍可能出现“输入框太窄、弹框过小、长文本撑爆、UI 变形”的位置和修复方案。后续 Agent 应按本清单逐项处理，并优先建立统一弹框工具函数，避免继续零散修补。

## 总体问题

当前主界面已经完成较大幅度优化，但部分弹框仍使用 Fyne 默认尺寸或默认确认框：

- `dialog.ShowForm(...)` 默认宽度偏窄，输入框很短。
- `dialog.ShowFileOpen(...)` / `dialog.ShowFileSave(...)` 如果不 `Resize`，文件路径较长时体验差。
- `dialog.ShowConfirm(...)` 对长流程名、长环境名、长变量名不做截断/换行，容易撑宽或挤压按钮。
- 环境变量值、说明、路径等长文本仍使用单行 Entry 或 Label，容易导致布局变形。

## P0-1. 建立统一弹框尺寸规范

### 建议尺寸

所有自定义弹框不要依赖 Fyne 默认尺寸。

推荐规范：

```text
单字段表单：       480 x 180
两字段表单：       560 x 240
三到四字段表单：   640 x 320
文件打开/保存：    720 x 520
确认弹框：         520 x 180，长文本自动换行
模板/复杂选择器：  720 x 420 或更大
```

### 建议新增工具函数

建议新增文件：

```text
internal/ui/dialogs.go
```

提供统一 helper：

```go
func sizedEntry(placeholder string) *widget.Entry
func sizedMultiLineEntry(placeholder string, rows int) *widget.Entry
func showSizedFormDialog(title, confirm, cancel string, items []*widget.FormItem, size fyne.Size, cb func(bool), win fyne.Window)
func showWrappedConfirm(title, message, confirm, cancel string, size fyne.Size, cb func(bool), win fyne.Window)
func resizeFileDialog(d dialog.Dialog)
```

建议实现要点：

- 使用 `dialog.NewCustomConfirm` 替代 `dialog.ShowForm`。
- 表单内容使用 `container.NewVBox` + `widget.Form` 或显式 label/entry。
- `Entry` 外层可用 `container.NewGridWrap(fyne.NewSize(420, entry.MinSize().Height), entry)` 强制宽度。
- 长说明字段用 `MultiLineEntry`，不要普通单行 Entry。
- 确认弹框正文用 `widget.Label`，设置 `Wrapping = fyne.TextWrapWord`。
- 长名称展示前使用 `truncate(name, 80)` 或 `newTruncatingLabel`，避免撑爆弹框。

## P0-2. 环境配置弹框：全部替换 `dialog.ShowForm`

当前高风险位置集中在 `internal/ui/env_panel.go`。

### 1. 新建环境弹框

位置：

```text
internal/ui/env_panel.go:303-322
```

当前问题：

- 使用 `dialog.ShowForm("新建环境", ...)`。
- 输入框默认宽度偏窄。
- 环境名称稍长时体验差。

修复方案：

- 改成 `dialog.NewCustomConfirm`。
- 弹框尺寸建议 `480 x 180`。
- 环境名称输入框固定宽度 `360-420px`。

验收：

- 新建环境弹框中输入框不再短小。
- 中文长环境名输入时不会挤压按钮。

### 2. 编辑环境弹框

位置：

```text
internal/ui/env_panel.go:331-349
```

当前问题：

- 使用 `dialog.ShowForm("编辑环境", ...)`。
- “说明”使用单行 Entry，长说明不适合。
- 输入框宽度默认偏窄。

修复方案：

- 改成 `dialog.NewCustomConfirm`。
- 弹框尺寸建议 `560 x 260`。
- 环境名称使用单行 Entry。
- 环境说明使用 `MultiLineEntry`，至少 3 行高度。
- 内容区可滚动，避免较小窗口下撑出屏幕。

验收：

- 环境名称和说明都能舒适编辑。
- 说明较长时弹框不横向变形。

### 3. 新增变量弹框

位置：

```text
internal/ui/env_panel.go:513-540
```

当前问题：

- 使用 `dialog.ShowForm("新增变量", ...)`。
- 变量名和变量值输入框默认过窄。
- 变量值可能很长，例如 URL、token、JSON 片段。

修复方案：

- 改成 `dialog.NewCustomConfirm`。
- 弹框尺寸建议 `640 x 300`。
- 变量名 Entry 固定宽度。
- 变量值建议使用 `MultiLineEntry` 或至少更宽的 Entry。
- 敏感变量 Check 不要占用表单 label 列导致错位，可放在输入框下方。

验收：

- 长 URL / token 输入时不会撑爆或难以编辑。
- 敏感变量复选框布局自然，不挤压输入框。

### 4. 编辑变量弹框

位置：

```text
internal/ui/env_panel.go:549-577
```

当前问题：

- 使用 `dialog.ShowForm("编辑变量", ...)`。
- 变量值和说明都是单行 Entry。
- 长变量值、长说明会导致编辑困难和 UI 变形。

修复方案：

- 改成 `dialog.NewCustomConfirm`。
- 弹框尺寸建议 `680 x 360`。
- 变量值使用 `MultiLineEntry`，至少 3 行。
- 说明使用 `MultiLineEntry`，至少 2 行。
- 如果变量是敏感变量，可考虑 `PasswordEntry` 或“显示/隐藏”切换。

验收：

- 编辑长 token、长 URL、JSON 值时可用。
- 说明字段不会横向撑宽弹框。

## P0-3. 环境配置导入/导出文件弹框需要固定尺寸

### 1. 导入环境配置

位置：

```text
internal/ui/env_panel.go:425-450
```

当前问题：

- 使用 `dialog.NewFileOpen` 但没有 `Resize`。
- 路径长、文件多时默认弹框偏小。

修复方案：

```go
fd.Resize(fyne.NewSize(720, 520))
```

验收：

- 文件选择器有足够空间展示目录和文件名。

### 2. 导出环境配置

位置：

```text
internal/ui/env_panel.go:453-468
```

当前问题：

- 使用 `dialog.NewFileSave` 但没有 `Resize`。
- 默认文件名和路径编辑区域可能偏窄。

修复方案：

```go
fd.Resize(fyne.NewSize(720, 520))
```

验收：

- 文件名 `go-chrome-env-config.json` 显示完整。
- 保存路径选择不局促。

## P0-4. 流程导出文件弹框需要固定尺寸

位置：

```text
internal/ui/main_window.go:332-345
```

当前问题：

- `exportFlow()` 使用 `dialog.ShowFileSave`，无法设置尺寸。
- 导出路径或文件名较长时弹框偏小。

修复方案：

- 改用 `dialog.NewFileSave`。
- 设置默认文件名，例如：

```text
{flow-name}.json
```

- 设置尺寸：

```go
d.Resize(fyne.NewSize(720, 520))
```

验收：

- 流程导出弹框尺寸与导入弹框一致。
- 长流程名默认文件名不撑爆，可截断或清理非法文件名。

## P1-1. 确认弹框长文本需要自定义换行和截断

当前若干确认弹框使用 `dialog.ShowConfirm`，长名称可能撑宽弹框或挤压按钮。建议统一替换为 `showWrappedConfirm(...)`。

### 1. 删除环境确认

位置：

```text
internal/ui/env_panel.go:386-410
```

问题：

- `fmt.Sprintf("确定删除环境 [%s] 吗？", env.Name)` 中环境名过长会撑宽弹框。

方案：

- 使用换行 label。
- 环境名用 `truncate(env.Name, 80)`。
- 弹框尺寸 `520 x 180`。

### 2. 导出敏感变量确认

位置：

```text
internal/ui/env_panel.go:493-497
```

问题：

- 默认确认框能用，但这是高风险操作，应有更清楚的视觉层级。

方案：

- 自定义确认框。
- 文案明确：

```text
导出文件将包含敏感变量的明文值。
请确认该文件只保存在可信位置。
```

- 按钮：`继续导出` / `取消`。

### 3. 删除变量确认

位置：

```text
internal/ui/env_panel.go:586-595
```

问题：

- 变量名很长时可能撑宽弹框。

方案：

- 使用 `showWrappedConfirm`。
- 变量名截断。

### 4. 删除流程确认

位置：

```text
internal/ui/main_window.go:831-844
```

问题：

- 流程名很长时可能撑宽弹框。

方案：

- 使用 `showWrappedConfirm`。
- 流程名截断。

### 5. 未应用步骤修改确认

位置：

```text
internal/ui/main_window.go:879-888
```

问题：

- 目前 `dialog.ShowConfirm` 两按钮语义偏弱。
- 这是编辑流程中的常见弹框，应与未保存流程弹框风格一致。

方案：

- 改为自定义弹框。
- 按钮建议：
  - `应用并切换`
  - `放弃修改`
  - `取消`
- 尺寸建议 `520 x 200`。

## P1-2. 未保存修改弹框仍需检查长流程名

位置：

```text
internal/ui/main_window.go:785-799
```

当前已有自定义弹框和 `Resize(480, 180)`，但正文包含当前流程名：

```go
"当前流程 [%s] 有未保存的修改，要如何处理？"
```

问题：

- 超长流程名仍可能造成正文高度增加，按钮区挤压。

方案：

- 流程名使用 `truncate(name, 80)`。
- 尺寸可调整为 `560 x 220`。
- 按钮使用 `container.NewGridWithColumns(3, ...)` 保持均分宽度。

验收：

- 超长流程名不会让按钮贴边或溢出。

## P1-3. 模板选择弹框需要长文本截断

位置：

```text
internal/ui/template_dialog.go:28-106
```

当前已设置：

```go
d.Resize(fyne.NewSize(720, 420))
```

但仍有潜在问题：

- 左侧模板名 + 标签使用普通 `widget.Label`。
- 步骤预览列表使用普通 `widget.Label`。
- 模板名、标签、步骤名较长时可能横向溢出。

方案：

- 列表 item 改用 `newTruncatingLabel("")`。
- 右侧说明保留 word wrap。
- 步骤预览也使用 truncating label。

验收：

- 长模板名、长步骤名不撑宽弹框。

## P1-4. 添加步骤弹框尺寸基本可用，但建议统一化

位置：

```text
internal/ui/step_table.go:170-195
```

当前已有：

```go
d.Resize(fyne.NewSize(420, 180))
```

建议：

- 可保留当前尺寸。
- 后续如字体变大或按钮文案更长，建议调整为 `480 x 200`。
- 使用统一 `showSizedFormDialog` helper，避免尺寸散落。

## P2-1. 信息弹框和错误弹框的长文本策略

以下弹框通常问题不大，但如果错误信息包含长路径、长 XPath、长 JSON，也可能变形：

- `dialog.ShowInformation(...)`
- `dialog.ShowError(...)`

建议：

- 对可预期的长路径错误，改用自定义错误弹框，正文换行并提供复制按钮。
- 对运行产物路径、导入失败路径、Chrome 路径等，尽量在日志中显示并可复制，不完全依赖弹框。

## 推荐实施顺序

1. 新增 `internal/ui/dialogs.go`，实现统一 sized form / confirm helper。
2. 替换 `env_panel.go` 中 4 个 `dialog.ShowForm`。
3. 给环境导入/导出文件弹框添加 `Resize(720, 520)`。
4. 将流程导出从 `dialog.ShowFileSave` 改为 `dialog.NewFileSave` 并设置尺寸。
5. 替换长名称相关 `ShowConfirm`。
6. 模板弹框列表 item 改为 truncating label。
7. 手工验证 Windows 实机所有弹框。

## 手工验收清单

- 新建环境：输入框宽度正常。
- 编辑环境：名称和说明输入舒适，说明可多行。
- 新增变量：变量名和值输入框不窄。
- 编辑变量：长 token / URL / JSON 值可编辑，不撑爆弹框。
- 导入环境配置：文件选择器尺寸足够。
- 导出环境配置：保存弹框尺寸足够，默认文件名显示完整。
- 导出流程：保存弹框尺寸足够。
- 删除超长环境名：确认弹框不横向撑宽。
- 删除超长变量名：确认弹框不横向撑宽。
- 删除超长流程名：确认弹框不横向撑宽。
- 未保存修改弹框：超长流程名不会挤压按钮。
- 模板选择弹框：超长模板名、标签、步骤名不会溢出。
- 1280x720 和 1400x900 两种窗口尺寸下弹框都可用。

## 注意事项

- 不要只靠 `ShowForm` 快速补功能；涉及输入框的弹框都应显式设置尺寸。
- 不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制或运行产物。
- 修改弹框时不要破坏已有环境导入/导出、流程导入/导出和未保存修改保护逻辑。
