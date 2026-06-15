# 桌面快捷方式一键执行设计文档

## 1. 背景与目标

当前 `go-chrome` 的 UI 已经支持选择流程、选择环境配置后点击“运行”。为了提升常用流程的启动效率，需要新增“生成桌面快捷方式”功能：

- 用户指定一个流程 + 一个环境配置，生成一个 Windows 桌面快捷方式（`.lnk`）。
- 双击快捷方式后，一键打开程序并自动用指定环境执行指定流程。
- 如果程序已经运行，应复用已有程序实例及其托管的 Chrome，而不是启动新实例。

## 2. 默认假设

本设计基于用户给出的 5 条约束：

1. 双击快捷方式后**打开 UI 再自动执行**，而不是静默后台执行。
2. 快捷方式名称**允许编辑**；默认格式为 `流程名-环境配置名-编号.lnk`，编号从空开始，遇到同名冲突时递增为 `-1`、`-2` 等。
3. 流程执行完成后**保持窗口打开**，运行面板显示结果。
4. 命令行参数使用**简短风格**：`--flow=<flowID>`、`--env=<envID>`。
5. 如果 `go-chrome.exe` 已有实例在运行，**唤醒该实例执行新命令**，不启动新程序。

额外工程假设：

- 目标平台为 Windows（项目本身目标平台即为 Windows 10+/Server 2016 Desktop Experience）。
- 环境变量值仍保存在 SQLite 中，快捷方式只保存流程 ID 与环境 ID，不保存变量值。
- 快捷方式图标复用程序主图标；工作目录指向程序所在目录。

## 3. 架构概览

新增/修改以下模块：

| 模块 | 说明 |
|------|------|
| `cmd/go-chrome/main.go` | 解析 `--flow` / `--env`；启动单实例守护；向 UI 传递自动执行请求。 |
| `internal/singleinstance` | 新增包：命名互斥体 + TCP 回环 IPC，实现单实例与实例间参数传递。 |
| `internal/shortcut` | 新增包：Windows COM 创建 `.lnk` 快捷方式。 |
| `internal/ui/main_window.go` | 接收自动执行请求，选中流程/环境并触发执行；处理运行中冲突。 |
| `internal/ui/flow_library.go` | 增加“生成桌面快捷方式”右键菜单入口。 |
| `internal/ui/global_toolbar.go` | 增加“生成桌面快捷方式”按钮（可选）。 |
| `internal/db/env_repo.go` | 若缺少 `GetByID`，需补充，用于根据 ID 取环境名。 |

## 4. 详细设计

### 4.1 命令行参数

新增参数：

```text
--flow=<flowID>
--env=<envID>
```

- 两个参数同时存在时，进入自动执行模式。
- 任一缺失时，仅按现有逻辑打开 UI。
- 使用 ID 而非名称，避免流程/环境重命名后快捷方式失效。

### 4.2 单实例与 IPC

`internal/singleinstance` 包职责：

1. **首次实例**：
   - 创建 Windows 命名互斥体 `Global\go-chrome-single-instance`。
   - 监听 `127.0.0.1:0`（系统分配端口）。
   - 将监听端口写入 `data/instance-port`。
   - 提供 `Listen(ctx, handler)`，收到参数后回调处理函数。

2. **后续实例**：
   - 打开互斥体失败，判定已有实例运行。
   - 读取 `data/instance-port`。
   - 通过 TCP 发送 JSON 行：`{"flowID":"...","envID":"..."}\n`。
   - 发送成功后直接退出。

3. **跨平台 stub**：
   - Windows 提供完整实现。
   - 非 Windows 提供不限制实例的 stub，直接让新实例启动执行（Linux 本地开发测试可用）。

### 4.3 自动执行流程

UI 初始化完成后，若从命令行或 IPC 收到自动执行请求：

1. 使用 `fyne.Do` 切回主线程。
2. 根据 `flowID` 选中流程（`a.currentFlow`）。
3. 根据 `envID` 设置工具栏环境选择（`envSelect.Selected`）。
4. 重置运行面板（`a.runPanel.reset()`）。
5. 如果 `runner.IsRunning()` 为 true，先调用 `runner.Stop()` 停止当前执行。
6. 调用 `runCurrentFlow()` 的等价逻辑启动执行。
7. 执行完成后保持窗口打开，`runPanel` 显示结果摘要。

