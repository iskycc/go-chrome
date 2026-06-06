# Go Chrome CDP GUI 自动化项目计划书

## 状态：核心功能与扩展功能均已完成

本项目基于 Go + Fyne + chromedp 构建。

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
- [x] 每次运行使用独立的 Chrome profile (`StartReplay`)

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
- [x] Retry 策略实现（最多 2 次重试）

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
- [x] 流程列表、搜索、标签筛选 (`ui.flowListPanel`)
- [x] 流程新建、编辑、删除（含确认弹窗）
- [x] 流程属性编辑器（名称、描述、标签）(`ui.flowEditorPanel`)
- [x] 步骤表格 (`ui.stepEditorPanel`)
- [x] 步骤属性编辑器（含中文标签）
- [x] 操作类型下拉（中文）
- [x] XPath 输入框
- [x] 输入内容框 + 模板预览按钮 + 校验按钮
- [x] 等待时间 / 超时时间输入
- [x] 失败策略选择（中文）
- [x] 步骤上移、下移、复制、禁用
- [x] 保存前校验错误展示
- [x] 浏览器启动状态展示

### Phase 6: 重放和运行面板
- [x] 流程运行按钮（完整执行）
- [x] 从指定步骤运行
- [x] **真正的单步执行**（StepRunner：初始化 → 下一步 → 下一步...）
- [x] 停止执行
- [x] 运行日志面板 (`ui.runPanel`)
- [x] 运行历史列表 (`ui.historyPanel`)
- [x] 失败截图入口
- [x] 运行结果统计
- [x] 运行历史持久化 (`runner.HistoryStore`)
- [x] 运行历史自动清理

### Phase 7: 体验优化
- [x] 自定义 Fyne 主题颜色、字体大小和控件间距（基础主题已覆盖）
- [x] 给关键按钮增加图标（工具栏和运行面板按钮已添加 Fyne 主题图标）
- [x] 对长 XPath 和输入内容做省略展示（步骤列表和流程列表已添加截断逻辑）
- [x] 对危险操作增加确认弹窗
- [x] 保存前展示校验错误
- [x] 支持最近打开流程
- [x] 支持流程标签
- [x] 支持窗口大小和布局记忆
- [ ] 支持中英文文案后续扩展（预留 i18n 接口）

### Phase 8: 测试
- [x] 单元测试：配置读写
- [x] 单元测试：流程 JSON 读写
- [x] 单元测试：流程校验
- [x] 单元测试：schema 迁移
- [x] 单元测试：Chrome 路径发现
- [x] 单元测试：输入模板数字范围
- [x] 单元测试：输入模板保留前导零
- [x] 单元测试：输入模板枚举随机
- [x] 单元测试：输入模板随机字符串、UUID、日期时间
- [x] 单元测试：输入模板变量复用
- [x] 单元测试：输入模板非法语法
- [x] 单元测试：启动参数构建、DevToolsActivePort 读取
- [x] 单元测试：运行历史保存、列表、清理
- [x] 单元测试：StepRunner 基础状态
- [ ] 集成测试：Chrome 下载（需实际 Chrome 环境）
- [ ] 集成测试：自定义远程链接 Chrome 下载
- [ ] 集成测试：自定义远程链接 SHA256 校验失败
- [ ] 集成测试：自定义远程链接失败后回退官方源
- [ ] 集成测试：Chrome 安装失败后旧版本回滚
- [ ] 集成测试：Chrome 启动和 CDP 连接
- [ ] 集成测试：点击、输入、等待、截图
- [ ] 集成测试：输入步骤使用 SP${11000-11099} 模板并成功输入
- [ ] 集成测试：同一流程内复用同一个随机输入值
- [ ] 手工测试：Windows 10 干净环境首次启动
- [ ] 手工测试：无网络环境已有 Chrome 包时运行
- [ ] 手工测试：端口冲突
- [ ] 手工测试：XPath 不存在
- [ ] 手工测试：用户中途停止

### Phase 9: 打包和发布
- [x] 编写 Windows 构建脚本
- [x] 设置 GUI 子系统编译参数，避免启动控制台窗口
- [x] 准备应用图标（assets/icon.png + embed + 程序设置）
- [x] 生成 zip 发布包脚本（package.bat / package.sh）
- [x] 提供离线包方案：发布包内预置 chrome 目录（package 脚本包含 chrome 目录占位）
- [x] 编写 README
- [x] 编写用户操作说明（USER_GUIDE.md）
- [x] 编写常见问题（FAQ.md）

