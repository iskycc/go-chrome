# UI 体验优化设计文档

## 背景与目标

当前 `go-chrome` 的 Fyne UI 已经覆盖流程编排、运行、环境、历史、设置等核心能力，但存在以下使用体验问题：

- 默认字体在 Windows 中文环境下部分文本变形。
- 运行相关操作集中在“运行”tab，切换页面后无法快速控制。
- 环境管理入口隐藏在运行页“更多”菜单中，用户难以发现。
- 页面布局偏默认控件堆叠，信息层级弱。
- 主题仅覆盖主色和背景，整体风格不统一。

本次优化目标：在不改变核心运行逻辑的前提下，把应用从“功能可用”提升到“信息结构清晰、常用操作固定可见、视觉稳定”的状态。

## 设计原则

1. **信息结构优先**：常用操作全局可见，页面内只保留本页专属内容。
2. **安静工作台式风格**：减少装饰，使用稳定布局、一致字号和合理留白。
3. **保留 Fyne 原生体验**：通过主题、字体、间距、分组提升专业度，不做复杂动画。
4. **不引入运行期网络依赖**：字体、模板、配置均打包或本地生成。
5. **最小侵入**：不改动流程模型、CDP 修复链路、Chrome 启动参数等核心逻辑。

## 总体架构

主窗口布局从

```text
顶部状态栏
Tab: 流程 / 步骤 / 运行 / 历史 / 设置
```

调整为

```text
顶部状态栏
全局操作栏：流程选择 | 保存 | 启动浏览器 | 运行 | 单步/下一步 | 停止 | 环境选择 | 进度
Tab: 流程 / 步骤 / 环境配置 / 历史 / 设置 / 运行详情
```

全局操作栏固定在最上方，保证用户在任何 tab 都能启动、停止、切换流程和环境。

## 变更模块

### 1. 字体与主题基础（P0-1、P1-2、P1-3）

#### 字体

- 资源目录：`assets/fonts/`
- 字体文件：`assets/fonts/CascadiaCode-SemiLight.ttf`
- 许可文件：`assets/fonts/LICENSE-CascadiaCode.txt`（OFL 1.1）
- `assets/embed.go` 增加字体嵌入，提供 `assets.CascadiaCodeSemiLight() fyne.Resource`。
- `internal/ui/theme.go` 的 `Font()` 在资源可用时返回内置字体，否则回退默认主题字体。
- 日志、XPath/目标输入、模板输入、环境变量 Key 等场景优先使用等宽字体；普通中文标题和按钮如果显示不佳可保留默认字体。

#### 主题扩展

扩展 `appTheme.Color()` 覆盖：

- Background
- Foreground
- Button
- Disabled
- Hover
- Selection
- InputBackground
- ScrollBar

扩展 `appTheme.Size()` 覆盖：

- `theme.SizeNameText`：14
- `theme.SizeNameCaptionText`：12
- `theme.SizeNameHeadingText`：16
- `theme.SizeNamePadding`：8
- `theme.SizeNameInlineIcon`：18

颜色保持浅色安静风格：

- 主色蓝仅用于主操作和选中态。
- 状态色：绿/黄/红/蓝职责清晰。
- 背景浅灰，输入区和列表轻微分层。
- 圆角 8px 或更小。

### 2. 全局操作栏（P0-2）

新增 `internal/ui/global_toolbar.go`。

组件结构：

```go
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
}
```

行为：

- 流程选择：显示流程名，内部维护 flow ID → name 映射；切换时复用 `App.onFlowSelected()`，有未保存修改时走 `promptSaveBefore()`。
- 保存：调用 `App.saveCurrentFlow()`。
- 启动浏览器：调用 `App.startBrowser()`。
- 运行：调用 `App.runCurrentFlow()`。
- 单步/下一步：调用 `App.onStepButton()`。
- 停止：调用 `App.stopCurrentRun()`。
- 环境选择：复用原 `runPanel.refreshEnvironments()` 逻辑；变更后调用 `envRepo.SetActive()`。
- 进度：显示轻量进度文本，如“第 2/6 步 · 输入用户名”。

状态同步：

- 新建/导入/删除流程后刷新流程下拉。
- 环境配置 tab 修改环境后刷新环境下拉。
- 运行状态变化时更新按钮可用态和进度文本。
- 单步按钮在 `onStepButton()` 中根据 `stepRunner` 状态切换文本。

### 3. 环境配置 Tab（P0-3）

新增 `internal/ui/env_panel.go`，将 `internal/ui/env_dialog.go` 中的环境管理逻辑迁移为主页面组件。

组件结构：

```go
type envPanel struct {
    app    *App
    widget fyne.CanvasObject
    // 内部包含环境列表、变量表格、操作按钮
}
```

页面布局：

```text
┌─────────────────────┬────────────────────────────────────┐
│ 环境列表             │ 环境变量表格                        │
│ 搜索/新建/复制/删除  │ KEY | VALUE | 敏感 | 说明 | 操作      │
│ 默认环境 [当前]      │ 新增变量 编辑变量 删除变量            │
└─────────────────────┴────────────────────────────────────┘
```

功能：

- 左侧环境列表支持搜索、新建、重命名/说明、复制、删除、设为当前。
- 右侧变量表格支持新增、编辑、删除。
- 敏感变量在列表中显示 `******`，编辑弹窗中可查看和修改真实值。
- 导入/导出环境配置作为“更多”菜单中的非常用操作。

原 `showEnvManager()` 改为跳转到环境配置 tab，保留兼容性但不再作为唯一入口。

`App` 结构体新增 `envPanel` 和 `globalToolbar` 字段。

### 4. 运行页改名为“运行详情”（P0-4）

