# UI 体验优化任务书

本文档记录当前用户反馈的 UI / 字体 / 信息架构优化需求，供后续 Agent 继续实现。目标是把应用从“功能可用”提升到“长期使用不割裂、信息清晰、视觉稳定”的状态。

## 用户反馈摘要

1. 当前 UI 页面不够美观，整体观感偏默认控件堆叠。
2. 大量字体存在变形，希望内置一个字体；用户偏好 **Cascadia Code SemiLight**。
3. 顶部 tab 栏需要新增“环境配置”tab，用来展示和修改环境变量。
4. 流程切换、启动、停止等常用功能应在任何页面都能看到，不应必须切到“运行”页；调整后“运行”页中重复的常用功能应删除，只保留非常用功能和运行详情。
5. 其他需要优化的点包括排列布局、字体大小、间距、按钮层级、信息密度和页面一致性。

## 当前相关实现位置

- 主窗口与 tab 架构：`internal/ui/main_window.go`
- 顶部状态栏：`internal/ui/status_bar.go`
- 运行面板：`internal/ui/run_panel.go`
- 流程库：`internal/ui/flow_library.go`
- 环境管理弹窗：`internal/ui/env_dialog.go`
- 主题：`internal/ui/theme.go`
- 资源嵌入：`assets/embed.go`
- 环境变量存储：`internal/db/env_repo.go`
- 运行控制方法：
  - `App.runCurrentFlow()`
  - `App.onStepButton()`
  - `App.stopCurrentRun()`
  - `App.startBrowser()`
  - `App.closeManagedChrome()`

## 总体设计方向

应用是流程编排和运行工具，不适合做成营销页或大面积装饰视觉。优化方向应是：

- 信息结构清晰，常用操作固定可见。
- 视觉风格安静、工作台式、便于长期盯屏。
- 使用稳定布局，减少控件挤压、文本变形、按钮跳动。
- 保留 Fyne 原生体验，但通过主题、字体、间距、分组和全局操作栏提高专业度。
- 不做复杂动画，不引入网络依赖，不让模板、Chrome、运行控制散落在多个互相割裂的页面里。

## P0-1. 内置 Cascadia Code SemiLight 字体，修复字体变形

### 问题

当前 `internal/ui/theme.go` 的 `Font()` 直接返回 `theme.DefaultTheme().Font(style)`。在 Windows 中文环境和 Fyne 默认字体组合下，部分中文、英文、数字、日志等文本会出现观感不一致或变形。

### 具体方案

1. 在仓库新增字体资源目录：

```text
assets/fonts/
```

2. 放入字体文件，建议命名：

```text
assets/fonts/CascadiaCode-SemiLight.ttf
```

如果字体授权或来源有问题，不要随便提交第三方字体。应确认 Cascadia Code 的 OFL 许可允许随应用分发，并在仓库中保留许可说明，例如：

```text
assets/fonts/LICENSE-CascadiaCode.txt
```

3. 修改 `assets/embed.go`，嵌入字体：

```go
//go:embed icon.png fonts/CascadiaCode-SemiLight.ttf
var assetFS embed.FS
```

或新增独立 `fontFS`。提供方法：

```go
func CascadiaCodeSemiLight() fyne.Resource
```

4. 修改 `internal/ui/theme.go`：

```go
func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
    if res := assets.CascadiaCodeSemiLight(); res != nil {
        return res
    }
    return theme.DefaultTheme().Font(style)
}
```

5. 如果 Cascadia Code 对中文覆盖不足，应验证中文是否会回退正常。若 Fyne 不能可靠 fallback，可考虑：
   - UI 正文使用更适合中文的字体。
   - 日志、代码、XPath、模板输入使用 Cascadia Code。
   - 不要为了单一字体导致中文缺字或显示方块。

6. 日志面板、XPath、模板输入、环境变量 Key 等更适合等宽字体；普通按钮和中文标题如果显示不佳，可保留系统默认字体。

### 验收标准

- Windows 下中文、英文、数字不再明显变形。
- 日志、XPath、模板表达式、环境变量 Key 排版稳定。
- 字体资源被打包进二进制或发布包，不依赖系统安装字体。
- 字体许可文件随仓库保存。

## P0-2. 增加全局常用操作栏，任何页面都能切换流程、运行、停止