---

## 验证状态

```bash
go test ./internal/...
# ok  go-chrome/internal/browser
# ok  go-chrome/internal/config
# ok  go-chrome/internal/flow
# ok  go-chrome/internal/runner
# ok  go-chrome/internal/template

go vet ./internal/... ./cmd/go-chrome
# PASS
```

---

## 后续 UI 人类友好化成熟方案

### 设计原则

界面必须围绕用户真实操作链路组织，而不是围绕程序内部模块组织。固定主线为：

```text
选择流程 -> 编辑流程属性 -> 编排步骤 -> 设置步骤参数 -> 保存 -> 运行 -> 查看失败原因并修正
```

所有按钮必须归属到明确区域，禁止再出现流程、步骤、浏览器、运行按钮混在同一个顶部工具栏的设计。

### 成熟版信息架构

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ 顶部状态栏：产品名 / 当前流程 / 保存状态 / Chrome 状态 / 运行状态            │
├──────────────┬────────────────────────────────┬────────────────────────────┤
│ 流程库        │ 1. 流程属性                     │ 3. 步骤属性                 │
│ 搜索流程      │ 名称 / 描述 / 标签               │ 按操作类型动态显示字段        │
│ 标签筛选      ├────────────────────────────────┤ 校验错误 / 模板预览          │
│ 新建 保存 导入 │ 2. 步骤编排                     │ 应用到当前步骤               │
│ 导出 复制 删除 │ 步骤列表 / 新增 / 复制 / 排序     │                            │
├──────────────┴────────────────────────────────┴────────────────────────────┤
│ 4. 运行控制与日志：启动浏览器 / 运行整个流程 / 单步执行 / 停止 / 历史 / 产物  │
└────────────────────────────────────────────────────────────────────────────┘
```

推荐尺寸：

- 左侧流程库：260px 到 320px。
- 中间编排区：占主窗口剩余宽度的 50% 左右。
- 右侧属性区：360px 到 460px。
- 底部运行区：220px 到 280px，可折叠但默认展开。

### 当前已完成的 UI 修复

- [x] 主窗口标题中文化。
- [x] 顶部混杂工具栏已移除，改为说明型头部。
- [x] 流程操作归入左侧流程区：新建、保存、导入、导出、复制、删除。
- [x] 步骤操作归入步骤编排区：新增步骤、复制步骤、删除步骤、上移、下移。
- [x] 运行操作归入底部运行区：启动浏览器、运行整个流程、单步执行、停止。
- [x] 步骤属性归入右侧属性区。
- [x] 步骤类型 UI 中文显示，JSON 仍保存英文枚举。
- [x] 失败策略 UI 中文显示，JSON 仍保存英文枚举。
- [x] 输入模板预览、校验、日志脱敏文案中文化。

### 顶部状态栏方案

顶部只展示状态，不放主要操作按钮。

状态字段：

- 当前流程：未选择时显示 `未选择流程`。
- 保存状态：`未修改`、`有未保存修改`、`保存中`、`已保存`、`保存失败`。
- Chrome 状态：`未安装`、`已安装`、`下载中`、`启动中`、`已启动`、`启动失败`。
- 运行状态：`空闲`、`运行中 3/12`、`已完成`、`失败于第 5 步`。

颜色规范：

- 灰色：空闲或未开始。
- 蓝色：处理中。
- 绿色：成功。
- 黄色：有未保存修改或等待用户处理。
- 红色：失败。

### 流程库设计

流程列表项应展示：

- 流程名称。
- 标签。
- 步骤数量。
- 最近修改时间。
- 最近运行结果。

行为规则：

- 新建流程后自动选中新流程，并聚焦到流程属性区。
- 切换流程前，如果有未保存修改，弹出 `保存并切换`、`放弃修改`、`取消`。
- 删除流程必须显示流程名并二次确认。
- 导入流程必须先校验 schema，失败时不写入 `data/flows`。
- 保存按钮只有在有未保存修改时高亮。

### 步骤编排设计

成熟版应从简单 List 升级为表格视图：

```text
序号 | 状态 | 启用 | 步骤名称 | 操作类型 | 目标摘要 | 输入摘要 | 等待 | 失败处理
```

步骤状态：

- 未运行。
- 运行中。
- 成功。
- 失败。
- 跳过。

新增步骤流程：

1. 点击 `新增步骤`。
2. 先选择操作类型：打开网址、点击元素、输入文本、等待、断言、截图。
3. 在当前步骤后插入新步骤。
4. 自动选中新步骤。
5. 右侧属性区只显示该类型需要填写的字段。

### 步骤属性动态表单

字段显示矩阵：

```text
打开网址       显示：步骤名称、URL、执行后等待、超时、失败处理、备注
点击元素       显示：步骤名称、XPath、执行前等待、执行后等待、超时、失败处理、备注
输入文本       显示：步骤名称、XPath、输入内容、模板预览、日志脱敏、等待、超时、失败处理、备注
清空后输入     显示：同输入文本
等待元素出现   显示：步骤名称、XPath、超时、失败处理、备注
等待元素可见   显示：步骤名称、XPath、超时、失败处理、备注
固定等待       显示：步骤名称、等待毫秒数、备注
获取元素文本   显示：步骤名称、XPath、超时、失败处理、备注
断言元素存在   显示：步骤名称、XPath、超时、失败处理、备注
断言文本包含   显示：步骤名称、XPath、期望文本、超时、失败处理、备注
页面截图       显示：步骤名称、截图备注、失败处理、备注
```

校验规则：

- 步骤名称不能为空。
- XPath 类操作必须填写 XPath。
- 打开网址必须以 `http://` 或 `https://` 开头。
- 等待和超时必须是非负整数。
- 输入模板必须通过模板语法校验。
- 勾选日志脱敏时，日志和列表摘要不能显示真实输入值。

