# 已知问题与未解决清单

本文档记录 go-chrome 项目当前的**已修复**、**环境相关**和**待优化**问题。

## 运行环境

- Windows 10/11 + Go 1.26.0 + MinGW64
- Chrome for Testing 149.0.7827.55（`C:\Users\admin\Downloads\main\go-chrome\chrome\chrome-win64\chrome.exe`）
- 测试服务器：`http://localhost:18080`（运行中）

## 已修复的问题

| # | 问题 | 根因 | 修复 |
|---|------|------|------|
| 1 | `flow_library.go:172` 空指针 | 初始化顺序错误，`list` 在 `search.OnChanged`/`tagFilter` 之后才创建 | 调换初始化顺序，先创建 `list` 再绑定回调 |
| 2 | `manager.go:200` `os.Rename` Access Denied | Windows 下 `os.Rename` 不能覆盖已存在目录（即使是空目录） | rename 前 `os.RemoveAll(m.cfg.InstallDir)` |
| 3 | 新建/导入流程弹窗输入框过小 | `dialog.ShowForm` 默认尺寸，`dialog.ShowFileOpen` 不可缩放 | 改用 `dialog.NewCustomConfirm` + `Resize(440,160)`，导入用 `dialog.NewFileOpen` + `.json` 过滤 + `Resize(680,480)` |
| 4 | 添加步骤对话框过小 | 缺少 `Resize` | `showAddStepDialog` 加 `Resize(420,180)` |
| 5 | 流程编辑缺少 storage 导入 | `main_window.go` 没有 `doImportFlow` | 新增 `doImportFlow` 处理 JSON 文件选择和导入 |
| 6 | `validate.go` 校验过弱 | 仅检查步骤非空 | 加入必填 URL、OnError/InputMode/TargetStrategy 白名单、空步骤、非负等待、流程 ID/名称必填等规则 |
| 7 | `NewRunner(nil, ...)` / `NewStepRunner(nil, ...)` panic | 没有 nil config guard | 加默认值：`DefaultTimeoutMs=10000`、`DefaultWaitAfterMs=500`、`MaskInputValueInLogs=true` |
| 8 | `os.MkdirAll` 错误被吞 | `step_runner.go:Init()` 不检查错误 | 显式返回 |
| 9 | 示例流程密码被当作模板解析 | `InputTemplate` 模式 + `"Password123"` 触发模板解析 | 改为 `InputLiteral` |
| 10 | `runID` 太长且不易读 | 包含毫秒精度 | 改为 `20060102-150405` + UUID[:8] 后缀 |
| 11 | `manager.go:200` 同 #2 的关联 | 即使空目录也拒绝 | 同 #2 |
| 12 | `build.bat` 未设置 MinGW PATH | CGO 编译需要 gcc | 开头加 `set "MINGW_BIN=...mingw64\bin"`，PATH 前置 |
| 13 | `cdp.go` `err` 变量作用域错误 | `cdpErr` 在 if 内声明但循环外使用 | 引入 `lastErr` 变量 |
| 14 | CDP 连接重试过短 | 单次失败就放弃 | 改为 8 次重试，递增延迟（1s, 2s, 3s, ..., 7s） |
| 15 | `-32000` 错误也走完整重试浪费时间 | 不区分可恢复错误 | 检测到 `-32000`（no browser open）立即快速失败 |
| 16 | **Chrome 149 下 `Target.createTarget` 失败导致 CDP 连接阻塞** | `chromedp.NewRemoteAllocator` 的远程上下文默认会创建新 tab，触发环境中失败的 `Target.createTarget` 路径 | 启动参数追加 `about:blank` 保证存在 page target；`browser.Connect()` 先通过 `/json/list` 选择已有 page，并用 `chromedp.WithTargetID` 绑定，避免默认创建新 tab；无 page 时才用 HTTP `/json/new` 兜底 |
| 17 | **GUI 点击运行后"无操作"且日志超时**（user-reported） | 流程从 `ListSorted()` 加载时**不包含 steps**（DB 优化设计，避免 N+1 查询），`flow_library.go` 的 `OnSelected` 把不含 steps 的 flow 传给 runner，导致 runner 立即 `StatusFailed` | `flow_library.go:54` 在 `OnSelected` 时改用 `flowStore.Load(id)` 重新加载完整流程（含 steps） |
| 18 | **DevToolsActivePort 永远找不到**（user-reported） | `app-config.json` 的 `installDir` / `userDataDir` 是相对路径（如 `./chrome`、`./data/chrome-profile`），Chrome 解析 `--user-data-dir` 时使用 **Chrome.exe 自己的 CWD**（`chrome/chrome-win64/`），结果 `DevToolsActivePort` 写到了 `chrome/chrome-win64/data/replay-tmp/...`，而 Go 代码在 `data/replay-tmp/...` 等，30s 超时 | `config.go` 新增 `ResolvePaths(baseDir)` 方法；`cmd/go-chrome/main.go` 启动时用 `app.ExecutableDir()` 作为 base 把所有相对路径转成绝对路径 |
| 19 | **运行面板缺少一键关闭本程序启动的 Chrome 按钮**（user-reported） | 用户在外部已经打开了 Chrome 时，关闭按钮误伤用户自己的 Chrome | 新增 `closeManagedChrome()`：只 kill `Manager` 跟踪的 pid（用 `taskkill /F /T /PID` 杀进程树），不影响用户进程；按钮在 `ChromeRunning/ChromeStarting` 时启用，否则禁用；点击先弹确认框 |
| 20 | **顶部状态栏缺少字段名** | 状态只有 `未修改`、`已安装`、`已完成`，用户不知道是哪个对象的状态 | `status_bar.go` 重写为 `statusItem` 结构：固定字段名（`当前流程：`、`保存状态：`、`Chrome：`、`运行状态：`）+ 动态值 + 动态颜色圆点；状态切换时圆点颜色同步变化 |
| 21 | **启动后不恢复上次流程** | recent flow IDs 已存盘但启动时未使用 | `buildUI()` 末尾调用 `restoreLastFlowSelection()`：先尝试最近 ID，没有则选第一个，都没有则保持空状态；`selectFlow` 改返回 `bool` |
| 22 | **未保存修改弹窗语义错误** | `dialog.ShowConfirm` 的"取消"被理解为"放弃修改并继续"，且 `saveCurrentFlow` 失败后仍继续 `next()` | `promptSaveBefore` 改三选一弹窗：`保存并继续` / `放弃修改` / `取消`；`saveCurrentFlow` 拆为 UI 入口和 `saveCurrentFlowInternal() error`，保存失败时**不**继续 next |
| 23 | **保存成功状态被立即覆盖** | `setSave(SaveSuccess)` 后立即 `markClean()` 改回 `未修改` | `setSave(SaveSuccess)` 启动 2s 定时器，2s 后自动转为 `SaveUnmodified`；期间任何状态切换会取消定时器 |
| 24 | **没有"从模板创建"功能** | 只能导入本地 JSON 或写死导入示例 | 新增 `internal/flow/templates.go` 内置 4 个模板（登录测试、空白、表单填写、文本断言）；UI 增加"从模板创建"对话框和入口；空状态按钮改为"新建空白流程" + "从模板创建" |
| 25 | **单步执行时停止按钮无效** | `Runner.Stop` 只能停止完整运行；`StepRunner` 没有 `Stop()` | `StepRunner` 新增 `Stop()`：设 `stopped` 标志 + 关闭 CDP 取消 action executor context；`App.stopCurrentRun()` 区分处理 Runner 和 StepRunner；停止后按钮恢复"单步执行" |
| 26 | **Chrome 状态显示不准** | `Status()` 只看 `cfg.UserDataDir` 根目录的 DevToolsActivePort，重放 Chrome 状态永远不对 | `Manager` 新增 `activeUserDataDir` 字段，`Start`/`StartReplay` 设置，`Stop` 清空；`Status()` 优先读 active dir；新增单元测试 `TestManagerActiveUserDataDirTrackedAndCleared` |
| 27 | **退出时不关闭托管 Chrome** | `SetOnClosed` 只保存窗口大小和 recent flows | `SetOnClosed` 增加：先停 runner/stepRunner；如 `cfg.App.CloseManagedChromeOnExit=true` 则 `browserMgr.Stop()`；只杀 Manager 跟踪的 pid，不影响用户 Chrome |
| 28 | **步骤页面字数超过显示框就乱码**（user-reported） | `truncate` 用 `s[:max-3]` 按**字节**切片；中文 UTF-8 一个字符 3 字节，切到字符中间就产生无效 UTF-8 → Fyne 显示成"豆腐块" | 抽出 `internal/textutil` 包，新 `Truncate` 按 rune 计数，永远返回合法 UTF-8；`flow_library.go` 同样的 byte 截断也修复；新增 11 个单元测试 + 属性测试 |
| 29 | **截断后还是超出列框宽度**（user-reported） | Fyne 默认 `Label` 不会自动 ellipsis，超出列宽的字符会画到相邻列或被窗口裁掉；手算的字符上限（14/24）依赖字体/DPI 不准 | 抽出 `newTruncatingLabel()` helper（设 `Label.Truncation = fyne.TextTruncateEllipsis`，Fyne 2.4+）；`step_table.go` 表头模板、`flow_library.go` 列表模板、`run_panel.go` 的 `currentStep`/artifact 路径、`status_bar.go` 状态值都换成 truncating label；删掉手算的 `truncate(s, N)` 调用，让 Fyne 按实际宽度切 |
| 30 | **顶栏状态全部显示为 `...`**（user-reported，#29 副作用） | Fyne 2.7.4 `widget/richtext.go` 的 `textRenderer.MinSize`：当 `Truncation != Off` 且不在 scroll 容器里时，min width = `charMinSize`（1 个字符）。HBox 里其他 widget 竞争空间时，truncating label 被压到 1 字符宽就显示 `...` | `status_bar.go` 的 value label 包一层 `container.NewGridWrap(fixedWidth, label)` 给一个稳定的像素预算（flow=180, save=110, chrome=110, run=160），里面再用 `Truncation.Ellipsis`；`setValue` 加一个 `truncate(text, 200)` 安全网。`step_table.go` / `flow_library.go` / `run_panel.go` 不受影响（Table/List cell 宽度是确定的） |

