# UI 三次优化任务书

本文档是给后续 Agent 的最新需求说明。优先级高于 `Todo1.md` / `Todo2.md` 中与字体或全局按钮相关的旧表述。当前用户明确要求：

1. 使用 **JetBrains Code** 风格字体。
2. 中文和英文都要使用同一套字体风格，不能英文是 JetBrains、中文是系统默认 fallback。
3. 当前构建版本字体太细，不满足要求。
4. “停止当前流程”和“停止/关闭浏览器”是常用功能，需要和“启动浏览器”放在同一个全局位置。
5. 环境配置需要支持导入和导出。

## P0-1. 字体最终目标：JetBrains Code 风格，中英文统一且不能过细

### 用户要求

用户希望使用 **JetBrains Code** 字体风格，并且：

- 中文也要使用这套风格。
- 英文也要使用这套风格。
- 不能出现英文是 JetBrains 风格、中文回退系统字体的割裂效果。
- 当前 SemiLight 观感太细，不满足要求。

### 必须先澄清的技术事实

JetBrains 官方常见字体是 **JetBrains Mono**。它是否包含中文不能靠名称猜测，必须以实际字体文件为准。

后续 Agent 必须拿到真实字体文件后检查 CJK 覆盖。检查命令示例：

```bash
fc-scan --format '%{family}\n%{charset}\n' assets/fonts/JetBrainsCode-Regular-CJK.ttf
```

输出中必须包含或覆盖 CJK 常用区，例如：

```text
4e00-9fff
```

如果字体不覆盖 CJK，即使 `theme.Font()` 返回该字体，中文也不会真正使用它。

### 不接受的方案

以下方案不算完成：

- 只替换成官方 JetBrains Mono，但该字体不含中文。
- 英文用 JetBrains，中文用系统默认字体。
- 继续使用过细的 SemiLight 作为全局字体。
- 只改日志或代码区域字体，不改全局 UI 字体。
- 使用来源不明或许可证不允许分发的字体。

### 推荐方案

使用一套合法分发、覆盖中英文的 JetBrains Code / JetBrains Mono 风格 CJK 字体，至少包含：

```text
assets/fonts/JetBrainsCode-Regular-CJK.ttf
assets/fonts/JetBrainsCode-Medium-CJK.ttf
assets/fonts/LICENSE-JetBrainsCode-CJK.txt
```

如果找不到名为 JetBrains Code 的官方 CJK 版本，可使用合法的 JetBrains Mono + CJK 融合字体。要求是最终视觉风格接近 JetBrains Code / JetBrains Mono，且中英文在同一字体资源内统一。

### 字重要求

当前 UI 字体太细，必须提供字重层级：

- 普通 UI：Regular
- Tab、标题、重要按钮：Medium 或 SemiBold
- 不建议再用 SemiLight 做全局 UI 字体

`internal/ui/theme.go` 的 `Font(style)` 必须区分 `style.Bold`：

```go
func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
    if style.Bold {
        if f := assets.AppUIFontMedium(); f != nil {
            return f
        }
    }
    if f := assets.AppUIFontRegular(); f != nil {
        return f
    }
    return theme.DefaultTheme().Font(style)
}
```

### 代码改造建议

修改 `assets/embed.go`，不要再只暴露具体旧字体名。建议提供稳定语义入口：

```go
func AppUIFontRegular() fyne.Resource
func AppUIFontMedium() fyne.Resource
func CodeFont() fyne.Resource
```

要求：

- `AppUIFontRegular()` 返回中英文完整字体 Regular。
- `AppUIFontMedium()` 返回中英文完整字体 Medium/SemiBold。
- `CodeFont()` 如无特殊必要，也返回同一套中英文完整字体，避免日志中文割裂。
- `theme.Font()` 全局使用上述 App UI 字体。

### 验收标准

- Windows 实机启动后，中文按钮、Tab、表格、状态栏、环境配置、日志都使用统一字体风格。
- 中文不缺字，不显示方块。
- 中文和英文没有明显 fallback 割裂。
- 普通文字不发虚、不太细。
- 标题和重要按钮有明显字重。
- `fc-scan` 或等价工具确认最终 UI 字体覆盖 CJK。

## P0-2. 全局工具栏：停止当前流程、关闭托管浏览器必须和启动浏览器同级可见

### 当前问题

当前 UI 已经有全局工具栏，但停止/关闭类操作仍不够明确：

- “停止当前流程”需要常驻可见。
- “关闭托管 Chrome / 停止托管浏览器”需要和“启动 Chrome”放在同一个位置。
- 不要把关闭浏览器藏在“运行详情”页。

### 产品语义

用户说“停止所有浏览器”，但项目既有约束是不能误杀用户自己打开的系统 Chrome。因此按钮建议命名为：

```text
关闭托管 Chrome
```

或：

```text
停止托管浏览器
```

该按钮只调用现有 `App.closeManagedChrome()`，只关闭本程序管理的 Chrome。

如果未来真的需要“杀掉系统全部 Chrome”，必须作为单独危险功能处理，不能和普通关闭托管浏览器混用。

### 推荐布局

全局工具栏建议分组：

```text
流程    [流程下拉.................] [保存]
浏览器  [启动 Chrome] [关闭托管 Chrome]
执行    [运行] [单步/下一步] [停止当前流程]
环境    [环境下拉.................]
进度    第 2/6 步 · 输入用户名 [==========      ]
```

