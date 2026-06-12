# Go Chrome CDP Automation

Go Chrome CDP Automation 是一个 Windows 桌面 GUI 工具，用 Go、Fyne 和 chromedp 编写，用于编排并重放 Chrome 浏览器 UI 自动化流程。

项目运行时通过 Chrome DevTools Protocol（CDP）控制当前目录下的 Chrome for Testing，不依赖用户系统已经安装的 Chrome、Selenium Server 或 ChromeDriver。

## 核心特性

- Windows 10+ 桌面 GUI。
- 使用 CDP 直接控制 Chrome。
- 自动发现 `./chrome` 下的 `chrome.exe`。
- 本地已有 Chrome for Testing 时不会下载，也不会访问远程下载源。
- 本地缺少 Chrome 时，可从官方 Chrome for Testing Stable 或自定义远程 ZIP 链接下载。
- 支持自定义 Chrome ZIP 的 SHA256 校验、安装失败回滚、下载缓存。
- 每次重放都会启动新的托管 Chrome 实例。
- 每次重放默认使用无痕模式、最大化窗口，并忽略 HTTPS 证书错误。
- 支持流程新建、编辑、复制、删除、导入、导出和重放。
- 支持 XPath 点击、输入、清空后输入、等待、断言、截图等步骤。
- 支持输入模板，例如 `SP${11000-11099}`，重放时随机生成范围内数据。
- 流程以 JSON 结构化文件保存到 `./data/flows`。
- 执行失败时保存截图和页面 HTML 到 `./data/run-history`。

### UI 特性

- 内置 Cascadia Code SemiLight 字体，避免 Windows 中文环境字体变形。
- 全局操作栏：流程选择、保存、启动浏览器、运行、单步执行、停止、环境选择始终可见。
- 独立的“环境配置”tab，集中管理环境变量。
- “运行详情”tab 专注于日志、摘要和产物。

## 离线运行说明

编译后的程序可以在完全离线环境运行，但需要满足以下条件：

1. 程序目录下已经存在可用的 Chrome for Testing。
2. `chrome.exe` 位于以下任一位置：
   - `./chrome/chrome.exe`
   - `./chrome/chrome-win64/chrome.exe`
   - `./chrome/chrome-win32/chrome.exe`
   - 或者可被程序在 `./chrome` 下递归找到。
3. Go 依赖只在编译期需要；运行期不需要 Go 环境。

只有在本地找不到可用 Chrome 时，程序才会尝试联网下载 Chrome。除此之外，程序运行流程、流程读取、重放、日志、截图、模板生成均不需要访问互联网。自动化步骤中的 `navigate` 访问哪个网页，由用户编排的目标 URL 决定。

## 目录结构

```text
go-chrome/
  cmd/go-chrome/main.go
  internal/
    app/        # 应用目录初始化
    browser/    # Chrome 下载、安装、启动、CDP 连接
    config/     # 配置读写
    flow/       # 流程模型、校验、JSON 持久化
    logx/       # 日志
    runner/     # 自动化执行器
    template/   # 输入模板引擎
    ui/         # Fyne GUI
  data/
    app-config.json
    flows/
    run-history/
    chrome-profile/
  logs/
  chrome/
```

## 构建要求

推荐环境：

- Go 1.26 或与 `go.mod` 匹配的版本。
- Windows 10+ 64 位用于目标运行。
- Fyne 需要 CGO。

Windows 构建：

```bat
build.bat
```

`build.bat` 会先检查当前环境是否有 Go SDK。若未找到 `go.exe`，会从 `https://golang.google.cn/dl/` 下载 Go SDK 到项目本地 `.tools\go`，并临时加入 `PATH`。随后会设置：

```text
GOPROXY=https://goproxy.cn,direct
GOSUMDB=sum.golang.google.cn
```

并通过中国 Go module 镜像下载依赖后编译。

Linux 本机构建：

```bash
./build.sh
```

Linux 上构建 Fyne 需要系统图形开发依赖，例如：

```bash
sudo apt-get install -y pkg-config libgl1-mesa-dev xorg-dev
```

Linux 离线构建要求 Go module cache 已提前准备好。联网环境中可先执行：

```bash
go mod download
```

`build.sh` 使用 `-mod=readonly`，不会自动执行 `go mod tidy`，也不会主动下载依赖。`build.bat` 按 Windows 用户体验要求，会在缺少 Go SDK 时自动下载 SDK，并使用中国镜像下载模块依赖。

## 测试

运行全部测试：

```bash
go test ./...
```

只运行核心非 GUI 包：

```bash
go test ./internal/browser ./internal/runner ./internal/config ./internal/flow ./internal/template
```

## Chrome 配置

配置文件位于：

```text
./data/app-config.json
```

示例：

```json
{
  "chrome": {
    "downloadSource": "official_stable",
    "channel": "Stable",
    "fixedVersion": "",
    "customDownloadURL": "",
    "customDownloadSHA256": "",
    "customVersionLabel": "",
    "fallbackToOfficial": true,
    "installDir": "./chrome",
    "userDataDir": "./data/chrome-profile",
    "keepDownloadCache": true
  },
  "runner": {
    "defaultTimeoutMs": 10000,
    "defaultWaitAfterMs": 500,
    "templateRandomSeedMode": "system",
    "templatePreviewCount": 5,
    "maskInputValueInLogs": true,
    "autoScreenshotOnFinish": false
  },
  "app": {
    "closeManagedChromeOnExit": true,
    "logRetentionDays": 14,
    "theme": "default"
  }
}
```

下载源：

- `official_stable`：从 Chrome for Testing 官方 Stable 源下载。
- `custom_url`：从 `customDownloadURL` 指定的远程 ZIP 下载。
- `official_fixed_version`：字段已预留，当前尚未实现固定官方版本下载。

## 重放时 Chrome 参数

每次重放都会使用新的 Chrome 用户数据目录，并带上以下关键参数：

```text
--incognito
--start-maximized
--ignore-certificate-errors
--allow-insecure-localhost
--remote-debugging-port=0
```

这保证每次重放相对独立，且可以执行 HTTPS 证书异常的测试环境页面。

## 输入模板

输入步骤和清空后输入步骤支持模板表达式。

| 用法 | 示例 |
| --- | --- |
| 数字范围 | `SP${11000-11099}` |
| 固定位数范围 | `SHOP-${0001-9999}` |
| 枚举随机 | `${dev|test|stage}` |
| 随机数字 | `${number:6}` |
| 随机字母 | `${alpha:8}` |
| 随机字母数字 | `${alnum:10}` |
| UUID | `${uuid}` |
| 时间戳 | `${timestamp}` |
| 日期 | `${date:yyyyMMdd}` |
| 日期时间 | `${datetime:yyyyMMddHHmmss}` |
| 运行序号 | `${seq}` |
| 变量复用 | `${var:user=SP${11000-11099}}` 后续用 `${var:user}` |

## 支持的步骤类型

- `navigate`
- `click`
- `input`
- `clear_and_input`
- `wait_present`
- `wait_visible`
- `wait_fixed`
- `get_text`
- `assert_exists`
- `assert_text`
- `screenshot`

## 运行数据

默认运行数据目录：

- `./data/flows`：流程 JSON。
- `./data/run-history`：运行历史、失败截图、HTML 快照。
- `./data/chrome-profile`：Chrome 用户数据目录。
- `./logs`：应用日志。
- `./chrome`：Chrome for Testing。

这些目录均为运行期产物，默认不纳入版本管理。