## 待优化 / 环境相关

### 1. PowerShell 5.1 转义（无关功能）

- `&&` 在 PowerShell 5.1 上不被识别
- 解决方案：用 `cmd /c` 包装，或者 PowerShell 用 `; if ($?) { ... }`
- 已在测试命令中改用 `cmd /c "..."` 包装

### 2. Chrome 启动日志噪音（无关功能）

- Chrome 启动时输出大量 `ERROR:...` 日志（网络服务重启、Crashpad 注册失败等）
- 在 Sandbox 关闭 + 中文 Windows 10/11 上是已知问题，不影响功能
- 长期方案：把这些 stderr 重定向到日志文件，不在 stdout 输出

## 待修复 UI / 产品问题（2026-06-12 新增）

本节记录当前人工体验检查发现的问题，优先面向后续 Agent 实现。以下问题不属于 Chrome 149 / CDP 连接问题，而是 GUI 可理解性、流程恢复、模板能力和运行控制的产品缺口。

### P0-1. 顶部状态栏缺少字段名，用户看不懂状态含义

**用户反馈**：

- 程序顶部显示 `未修改`、`已安装`、`已完成` 等状态，但没有标注这些状态分别是什么。
- 用户无法判断 `未修改` 是流程保存状态，`已安装` 是 Chrome 状态，还是运行状态。

