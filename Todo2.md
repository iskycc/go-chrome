# UI 二次优化任务书

本文档基于当前已完成的 `Todo1.md` 优化继续补充。当前 UI 已有较大改善：已增加全局工具栏、环境配置 Tab、运行详情页、状态栏和字体资源。但用户仍明确反馈还有继续优化空间，尤其是字体、常用按钮位置和整体精修。

## 用户最新反馈

1. 当前 UI 已明显优化，但还可以继续提升美观度、布局密度和操作一致性。
2. 字体预期更新：用户希望改用 **JetBrains Code** 字体，并且中文和英文都要使用这一套字体风格。
3. 当前所有中文字体似乎没有切换到新字体，需要继续排查和修复。
4. “停止当前流程”和“停止所有浏览器/关闭浏览器”也属于常用按钮，应放到和“启动浏览器”相同的位置，而不是藏在运行详情页。

## 当前字体问题诊断

### 现状

- 字体文件已存在：

```text
assets/fonts/CascadiaCode-SemiLight.ttf
assets/fonts/LICENSE-CascadiaCode.txt
```

- `assets/embed.go` 已嵌入该字体。
- `internal/ui/theme.go` 的 `Font()` 已返回 `assets.CascadiaCodeSemiLight()`。

### 核心问题

当前嵌入的 `CascadiaCode-SemiLight.ttf` **不包含常用中文字形**，且字重过细，实际启动后观感不满足要求。

用 `fc-scan` 检查 charset 时，没有看到 CJK 常用区，例如：

```text
4e00-9fff
3400-4dbf
f900-faff
```

因此，即使 `theme.Font()` 已经返回 Cascadia Code SemiLight，中文也无法真正用这个字体渲染。Fyne 也不能像 CSS 那样稳定声明字体 fallback 栈，所以“返回 Cascadia Code + 期望系统自动 fallback 中文”不能满足用户预期。

另外，JetBrains 官方常见字体名是 **JetBrains Mono**。后续 Agent 需要确认用户所说的 **JetBrains Code** 是具体哪一个字体文件/字体项目。不要只按名称替换，必须拿到真实字体文件后检查 CJK 覆盖和许可证。

### 用户真实要求

后续 Agent 不要把目标理解为：

```text
英文/代码：JetBrains Code
中文：系统默认字体
```

真实目标是：

```text
中文和英文都要呈现为同一套 JetBrains Code 风格。
```

也就是说，最终用于 UI 的字体资源必须同时覆盖：

- ASCII / Latin
- 数字
- 常用中文
- 中文标点
- 常用符号

并且视觉上应接近 JetBrains Code / JetBrains Mono 的清晰、代码感、现代等宽风格。普通 UI 不要再使用过细字重，建议默认使用 Regular，标题和重要按钮使用 Medium 或 SemiBold。

## P0-1. 修复中英文字体统一问题

### 目标

应用全局中文和英文都使用同一套用户认可的 JetBrains Code 风格字体，不再出现中文回退到系统字体、英文是另一套字体的割裂效果。

### 推荐方案 A：使用合法的 JetBrains Code CJK 完整字体（首选）

如果可以获得或生成一个合法分发的字体文件，建议使用一个**包含 JetBrains Code / JetBrains Mono Latin glyph + CJK glyph** 的完整字体，例如：

```text
assets/fonts/JetBrainsCode-Regular-CJK.ttf
assets/fonts/JetBrainsCode-Medium-CJK.ttf
```

要求：

1. 必须确认许可证允许随本项目分发。
2. 必须保留字体许可文件：

```text
assets/fonts/LICENSE-JetBrainsCode-CJK.txt
```

3. 字体必须覆盖 CJK 常用区。
4. Fyne theme 必须全局返回这个 CJK 完整字体，而不是原始 `CascadiaCode-SemiLight.ttf`。
5. 必须至少提供 Regular 字重；如果有 Medium/SemiBold，标题和按钮应使用更粗字重。