如果一行太挤，可以拆成两行：

```text
第一行：流程组 + 浏览器组 + 环境组
第二行：执行组 + 进度
```

### 按钮状态

- `启动 Chrome`
  - Chrome 未启动：启用
  - Chrome 启动中/已启动：禁用
- `关闭托管 Chrome`
  - Chrome 未启动：禁用
  - Chrome 启动中/已启动：启用
- `运行`
  - 空闲且有当前流程：启用
  - 正在运行：禁用
- `单步/下一步`
  - 空闲：显示 `单步执行`
  - 单步中：显示 `下一步`
- `停止当前流程`
  - 空闲：可见但禁用
  - 完整运行/单步运行中：启用

不要用 `Hide()` 隐藏停止按钮，因为按钮出现/消失会导致布局跳动。使用 `Enable()` / `Disable()`。

### 运行详情页同步

当 `关闭托管 Chrome` 放到全局工具栏后，运行详情页应删除重复入口：

- 删除底部 `关闭本程序启动的 Chrome` 主按钮。
- 删除 `更多` 菜单中的关闭 Chrome 项。
- 运行详情页只保留日志、摘要、产物和低频诊断操作。

### 验收标准

- 任意 Tab 下都能看到：
  - 启动 Chrome
  - 关闭托管 Chrome
  - 运行
  - 单步/下一步
  - 停止当前流程
- 关闭托管 Chrome 和启动 Chrome 在同一组。
- 停止当前流程常驻，空闲禁用，运行中启用。
- 关闭托管 Chrome 不影响用户手动打开的 Chrome。

## P0-3. 环境配置必须支持导入和导出

### 用户需求

环境配置需要支持导入和导出，方便在不同机器、项目目录、测试环境之间迁移环境变量。

### 当前实现参考

当前 `internal/ui/env_panel.go` 中已经有类似入口：

```go
fyne.NewMenuItemWithIcon("导入配置", theme.DownloadIcon(), func() { p.showImportEnvDialog() })
fyne.NewMenuItemWithIcon("导出配置", theme.UploadIcon(), func() { p.showExportEnvDialog() })
```

后续 Agent 需要确认这些功能真实可用，并在 UI 中做成容易发现的一等功能。

### 推荐方案

1. 在“环境配置”Tab 顶部提供明显入口：

```text
[导入配置] [导出配置]
```

也可以放在环境操作菜单中，但页面上必须有清晰可见的入口或“更多”按钮。

2. 导出：
   - 复用 `EnvRepo.Export()`。
   - 默认文件名建议：`go-chrome-env-config.json`。
   - 如果导出内容包含敏感变量值，必须弹窗提示。

3. 导入：
   - 复用 `EnvRepo.Import()`。
   - 导入成功后刷新：
     - 环境列表
     - 当前变量列表
     - 全局环境下拉
     - 历史筛选环境列表
   - 导入失败要显示错误，不能破坏现有环境配置。

4. 重名环境策略必须明确：
   - 推荐：不覆盖现有环境，重名环境自动追加后缀，例如 `默认环境 导入`。

5. 敏感变量策略必须明确：
   - 如果导出明文敏感变量，导出前必须提示。
   - 可选增强：提供“包含敏感变量值”复选框。

### 验收标准

- 用户能在“环境配置”Tab 中找到导入/导出。
- 导出得到可读 JSON。
- 导入刚导出的 JSON 后，环境和变量能恢复。
- 导入后全局环境下拉同步刷新。
- 导入失败不会破坏已有配置。
- 敏感变量导出前有明确提示或明确策略说明。

## P1-1. 继续做视觉精修

### 优化方向

1. 全局工具栏分组更清楚，避免所有按钮堆成一排。
2. 流程下拉、环境下拉设置稳定宽度，长文本截断。
3. 运行进度显示不要挤压按钮。
4. 环境配置页变量表更像管理工具：
   - KEY
   - VALUE
   - 敏感
   - 说明
   - 操作
5. 路径、XPath、长流程名全部截断，并提供复制能力。
6. 主操作和危险操作的颜色/权重区分清楚。

## 推荐实施顺序

1. 先解决 JetBrains Code 风格 CJK 字体和字重。
2. 调整全局工具栏，把关闭托管 Chrome、停止当前流程放到常用区域。
3. 删除运行详情页中的重复关闭浏览器入口。
4. 强化环境配置导入/导出入口和验收。
5. 优化工具栏分组、下拉宽度、变量表和整体间距。

## 测试与验收清单

- `go test ./...`
- Windows 实机检查字体：
  - 顶部状态栏中文
  - Tab 中文
  - 全局工具栏按钮中文
  - 环境配置表格中文
  - 日志中文
- `fc-scan` 或等价工具确认最终 UI 字体覆盖 CJK。
- 空闲时停止当前流程按钮可见但禁用。
- 运行时停止当前流程按钮启用。
- 启动 Chrome 后关闭托管 Chrome 按钮启用。
- 关闭托管 Chrome 不影响用户手动打开的 Chrome。
- 环境配置能导出 JSON。
- 导入环境配置后环境列表、变量列表、全局环境下拉同步刷新。

## 注意事项

- 不要提交 `data/`、`logs/`、`chrome/`、`go-chrome` 二进制或运行产物。
- 字体文件必须确认许可证，不能提交来源不明的字体。
- 不要在程序启动时联网下载字体或模板。
- 不要默认杀掉系统全部 Chrome 进程。