**当前实现位置**：

- `internal/ui/status_bar.go`
- `newStatusBar()` 里直接把 4 个状态值放进 `container.NewHBox`：
  - `flowLabel = "未选择流程"`
  - `saveLabel = "未修改"`
  - `chromeLabel = "未安装"`
  - `runLabel = "空闲"`
- 当前 UI 只有状态圆点 + 状态值，没有 `当前流程：`、`保存状态：`、`Chrome：`、`运行状态：` 这类 label。
- `statusColorBlue/Green/Yellow/Red` 已定义但没有用于状态切换，圆点始终是灰色，状态颜色信息没有生效。

**建议方案**：

1. 将顶部状态栏改为明确的键值展示：
   - `当前流程：未选择流程`
   - `保存状态：未修改`
   - `Chrome：已安装`
   - `运行状态：空闲`
2. 每个状态项使用独立组件，至少包含：
   - 固定字段名 label
   - 动态状态值 label
   - 动态颜色圆点或状态 chip
3. 状态颜色规则：
   - 保存状态：
     - `未修改` / `已保存`：绿色或灰色
     - `有未保存修改`：黄色
     - `保存中`：蓝色
     - `保存失败`：红色
   - Chrome 状态：
     - `未安装`：黄色或灰色
     - `已安装` / `已启动`：绿色
     - `下载中` / `启动中`：蓝色
     - `启动失败`：红色
   - 运行状态：
     - `空闲`：灰色
     - `运行中`：蓝色
     - `已完成`：绿色
     - `失败`：红色