### 4.4 桌面快捷方式生成

`internal/shortcut` 包职责：

- 输入：可执行文件路径、参数、工作目录、图标路径、快捷方式保存路径、描述。
- 使用 Windows Shell COM（`WScript.Shell`）创建 `.lnk`。
- UI 层负责组装参数和选择桌面路径。

UI 层行为：

- 入口 1：流程库右键菜单 → “生成桌面快捷方式”。
- 入口 2：全局工具栏新增按钮（当前流程/环境已选时可用）。
- 点击后弹出对话框，默认名称为 `流程名-环境配置名-编号.lnk`。
- 用户可编辑名称；确认后生成到用户桌面目录。
- 若文件已存在，递增编号避免覆盖。

### 4.5 名称冲突处理

默认名称生成算法：

```text
base = "{flowName}-{envName}"
candidate = "{base}.lnk"
suffix = 1
while candidate 已存在:
    candidate = "{base}-{suffix}.lnk"
    suffix += 1
```

## 5. 数据流

### 5.1 生成快捷方式

```text
用户在流程库右键点击“生成桌面快捷方式”
  → UI 读取当前选中的 flowID / envID
  → 查询 flowName / envName
  → 弹出编辑对话框（默认 base = flowName-envName）
  → 根据冲突算法确定最终文件名
  → internal/shortcut.Create(...)
  → 在桌面生成 .lnk
```

### 5.2 双击快捷方式执行

```text
用户双击 .lnk
  → Windows 启动 go-chrome.exe --flow=<id> --env=<id>
  → singleinstance.TryStart() 发现互斥体已存在
  → 读取 data/instance-port
  → TCP 发送 {"flowID":"...","envID":"..."}
  → 新进程退出
  → 已有实例 IPC handler 收到消息
  → fyne.Do 切主线程
  → 选中流程/环境
  → 停止当前运行（如有）
  → runCurrentFlow() 自动执行
```

### 5.3 无已有实例时双击

```text
用户双击 .lnk
  → singleinstance.TryStart() 创建互斥体成功
  → 启动 TCP 监听
  → UI 初始化完成
  → main.go 检测到 --flow/--env
  → 通过通道通知 UI 自动执行
  → UI 选中流程/环境并 runCurrentFlow()
```

## 6. 错误处理

| 场景 | 行为 |
|------|------|
| 快捷方式生成失败（COM、权限、无桌面目录） | 弹窗提示具体错误。 |
| 双击时流程已被删除 | 程序启动后弹窗“流程不存在”。 |
| 双击时环境配置已被删除 | 程序启动后弹窗“环境配置不存在”。 |
| 环境变量缺失 | 复用现有 `checkEnvVars()` 逻辑弹窗提示。 |
| 当前已有流程运行 | 自动停止并重跑；日志中记录“用户通过快捷方式触发新执行，已停止当前运行”。 |
| IPC 发送失败（端口文件丢失、已有实例无响应） | 回退为启动新实例执行（降级保证可用性）。 |

## 7. 测试策略

- 单元测试：
  - `internal/singleinstance`：参数序列化/反序列化、端口文件读写。
  - `internal/shortcut`：在非 Windows 平台跳过；Windows 上使用临时目录验证 `.lnk` 属性（需要解析 COM）。
  - 快捷方式名称冲突算法：用临时目录模拟桌面，验证递增逻辑。
- 集成测试：
  - 命令行参数解析：`main.go` 启动时传入 `--flow` / `--env`，验证 UI 自动执行逻辑被触发。
  - 单实例：启动第一个实例后，再启动第二个带参数的实例，验证第二个退出且第一个收到参数。
- 手工测试：
  - 在 Windows 上生成快捷方式，双击验证执行。
  - 验证已有实例复用。

## 8. 待确认事项

无。用户已确认设计约束与方案。