### 运行与诊断设计

运行区不能只是文本日志，应提供完整诊断闭环：

- 进度：`5 / 18`。
- 当前步骤：显示步骤名称和操作类型。
- 汇总：成功数、失败数、跳过数、总耗时。
- 失败时自动选中失败步骤。
- 日志中显示截图和 HTML 快照路径。

日志示例：

```text
[10:31:02] 开始运行：后台登录
[10:31:03] 第 1 步 打开网址 成功，用时 812ms
[10:31:04] 第 2 步 输入用户名 成功，用时 124ms
[10:31:05] 第 3 步 点击登录 失败：未找到 XPath //button[@id='login']
[10:31:05] 已保存截图：data/run-history/.../screenshot.png
[10:31:05] 已保存页面 HTML：data/run-history/.../page.html
```

### 后续实施阶段

#### 阶段 A：状态栏与保存状态

- [ ] 实现顶部状态栏。
- [ ] 实现当前流程名展示。
- [ ] 实现保存状态跟踪。
- [ ] 实现 Chrome 状态展示。
- [ ] 实现运行状态展示。

#### 阶段 B：动态表单与字段校验

- [ ] 按步骤类型动态显示字段。
- [ ] 字段级实时校验。
- [ ] 切换步骤前提示未应用修改。
- [ ] 新增步骤时先选类型再插入。

#### 阶段 C：运行诊断闭环

- [ ] 步骤列表显示运行状态。
- [ ] 运行失败自动选中失败步骤。
- [ ] 日志展示失败截图和 HTML 快照路径。
- [ ] 增加进度条、当前步骤、耗时统计。

#### 阶段 D：新手引导

- [ ] 无流程时展示空状态页。
- [ ] 提供示例流程。
- [ ] 提供常用输入模板插入菜单。
- [ ] 首次启动检查 Chrome、数据目录、配置文件状态。

### 验收标准

- 新用户不看 README，也能按区域编号完成一次流程创建和运行。
- 所有用户可见文案为中文。
- 用户能明确判断按钮属于流程、步骤、属性还是运行。
- UI 不暴露 `clear_and_input`、`assert_text` 等内部枚举。
- 输入错误能在字段附近看到原因。
- 运行失败后能快速定位失败步骤、错误原因、截图和 HTML 快照。
- JSON schema 保持兼容，不因中文 UI 改动改变持久化结构。

---

## 新 Feature：环境变量输入引用、环境切换与 SQLite 持久化（破坏性改造）

### 背景

当前输入模板已经支持随机范围、枚举、UUID、日期、变量复用等运行期动态值，但还缺少“环境变量”能力。实际自动化流程通常需要在不同环境之间切换，例如开发环境、测试环境、预发环境、生产环境，每个环境的 URL、账号、密码、租户 ID、门店号、接口域名都不同。

本 Feature 同时进行数据层破坏性改造：新版本直接废弃分散 JSON 持久化，改为 SQLite 作为唯一运行时存储。无需兼容旧 JSON 数据，无需自动迁移旧数据，无需保留旧 Store 行为。旧 JSON 只可作为手工导入来源，且不是启动兼容路径。