4. 字段名不要随状态变化，动态变化只发生在状态值和颜色上。
5. 对状态项增加 tooltip（如果 Fyne 当前控件实现方便）：
   - 保存状态：说明当前流程是否有未保存修改
   - Chrome：说明本地 Chrome 安装/托管启动状态
   - 运行状态：说明当前流程执行状态

**验收标准**：

- 用户只看顶部就能知道每个状态对应的对象。
- `未修改` 前必须有 `保存状态：` 或等价文案。
- `已安装` 前必须有 `Chrome：` 或等价文案。
- `已完成` 前必须有 `运行状态：` 或等价文案。
- 状态变化时颜色同步变化，不再所有圆点固定灰色。

### P0-2. 程序启动后不恢复上次选择的流程，也不默认打开流程

**用户反馈**：

- 程序打开后不会记录/恢复上次选择的流程。
- 如果已有流程，也不会默认打开一个流程，用户需要手动重新选择。

**当前实现位置**：

- `internal/ui/main_window.go`
  - `initDeps()` 中通过 `recentRepo.Load()` 读取了 recent flow IDs。
  - `setCurrentFlow()` 中会 `recentStore.Touch(f.ID)`，说明最近流程链路已经有一半实现。
  - `SetOnClosed()` 中会 `recentRepo.Save(recentStore.FlowIDs)`，退出时保存 recent flow IDs。
  - `buildUI()` 末尾只调用：
    - `refreshFlowList()`
    - `runPanel.refreshEnvironments()`
    - `historyPanel.refreshFilters()`
  - 但没有根据 recent flow IDs 自动 `selectFlow()`。
- `internal/ui/flow_library.go`
  - 已有 `selectFlow(id string)` 方法，可复用。

**根因**：

- recent flow 已保存，但启动后没有恢复选择。
- `ListSorted()` 返回的流程列表和 recent IDs 没有在 UI 初始化后建立选择关系。
- 没有 fallback：recent flow 不存在时，应自动选择第一个可用流程。

**建议方案**：

1. 在 `buildUI()` 完成 `refreshFlowList()` 后调用 `restoreLastFlowSelection()`。
2. 新增方法建议：