修改建议：

```go
func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
    if f := assets.AppUIFont(); f != nil {
        return f
    }
    return theme.DefaultTheme().Font(style)
}
```

`assets.AppUIFont()` 返回最终确认的中英文完整字体。

验收命令：

```bash
fc-scan --format '%{family}\n%{charset}\n' assets/fonts/JetBrainsCode-Regular-CJK.ttf
```

输出中必须包含或覆盖：

```text
4e00-9fff
```

### 推荐方案 B：使用 JetBrains Mono + CJK 合成/融合字体

如果“JetBrains Code”实际指的是 JetBrains Mono，而官方字体文件不含中文，则需要使用合法的 CJK 融合版本。候选方向：

- JetBrains Mono Latin glyph + 合法 CJK glyph 的融合字体。
- JetBrains Maple Mono 之类明确说明合并 JetBrains Mono 与 CJK 字形的项目。
- 其他明确包含 CJK 且风格接近 JetBrains Mono 的字体。

注意：这不是“中文用另一个 fallback 字体”，而是**整个 UI 都使用同一个 CJK 完整字体资源**。

如果采用该方案，应在文档和代码注释中说明：

- 原始 JetBrains Mono / JetBrains Code 字体是否包含中文，必须以实际 `fc-scan` 或字体工具检查为准。
- 该字体是为了满足“中英文统一风格”而选用。
- 字体许可允许分发。

### 不接受的方案

以下方案不满足用户要求：

- 继续只嵌入原始 `CascadiaCode-SemiLight.ttf`。
- 只嵌入一个不含 CJK 的 JetBrains Mono / JetBrains Code 字体。
- 英文用 JetBrains，中文用系统默认字体。
- 在不同控件里手动混用多个字体，导致中文英文视觉不统一。
- 使用来源不明、许可不清楚的字体文件。
- 使用过细字重作为全局字体，导致中文和按钮文字发虚。

### 代码改造建议

1. `assets/embed.go` 不要暴露具体字体名作为唯一 UI 字体入口，建议改为：

```go
func AppUIFont() fyne.Resource
func CodeFont() fyne.Resource
```

2. `AppUIFont()` 返回中英文完整字体。
3. `CodeFont()` 可以返回同一套 JetBrains Code CJK 完整字体的 Regular 或 Medium 字重。除非用户明确接受，否则不要让日志/代码区域和普通 UI 的中文字体割裂。
4. `theme.Font()` 应返回 `AppUIFont()`。
5. 如果后续需要局部代码字体，必须确保不会导致中文显示割裂。
6. `theme.Font(style)` 必须区分字重：

```go
func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
    if style.Bold {
        if f := assets.AppUIFontMedium(); f != nil {
            return f
        }
    }
    if f := assets.AppUIFontRegular(); f != nil {
        return f
    }
    return theme.DefaultTheme().Font(style)
}
```

### 验收标准

- 应用内中文、英文、数字视觉风格统一。
- 中文不再回退到明显不同的系统字体。
- 中文不缺字、不显示方块。
- `fc-scan` 或等价检查确认 UI 字体覆盖 CJK。
- Windows 环境下实际运行确认中文按钮、Tab、表格、日志都使用 JetBrains Code 风格字体。
- 普通字体不能过细；标题、Tab、主按钮必须有明显字重层级。

## P0-2. 将“停止当前流程”和“停止/关闭托管浏览器”提升到全局工具栏

### 当前问题

当前全局工具栏已经有：

- 启动浏览器
- 运行
- 单步执行
- 停止

但“关闭本程序启动的 Chrome”仍主要在 `运行详情` 页中：

```text
internal/ui/run_panel.go
```

用户认为停止当前流程、停止浏览器和启动浏览器一样常用，应该放在同一个全局位置。

### 产品语义澄清

用户说“停止所有浏览器”，但项目历史约束是：

