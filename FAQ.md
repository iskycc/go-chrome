# Go Chrome 自动化编排工具 — 常见问题

## 1. 首次启动时为什么需要下载 Chrome？

本工具使用 **Chrome for Testing** 作为自动化浏览器，不依赖您电脑上已安装的 Chrome。首次启动时会自动从官方源下载 Stable 版本的 Chrome for Testing，解压到 `./chrome` 目录中。下载完成后即可重复使用，无需每次都下载。

## 2. 如何配置自定义 Chrome 下载源？

编辑 `data/app-config.json`，修改 `chrome` 部分：

```json
{
  "chrome": {
    "downloadSource": "custom_url",
    "customDownloadURL": "https://your-company.com/chrome-for-testing-win64.zip",
    "customDownloadSHA256": "abc123...",
    "fallbackToOfficial": true
  }
}
```

- `customDownloadURL`：指向 Windows 64 位 Chrome for Testing 的 `.zip` 文件。
- `customDownloadSHA256`：（可选）用于校验下载文件完整性。
- `fallbackToOfficial`：自定义下载失败时是否回退到官方 Stable 源。

## 3. 如何离线使用？

在有网络的环境先启动一次，等待 Chrome 下载完成。然后将整个项目目录（包含 `./chrome` 子目录）复制到离线环境即可。Chrome 包和流程文件都保存在本地，不需要联网。

## 4. 流程文件保存在哪里？

每个流程保存为独立的 JSON 文件，位于 `./data/flows/{flow-id}.json`。您可以：
- 直接复制这些文件进行备份或迁移。
- 使用 GUI 的"导入"/"导出"功能。
- 在版本控制中管理流程文件。

## 5. 输入模板 `${...}` 是什么？

输入模板用于在自动化执行时动态生成数据，支持：
- 数字范围：`SP${11000-11099}` → `SP11042`
- 固定位数：`SHOP-${0001-9999}` → `SHOP-0042`
- 枚举随机：`${dev|test|stage}` → `test`
- 随机字符串：`${alnum:8}` → `a3B9xK2m`
- 变量复用：`${var:user=SP${11000-11099}}` 后可用 `${var:user}` 读取同一值

在步骤属性编辑器的"输入内容"框旁有"预览"和"校验"按钮，可提前查看生成效果。

## 6. 单步执行和运行整个流程有什么区别？

- **运行整个流程**：从第 1 步（或选中步骤）开始连续执行到结束。
- **单步执行**：点击"单步执行"初始化环境，然后每次点击"下一步"执行一个步骤。适合调试和观察每一步的效果。

## 7. 执行失败时如何排查？

失败时会自动记录：
- 错误信息（显示在底部日志面板）
- 失败截图（保存在 `./data/run-history/{flow-id}/{timestamp}/`）
- 页面 HTML 快照（同一目录）

您可以在"执行历史"面板中查看每次运行的结果，并打开对应的截图和 HTML 文件进行分析。

## 8. 如何修改流程的 XPath？

在步骤列表中选中步骤，右侧"步骤属性"面板中的"目标"字段即为 XPath。支持标准的 XPath 1.0 语法，例如：
- `//button[contains(., '登录')]`
- `//input[@id='username']`
- `//*[@data-testid='submit']`

## 9. 敏感数据（如密码）会保存到日志吗？

默认不会。步骤属性中有"日志脱敏"选项，勾选后该步骤的输入值在日志中显示为 `[输入值已脱敏]`。此外，全局配置 `runner.maskInputValueInLogs` 默认为 `true`，所有输入值默认只显示前几位。

## 10. 如何清理旧的运行历史？

运行历史会自动保留（默认 14 天），超过保留期限的历史记录会在程序启动时自动清理。您也可以在配置中修改 `app.logRetentionDays` 调整保留天数。

## 11. 为什么程序退出后 Chrome 还在运行？

如果 Chrome 是由本工具启动的，程序退出时会尝试关闭它。但如果 Chrome 是之前独立启动的，或者进程被其他程序占用，可能无法自动关闭。您可以手动结束 `chrome.exe` 进程，或在配置中将 `app.closeManagedChromeOnExit` 设为 `true`（默认已启用）。

## 12. 支持 Windows 32 位吗？

当前优先支持 Windows 64 位。Chrome for Testing 官方提供 win32 包，但下载逻辑目前默认查找 win64。如需 32 位支持，可配置自定义下载源指向 win32 包。

## 13. 如何添加自定义操作类型？

当前支持的操作类型在 `internal/flow/model.go` 中定义。如需扩展，需要：
1. 在 `flow.StepType` 中添加新类型。
2. 在 `internal/runner/actions.go` 的 `ExecuteStep` 中实现对应逻辑。
3. 在 `internal/ui/labels.go` 中添加中文标签。
4. 在 `internal/ui/step_editor.go` 中更新表单展示逻辑。

## 14. 数据目录结构说明

```
./data/
  app-config.json          # 应用配置
  recent-flows.json        # 最近打开流程
  chrome-profile/          # Chrome 用户数据
  flows/                   # 流程 JSON 文件
  run-history/             # 执行历史与截图
./logs/
  2026-06-05.log           # 按日期滚动的日志
./chrome/
  chrome.exe               # Chrome for Testing
  chrome-version.json      # 版本清单
```