```go
func (a *App) restoreLastFlowSelection() {
    if a.flowLibrary == nil || len(a.flowLibrary.flows) == 0 {
        a.setCurrentFlow(nil)
        return
    }
    if a.recentStore != nil {
        for _, id := range a.recentStore.FlowIDs {
            if a.flowLibrary.selectFlow(id) {
                return
            }
        }
    }
    a.flowLibrary.selectFlow(a.flowLibrary.flows[0].ID)
}
```

3. 将 `flowLibraryPanel.selectFlow(id string)` 改为返回 `bool`，表示是否找到并选中。
4. 注意 `widget.List.Select(i)` 会触发 `OnSelected`，而 `OnSelected` 已经会通过 `flowStore.Load(id)` 重新加载完整流程（含 steps），不要绕过这条路径。
5. 如果流程库为空，显示 empty state。
6. 新建流程、导入流程、导入示例后继续自动选中新流程，保持当前行为。

**验收标准**：

- 关闭程序前选中流程 A，重启后自动打开流程 A。
- 如果流程 A 已被删除，重启后自动打开列表中的第一个流程。
- 如果没有任何流程，仍显示“暂无流程”空状态。
- 自动恢复的流程必须包含 steps，运行时不能再出现空 steps 问题。

### P0-3. 没有“下载模板 / 模板库”功能，只能导入本地 JSON

**用户反馈**：

- 目前模板只能导入，没有下载模板功能。
- 用户期望可以从模板库选择并下载/创建流程。

**当前实现位置**：

- `internal/ui/flow_library.go`
  - “更多”菜单里只有 `导入流程`、`导出当前流程`、`复制当前流程`、`删除当前流程`。
- `internal/ui/onboarding.go`
  - 空状态中有 `导入示例`，但这是写死的 `flow.NewExampleLoginFlow()`，不是模板库。
- `cmd/import-example/main.go`
  - 只有命令行导入示例流程。
- `internal/flow/model.go`
  - 只有一个 `NewExampleLoginFlow()` 示例。

**约束**：

- 根据 `AGENTS.md`，运行期不要无条件依赖互联网。
- 不能在程序启动时自动联网拉模板。
- 远程模板下载必须由用户主动点击触发。
- 离线环境应至少可用内置模板。

**建议方案：分两阶段实现**

**阶段 1：内置模板库（优先实现，离线可用）**

1. 新增包或文件，例如：
   - `internal/flow/templates.go`
   - 或 `internal/template_catalog/catalog.go`（如果想与输入模板引擎区分）
2. 定义模板元数据：

```go
type FlowTemplate struct {
    ID          string
    Name        string
    Description string
    Tags        []string
    FlowFactory func() *flow.Flow
}
```

3. 提供 `ListBuiltinTemplates() []FlowTemplate`。
4. 首批至少内置：
   - 登录测试模板（现有 `NewExampleLoginFlow()`）
   - 空白流程模板
   - 表单填写模板（navigate + input + screenshot）
   - 断言页面文本模板（navigate + assert_text + screenshot）
5. UI 增加入口：
   - 流程库 “更多” 菜单新增 `从模板创建`
   - 空状态按钮 `导入示例` 改成 `从模板创建`
6. 新增模板选择弹窗：
   - 左侧/上方列表展示模板名、说明、标签
   - 右侧/下方展示步骤预览（步骤名称 + 类型）
   - 按钮：`创建流程`、`取消`
7. 创建时复制模板 flow，生成新 ID，不覆盖现有流程。
8. 创建成功后自动选中新流程并进入流程编辑页。

**阶段 2：远程模板下载（用户主动触发）**

1. 配置中新增可选模板源：

```json
{
  "app": {
    "templateCatalogURL": ""
  }
}
```

或单独放在 `templates` 配置段。

2. UI 在模板弹窗中提供 `刷新远程模板` 或 `从远程下载模板` 按钮。
3. 远程模板建议格式：

