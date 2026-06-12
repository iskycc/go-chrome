# UI 二次优化设计文档

## 背景

`Todo1.md` 优化已完成，但用户反馈：
1. 中英文字体没有真正统一（Cascadia Code SemiLight 不含中文，系统 fallback 导致割裂）。
2. “停止当前流程”和“关闭托管 Chrome”应和“启动 Chrome”一样放在全局工具栏。
3. 全局工具栏需要更强的视觉分组和布局稳定性。
4. 流程下拉同名流程会互相覆盖。
5. 环境配置页变量列表应更像管理工具，导入/导出需要更明确。

## 设计原则

- 中英文必须由同一个字体资源渲染，不再依赖系统 fallback。
- 常用操作必须在任何 tab 可见且位置固定。
- 危险操作需要明确命名和稳定位置，但不默认执行。
- 布局在 1280x720 和 1400x900 下都稳定。

## 方案

### 字体方案（选定）

使用微软官方 **Cascadia Next SC** 可变字体作为全局 UI 字体：
- 来源：`https://github.com/microsoft/cascadia-code/releases/download/cascadia-next/CascadiaNextSC.wght.ttf`
- 许可：SIL Open Font License 1.1（© Microsoft Corporation）
- 覆盖：ASCII、拉丁、数字、GB2312 扩展简体中文、中文标点、常用符号。
- `fc-scan` 确认包含 `4e00-9fff` 区段。

保留原始 `CascadiaCode-SemiLight.ttf` 作为代码场景字体（日志、XPath、模板输入），但 UI 正文字体统一使用 Cascadia Next SC。

### 全局工具栏

布局改为两行或清晰分组：

```text
第一行：流程 [下拉] [保存]    浏览器 [启动 Chrome] [关闭托管]    环境 [下拉]
第二行：执行 [运行] [单步/下一步] [停止当前流程]    进度 [第 2/6 步 · 名称] [======  ]
```

- 流程下拉：使用 `flowSelectOption{Label, ID}`，同名流程 Label 后附加 ID 前 6 位区分。
- 浏览器按钮组：启动 Chrome / 关闭托管 Chrome。
- 执行按钮组：运行 / 单步 / 停止当前流程（空闲时禁用，不隐藏）。
- 环境下拉：固定宽度，长名称截断。
- 进度：单独一行或固定区域，避免挤压按钮。

### 运行详情页

移除底部“关闭本程序启动的 Chrome”按钮和“更多”菜单中的关闭 Chrome 入口。保留：清空日志、复制日志、打开产物目录、浏览器下载配置跳转、日志、摘要、当前步骤、产物。

### 环境配置页

- 变量列表改为 `widget.Table`，列：KEY、VALUE、敏感、说明、操作。
- VALUE 和说明截断显示。
- 导入/导出按钮放在顶部右侧，不再只藏在菜单中。
- 导出包含敏感变量时弹窗提示。
- 导入失败弹窗错误，不破坏现有配置。

## 接口变更

- `assets/embed.go`：新增 `AppUIFont()` 返回 Cascadia Next SC；`CodeFont()` 返回原始 Cascadia Code SemiLight。
- `internal/ui/theme.go`：`Font()` 返回 `assets.AppUIFont()`。
- `internal/ui/global_toolbar.go`：新增 `stopChromeBtn`，重组布局，修复同名流程处理。
- `internal/ui/run_panel.go`：移除关闭 Chrome 入口。
- `internal/ui/env_panel.go`：变量列表改为表格，强化导入/导出入口。
- 文档：`README.md`、`USER_GUIDE.md`、`FAQ.md`、`problem.md` 同步更新。

## 验收

- `fc-scan` 确认 UI 字体覆盖 `4e00-9fff`。
- `go build -mod=readonly ./...` 通过。
- 核心非 GUI 测试通过。
- 同名流程下拉可区分。
- 全局工具栏在 1280x720 下可用。
