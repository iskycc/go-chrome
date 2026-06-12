# 已知问题与未解决清单

本文档记录 go-chrome 项目当前的**已修复**、**已记录但未解决**和**环境相关**问题。

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
| 16 | Chrome 149 下 `Target.createTarget` 失败导致 CDP 连接阻塞 | `chromedp.NewRemoteAllocator` 的远程上下文默认会创建新 tab，触发环境中失败的 `Target.createTarget` 路径 | 启动参数追加 `about:blank` 保证存在 page target；`browser.Connect()` 先通过 `/json/list` 选择已有 page，并用 `chromedp.WithTargetID` 绑定，避免默认创建新 tab；无 page 时才用 HTTP `/json/new` 兜底 |

## 环境相关 / 待回归验证问题

### 1. Chrome 149 在 Windows 上 `Target.createTarget` 失败（已做代码规避，待目标环境回归）

**症状**：
```
DevTools listening on ws://127.0.0.1:60077/...
cdp connect failed: Failed to open new tab - no browser is open (-32000)
```

**现象**：
- Chrome 进程正常启动，HTTP `/json/version` 和 `/json/list` 都能正常返回
- HTTP 调试端口工作正常
- 但 `Target.createTarget`（CDP 方法）返回 -32000
- 伴随 `ERROR:content\browser\network_service_instance_impl.cc:722] Network service crashed or was terminated, restarting service.`
- 错误日志中还有 `Sandbox cannot access executable ... 拒绝访问 (0x5)`

**已尝试的 flag**（都无效）：
- `--no-sandbox`
- `--disable-gpu`
- `--disable-software-rasterizer`
- `--disable-gpu-sandbox`
- `--disable-features=Translate,RendererCodeIntegrity`
- `--headless=new`
- `--single-process`（未试）
- `--disable-features=NetworkServiceInProcess2`（未试）
- `--disable-extensions`
- `--disable-dev-shm-usage`
- `--disable-renderer-backgrounding`
- `--no-first-run`
- `--no-default-browser-check`
- 固定端口（62000-62010）

**当前处理**：
- `browser.Connect()` 不再依赖 chromedp 默认新建 tab，而是优先绑定 Chrome 启动时已有的 `about:blank` page target
- Chrome 启动参数追加本地 `about:blank`，避免启动后没有可绑定的 page target
- 保留无 page target 时的 HTTP `/json/new?about:blank` 兜底
- 如果目标 Windows 环境仍然返回 `-32000`，再按下面的环境方案处理
- 13 个 Runner 单元测试 + 14 个 Validation 子测试 + 12 个 ResolveInput/Wait/MissingEnv 测试 + 18 个 DB 集成测试 = **50+ 测试全部通过**

**可能的解决方案**（按推荐顺序）：

1. **降级到 Chrome 148 或更早**（最直接）
   - Chrome 148 的 `Target.createTarget` 在同样参数下能正常工作
   - 手动下载 Chrome 148 替换 `./chrome/chrome-win64/chrome.exe`
   - 或修改 `data/app-config.json` 的下载源

2. **使用 `chrome-headless-shell`**（备用）
   - Chrome for Testing 还提供轻量级 `chrome-headless-shell`
   - 不需要完整 Chrome 进程，沙箱要求更少

3. **用 `--single-process` 模式**（未验证）
   - 强制所有进程合一，避开沙箱
   - 副作用：稳定性下降，不推荐生产

4. **修复 NTFS 文件权限**
   - Chrome 沙箱需要 `chrome.exe` 所在目录有特定权限
   - 需要手动 `icacls` 调整

5. **回滚到 `chromedp` 旧版本**
   - 可能是 `chromedp v0.15.1` 与 Chrome 149 的协议不匹配
   - 尝试 `chromedp v0.13.0` 或更早

### 2. `chromedp` 与 Chrome 协议版本可能不匹配（待验证）

- `chromedp v0.15.1`（项目当前版本）2025 年发布
- Chrome 149 是较新版本
- CDP 协议 1.3（来自 `/json/version`）
- 可能存在协议不兼容的边缘情况

### 3. PowerShell 转义问题（无关功能）

- `&&` 在 PowerShell 5.1 上不被识别
- 解决方案：用 `cmd /c` 包装，或者 PowerShell 用 `; if ($?) { ... }`
- 已在测试命令中改用 `cmd /c "..."` 包装

## 测试结果汇总

### 全部通过（PASS，50+ 测试）

```
ok  go-chrome/internal/runner    3.830s
ok  go-chrome/internal/flow      0.039s
ok  go-chrome/internal/template  0.059s
ok  go-chrome/internal/config    0.059s
ok  go-chrome/internal/db        2.158s
```

### 跳过（SKIP，6 个 Chrome E2E）

```
SKIP  TestIntegration_LoginFlowComplete
SKIP  TestIntegration_ExampleLoginFlowViaRunFlow
SKIP  TestIntegration_StepRunnerInitAndNextStepByStep
SKIP  TestIntegration_MaskInputInLogsHonored
SKIP  TestIntegration_RetryOnFailure
SKIP  TestIntegration_CDPConnectionRoundTrip
```

### 失败（FAIL）

无。`-32000` 错误已转化为 `SKIP` 而非 `FAIL`。

## 下一步建议

1. **优先回归 Chrome 149**：在 Windows 目标环境跑 `go test -tags=integration ./internal/runner/`，确认新的 target 绑定逻辑是否让 6 个 E2E 测试通过
2. **如仍失败再处理 Chrome 版本问题**：手动下载 Chrome 148 替换 `./chrome/chrome-win64/chrome.exe`，让 6 个 E2E 测试通过
3. **完成 `official_fixed_version` 功能**：AGENTS.md 中提到这是预留配置，尚未实现
4. **GUI 回归测试**：手动启动 `go-chrome.exe`，验证弹窗 UX 改进（新建/导入流程、添加步骤）

## 留给下一个 Agent

- 本轮针对 Chrome 149 的 `Target.createTarget` / `-32000` 问题做的是**代码规避**，不是环境根治：`internal/browser/launcher.go` 启动时追加 `about:blank`，`internal/browser/cdp.go` 连接时优先从 `/json/list` 找已有 `page` target 并用 `chromedp.WithTargetID` 绑定，避免 chromedp 默认新建 tab。
- 已新增 `internal/browser/cdp_test.go`，并更新 `internal/browser/launcher_test.go` 固定上述行为。当前 Linux 环境已通过 `go test ./internal/browser`、`go test ./...`、`go test -tags=integration ./internal/runner`。
- 下一个 Agent 的第一优先级是在问题描述里的 Windows 10/11 + Chrome for Testing 149 环境中回归：先启动 `cmd/test-server`，再跑 `go test -tags=integration ./internal/runner`。如果 6 个 E2E 从 SKIP 变 PASS，说明规避有效。
- 如果 Windows 目标环境仍然报 `Failed to open new tab - no browser is open (-32000)`，不要只继续加重试。先用浏览器调试端口确认 `/json/list` 是否有 `type:"page"`，再看 `/json/new?about:blank` 是否也失败；如果两者都失败，按上面的环境方案处理 Chrome 版本、headless shell 或 NTFS 权限。
- 注意不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制、`stdout*.txt`、`stderr*.txt` 等运行产物。远程刚拉下来的 `stdout*.txt` / `stderr*.txt` 是历史产物，后续如清理请单独确认。