```json
{
  "version": 1,
  "templates": [
    {
      "id": "login-basic",
      "name": "登录测试",
      "description": "打开登录页，输入用户名密码并断言欢迎文案",
      "tags": ["登录", "示例"],
      "downloadURL": "https://example.com/templates/login-basic.json",
      "sha256": "..."
    }
  ]
}
```

4. 下载单个模板 JSON 后必须：
   - 校验 HTTP 状态码
   - 限制响应大小
   - 如提供 SHA256，必须校验
   - 调用 `flow.Validate()` 校验 schema
   - 重新生成 flow ID，避免覆盖
5. 远程下载失败时只提示错误，不影响内置模板和本地导入。
6. 不要在程序启动时自动刷新远程模板。

**验收标准**：

- 离线环境下用户可以从内置模板创建流程。
- 用户点击“从模板创建”后能看到模板列表和说明，不需要提前准备 JSON 文件。
- 创建出的流程自动保存、自动选中、可直接运行或编辑。
- 远程模板下载必须是用户主动触发，且失败不影响程序其他功能。

### P1-1. 未保存修改弹窗语义错误，有数据丢失风险

**当前实现位置**：

- `internal/ui/main_window.go`
- `promptSaveBefore(next func())` 使用 `dialog.ShowConfirm()`。

**问题**：

- 弹窗只有确认/取消两个按钮，但代码语义是：
  - 确认：保存后继续
  - 取消：直接 `markClean()` 并继续
- 用户通常会把“取消”理解为取消切换/取消操作，而不是“放弃修改并继续”。
- `saveCurrentFlow()` 没有返回 error/bool，保存失败后 `next()` 仍可能继续执行，导致用户以为保存了，但实际切换走了。

**建议方案**：

1. 将 `saveCurrentFlow()` 拆成：
   - `saveCurrentFlow()`：UI 入口
   - `saveCurrentFlowInternal() error`：实际保存逻辑
2. 将 `promptSaveBefore` 改成三选一弹窗：
   - `保存并继续`
   - `放弃修改`
   - `取消`
3. 行为：
   - 保存并继续：保存成功才执行 `next()`；保存失败停留当前流程并显示错误。
   - 放弃修改：重新加载当前流程或清除 dirty 后执行 `next()`。
   - 取消：不执行 `next()`，保持当前状态。

**验收标准**：

- 点击取消不会切换流程、不会新建、不会导入。
- 保存失败不会继续切换。
- 放弃修改的按钮文案必须明确包含“放弃”。

### P1-2. 保存成功状态会被立即覆盖，用户看不到“已保存”

**当前实现位置**：

- `internal/ui/main_window.go`
- `saveCurrentFlow()` 中先 `statusBar.setSave(SaveSuccess)`，随后调用 `markClean()`。
- `markClean()` 会设置 `SaveUnmodified`，导致 `已保存` 很快变成 `未修改`。

**建议方案**：

1. 区分“保存成功瞬时提示”和“干净状态”：
   - 保存成功后显示 `已保存`
   - 例如 2 秒后自动变为 `未修改`
2. 或者简化为保存成功后一直显示 `已保存`，下一次编辑时变为 `有未保存修改`。
3. 不要在同一个同步调用链里先设置 `已保存` 又立即设置 `未修改`。

**验收标准**：

- 用户点击保存后能看到明确的 `已保存` 状态。
- 再次编辑后状态变为 `有未保存修改`。

### P1-3. 单步执行时“停止”按钮不停止 StepRunner

**当前实现位置**：

- `internal/ui/run_panel.go`
  - 停止按钮只调用 `p.app.runner.Stop()`。
- `internal/ui/main_window.go`
  - 单步执行使用 `a.stepRunner`。

**问题**：

- 完整运行和单步运行使用不同 runner。
- 停止按钮只对完整运行有效。
- 单步执行过程中点击停止可能没有用户预期的效果。

**建议方案**：

