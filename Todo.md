# Go Chrome CDP GUI 自动化项目计划书

## 状态：已实现核心骨架与功能

本项目基于 Go + Fyne + chromedp 构建，实现了 Todo.md 中定义的核心功能。

---

## 已完成的模块

### Phase 0: 工程骨架
- [x] 初始化 Go module (`go.mod`)
- [x] 建立 `cmd/` 和 `internal/` 目录结构
- [x] 配置加载 (`internal/config`) 与默认配置生成
- [x] 日志模块 (`internal/logx`) — 结构化日志、按日期滚动、保留清理
- [x] 应用目录初始化 (`internal/app`)

### Phase 1: Chrome for Testing 管理
- [x] Chrome manifest 结构 (`browser.VersionManifest`)
- [x] Stable 版本信息获取 (`FetchStableInfo`)
- [x] 下载源枚举：官方 Stable、自定义远程链接
- [x] 自定义远程链接下载 (`browser.DownloadFile`)
- [x] SHA256 校验 (`browser.VerifySHA256`)
- [x] ZIP 校验和解压 (`browser.ExtractZIP`)
- [x] `chrome.exe` 路径发现 (`browser.FindChromeExe`)
- [x] 原子替换、备份与失败回滚 (`browser.Manager.Install`)
- [x] 自定义远程链接失败后官方源回退 (`FallbackToOfficial`)
- [x] 启动 Chrome (`browser.Launch`)
- [x] 读取 `DevToolsActivePort`
- [x] Chrome 存活检测 (`browser.IsChromeRunning`)
- [x] 退出时关闭程序托管的 Chrome
- [x] 本地 `chrome.exe` 已存在时跳过下载，不再访问远程下载源
- [x] 重放时每次启动新的托管 Chrome 实例
- [x] 重放 Chrome 默认无痕模式：`--incognito`
- [x] 重放 Chrome 默认最大化窗口：`--start-maximized`
- [x] 重放 Chrome 默认忽略 HTTPS 证书错误：`--ignore-certificate-errors`
- [x] 重放 Chrome 默认允许本地不安全证书：`--allow-insecure-localhost`

### Phase 2: 输入模板引擎
- [x] 数字范围：`${11000-11099}`
- [x] 保留前导零：`${001-999}`
- [x] 枚举随机：`${A|B|C}`
- [x] 随机数字、字母、字母数字
- [x] UUID、时间戳、日期时间占位符
- [x] 运行序号：`${seq}`
- [x] 同一次流程运行内变量复用：`${var:user=...}` / `${var:user}`
- [x] 模板预览能力 (`template.Preview`)
- [x] 模板语法校验 (`template.Validate`)
- [x] 嵌套 `${}` 支持

### Phase 3: CDP 自动化执行核心
- [x] 打开网址 (`navigate`)
- [x] XPath 点击 (`click`)
- [x] 输入文本 / 清空后输入 (`input` / `clear_and_input`)
- [x] 等待元素出现 / 可见 (`wait_present` / `wait_visible`)
- [x] 等待固定时间 (`wait_fixed`)
- [x] 获取元素文本 (`get_text`)
- [x] 断言元素存在 / 文本包含 (`assert_exists` / `assert_text`)
- [x] 截图 (`screenshot`)
- [x] 失败时保存截图和 HTML (`captureScreenshot` / `captureHTML`)
- [x] 超时控制（基于 `context.WithTimeout`）
- [x] 停止信号（channel）

### Phase 4: 流程模型和持久化
- [x] Flow、Step、Target、Input、RunResult 模型
- [x] JSON 保存与读取 (`flow.Store`)
- [x] schema 校验 (`flow.Validate`)
- [x] 流程导入 / 导出
- [x] 流程复制 (`Flow.Clone`)
- [x] 流程删除
- [x] schema 迁移框架 (`flow.Migrate`)

### Phase 5: GUI 初版 (Fyne)
- [x] 主窗口 (`ui.App`)
- [x] 流程列表、搜索 (`ui.flowListPanel`)
- [x] 流程新建、编辑、删除
- [x] 步骤表格 (`ui.stepEditorPanel`)
- [x] 步骤属性编辑器
- [x] 操作类型下拉
- [x] XPath 输入框
- [x] 输入内容框 + 模板预览按钮 + 校验按钮
- [x] 等待时间 / 超时时间输入
- [x] 失败策略选择
- [x] 步骤上移、下移、复制、禁用
- [x] 保存按钮
- [x] 浏览器启动状态展示

### Phase 6: 重放和运行面板
- [x] 流程运行按钮
- [x] 从指定步骤运行
- [x] 单步执行（简化版：从选定步骤开始）
- [x] 停止执行
- [x] 运行日志面板 (`ui.runPanel`)
- [x] 失败截图入口
- [x] 运行结果统计

### Phase 7: 测试
- [x] 单元测试：配置读写
- [x] 单元测试：流程 JSON 读写
- [x] 单元测试：流程校验
- [x] 单元测试：输入模板数字范围、前导零、枚举、随机字符串、UUID、日期时间、变量复用、非法语法
- [x] 单元测试：Chrome 路径发现、SHA256 校验
- [x] 单元测试：Chrome 启动参数包含无痕、最大化、忽略证书
- [x] 单元测试：本地已有 `chrome.exe` 时 `Install()` 直接跳过下载
- [x] 单元测试：兼容读取 `DevToolsActivePort` 的根目录和 `Default` 目录位置

### Phase 8: 打包
- [x] Windows 构建脚本 (`build.bat`)，含 `-H=windowsgui`
- [x] 通用构建脚本 (`build.sh`)
- [x] README.md
- [x] AGENTS.md

---

## 已知限制与后续扩展

1. **Windows 编译环境**：Fyne 依赖 CGO，当前 Linux 开发环境无 mingw-w64，因此 UI 包需在 Windows 或配备 mingw 的 Linux 上编译。所有非 UI 包已在 Linux 验证编译与测试通过。
2. **单步执行**：当前实现为“从选定步骤运行到结束”，真正的逐步行进需额外状态机。
3. **重试策略**：`retry` 策略目前仅作为字段保存，运行器暂未实现自动重试次数控制。
4. **CSS Selector**：初版仅支持 XPath。
5. **多标签页 / iframe**：暂未支持。
6. **GUI 主题深度定制**：当前仅基础主题色覆盖。
7. **全量 Linux 测试限制**：当前环境缺少 Fyne/GLFW 所需的 `pkg-config` 与 X11 头文件，`go test ./...` 会在 UI 包构建阶段失败；非 UI 核心包测试已通过。