### 问题

当前运行相关按钮主要在“运行”tab 中。用户在“流程”“步骤”“历史”“设置”等页面时，如果想运行、停止、切换流程，需要切换页面，造成明显割裂。

### 目标

常用操作必须在任意页面可见：

- 当前流程选择/切换。
- 保存当前流程。
- 启动浏览器。
- 运行整个流程。
- 单步执行 / 下一步。
- 停止。
- 当前环境选择。
- 运行进度简要状态。

### 具体方案

1. 在 `internal/ui/main_window.go` 中把主窗口布局改为：

```text
┌────────────────────────────────────────────────────────────┐
│ 顶部状态栏：产品名 + 状态信息                               │
├────────────────────────────────────────────────────────────┤
│ 全局操作栏：流程选择 保存 启动浏览器 运行 单步 停止 环境选择 │
├────────────────────────────────────────────────────────────┤
│ Tab：流程 / 步骤 / 环境配置 / 历史 / 设置 / 运行详情         │
└────────────────────────────────────────────────────────────┘
```

2. 新增组件文件建议：

```text
internal/ui/global_toolbar.go
```

3. `globalToolbar` 建议包含：

```go
type globalToolbar struct {
    app *App
    widget fyne.CanvasObject
    flowSelect *widget.Select
    envSelect *widget.Select
    saveBtn *widget.Button
    startChromeBtn *widget.Button
    runBtn *widget.Button
    stepBtn *widget.Button
    stopBtn *widget.Button
    progress *widget.ProgressBar
    progressText *widget.Label
}
```

4. 流程选择：
   - 使用 `widget.Select` 或更适合长列表的搜索选择控件。
   - 选项显示流程名，内部需要能映射 flow ID。
   - 切换时复用现有 `onFlowSelected()`，不能绕过完整 flow load。
   - 有未保存修改时仍走 `promptSaveBefore()`。

5. 环境选择：
   - 复用 `runPanel.refreshEnvironments()` 逻辑，但迁移到全局操作栏。
   - 当前环境变更后调用 `envRepo.SetActive()`。
   - 环境配置 tab 修改环境后要刷新全局环境下拉。

6. 运行按钮：
   - 调用现有 `runCurrentFlow()`。
   - 单步按钮调用 `onStepButton()`。
   - 停止按钮调用 `stopCurrentRun()`。
   - 启动浏览器按钮调用 `startBrowser()`。
   - 关闭托管 Chrome 不是最高频操作，可放入更多菜单。

7. 进度：
   - 全局操作栏显示轻量级进度，例如 `就绪`、`第 2/6 步：输入用户名`。
   - 详细日志、截图、HTML 产物仍在“运行详情”页。

8. 移除或弱化运行页中的重复常用按钮：
   - “运行整个流程”
   - “单步执行”
   - “停止”
   - “当前环境”
   - “启动浏览器”
   这些放到全局操作栏后，运行页只保留运行详情、日志、摘要、产物、非常用操作。

### 验收标准

- 在任意 tab 中都能看到当前流程、当前环境、运行、单步、停止。
- 用户在“步骤”tab 修改步骤后可以直接点击全局运行按钮。
- 用户在“历史”tab 查看记录时也能停止当前运行。
- “运行详情”页不再重复展示一整套常用控制按钮。

## P0-3. 新增“环境配置”Tab，替代环境管理弹窗作为主入口

### 问题

当前环境变量管理通过 `showEnvManager()` 弹窗实现，入口藏在运行页“更多”菜单里。用户反馈顶部 tab 需要有“环境配置”，用于展示和修改环境变量。

### 具体方案

1. 新增面板文件：

```text
internal/ui/env_panel.go
```

2. 将 `internal/ui/env_dialog.go` 中的主要内容拆成可复用组件：

```go
type envPanel struct {
    app *App
    widget fyne.CanvasObject
}

func newEnvPanel(app *App) *envPanel
func (p *envPanel) refresh()
```

3. 在 `App` 结构体中新增：

```go
envPanel *envPanel
globalToolbar *globalToolbar
```

4. 在 `buildUI()` 中新增 tab：

```go
container.NewTabItemWithIcon("环境配置", theme.SettingsIcon(), a.envPanel.widget)
```

推荐 tab 顺序：

