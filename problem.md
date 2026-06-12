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

## 待优化 / 环境相关

### 1. PowerShell 5.1 转义（无关功能）

- `&&` 在 PowerShell 5.1 上不被识别
- 解决方案：用 `cmd /c` 包装，或者 PowerShell 用 `; if ($?) { ... }`
- 已在测试命令中改用 `cmd /c "..."` 包装

### 2. Chrome 启动日志噪音（无关功能）

- Chrome 启动时输出大量 `ERROR:...` 日志（网络服务重启、Crashpad 注册失败等）
- 在 Sandbox 关闭 + 中文 Windows 10/11 上是已知问题，不影响功能
- 长期方案：把这些 stderr 重定向到日志文件，不在 stdout 输出

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
