# Agent Notes

本文档给后续编码代理和维护者使用，说明本项目的关键约束、构建方式和容易踩坑的地方。

## 项目目标

本项目是 Go 编写的 Windows GUI 自动化工具：

- GUI 框架：Fyne v2。
- 浏览器控制：chromedp + Chrome DevTools Protocol。
- 浏览器来源：当前目录 `./chrome` 下的 Chrome for Testing。
- 运行期不依赖系统已安装 Chrome、Selenium Server 或 ChromeDriver。
- 流程数据使用 JSON 文件持久化。

## 强约束

- 不要在运行期依赖互联网，除非本地没有 Chrome 且用户触发 Chrome 下载。
- 本地 `./chrome` 下已有 `chrome.exe` 时，禁止再次下载 Chrome。
- 每次重放都必须启动新的托管 Chrome 实例。
- 每次重放 Chrome 必须带以下参数：
  - `--incognito`
  - `--start-maximized`
  - `--ignore-certificate-errors`
  - `--allow-insecure-localhost`
  - `--remote-debugging-port=0`
- 不要读取或污染用户系统 Chrome profile。
- 不要把流程运行时生成的 input 明文无条件写入日志；尊重 `maskInLogs` 和配置默认值。
- 不要把 `data/`、`logs/`、`chrome/`、构建产物提交进版本库。

## 运行环境兼容性

目标平台：

- Windows 10（最低版本，所有发行版）
- Windows Server 2016 **Desktop Experience**（Server Core 不带 GUI，无法运行 Fyne 应用）

关键依赖的最低系统要求：

- Go 1.26：官方支持 Windows 10 / Server 2016。
- Fyne v2.7.4：支持 Windows 7+，需要桌面体验和 OpenGL/DirectX 图形驱动。
- chromedp v0.15.1：纯 Go，通过 Chrome DevTools Protocol 通信，无额外 Windows 版本限制。
- gopsutil v3.24.5：进程信息通过 Windows WMI / PDH 获取，需要 WMI 服务运行（Server 2016 默认启用）。
- `taskkill /F /T /PID`：Windows XP+ 可用，用于关闭托管 Chrome 进程树。
- `build.bat` 使用 PowerShell 5.1 语法（Windows 10 / Server 2016 默认内置），`Invoke-WebRequest` / `Expand-Archive` 要求 PowerShell 3.0+。

已知限制：

- Windows Server 2016 Server Core（无桌面体验）缺少 Fyne 所需的 GUI 子系统，无法运行本程序。
- Windows Server 2016 默认安全策略可能阻止未签名可执行文件或限制网络下载，首次运行 `build.bat` 下载 Go SDK / 字体时需要管理员权限或相应安全策略放行。

## 构建

构建脚本要求：

- 不要在 `build.sh` 或 `build.bat` 中自动运行 `go mod tidy`。
- `build.sh` 保持离线友好，使用本机已有 Go SDK 和 module cache 构建。
- `build.bat` 必须检查 Windows 环境是否存在 Go SDK。
- 如果 Windows 环境没有 Go SDK，`build.bat` 必须从 `https://golang.google.cn/dl/` 下载 Go SDK 到项目本地 `.tools\go`。
- `build.bat` 编译前必须设置 `GOPROXY=https://goproxy.cn,direct`，并使用中国 Go module 镜像下载依赖。
- 构建使用 `go build -mod=readonly`。

Windows：

```bat
build.bat
```

Linux 本机：

```bash
./build.sh
```

Linux 构建 Fyne 需要 CGO 和图形开发包，例如：

```bash
sudo apt-get install -y pkg-config libgl1-mesa-dev xorg-dev
```

## 测试

全量测试：

```bash
go test ./...
```

核心非 GUI 包：

```bash
go test ./internal/browser ./internal/runner ./internal/config ./internal/flow ./internal/template
```

如果 Linux 环境缺少 Fyne/GLFW 依赖，全量测试会在 UI 包构建阶段失败。优先补系统依赖，而不是绕过 UI 包。

## 关键模块

- `internal/browser`
  - Chrome 下载、SHA256 校验、ZIP 解压、安装回滚。
  - `Manager.Install()` 负责"已有 Chrome 则跳过下载"。
  - `Manager.StartReplay()` 负责每次重放启动新的 Chrome。
  - `Manager.Stop()` 负责关闭本程序启动的 Chrome 进程树（Windows 用 `taskkill /F /T /PID`，非 Windows 用 `kill -9`），只杀 `m.proc` 跟踪的 pid，不影响用户自己打开的 Chrome。
  - `Launch()` 负责 Chrome 启动参数。
- `internal/runner`
  - `Runner.RunFlow()` 串联 Chrome 准备、CDP 连接、步骤执行和事件输出。
- `internal/runner/actions.go`
  - 单步 CDP 操作实现。
- `internal/template`
  - 输入模板解析和生成。
  - 支持 `SP${11000-11099}`、枚举、随机字符串、日期、UUID、变量复用。
- `internal/flow`
  - 流程模型、校验、JSON 存储、导入导出。
- `internal/ui`
  - Fyne GUI。
  - 后台 goroutine 更新 UI 时必须使用 `fyne.Do` 或等价主线程调度。
  - 运行面板的"关闭本程序启动的 Chrome"按钮只杀 `Manager` 跟踪的 Chrome 进程，不影响用户手动打开的 Chrome。

## 离线运行检查点

在离线环境验证前，确认：

1. `go-chrome.exe` 已经构建完成。
2. 程序目录存在 `./chrome`。
3. `./chrome` 下能找到 `chrome.exe`。
4. `./data/app-config.json` 不要求强制更新 Chrome。
5. 用户编排的流程访问目标如果是外部网站，离线环境下页面本身不可达；这不属于程序运行期依赖互联网。

## 常见风险

- `DevToolsActivePort` 位置可能是 user-data-dir 根目录，也可能在 `Default` 子目录；读取逻辑兼容两者（`ReadDevToolsPort` 会依次尝试）。
- Fyne 控件不能随意从后台 goroutine 直接更新；已使用 `fyne.Do` 包装 UI 更新。
- `official_fixed_version` 目前是预留配置，尚未实现。
- `retry` 失败策略已实现：当步骤的 `onError` 为 `retry` 时，最多自动重试 2 次。
- 单步执行已实现为真正的逐步执行：`StepRunner.Init()` 初始化环境，`Next()` 每次执行一步。UI 中"单步执行"按钮初始化后会变为"下一步"。
- 输入模板支持嵌套 `${}`（如 `${var:user=SP${11000-11099}}`），解析器使用栈深度匹配括号。