```text
流程 / 步骤 / 环境配置 / 历史 / 设置 / 运行详情
```

5. 环境配置页面布局建议：

```text
┌─────────────────────┬────────────────────────────────────┐
│ 环境列表             │ 环境变量表格                        │
│ 搜索/新建/复制/删除  │ KEY | VALUE | 敏感 | 说明 | 操作      │
│ 默认环境 [当前]      │ 新增变量 编辑变量 删除变量            │
└─────────────────────┴────────────────────────────────────┘
```

6. 交互要求：
   - 左侧选择环境，右侧显示变量。
   - 可新增、重命名、复制、删除环境。
   - 可设为当前环境。
   - 可新增、编辑、删除变量。
   - 敏感变量默认显示 `******`，提供“显示/隐藏”或编辑时可见。
   - 支持导入/导出环境配置，作为非常用操作放在更多菜单。

7. 原 `showEnvManager()` 可保留作为兼容入口，但应改成跳转到“环境配置”tab：

```go
func (a *App) showEnvManager() {
    if a.moduleTabs != nil {
        a.moduleTabs.Select(envTab)
    }
}
```

或者删除弹窗入口，避免两个环境管理界面长期分叉。

### 验收标准

- 顶部 tab 中存在“环境配置”。
- 用户不进入运行页也能查看、修改环境变量。
- 修改环境变量后，全局环境下拉和运行逻辑使用最新环境。
- 敏感变量不会在列表中明文暴露。

## P0-4. 运行页改成“运行详情”，只保留非常用功能和详情

### 问题

运行页现在同时承载常用控制和运行详情。全局操作栏实现后，运行页继续保留重复按钮会造成两个入口并存，增加割裂感。

### 具体方案

1. tab 名称建议从 `运行` 改成：

```text
运行详情
```

2. `internal/ui/run_panel.go` 中移除常用控制区域：
   - 运行整个流程按钮
   - 单步执行按钮
   - 停止按钮
   - 当前环境下拉
   - 启动浏览器入口

3. 运行详情页保留：
   - 运行摘要
   - 总进度
   - 当前步骤
   - 日志
   - 失败截图和 HTML 路径
   - 清空日志
   - 打开产物目录
   - 复制日志
   - 非常用操作：关闭托管 Chrome、浏览器下载配置跳转

4. 如果保留更多菜单，菜单内不应重复全局工具栏已有的“运行/停止/单步/环境选择”。

### 验收标准

- 常用运行操作只在全局操作栏出现。
- 运行详情页聚焦日志、摘要和产物。
- 切换到任意页面时，运行状态不会丢失，仍可从全局操作栏停止。

## P1-1. 全局布局与视觉层级优化

### 问题

当前 UI 大量使用默认 `VBox/HBox/Border`，功能能用但视觉层级弱，页面像控件堆叠，不够美观。

### 具体方案

1. 建立统一尺寸规范：
   - 主窗口默认：`1400 x 900` 可保留。
   - 左侧流程库宽度：`280-340px`。
   - 右侧属性区宽度：`360-460px`。
   - 底部/详情日志区最小高度：`220px`。
2. 不要把大区域做成嵌套卡片。页面大区域用 split 和 border，重复项/弹窗/模板项可用轻量 card。
3. 页面标题使用一致样式：
   - 主模块标题 16-18px / bold。
   - 区块标题 14-15px / bold。
   - 表格和列表正文 13-14px。
4. 统一按钮层级：
   - 主操作：运行、保存、创建，使用 HighImportance 或主色。
   - 次操作：复制、导入、导出，普通按钮。
   - 危险操作：删除、停止、关闭 Chrome，使用 DangerImportance 或确认弹窗。
5. 避免按钮一行过多。常用按钮最多 5-6 个，其他放 `更多` 菜单。
6. 所有长文本必须截断或换行：
   - 流程名
   - XPath
   - 输入模板
   - 日志路径
   - 截图/HTML 路径
7. 已有 `internal/textutil` 和 `newTruncatingLabel` 可复用，避免每个地方手写 `name[:N]`。

### 验收标准

- 窗口缩小时文本不互相覆盖。
- 流程名、XPath、产物路径不会撑爆布局。
- 主操作和危险操作视觉层级明确。
- 页面在 1400x900 和 1280x720 下都可用。