- 不要污染或误杀用户自己的系统 Chrome。
- 关闭按钮应只影响本程序启动和管理的 Chrome。

因此建议按钮文案使用更安全、明确的名称：

```text
关闭托管 Chrome
```

或：

```text
停止托管浏览器
```

如果产品最终真的要“停止所有 Chrome 进程”，必须单独做成危险操作，并且需要非常明确的二次确认，不应作为普通常用按钮默认执行。

### 具体方案

1. 修改 `internal/ui/global_toolbar.go`。
2. 在 `globalToolbar` 中增加按钮字段：

```go
stopChromeBtn *widget.Button
```

3. 将浏览器相关按钮放到同一组：

```text
浏览器：启动 Chrome | 关闭托管 Chrome
```

4. 将流程执行相关按钮放到另一组：

```text
执行：运行 | 单步执行/下一步 | 停止当前流程
```

5. `关闭托管 Chrome` 调用现有：

```go
app.closeManagedChrome()
```

6. `停止当前流程` 调用现有：

```go
app.stopCurrentRun()
```

7. 按钮状态建议：

- `启动 Chrome`
  - 未启动时启用
  - 已启动/启动中时禁用
- `关闭托管 Chrome`
  - 未启动时禁用
  - 已启动/启动中时启用
- `停止当前流程`
  - 空闲时禁用但保持可见
  - 运行中/单步执行中启用

8. 不建议使用 `Hide()` 隐藏停止按钮，因为按钮出现/消失会导致布局跳动。应使用 `Enable()` / `Disable()`。

### 运行详情页同步调整

当 `关闭托管 Chrome` 移到全局工具栏后，`运行详情` 页中应删除重复按钮：

- 底部 `关闭本程序启动的 Chrome`
- `更多` 菜单里的同类入口

运行详情页只保留：

- 清空日志
- 复制日志
- 打开产物目录
- 浏览器下载配置跳转
- 运行摘要、当前步骤、产物路径、日志

### 验收标准

- 任意 Tab 下都能看到：
  - 启动 Chrome
  - 关闭托管 Chrome
  - 运行
  - 单步执行 / 下一步
  - 停止当前流程
- 停止当前流程按钮空闲时禁用但位置保留。
- 关闭托管 Chrome 按钮和启动 Chrome 按钮在同一组。
- 运行详情页不再重复放置关闭 Chrome 主按钮。
- 关闭托管 Chrome 不会影响用户手动打开的系统 Chrome。

## P0-3. 全局工具栏继续做视觉分组和布局稳定

### 当前问题

全局工具栏功能已存在，但仍偏控件并排。下一步应强化分组，让用户一眼知道哪些按钮属于流程、浏览器、执行、环境。