1. 新增 `App.stopCurrentRun()`：
   - 如果 `runner.IsRunning()`，调用 `runner.Stop()`。
   - 如果 `stepRunner != nil` 且未完成，调用 `stepRunner.Close()`，置空，按钮文案恢复 `单步执行`，运行面板 `setRunning(false)`。
2. 停止按钮改为调用 `p.app.stopCurrentRun()`。
3. 如有正在执行的单步 CDP action，需要考虑 context cancellation。当前 `StepRunner.Close()` 如果只关闭 CDP，至少能释放资源；更完整方案是给 StepRunner 增加 cancel context。

**验收标准**：

- 单步执行初始化后点击停止，按钮恢复为 `单步执行`。
- 后续可以重新开始单步执行。
- 不遗留托管 Chrome 或 CDP session。

### P1-4. 运行重放后的 Chrome 状态可能显示不准

**当前实现位置**：

- `internal/browser/manager.go`
  - `StartReplay()` 使用 `filepath.Join(m.cfg.UserDataDir, "replay", runID)` 作为 profile。
  - `Status()` 只检查 `m.cfg.UserDataDir` 根目录下的 `DevToolsActivePort`。

**问题**：

- 重放 Chrome 已启动时，`Status()` 可能读不到 replay profile 的 DevToolsActivePort。
- UI 顶部 Chrome 状态可能显示 `启动中` 或不准确。

**建议方案**：

1. 在 `Manager` 中记录当前托管实例的 user data dir：

```go
type Manager struct {
    cfg *config.ChromeConfig
    manifest VersionManifest
    proc *os.Process
    activeUserDataDir string
}
```

2. `Start()` 成功后设置 `activeUserDataDir = m.cfg.UserDataDir`。
3. `StartReplay()` 成功后设置 `activeUserDataDir = userDataDir`。
4. `Stop()` 后清空 `activeUserDataDir`。
5. `Status()` 优先检查 `activeUserDataDir`，而不是固定根 `UserDataDir`。

**验收标准**：

- 完整运行/单步执行启动 Chrome 后，顶部 Chrome 状态能正确显示 `已启动`。
- 关闭托管 Chrome 后，状态回到 `已安装`。

### P1-5. 退出时没有按配置关闭托管 Chrome

**当前实现位置**：

- `internal/config/config.go`
  - `AppConfig.CloseManagedChromeOnExit` 已存在，默认 true。
- `internal/ui/main_window.go`
  - `SetOnClosed()` 只保存窗口大小、recent flows，并关闭 ticker channel。
  - 没有根据 `CloseManagedChromeOnExit` 调用 `browserMgr.Stop()`。

**建议方案**：

1. 在 `SetOnClosed()` 中增加：

```go
if a.cfg.App.CloseManagedChromeOnExit && a.browserMgr != nil {
    _ = a.browserMgr.Stop()
}
```

2. 如果 runner 正在运行，先 `runner.Stop()`。
3. 如果 stepRunner 存在，先 `stepRunner.Close()`。
4. 注意关闭顺序：停止 runner / stepRunner -> Stop Chrome -> 关闭 ticker。

**验收标准**：

- 配置为 true 时，退出程序会关闭本程序启动的 Chrome。
- 配置为 false 时，退出程序不关闭 Chrome。
- 不影响用户手动打开的 Chrome。

## 推荐实施顺序

1. **先做 P0-1 顶部状态栏可读性**：这是用户第一眼可见问题，改动集中，风险低。
2. **再做 P0-2 恢复最近流程/默认流程**：直接改善启动体验，并复用现有 recent store。
3. **同时修 P1-1 / P1-2 保存状态和未保存弹窗**：避免做默认打开流程后引入误切换数据丢失。
4. **做 P0-3 内置模板库第一阶段**：先满足离线可用和“从模板创建”，远程下载可放第二阶段。
5. **最后做 P1-3 / P1-4 / P1-5 运行控制修正**：完善运行体验和 Chrome 生命周期状态。