## P1-2. 主题和颜色系统优化

### 当前问题

`internal/ui/theme.go` 只设置了主色和背景色，其余仍依赖默认主题，导致页面整体风格不统一。

### 具体方案

1. 扩展 `appTheme.Color()`：
   - Background
   - Foreground
   - Button
   - Disabled
   - Hover
   - Selection
   - InputBackground
2. 避免单一蓝色铺满页面。推荐主色蓝只用于主操作和选中态，状态色使用绿/黄/红/蓝。
3. 背景保持浅灰，输入区和列表保持白色或轻微分层。
4. 使用 8px 或更小的圆角，不要大圆角装饰。
5. 保持工具型应用的安静风格，不要做大面积渐变或装饰图形。

### 验收标准

- 应用不再像完全默认 Fyne 控件。
- 主色、状态色、背景色职责清晰。
- 深色/浅色主题如果暂不支持，至少浅色主题统一稳定。

## P1-3. 字体大小和密度调整

### 问题

部分文本过小或过密，部分区域按钮和标签大小不一致。字体变形修复后，还需要调整字号和控件密度。

### 具体方案

1. 在 `appTheme.Size()` 中统一调整：
   - `theme.SizeNameText`
   - `theme.SizeNameCaptionText`
   - `theme.SizeNameHeadingText`
   - `theme.SizeNamePadding`
   - `theme.SizeNameInlineIcon`
2. 建议：
   - 正文 13-14
   - 标题 16
   - 日志 12-13 等宽
   - 表格行高不要太矮
3. 不要使用随窗口宽度动态缩放的字体。
4. 中文按钮文字要留足宽度，避免按钮内文字挤压。

### 验收标准

- 中文按钮文字完整显示。
- 表格和列表可扫描，不显得过密。
- 日志区域信息密度高但可读。

## P1-4. Tab 信息架构调整

### 建议 tab

```text
流程
步骤
环境配置
历史
设置
运行详情
```

### 说明

- `流程`：流程库 + 流程属性，负责选择和基本信息。
- `步骤`：步骤表格 + 步骤属性，负责编排。
- `环境配置`：环境列表 + 环境变量表。
- `历史`：历史运行记录和筛选。
- `设置`：Chrome 下载、路径、保留策略、主题等。
- `运行详情`：日志、摘要、截图、HTML、产物。

### 验收标准

- “环境配置”不是弹窗主入口，而是一级 tab。
- “运行详情”不再承担全局运行入口。
- 用户完成主要链路时不需要频繁来回跳 tab 找按钮。

## P1-5. 需要同步更新的文档

实现 UI 优化后，需要同步更新：

- `README.md`
- `USER_GUIDE.md`
- `FAQ.md` 如涉及环境配置或字体说明
- `problem.md` 中已完成的问题状态

尤其是 `USER_GUIDE.md` 当前仍可能描述旧工具栏，应改成全局操作栏和新 tab 架构。

## 推荐实施顺序

1. 字体资源与主题基础改造：先解决字体变形和默认观感。
2. 新增全局操作栏：让流程、环境、运行、停止任意页面可见。
3. 增加“环境配置”tab：把现有弹窗逻辑迁移为主页面。
4. 瘦身运行页并改名“运行详情”。
5. 做布局细节和字号/间距统一。
6. 更新文档和手工测试清单。

## 手工验收清单

- Windows 下启动程序，确认字体不变形、不缺字。
- 任意 tab 中都能看到并使用：流程选择、保存、启动浏览器、运行、单步、停止、环境选择。
- “环境配置”tab 可以新建环境、设置当前环境、新增/编辑/删除变量。
- 敏感变量在列表中不明文展示。
- “运行详情”页没有重复的运行/停止/环境选择主控件。
- 运行中切换到“流程”“步骤”“历史”“设置”任意页面，仍能从全局操作栏停止。
- 1280x720、1400x900 两种窗口尺寸下，按钮文字不截断，长路径不撑爆布局。
- `go test ./...` 通过。

## 注意事项

- 不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制或运行产物。
- 字体文件必须确认许可；不能使用来源不明的字体。
- 不要让应用启动时自动联网下载模板、字体或配置。
- UI 改造过程中要保持现有流程运行能力，不要破坏 Chrome 149 的 CDP 修复链路。