### 建议布局

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ 流程 [流程下拉................] [保存]  浏览器 [启动 Chrome] [关闭托管]    │
│ 执行 [运行] [单步/下一步] [停止当前流程]  环境 [环境下拉........]          │
│ 进度 第 2/6 步 · 输入用户名  [====================              ]          │
└─────────────────────────────────────────────────────────────────────────────┘
```

如果一行太挤，可以使用两行：

```text
第一行：流程组 + 浏览器组 + 环境组
第二行：执行组 + 进度
```

### 具体建议

1. 给每组加固定 label：
   - `流程`
   - `浏览器`
   - `执行`
   - `环境`
2. 每组内按钮数量控制在 2-3 个。
3. 使用固定宽度或 `GridWrap` 限制流程下拉、环境下拉，避免长名称撑爆。
4. 长流程名用截断显示。
5. 分组之间加间距或浅色分隔线。
6. 危险/停止类按钮使用红色或 DangerImportance，但不要过分刺眼。

### 验收标准

- 1280x720 下工具栏不换成混乱布局。
- 流程名很长时不会挤压执行按钮。
- 浏览器启动/停止和流程运行/停止在视觉上能区分。

## P0-4. 修复流程下拉同名覆盖问题

### 当前问题

`globalToolbar.refreshFlows()` 当前使用流程名作为 map key：

```go
flowByName map[string]*flow.Flow
```

如果两个流程同名，后一个会覆盖前一个，用户无法可靠选择指定流程。

### 具体方案

1. 改为保存选项结构：

```go
type flowSelectOption struct {
    Label string
    ID string
}
```

2. `globalToolbar` 使用：

```go
flowOptions []flowSelectOption
flowByLabel map[string]string // label -> id
flowByID map[string]*flow.Flow
```

3. 生成 Label 时处理同名：

```text
登录流程
登录流程 · a1b2c3
登录流程 · d4e5f6
```

或显示最近修改时间/步骤数辅助区分。

4. 选择后按 ID 加载完整 flow，不要只拿列表里的浅 flow。

### 验收标准

- 存在两个同名流程时，下拉可以区分并正确选中。
- 当前流程状态栏和编辑区显示正确。

## P1-1. 环境配置页继续优化成表格工具页

### 当前问题

`internal/ui/env_panel.go` 已经存在，但变量列表仍是 `widget.List + GridWithColumns`，长 VALUE 或说明可能挤压。

### 建议

1. 改成 `widget.Table` 或自定义稳定列宽布局。
2. 列建议：

```text
KEY | VALUE | 敏感 | 说明 | 操作
```

3. VALUE 和说明必须截断，不撑开布局。
4. 敏感变量显示：

```text
******
```

5. 编辑入口可以是按钮或双击行。

### 验收标准

- 长环境变量值不会撑爆界面。
- 敏感变量不会在列表中明文展示。
- 变量表看起来像管理工具，而不是普通列表。

## P1-2. 环境配置必须支持导入和导出

### 新增需求

环境配置不仅要能在 UI 中编辑，还必须支持导入和导出，方便用户在不同机器、不同项目目录、离线环境或测试环境之间迁移环境变量。

### 当前实现参考

当前 `internal/ui/env_panel.go` 中已经存在菜单项：

```go
fyne.NewMenuItemWithIcon("导入配置", theme.DownloadIcon(), func() { p.showImportEnvDialog() })
fyne.NewMenuItemWithIcon("导出配置", theme.UploadIcon(), func() { p.showExportEnvDialog() })
```

后续 Agent 需要确认这些入口真实可用，并把它们提升为明确的一等功能，而不是隐藏在难发现的位置。

### 具体方案

1. 在“环境配置”Tab 中保留并强化导入/导出入口。
2. 推荐位置：
   - 环境配置页顶部右侧：`导入配置`、`导出配置`
   - 或环境操作菜单中保留，但按钮区必须能明显看到“更多/导入导出”入口。
3. 导出格式应复用现有 `EnvRepo.Export()`，输出 JSON。
4. 导入格式应复用现有 `EnvRepo.Import()`，读取 JSON。
5. 导入前后需要明确反馈：
   - 导入成功：刷新环境列表、变量列表、全局环境下拉。
   - 导入失败：弹窗展示错误，不修改当前 UI 选择状态。
6. 导出时建议默认文件名：

```text
go-chrome-env-config.json
```

7. 导入成功后，如果导入内容包含 active 环境，应同步全局环境选择；否则保持当前 active 环境或选中默认环境。
8. 敏感变量导出策略需要明确：
   - 当前如果 `EnvRepo.Export()` 会导出敏感变量明文，UI 必须在导出前给出提示。
   - 可选增强：导出弹窗提供“包含敏感变量值”复选框，默认关闭或由产品确认。
   - 不要在没有提示的情况下让用户误以为敏感变量不会被导出。
9. 导入时必须校验：
   - JSON 格式错误要清楚提示。
   - 空 key 或非法 key 应拒绝或跳过并提示。
   - 重名环境如何处理要明确：覆盖、重命名导入、合并变量，三者只能选一个默认策略。

### 推荐默认策略

- 导出：导出所有环境和变量；如果包含敏感变量，先弹确认提示。
- 导入：不覆盖现有环境，重名环境自动追加后缀，例如 `默认环境 导入`。
- 导入完成：刷新环境配置页、全局环境下拉、历史筛选环境列表。

### 验收标准

- 用户可以在“环境配置”Tab 中找到导入/导出配置入口。
- 导出后得到可读 JSON 文件。
- 导入刚导出的 JSON 后，环境和变量能恢复。
- 导入成功后全局环境下拉同步刷新。
- 导入失败不会破坏已有环境配置。
- 敏感变量导出前有明确提示或明确导出策略。

## P1-3. 运行详情页继续瘦身

### 当前问题

运行详情页已经拆掉主要运行控制，但仍保留关闭 Chrome 入口。全局工具栏完善后，运行详情页应只做详情展示和低频操作。

### 建议保留

- 清空日志
- 复制日志
- 打开产物目录
- 浏览器下载配置跳转
- 日志
- 运行摘要
- 当前步骤
- 截图 / HTML 产物

### 建议删除

- 关闭本程序启动的 Chrome 主按钮
- 关闭 Chrome 菜单项
- 任何运行/停止/单步/环境选择重复入口

### 验收标准

- 用户不会在运行详情页和全局工具栏看到重复控制。
- 运行详情页聚焦观察和诊断。

## P1-4. 视觉细节继续打磨

### 建议

1. 全局工具栏背景与 Tab 内容区做轻微分层。
2. Tab 内容顶部增加一致的页面标题和说明，但不要写大量使用说明。
3. 表格/列表行高统一。
4. 按钮文案尽量短：
   - `启动 Chrome`
   - `关闭托管`
   - `停止流程`
   - `运行`
   - `单步`
5. 日志字体和 UI 字体如果最终不同，必须保证中文不割裂；如做不到，日志也使用同一个中英文完整字体。
6. 所有路径类文本必须截断，并提供复制能力。

## 推荐实施顺序

1. 先解决字体：换成或生成一个覆盖 CJK 的 JetBrains Code 风格字体，并让 `theme.Font()` 全局使用它。
2. 将 `关闭托管 Chrome` 提升到全局工具栏，与 `启动 Chrome` 同组。
3. 将 `停止当前流程` 改为常驻按钮，空闲时禁用，不隐藏。
4. 删除运行详情页中的重复关闭 Chrome 入口。
5. 修复流程下拉同名覆盖问题。
6. 优化全局工具栏分组和窄窗口表现。
7. 优化环境配置变量表。
8. 验证并强化环境配置导入/导出。

## 测试与验收清单

- `go test ./...`
- Windows 实机检查字体：
  - Tab 中文
  - 按钮中文
  - 流程名中文
  - 环境变量中文说明
  - 日志中文
- `fc-scan` 或等价工具确认最终 UI 字体覆盖 CJK。
- 创建两个同名流程，确认全局流程下拉可以区分。
- 启动 Chrome 后，全局工具栏中 `启动 Chrome` 禁用，`关闭托管 Chrome` 启用。
- 运行流程时，`停止当前流程` 启用。
- 空闲时，`停止当前流程` 可见但禁用。
- 运行详情页不再出现重复的关闭 Chrome 主入口。
- 关闭托管 Chrome 只关闭本程序管理的 Chrome，不影响用户手动打开的 Chrome。
- 环境配置可以导出为 JSON。
- 导入环境配置 JSON 后，环境列表、变量列表和全局环境下拉同步刷新。
- 包含敏感变量时，导出前有明确提示或已在 UI 中说明导出策略。

## 注意事项

- 不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制或运行产物。
- 字体文件必须确认许可，不能提交来源不明的字体。
- 不要在程序启动时联网下载字体。
- 不要实现默认杀掉系统所有 Chrome 进程的功能；如必须做，需要单独危险操作和强确认。