新 Feature 目标：

- 输入值、URL、XPath 或断言文本可以引用当前选中环境中的变量。
- 用户可以维护多套环境配置，并在运行前切换当前环境。
- 环境配置必须持久化保存。
- 流程、步骤、环境配置、运行历史统一保存到 SQLite。
- 破坏性删除 JSON 作为主存储的实现和假设。

### 非目标

- 不做旧版本 JSON 数据自动迁移。
- 不保留 `data/flows/*.json` 作为运行时数据源。
- 不保持 flow.Store 的 JSON 持久化语义。
- 不要求旧运行历史自动导入 SQLite。
- 不要求旧 `recent-flows.json` 自动导入 SQLite。
- 不实现双写 JSON + SQLite。

### 使用场景

1. 同一套流程在测试环境和预发环境运行，只切换环境配置，不复制流程。
2. 登录步骤引用环境账号密码，例如用户名来自 `${env:USERNAME}`，密码来自 `${env:PASSWORD}`。
3. 打开网址步骤引用环境域名，例如 `${env:BASE_URL}/login`。
4. 输入业务编号时组合环境变量和随机模板，例如 `${env:SHOP_PREFIX}-${number:6}`。
5. 断言文本按环境不同而不同，例如 `${env:WELCOME_TEXT}`。

### 模板语法设计

新增环境变量占位符：

```text
${env:变量名}
```

示例：

```text
${env:BASE_URL}/login
${env:USERNAME}
${env:PASSWORD}
SP-${env:SHOP_CODE}-${11000-11099}
```

变量命名规则：

- 只允许字母、数字、下划线。
- 推荐大写，例如 `BASE_URL`、`USERNAME`、`PASSWORD`。
- 变量名不能为空。
- 变量名区分大小写；UI 中默认转大写，降低误用。

解析规则：

- `${env:NAME}` 从当前选中环境读取变量 `NAME`。
- 找不到当前环境时，流程运行失败，错误提示：`未选择运行环境`。
- 找不到变量时，步骤失败，错误提示：`当前环境缺少变量 NAME`。
- 环境变量解析必须和现有模板能力组合工作，例如 `${env:PREFIX}-${number:6}`。
- 环境变量解析和随机模板解析在同一个模板引擎中完成，最终只生成一次输入值。
- 敏感变量参与日志输出时必须按脱敏规则隐藏。

### 环境配置模型

Environment：

```text
id            string/uuid
name          string        // 测试环境、预发环境、生产环境
description   string
is_active     bool          // 当前默认环境
created_at    datetime
updated_at    datetime
```

EnvironmentVariable：

```text
id             string/uuid
environment_id string
key            string       // BASE_URL、USERNAME、PASSWORD
value          string
is_secret      bool         // 是否敏感
description    string
created_at     datetime
updated_at     datetime
```

敏感变量规则：

- `is_secret=true` 时，UI 默认显示为 `******`。
- 日志中永远不输出真实值。
- 运行历史中只保存脱敏后的 generated input。
- 初版可以明文存 SQLite，但字段和 Repository 要隔离，后续可替换为 Windows DPAPI 或主密码加密。

### SQLite 持久化方案

数据库文件：

```text
./data/go-chrome.db
```

SQLite 是唯一运行时持久化存储。新版本启动时只读取 SQLite。若数据库不存在，则创建空库和默认数据；不会扫描、导入或迁移旧 JSON。

启动规则：

1. 确保 `./data` 存在。
2. 打开 `./data/go-chrome.db`。
3. 执行 schema migration。
4. 如果没有环境，创建默认环境 `默认环境`。
5. 如果没有流程，显示空状态或引导创建示例流程。

推荐表结构：