- `runPanel` 移除常用控制区域：运行整个流程、单步执行、停止、当前环境下拉、启动浏览器入口。
- 保留运行摘要、总进度、当前步骤、日志、失败截图/HTML 路径、清空日志、打开产物目录、复制日志、关闭托管 Chrome（更多菜单）、浏览器下载配置跳转。
- tab 名称从“运行”改为“运行详情”。
- 更多菜单内不再重复全局工具栏已有的运行/停止/单步/环境选择。

### 5. Tab 信息架构调整（P1-4）

最终 tab 顺序：

```text
流程 / 步骤 / 环境配置 / 历史 / 设置 / 运行详情
```

职责：

- 流程：流程库 + 流程属性，负责选择和基本信息。
- 步骤：步骤表格 + 步骤属性，负责编排。
- 环境配置：环境列表 + 环境变量表。
- 历史：运行记录和筛选。
- 设置：Chrome 下载、路径、保留策略等。
- 运行详情：日志、摘要、截图、HTML、产物。

### 6. 全局布局与视觉层级优化（P1-1）

- 主窗口默认 1400x900 可保留。
- 流程库左侧宽度 280-340px；步骤属性右侧 360-460px；底部/详情日志区最小高度 220px。
- 页面大区域使用 split 和 border；重复项/弹窗/模板项使用轻量 card。
- 标题层级：主模块 16px bold，区块 14-15px bold，表格正文 13-14px。
- 按钮层级：
  - 主操作（运行、保存、创建）：HighImportance。
  - 次操作（复制、导入、导出）：普通按钮。
  - 危险操作（删除、停止、关闭 Chrome）：DangerImportance 或确认弹窗。
- 一行按钮不超过 5-6 个，其余放入“更多”菜单。
- 所有长文本使用 `newTruncatingLabel` 截断或换行：流程名、XPath、输入模板、日志路径、截图/HTML 路径。

## 接口变更

### App 结构体

```go
type App struct {
    // ... 已有字段 ...
    envPanel      *envPanel
    globalToolbar *globalToolbar
    // stepBtn 从 App 移除，迁移到 globalToolbar
}
```

### 新增公开方法

- `assets.CascadiaCodeSemiLight() fyne.Resource`
- `newGlobalToolbar(app *App) *globalToolbar`
- `newEnvPanel(app *App) *envPanel`

### 修改方法

- `App.buildUI()`：增加全局工具栏和环境配置 tab，调整 tab 顺序。
- `App.currentEnvProvider()`：从 `runPanel.envSelect.Selected` 改为 `globalToolbar.envSelect.Selected`。
- `App.onStepButton()`：操作 `globalToolbar.stepBtn` 而不是 `App.stepBtn`。
- `App.stopCurrentRun()`：操作 `globalToolbar.stepBtn` 和 `globalToolbar.stopBtn`。
- `App.handleRunnerEvents()`：更新 `globalToolbar` 的进度和按钮状态。
- `App.startBrowser()` / `App.closeManagedChrome()`：日志输出仍使用 `runPanel.log()`，不改动 Chrome 逻辑。
- `App.refreshFlowList()`：同步刷新全局工具栏的流程下拉。
- `runPanel`：移除控制按钮，保留详情展示。
- `env_dialog.go`：`showEnvManager()` 改为跳转到环境配置 tab。

## 数据流

1. 启动：`buildUI()` 初始化 `globalToolbar`、`envPanel`，并同步一次流程/环境列表。
2. 流程切换：`globalToolbar.flowSelect.OnChanged` → `App.onFlowSelected()` → `setCurrentFlow()` → 刷新编辑器、步骤表、历史、运行摘要。
3. 环境切换：`globalToolbar.envSelect.OnChanged` → `envRepo.SetActive()`。
4. 运行：`globalToolbar.runBtn` → `App.runCurrentFlow()` → runner 事件 → `globalToolbar` 更新进度/按钮，`runPanel` 更新日志/摘要/产物。
5. 环境配置修改：`envPanel` 操作 `envRepo` → 刷新 `globalToolbar` 环境下拉和历史页环境筛选。

## 错误处理

- 字体文件缺失时优雅回退到默认主题，不阻塞启动。
- 流程/环境列表为空时，下拉保留占位项并禁用相关操作。
- 未选择流程时点击运行/保存给出提示。
- 环境配置 tab 中删除当前环境后自动激活剩余环境或创建默认环境。

## 测试策略

- 单元测试：
  - `internal/textutil` 已有截断测试，保持不变。
  - 主题 `Size()` / `Color()` 返回值符合预期。
- 构建测试：
  - `go build ./...` 通过。
  - `go test ./internal/browser ./internal/runner ./internal/config ./internal/flow ./internal/template` 通过。
- 手工验收（Windows）：
  - 字体不变形、不缺字。
  - 任意 tab 可见并使用全局操作栏。
  - 环境配置 tab 可管理环境和变量。
  - 敏感变量不明文展示。
  - 运行详情页没有重复控件。
  - 运行中切换页面仍可停止。
  - 1280x720 和 1400x900 下布局可用。

## 依赖与约束

- 不新增运行时网络依赖。
- 字体使用 Cascadia Code（OFL 1.1），仓库保留许可证文件。
- 不提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制或运行产物。
- 保持 Chrome 149 CDP 修复链路和 `--incognito` 等启动参数不变。

## 实施顺序

1. 字体资源与主题基础改造。
2. 新增全局操作栏。
3. 新增“环境配置”tab。
4. 瘦身运行页并改名“运行详情”。
5. 统一布局、字号、间距、按钮层级。
6. 更新 `README.md`、`USER_GUIDE.md`、`FAQ.md`、`problem.md`。
7. 构建测试与手工验收。