## 建议测试清单

- `go test ./internal/browser ./internal/runner ./internal/config ./internal/flow ./internal/template`
- `go test ./...`
- 如修改 Chrome 生命周期：`go test -tags=integration ./internal/runner`
- 手工 GUI 验证：
  - 顶部状态栏字段名和颜色是否清楚。
  - 选择流程 A，退出，重启后是否自动打开 A。
  - 删除最近流程后重启，是否自动打开第一个流程。
  - 无流程时是否显示空状态。
  - 编辑流程后切换流程，三选一弹窗是否符合语义。
  - 保存失败时是否不会继续切换。
  - 从内置模板创建流程后是否自动选中并可保存/运行。
  - 单步执行时停止按钮是否能恢复状态。
  - 退出程序是否按配置关闭托管 Chrome。

## 测试结果汇总（2026-06-12）

### 全部通过（PASS，60+ 测试，0 SKIP，0 FAIL）

```
ok  go-chrome/internal/browser  0.391s   (4 new tests for target discovery)
ok  go-chrome/internal/runner   ~30s    (6 Chrome E2E PASS after fix)
ok  go-chrome/internal/flow     0.051s
ok  go-chrome/internal/template 0.083s
ok  go-chrome/internal/config   0.072s
ok  go-chrome/internal/db       1.439s
```

### Chrome E2E 关键测试（全部通过）

- `TestIntegration_LoginFlowComplete` (4.8s)
- `TestIntegration_ExampleLoginFlowViaRunFlow` (4.8s)
- `TestIntegration_StepRunnerInitAndNextStepByStep` (4.8s)
- `TestIntegration_MaskInputInLogsHonored` (3.7s)
- `TestIntegration_RetryOnFailure` (4.2s)
- `TestIntegration_CDPConnectionRoundTrip` (1.3s)

## 下一步建议

1. **完成 `official_fixed_version` 功能**：AGENTS.md 中提到这是预留配置，尚未实现
2. **GUI 回归测试**：手动启动 `go-chrome.exe`，验证弹窗 UX 改进（新建/导入流程、添加步骤）
3. **清理远程分支的运行产物**（`stdout*.txt`、`stderr*.txt` 是历史产物，可考虑删除）
4. **CI 集成**：把 `go test -tags=integration ./...` 加到 CI pipeline，确保 Chrome 149 兼容性不退化

## 留给下一个 Agent

- 当前所有 6 个 Chrome E2E 测试通过，代码规避已生效。
- 核心改动：
  - `internal/browser/launcher.go` 启动参数追加 `about:blank`
  - `internal/browser/cdp.go` 新增 `ensurePageTarget()` 函数，优先从 `/json/list` 选择已有 page target 并用 `chromedp.WithTargetID` 绑定
  - 新增 `internal/browser/cdp_test.go` 单元测试
- 如未来 Chrome 升级到 150+ 仍出问题，先确认 `/json/list` 是否能正常返回 page target。如果能，那只是启动参数问题；如果不能，则需要排查更底层的 Chrome sandbox / 网络服务问题。
- 注意不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制、`stdout*.txt`、`stderr*.txt` 等运行产物。

## UI 优化完成项（2026-06-12 UI 优化）

- [x] UI 页面不够美观、字体变形
- [x] 常用运行操作必须切到运行页
- [x] 环境管理入口不明显

## UI 二次优化完成项（2026-06-12 UI 二次优化）

- [x] 中英文字体不统一（替换为 Cascadia Next SC，覆盖 CJK）
- [x] “关闭托管 Chrome”和“停止当前流程”未放在全局工具栏
- [x] 全局工具栏缺少明确分组
- [x] 同名流程下拉覆盖无法区分
- [x] 运行详情页仍有重复的关闭 Chrome 入口
- [x] 环境配置变量列表不是表格
- [x] 环境配置导入/导出入口不够明显