```sql
CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE app_state (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE TABLE flows (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  schema_version INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE steps (
  id TEXT PRIMARY KEY,
  flow_id TEXT NOT NULL,
  sort_order INTEGER NOT NULL,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  type TEXT NOT NULL,
  target_strategy TEXT NOT NULL DEFAULT 'xpath',
  target_value TEXT NOT NULL DEFAULT '',
  input_mode TEXT NOT NULL DEFAULT 'template',
  input_text TEXT NOT NULL DEFAULT '',
  input_mask_in_logs INTEGER NOT NULL DEFAULT 0,
  wait_before_ms INTEGER NOT NULL DEFAULT 0,
  wait_after_ms INTEGER NOT NULL DEFAULT 500,
  timeout_ms INTEGER NOT NULL DEFAULT 10000,
  on_error TEXT NOT NULL DEFAULT 'stop',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(flow_id) REFERENCES flows(id) ON DELETE CASCADE
);

CREATE TABLE environments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  is_active INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE environment_variables (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL DEFAULT '',
  is_secret INTEGER NOT NULL DEFAULT 0,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(environment_id, key),
  FOREIGN KEY(environment_id) REFERENCES environments(id) ON DELETE CASCADE
);

CREATE TABLE run_results (
  id TEXT PRIMARY KEY,
  flow_id TEXT NOT NULL,
  environment_id TEXT,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  total_steps INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  skipped_count INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  snapshot_dir TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(flow_id) REFERENCES flows(id) ON DELETE CASCADE,
  FOREIGN KEY(environment_id) REFERENCES environments(id) ON DELETE SET NULL
);

CREATE TABLE run_step_results (
  id TEXT PRIMARY KEY,
  run_result_id TEXT NOT NULL,
  step_id TEXT,
  step_order INTEGER NOT NULL,
  step_name TEXT NOT NULL,
  step_type TEXT NOT NULL,
  status TEXT NOT NULL,
  error TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  screenshot_path TEXT NOT NULL DEFAULT '',
  html_snapshot_path TEXT NOT NULL DEFAULT '',
  generated_input_masked TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  FOREIGN KEY(run_result_id) REFERENCES run_results(id) ON DELETE CASCADE
);
```

索引：

```sql
CREATE INDEX idx_steps_flow_order ON steps(flow_id, sort_order);
CREATE INDEX idx_env_vars_environment ON environment_variables(environment_id);
CREATE INDEX idx_run_results_flow_started ON run_results(flow_id, started_at DESC);
CREATE INDEX idx_run_step_results_run_order ON run_step_results(run_result_id, step_order);
```

### 破坏性数据层改造要求

必须删除或重写以下 JSON 主存储路径：

- `internal/flow/store.go` 不再以 `data/flows/*.json` 作为主存储。
- `internal/flow/recent.go` 不再使用 `recent-flows.json`，改用 `app_state` 或专表。
- `internal/runner/history.go` 不再以 JSON 摘要作为运行历史主存储，改用 `run_results` 和 `run_step_results`。
- UI 中所有流程列表、保存、删除、复制、最近流程、运行历史查询都必须走 SQLite repository。

允许保留的 JSON 能力：

- `导入流程 JSON`：用户手工选择 JSON 文件导入 SQLite。
- `导出流程 JSON`：从 SQLite 读取流程并导出为 JSON。
- `app-config.json` 可暂时保留用于 Chrome 下载源、窗口大小、主题等应用级配置；流程数据和环境数据不得再写入 JSON。

删除兼容逻辑：

- 不扫描 `data/flows` 自动导入。
- 不生成 `legacy-json-backup`。
- 不做 JSON 与 SQLite 双写。
- 不保证旧 JSON 运行历史可见。

### UI 设计

新增环境选择区域，建议放在顶部状态栏或流程库上方：

```text
当前环境：[测试环境 v]  [管理环境]
```

点击“管理环境”打开环境管理面板：

```text
环境列表：
- 默认环境
- 测试环境
- 预发环境
- 生产环境

环境变量表：
变量名 | 变量值 | 敏感 | 说明
BASE_URL | https://test.example.com | 否 | 测试环境域名
USERNAME | test_user | 否 | 登录账号
PASSWORD | ****** | 是 | 登录密码
```

环境管理功能：

- 新建环境。
- 复制环境。
- 删除环境。
- 重命名环境。
- 设置为当前环境。
- 新增变量。
- 编辑变量。
- 删除变量。
- 标记变量为敏感。
- 导入/导出环境配置。

运行前要求：

- 如果流程未引用 `${env:...}`，允许无环境变量运行，但仍要有当前环境记录。
- 如果流程引用 `${env:...}`，运行前必须检查当前环境变量是否完整。
- 缺失变量时阻止执行，并列出缺失变量名。

### 模板引擎改造方案

现有 `template.Engine` 新增环境变量上下文：

```go
type EnvProvider interface {
    GetEnvValue(key string) (value string, found bool, secret bool)
}

type Engine struct {
    vars map[string]string
    seq int
    env EnvProvider
}
```

新增构造：

```go
func NewEngineWithEnv(env EnvProvider) *Engine
```

新增解析：

```go
${env:BASE_URL}
```

错误类型：

- `ErrEnvironmentNotSelected`
- `ErrEnvironmentVariableNotFound`
- `ErrInvalidEnvironmentVariableName`

日志脱敏：

- 模板解析结果要返回真实值和脱敏值。
- 如果任一参与变量是 secret，则整段 generated input 默认脱敏。
- `StepResult.GeneratedInput` 不再保存真实值，改为 `GeneratedInputMasked`。

### 运行器改造方案

Runner 启动时需要知道当前环境：

```go
type RunOptions struct {
    StartStep int
    EnvironmentID string
}
```

破坏性修改接口：

```go
RunFlow(f *flow.Flow, opts RunOptions)
```

不保留旧接口：

```go
RunFlow(f *flow.Flow, startStep int)
```

运行前校验：

- 扫描流程步骤中的 `${env:...}` 引用。
- 获取当前环境变量集合。
- 如果缺失变量，运行前失败并在 UI 显示缺失列表。
- 将 environment_id 写入 run_results。

### 实施阶段

#### 阶段 1：SQLite 基础设施

- [ ] 引入 SQLite driver。
- [ ] 新增 `internal/db` 包。
- [ ] 创建 `./data/go-chrome.db`。
- [ ] 实现 migration 表和迁移执行器。
- [ ] 建立 flows、steps、environments、environment_variables、run_results、run_step_results 表。
- [ ] 创建默认环境。
- [ ] 单元测试：空库初始化。
- [ ] 单元测试：重复 migration 不报错。

#### 阶段 2：破坏性替换流程存储

- [ ] 删除 JSON flow.Store 主存储实现。
- [ ] 实现 SQLite FlowRepository。
- [ ] 实现 SQLite StepRepository。
- [ ] 流程保存、读取、删除、复制全部改走 SQLite。
- [ ] 最近流程改写到 SQLite。
- [ ] 保留手工 JSON 导入导出。
- [ ] 单元测试：流程保存、读取、删除、复制。
- [ ] 单元测试：步骤排序持久化。

#### 阶段 3：环境配置能力

- [ ] 定义 Environment 和 EnvironmentVariable 模型。
- [ ] 实现 EnvironmentRepository。
- [ ] 实现当前环境选择和持久化。
- [ ] 实现环境复制、删除、变量增删改查。
- [ ] 单元测试：环境切换后持久化。
- [ ] 单元测试：敏感变量不在日志中明文出现。

#### 阶段 4：模板引擎支持 env

- [ ] `template.Engine` 支持 EnvProvider。
- [ ] 支持 `${env:NAME}`。
- [ ] 支持 `${env:NAME}` 和随机模板组合。
- [ ] 运行前扫描缺失变量。
- [ ] 单元测试：正常引用。
- [ ] 单元测试：缺失环境。
- [ ] 单元测试：缺失变量。
- [ ] 单元测试：secret 变量触发脱敏。

#### 阶段 5：UI 环境管理

- [ ] 顶部状态栏增加当前环境下拉框。
- [ ] 增加环境管理弹窗或面板。
- [ ] 增加环境变量表格。
- [ ] 增加敏感变量显示/隐藏切换。
- [ ] 增加导入/导出环境配置。
- [ ] 运行前缺失变量提示。

#### 阶段 6：运行历史 SQLite 化

- [ ] 将 RunResult 写入 SQLite。
- [ ] 将 StepResult 写入 SQLite。
- [ ] 运行历史列表从 SQLite 查询。
- [ ] 截图和 HTML 文件仍保留文件系统路径。
- [ ] 支持按流程、环境、状态筛选运行历史。

### 验收标准

- 删除 `data/flows` 后程序仍能正常运行，因为主存储为 `./data/go-chrome.db`。
- 新装启动会创建 `./data/go-chrome.db` 和默认环境。
- 用户可以创建至少两套环境，例如 `测试环境` 和 `预发环境`。
- 用户可以在 UI 中切换当前环境，关闭程序后再次打开仍保持上次选择。
- 输入值支持 `${env:USERNAME}`。
- 打开网址支持 `${env:BASE_URL}/login`。
- `${env:SHOP_CODE}-${11000-11099}` 能正常组合环境变量和随机范围。
- 缺失环境变量时，运行前阻止执行，并明确列出缺失变量名。
- 敏感变量在 UI 和日志中默认脱敏。
- 流程、步骤、环境配置、运行历史保存到 `./data/go-chrome.db`。
- 程序不会自动扫描旧 JSON 流程目录。
- 仍可手工导入和导出单个流程 JSON。
- SQLite migration 可重复执行且不破坏已有 SQLite 数据。
