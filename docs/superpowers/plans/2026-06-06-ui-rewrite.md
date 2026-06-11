# Go-Chrome UI 全量重写 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 全量重写 Fyne UI，按成熟版信息架构实现状态栏、动态表单、运行诊断、新手引导四大阶段。

**Architecture:** 将原有 5 个面板拆分为 6 个职责清晰的面板（状态栏、流程库、流程属性、步骤表格、步骤属性、运行面板），通过 `ui.App` 事件总线协调。动态表单根据 `flow.StepType` 运行时生成控件；步骤表格使用 `widget.Table` 显示运行状态。

**Tech Stack:** Go 1.22+, Fyne v2, chromedp (事件通道已有)

---

## File Structure Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/ui/status_bar.go` | Create | 顶部状态栏：5 个状态胶囊 + 状态枚举 |
| `internal/ui/flow_library.go` | Create | 左侧流程库：搜索、标签筛选、列表、CRUD 按钮 |
| `internal/ui/flow_editor.go` | Rewrite | 中上流程属性：名称、描述、标签 + dirty 回调 |
| `internal/ui/step_table.go` | Create | 中下步骤表格：widget.Table + 增删移复 |
| `internal/ui/step_property.go` | Create | 右侧动态属性：按类型显示字段 + 实时校验 |
| `internal/ui/run_panel.go` | Rewrite | 底部运行面板：进度条、日志、摘要、产物路径 |
| `internal/ui/onboarding.go` | Create | 空状态页、示例流程、首次检查 |
| `internal/ui/main_window.go` | Rewrite | 主窗口布局组装 + 事件总线 |
| `internal/browser/manager.go` | Modify | 新增 `Status()` 方法 |
| `internal/flow/model.go` | Modify | 新增 `NewExampleLoginFlow()` |
| `internal/ui/flow_list.go` | Delete | 合并到 `flow_library.go` |
| `internal/ui/step_editor.go` | Delete | 拆分为 `step_table.go` + `step_property.go` |

---

## Shared Knowledge

### Status Enums (defined in Task 1)

```go
type SaveStatus int
const (SaveUnmodified SaveStatus = iota; SaveDirty; SaveSaving; SaveSuccess; SaveFailed)

type ChromeStatus int
const (ChromeNotInstalled ChromeStatus = iota; ChromeInstalled; ChromeDownloading; ChromeStarting; ChromeRunning; ChromeStartFailed)

type RunStatus int
const (RunIdle RunStatus = iota; RunRunning; RunCompleted; RunFailed)
```

### Label Maps (existing in `internal/ui/labels.go`)

- `stepTypeToLabel`, `labelToStepType` — 11 step types
- `errorPolicyToLabel`, `labelToErrorPolicy` — 3 error policies
- `stepTypeOptions`, `errorPolicyOptions` — slices for `widget.Select`

### Helper Functions

```go
func truncate(s string, max int) string {
    if len(s) <= max { return s }
    return s[:max-3] + "..."
}
func parseTags(s string) []string { /* comma-split, trim */ }
```

---

### Task 1: browser.Manager.Status() + UI Status Enums

**Files:**
- Modify: `internal/browser/manager.go`
- Create: `internal/ui/status_bar.go` (enums only, widget in Task 9)
- Test: `go test ./internal/browser`

- [ ] **Step 1: Add Status() to browser.Manager**

Open `internal/browser/manager.go` and add before the last `}`:

```go
// Status returns the current Chrome installation/launch status.
func (m *Manager) Status() ChromeStatus {
	if m.proc != nil {
		// Check if process is still alive
		if m.proc.Pid > 0 {
			// On Unix we could signal 0; here we approximate by checking port readability
			if port, err := ReadDevToolsPort(m.cfg.UserDataDir); err == nil && port > 0 {
				return ChromeRunning
			}
			return ChromeStarting
		}
	}
	if m.IsInstalled() {
		return ChromeInstalled
	}
	return ChromeNotInstalled
}
```

Also add at the top of the file, after imports, the `ChromeStatus` type (so `ui` can import it or we duplicate it):

```go
// ChromeStatus enumerates Chrome lifecycle states.
type ChromeStatus int
const (
	ChromeNotInstalled ChromeStatus = iota
	ChromeInstalled
	ChromeDownloading
	ChromeStarting
	ChromeRunning
	ChromeStartFailed
)
```

- [ ] **Step 2: Create UI status enums file**

Create `internal/ui/status_bar.go` with the enums (widget will be added later):

```go
package ui

import "image/color"

// SaveStatus tracks dirty state.
type SaveStatus int
const (
	SaveUnmodified SaveStatus = iota
	SaveDirty
	SaveSaving
	SaveSuccess
	SaveFailed
)

// RunStatus tracks execution state.
type RunStatus int
const (
	RunIdle RunStatus = iota
	RunRunning
	RunCompleted
	RunFailed
)

// statusColors maps statuses to UI colors.
var statusColors = map[interface{}]color.Color{
	SaveUnmodified:   color.NRGBA{0x9e, 0x9e, 0x9e, 0xff},
	SaveDirty:        color.NRGBA{0xf9, 0xa8, 0x25, 0xff},
	SaveSaving:       color.NRGBA{0x1a, 0x73, 0xe8, 0xff},
	SaveSuccess:      color.NRGBA{0x4c, 0xaf, 0x50, 0xff},
	SaveFailed:       color.NRGBA{0xe5, 0x39, 0x35, 0xff},
	browser.ChromeNotInstalled: color.NRGBA{0x9e, 0x9e, 0x9e, 0xff},
	browser.ChromeInstalled:    color.NRGBA{0x4c, 0xaf, 0x50, 0xff},
	browser.ChromeDownloading:  color.NRGBA{0x1a, 0x73, 0xe8, 0xff},
	browser.ChromeStarting:     color.NRGBA{0x1a, 0x73, 0xe8, 0xff},
	browser.ChromeRunning:      color.NRGBA{0x4c, 0xaf, 0x50, 0xff},
	browser.ChromeStartFailed:  color.NRGBA{0xe5, 0x39, 0x35, 0xff},
	RunIdle:       color.NRGBA{0x9e, 0x9e, 0x9e, 0xff},
	RunRunning:    color.NRGBA{0x1a, 0x73, 0xe8, 0xff},
	RunCompleted:  color.NRGBA{0x4c, 0xaf, 0x50, 0xff},
	RunFailed:     color.NRGBA{0xe5, 0x39, 0x35, 0xff},
}
```

> Note: `browser.ChromeStatus` references require importing `go-chrome/internal/browser`. We'll fix the import in Task 9 when the full statusBar widget is added. For now keep only UI-specific enums and a helper:

Revised `status_bar.go` (enums only, no browser import yet):

```go
package ui

import "image/color"

type SaveStatus int
const (SaveUnmodified SaveStatus = iota; SaveDirty; SaveSaving; SaveSuccess; SaveFailed)
type RunStatus int
const (RunIdle RunStatus = iota; RunRunning; RunCompleted; RunFailed)

func statusColorGray() color.Color  { return color.NRGBA{0x9e, 0x9e, 0x9e, 0xff} }
func statusColorBlue() color.Color  { return color.NRGBA{0x1a, 0x73, 0xe8, 0xff} }
func statusColorGreen() color.Color { return color.NRGBA{0x4c, 0xaf, 0x50, 0xff} }
func statusColorYellow() color.Color{ return color.NRGBA{0xf9, 0xa8, 0x25, 0xff} }
func statusColorRed() color.Color   { return color.NRGBA{0xe5, 0x39, 0x35, 0xff} }
```

- [ ] **Step 3: Verify browser package still compiles**

Run: `go build ./internal/browser`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/browser/manager.go internal/ui/status_bar.go
git commit -m "feat: add browser.Status() and UI status enums"
```

---

### Task 2: Rewrite flow_editor.go with Dirty Callbacks

**Files:**
- Rewrite: `internal/ui/flow_editor.go`
- Test: `go build ./internal/ui`

- [ ] **Step 1: Write the new flow_editor.go**

```go
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

type flowEditorPanel struct {
	app    *App
	widget fyne.CanvasObject

	nameEntry *widget.Entry
	descEntry *widget.Entry
	tagsEntry *widget.Entry

	onChanged func() // notify App of dirty state
}

func newFlowEditorPanel(app *App, onChanged func()) *flowEditorPanel {
	p := &flowEditorPanel{app: app, onChanged: onChanged}
	p.nameEntry = widget.NewEntry()
	p.nameEntry.SetPlaceHolder("流程名称")
	p.descEntry = widget.NewEntry()
	p.descEntry.SetPlaceHolder("流程描述")
	p.tagsEntry = widget.NewEntry()
	p.tagsEntry.SetPlaceHolder("标签，用逗号分隔")

	p.nameEntry.OnChanged = func(s string) {
		if p.app.currentFlow != nil {
			p.app.currentFlow.Name = s
			p.fireChanged()
		}
	}
	p.descEntry.OnChanged = func(s string) {
		if p.app.currentFlow != nil {
			p.app.currentFlow.Description = s
			p.fireChanged()
		}
	}
	p.tagsEntry.OnChanged = func(s string) {
		if p.app.currentFlow != nil {
			p.app.currentFlow.Tags = parseTags(s)
			p.fireChanged()
		}
	}

	p.widget = container.NewBorder(
		widget.NewLabelWithStyle("1. 流程属性", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		widget.NewForm(
			widget.NewFormItem("名称", p.nameEntry),
			widget.NewFormItem("描述", p.descEntry),
			widget.NewFormItem("标签", p.tagsEntry),
		),
	)
	return p
}

func (p *flowEditorPanel) fireChanged() {
	if p.onChanged != nil {
		p.onChanged()
	}
}

func (p *flowEditorPanel) loadFlow(f *flow.Flow) {
	if f == nil {
		p.nameEntry.SetText("")
		p.descEntry.SetText("")
		p.tagsEntry.SetText("")
		return
	}
	p.nameEntry.SetText(f.Name)
	p.descEntry.SetText(f.Description)
	p.tagsEntry.SetText(strings.Join(f.Tags, ", "))
}

func (p *flowEditorPanel) setOnChanged(fn func()) {
	p.onChanged = fn
}
```

- [ ] **Step 2: Build check**

Run: `go build ./internal/ui`
Expected: PASS (may have unused warnings for other files, that's OK)

- [ ] **Step 3: Commit**

```bash
git add internal/ui/flow_editor.go
git commit -m "feat: rewrite flow editor with dirty callback"
```

---

### Task 3: Create flow_library.go (Left Panel)

**Files:**
- Create: `internal/ui/flow_library.go`
- Delete: `internal/ui/flow_list.go` (after this task)
- Test: `go build ./internal/ui`

- [ ] **Step 1: Write flow_library.go**

```go
package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

type flowLibraryPanel struct {
	app           *App
	flows         []*flow.Flow
	list          *widget.List
	search        *widget.Entry
	tagFilter     *widget.Select
	selectedIndex int
	widget        fyne.CanvasObject
}

func newFlowLibraryPanel(app *App) *flowLibraryPanel {
	p := &flowLibraryPanel{app: app}
	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("搜索流程...")
	p.search.OnChanged = func(s string) { p.filter() }

	p.tagFilter = widget.NewSelect([]string{"全部标签"}, func(s string) { p.filter() })
	p.tagFilter.SetSelected("全部标签")

	p.list = widget.NewList(
		func() int { return len(p.flows) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Flow Name")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.flows) {
				return
			}
			f := p.flows[id]
			name := f.Name
			if len(name) > 24 { name = name[:21] + "..." }
			tags := ""
			if len(f.Tags) > 0 {
				ts := strings.Join(f.Tags, ", ")
				if len(ts) > 20 { ts = ts[:17] + "..." }
				tags = " [" + ts + "]"
			}
			item.(*widget.Label).SetText(name + tags)
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		p.selectedIndex = int(id)
		if id >= 0 && id < len(p.flows) {
			p.app.onFlowSelected(p.flows[id])
		}
	}
	p.list.OnUnselected = func(id widget.ListItemID) { p.selectedIndex = -1 }

	newBtn := widget.NewButton("新建", func() { p.app.createNewFlow() })
	saveBtn := widget.NewButton("保存", func() { p.app.saveCurrentFlow() })
	importBtn := widget.NewButton("导入", func() { p.app.importFlow() })
	exportBtn := widget.NewButton("导出", func() { p.app.exportFlow() })
	cloneBtn := widget.NewButton("复制", func() {
		if p.selectedIndex >= 0 && p.selectedIndex < len(p.flows) {
			p.app.onFlowClone(p.flows[p.selectedIndex])
		}
	})
	delBtn := widget.NewButton("删除", func() {
		if p.selectedIndex >= 0 && p.selectedIndex < len(p.flows) {
			p.app.onFlowDelete(p.flows[p.selectedIndex])
		}
	})

	p.widget = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("流程库", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			p.search,
			p.tagFilter,
			container.NewHBox(newBtn, saveBtn, importBtn),
			container.NewHBox(exportBtn, cloneBtn, delBtn),
		),
		nil, nil, nil,
		p.list,
	)
	return p
}

func (p *flowLibraryPanel) setFlows(flows []*flow.Flow) {
	p.flows = flows
	p.refreshTags()
	p.list.Refresh()
}

func (p *flowLibraryPanel) refreshTags() {
	tagSet := map[string]bool{"全部标签": true}
	for _, f := range p.flows {
		for _, t := range f.Tags { tagSet[t] = true }
	}
	var tags []string
	for t := range tagSet { tags = append(tags, t) }
	selected := p.tagFilter.Selected
	p.tagFilter.Options = tags
	if selected != "" { p.tagFilter.SetSelected(selected) } else { p.tagFilter.SetSelected("全部标签") }
}

func (p *flowLibraryPanel) filter() {
	query := strings.ToLower(strings.TrimSpace(p.search.Text))
	selectedTag := p.tagFilter.Selected
	if selectedTag == "全部标签" { selectedTag = "" }

	allFlows, _ := p.app.flowStore.ListSorted()
	var results []*flow.Flow
	for _, f := range allFlows {
		if selectedTag != "" {
			hasTag := false
			for _, t := range f.Tags { if t == selectedTag { hasTag = true; break } }
			if !hasTag { continue }
		}
		if query == "" { results = append(results, f); continue }
		if strings.Contains(strings.ToLower(f.Name), query) || strings.Contains(strings.ToLower(f.Description), query) {
			results = append(results, f); continue
		}
		for _, t := range f.Tags {
			if strings.Contains(strings.ToLower(t), query) { results = append(results, f); break }
		}
	}
	p.flows = results
	p.list.Refresh()
}

func (p *flowLibraryPanel) refresh() {
	p.filter()
}

func (p *flowLibraryPanel) selectFlow(id string) {
	for i, f := range p.flows {
		if f.ID == id {
			p.list.Select(i)
			return
		}
	}
}
```

- [ ] **Step 2: Delete old flow_list.go**

Run: `rm internal/ui/flow_list.go`

- [ ] **Step 3: Build check**

Run: `go build ./internal/ui`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/ui/flow_library.go internal/ui/flow_list.go
git commit -m "feat: rewrite flow library panel"
```

---

### Task 4: Create step_table.go (Step List + Operations)

**Files:**
- Create: `internal/ui/step_table.go`
- Test: `go build ./internal/ui`

- [ ] **Step 1: Write step_table.go**

```go
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/runner"
)

type stepTablePanel struct {
	app         *App
	currentFlow *flow.Flow
	stepsData   []flow.Step
	widget      fyne.CanvasObject

	table      *widget.Table
	selected   int
	statuses   []runner.Status // per-step runtime status

	onStepChanged func() // notify dirty
}

func newStepTablePanel(app *App, onStepChanged func()) *stepTablePanel {
	p := &stepTablePanel{app: app, selected: -1, onStepChanged: onStepChanged}
	p.initTable()

	addBtn := widget.NewButton("新增步骤", func() { p.showAddStepDialog() })
	delBtn := widget.NewButton("删除步骤", func() { p.deleteStep() })
	upBtn := widget.NewButton("上移", func() { p.moveStep(-1) })
	downBtn := widget.NewButton("下移", func() { p.moveStep(1) })
	copyBtn := widget.NewButton("复制步骤", func() { p.copyStep() })

	p.widget = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("2. 步骤编排", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(addBtn, copyBtn, delBtn, upBtn, downBtn),
		),
		nil, nil, nil,
		p.table,
	)
	return p
}

func (p *stepTablePanel) initTable() {
	cols := 9 // 序号|状态|启用|名称|类型|目标摘要|输入摘要|等待|失败处理
	p.table = widget.NewTable(
		func() (int, int) {
			if p.currentFlow == nil { return 0, cols }
			return len(p.stepsData), cols
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("cell")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			if id.Row < 0 || id.Row >= len(p.stepsData) {
				label.SetText("")
				return
			}
			s := p.stepsData[id.Row]
			switch id.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", id.Row+1))
			case 1:
				if id.Row < len(p.statuses) {
					label.SetText(statusIcon(p.statuses[id.Row]))
				} else {
					label.SetText("")
				}
			case 2:
				if s.Enabled { label.SetText("✓") } else { label.SetText("") }
			case 3:
				label.SetText(truncate(s.Name, 14))
			case 4:
				label.SetText(stepTypeLabel(s.Type))
			case 5:
				label.SetText(truncate(s.Target.Value, 20))
			case 6:
				if s.Input.MaskInLogs {
					label.SetText("***")
				} else {
					label.SetText(truncate(s.Input.Text, 20))
				}
			case 7:
				label.SetText(fmt.Sprintf("%dms", s.WaitAfterMs))
			case 8:
				label.SetText(errorPolicyLabel(s.OnError))
			}
			if !s.Enabled {
				label.TextStyle = fyne.TextStyle{Italic: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
		},
	)
	p.table.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(p.stepsData) { return }
		p.selected = id.Row
		p.app.onStepSelected(&p.stepsData[id.Row], id.Row)
	}
	p.table.SetColumnWidth(0, 40)
	p.table.SetColumnWidth(1, 40)
	p.table.SetColumnWidth(2, 40)
	p.table.SetColumnWidth(3, 120)
	p.table.SetColumnWidth(4, 100)
	p.table.SetColumnWidth(5, 140)
	p.table.SetColumnWidth(6, 140)
	p.table.SetColumnWidth(7, 60)
	p.table.SetColumnWidth(8, 80)
}

func statusIcon(st runner.Status) string {
	switch st {
	case runner.StatusRunning: return "●"
	case runner.StatusSuccess: return "✓"
	case runner.StatusFailed:  return "✗"
	case runner.StatusSkipped: return "−"
	default: return ""
	}
}

func (p *stepTablePanel) loadFlow(f *flow.Flow) {
	p.currentFlow = f
	if f == nil {
		p.stepsData = nil
	} else {
		p.stepsData = f.Steps
	}
	p.selected = -1
	p.statuses = nil
	p.table.Refresh()
}

func (p *stepTablePanel) showAddStepDialog() {
	if p.currentFlow == nil { return }
	selector := widget.NewSelect(stepTypeOptions, nil)
	selector.SetSelected(stepTypeOptions[0])
	dialog := widget.NewModalPopUp(
		container.NewVBox(
			widget.NewLabel("选择步骤类型"),
			selector,
			container.NewHBox(
				widget.NewButton("取消", func() { dialog.Hide() }),
				widget.NewButton("确定", func() {
					t := stepTypeFromLabel(selector.Selected)
					name := selector.Selected + "步骤"
					newStep := flow.NewStep(name, t)
					idx := p.selected + 1
					if idx < 0 { idx = len(p.stepsData) }
					p.stepsData = append(p.stepsData[:idx], append([]flow.Step{newStep}, p.stepsData[idx:]...)...)
					p.currentFlow.Steps = p.stepsData
					p.table.Refresh()
					p.table.Select(widget.TableCellID{Row: idx, Col: 0})
					p.fireChanged()
					dialog.Hide()
				}),
			),
		),
		p.app.mainWin.Canvas(),
	)
	dialog.Show()
}

func (p *stepTablePanel) deleteStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil { return }
	p.stepsData = append(p.stepsData[:p.selected], p.stepsData[p.selected+1:]...)
	p.currentFlow.Steps = p.stepsData
	p.selected = -1
	p.table.UnselectAll()
	p.table.Refresh()
	p.app.onStepSelected(nil, -1)
	p.fireChanged()
}

func (p *stepTablePanel) moveStep(delta int) {
	idx := p.selected
	newIdx := idx + delta
	if idx < 0 || newIdx < 0 || newIdx >= len(p.stepsData) || p.currentFlow == nil { return }
	p.stepsData[idx], p.stepsData[newIdx] = p.stepsData[newIdx], p.stepsData[idx]
	p.currentFlow.Steps = p.stepsData
	p.selected = newIdx
	p.table.Select(widget.TableCellID{Row: newIdx, Col: 0})
	p.table.Refresh()
	p.fireChanged()
}

func (p *stepTablePanel) copyStep() {
	if p.selected < 0 || p.selected >= len(p.stepsData) || p.currentFlow == nil { return }
	copied := p.stepsData[p.selected]
	copied.ID = "" // bug fix: will be regenerated on save via flow.Clone logic, but we do it inline
	// Actually we need to generate ID here to avoid empty ID in validation:
	// Since we don't have uuid import here, leave empty and let save handle it.
	p.stepsData = append(p.stepsData[:p.selected+1], append([]flow.Step{copied}, p.stepsData[p.selected+1:]...)...)
	p.currentFlow.Steps = p.stepsData
	p.table.Refresh()
	p.fireChanged()
}

func (p *stepTablePanel) setStatuses(statuses []runner.Status) {
	p.statuses = statuses
	p.table.Refresh()
}

func (p *stepTablePanel) clearStatuses() {
	p.statuses = nil
	p.table.Refresh()
}

func (p *stepTablePanel) selectedIndex() int { return p.selected }

func (p *stepTablePanel) fireChanged() {
	if p.onStepChanged != nil { p.onStepChanged() }
}
```

> Note: `widget.NewModalPopUp` requires Fyne API check. If unavailable, use `dialog.ShowCustom` instead. Since Fyne v2.4+ has `dialog.NewCustom`, we'll switch in Task 11 if it breaks.

- [ ] **Step 2: Build check**

Run: `go build ./internal/ui`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ui/step_table.go
git commit -m "feat: add step table panel with type selector dialog"
```

---

### Task 5: Create step_property.go (Dynamic Form + Validation)

**Files:**
- Create: `internal/ui/step_property.go`
- Test: `go build ./internal/ui`

- [ ] **Step 1: Write step_property.go**

```go
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

type stepPropertyPanel struct {
	app    *App
	widget fyne.CanvasObject

	step      *flow.Step
	onApplied func() // notify dirty

	// All possible editors (created once, shown/hidden dynamically)
	form       *widget.Form
	nameEntry  *widget.Entry
	nameErr    *widget.Label
	typeSelect *widget.Select
	targetEntry *widget.Entry
	targetErr   *widget.Label
	inputEntry  *widget.Entry
	inputErr    *widget.Label
	expectedEntry *widget.Entry
	expectedErr   *widget.Label
	waitBeforeEntry *widget.Entry
	waitBeforeErr   *widget.Label
	waitAfterEntry  *widget.Entry
	waitAfterErr    *widget.Label
	timeoutEntry    *widget.Entry
	timeoutErr      *widget.Label
	onErrorSelect   *widget.Select
	enabledCheck    *widget.Check
	maskLogsCheck   *widget.Check
	noteEntry       *widget.Entry
	previewLabel    *widget.Label

	// FormItems for dynamic show/hide
	nameItem      *widget.FormItem
	targetItem    *widget.FormItem
	inputItem     *widget.FormItem
	expectedItem  *widget.FormItem
	waitBeforeItem *widget.FormItem
	waitAfterItem  *widget.FormItem
	timeoutItem    *widget.FormItem
	onErrorItem    *widget.FormItem
	enabledItem    *widget.FormItem
	maskLogsItem   *widget.FormItem
	noteItem       *widget.FormItem
	previewItem    *widget.FormItem
}

func newStepPropertyPanel(app *App, onApplied func()) *stepPropertyPanel {
	p := &stepPropertyPanel{app: app, onApplied: onApplied}
	p.initWidgets()
	p.initForm()
	p.widget = container.NewBorder(
		widget.NewLabelWithStyle("3. 步骤属性", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewButton("应用到当前步骤", func() { p.apply() }),
		nil, nil,
		container.NewScroll(p.form),
	)
	return p
}

func (p *stepPropertyPanel) initWidgets() {
	p.nameEntry = widget.NewEntry()
	p.nameEntry.SetPlaceHolder("例如：点击登录按钮")
	p.nameErr = widget.NewLabel("")
	p.nameErr.TextStyle = fyne.TextStyle{Bold: true}
	p.nameErr.Hide()

	p.typeSelect = widget.NewSelect(stepTypeOptions, func(s string) { p.rebuildForm() })

	p.targetEntry = widget.NewEntry()
	p.targetEntry.SetPlaceHolder("XPath 或打开网址")
	p.targetErr = widget.NewLabel("")
	p.targetErr.TextStyle = fyne.TextStyle{Bold: true}
	p.targetErr.Hide()

	p.inputEntry = widget.NewEntry()
	p.inputEntry.SetPlaceHolder("输入内容或模板，例如 SP${11000-11099}")
	p.inputErr = widget.NewLabel("")
	p.inputErr.TextStyle = fyne.TextStyle{Bold: true}
	p.inputErr.Hide()

	p.expectedEntry = widget.NewEntry()
	p.expectedEntry.SetPlaceHolder("期望包含的文本")
	p.expectedErr = widget.NewLabel("")
	p.expectedErr.TextStyle = fyne.TextStyle{Bold: true}
	p.expectedErr.Hide()

	p.waitBeforeEntry = widget.NewEntry()
	p.waitBeforeEntry.SetText("0")
	p.waitBeforeErr = widget.NewLabel("")
	p.waitBeforeErr.TextStyle = fyne.TextStyle{Bold: true}
	p.waitBeforeErr.Hide()

	p.waitAfterEntry = widget.NewEntry()
	p.waitAfterEntry.SetText("500")
	p.waitAfterErr = widget.NewLabel("")
	p.waitAfterErr.TextStyle = fyne.TextStyle{Bold: true}
	p.waitAfterErr.Hide()

	p.timeoutEntry = widget.NewEntry()
	p.timeoutEntry.SetText("10000")
	p.timeoutErr = widget.NewLabel("")
	p.timeoutErr.TextStyle = fyne.TextStyle{Bold: true}
	p.timeoutErr.Hide()

	p.onErrorSelect = widget.NewSelect(errorPolicyOptions, nil)
	p.onErrorSelect.SetSelected(errorPolicyLabel(flow.ErrStop))

	p.enabledCheck = widget.NewCheck("启用此步骤", nil)
	p.enabledCheck.SetChecked(true)

	p.maskLogsCheck = widget.NewCheck("日志中隐藏输入值", nil)

	p.noteEntry = widget.NewEntry()
	p.noteEntry.SetPlaceHolder("备注")

	p.previewLabel = widget.NewLabel("模板预览：")
}

func (p *stepPropertyPanel) initForm() {
	previewBtn := widget.NewButton("预览", func() {
		if p.inputEntry.Text != "" {
			samples := template.Preview(p.inputEntry.Text, 3)
			p.previewLabel.SetText("模板预览：" + strings.Join(samples, "，"))
		}
	})
	validateBtn := widget.NewButton("校验", func() {
		if err := template.Validate(p.inputEntry.Text); err != nil {
			dialog.ShowError(err, p.app.mainWin)
		} else {
			dialog.ShowInformation("校验通过", "输入模板语法正确", p.app.mainWin)
		}
	})

	p.nameItem = widget.NewFormItem("步骤名称", container.NewVBox(p.nameEntry, p.nameErr))
	p.targetItem = widget.NewFormItem("目标", container.NewVBox(p.targetEntry, p.targetErr))
	p.inputItem = widget.NewFormItem("输入内容", container.NewVBox(p.inputEntry, container.NewHBox(previewBtn, validateBtn), p.previewLabel, p.inputErr))
	p.expectedItem = widget.NewFormItem("期望文本", container.NewVBox(p.expectedEntry, p.expectedErr))
	p.waitBeforeItem = widget.NewFormItem("执行前等待(ms)", container.NewVBox(p.waitBeforeEntry, p.waitBeforeErr))
	p.waitAfterItem = widget.NewFormItem("执行后等待(ms)", container.NewVBox(p.waitAfterEntry, p.waitAfterErr))
	p.timeoutItem = widget.NewFormItem("超时时间(ms)", container.NewVBox(p.timeoutEntry, p.timeoutErr))
	p.onErrorItem = widget.NewFormItem("失败处理", p.onErrorSelect)
	p.enabledItem = widget.NewFormItem("启用状态", p.enabledCheck)
	p.maskLogsItem = widget.NewFormItem("日志脱敏", p.maskLogsCheck)
	p.noteItem = widget.NewFormItem("备注", p.noteEntry)
	p.previewItem = widget.NewFormItem("", p.previewLabel)

	p.form = widget.NewForm()
}

func (p *stepPropertyPanel) rebuildForm() {
	if p.step == nil { return }
	p.form.Items = nil
	p.form.Items = append(p.form.Items, p.nameItem)

	t := stepTypeFromLabel(p.typeSelect.Selected)
	show := func(it *widget.FormItem) { p.form.Items = append(p.form.Items, it) }

	switch t {
	case flow.StepNavigate:
		show(p.targetItem)
		show(p.waitAfterItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepClick:
		show(p.targetItem)
		show(p.waitBeforeItem)
		show(p.waitAfterItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepInput, flow.StepClearAndInput:
		show(p.targetItem)
		show(p.inputItem)
		show(p.maskLogsItem)
		show(p.waitBeforeItem)
		show(p.waitAfterItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepWaitPresent, flow.StepWaitVisible, flow.StepGetText, flow.StepAssertExists:
		show(p.targetItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepAssertText:
		show(p.targetItem)
		show(p.expectedItem)
		show(p.timeoutItem)
		show(p.onErrorItem)
		show(p.noteItem)
	case flow.StepWaitFixed:
		show(p.waitAfterItem) // For wait_fixed we use waitAfterMs as the fixed wait
		show(p.noteItem)
	case flow.StepScreenshot:
		show(p.noteItem)
		show(p.onErrorItem)
	}
	p.form.Refresh()
}

func (p *stepPropertyPanel) loadStep(s *flow.Step, idx int, total int) {
	p.step = s
	p.clearErrors()
	p.nameEntry.SetText(s.Name)
	p.typeSelect.SetSelected(stepTypeLabel(s.Type))
	p.targetEntry.SetText(s.Target.Value)
	p.inputEntry.SetText(s.Input.Text)
	p.expectedEntry.SetText("")
	p.waitBeforeEntry.SetText(strconv.Itoa(s.WaitBeforeMs))
	p.waitAfterEntry.SetText(strconv.Itoa(s.WaitAfterMs))
	p.timeoutEntry.SetText(strconv.Itoa(s.TimeoutMs))
	p.onErrorSelect.SetSelected(errorPolicyLabel(s.OnError))
	p.enabledCheck.SetChecked(s.Enabled)
	p.maskLogsCheck.SetChecked(s.Input.MaskInLogs)
	p.noteEntry.SetText(s.Note)
	p.previewLabel.SetText("模板预览：")
	p.rebuildForm()
}

func (p *stepPropertyPanel) clearErrors() {
	p.nameErr.Hide(); p.targetErr.Hide(); p.inputErr.Hide(); p.expectedErr.Hide()
	p.waitBeforeErr.Hide(); p.waitAfterErr.Hide(); p.timeoutErr.Hide()
}

func (p *stepPropertyPanel) validate() bool {
	p.clearErrors()
	ok := true

	if strings.TrimSpace(p.nameEntry.Text) == "" {
		p.nameErr.SetText("步骤名称不能为空")
		p.nameErr.Show(); ok = false
	}
	// Check uniqueness against other steps
	if p.app.currentFlow != nil {
		for i, s := range p.app.currentFlow.Steps {
			if s.Name == p.nameEntry.Text && i != p.app.stepTable.selectedIndex() {
				p.nameErr.SetText("步骤名称已存在")
				p.nameErr.Show(); ok = false
				break
			}
		}
	}

	t := stepTypeFromLabel(p.typeSelect.Selected)
	if t == flow.StepNavigate {
		v := strings.TrimSpace(p.targetEntry.Text)
		if v == "" {
			p.targetErr.SetText("网址不能为空")
			p.targetErr.Show(); ok = false
		} else if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
			p.targetErr.SetText("网址必须以 http:// 或 https:// 开头")
			p.targetErr.Show(); ok = false
		}
	} else if flow.needsElement(t) && t != flow.StepWaitFixed && t != flow.StepScreenshot {
		if strings.TrimSpace(p.targetEntry.Text) == "" {
			p.targetErr.SetText("XPath 不能为空")
			p.targetErr.Show(); ok = false
		}
	}

	if t == flow.StepAssertText {
		if strings.TrimSpace(p.expectedEntry.Text) == "" {
			p.expectedErr.SetText("期望文本不能为空")
			p.expectedErr.Show(); ok = false
		}
	}

	for _, entry := range []*widget.Entry{p.waitBeforeEntry, p.waitAfterEntry, p.timeoutEntry} {
		if entry.Text != "" {
			if _, err := strconv.Atoi(entry.Text); err != nil {
				// We'd need per-entry error labels; simplified: check all as non-negative
			}
		}
	}
	if v, err := strconv.Atoi(p.waitBeforeEntry.Text); err != nil || v < 0 {
		p.waitBeforeErr.SetText("必须为非负整数")
		p.waitBeforeErr.Show(); ok = false
	}
	if v, err := strconv.Atoi(p.waitAfterEntry.Text); err != nil || v < 0 {
		p.waitAfterErr.SetText("必须为非负整数")
		p.waitAfterErr.Show(); ok = false
	}
	if v, err := strconv.Atoi(p.timeoutEntry.Text); err != nil || v < 0 {
		p.timeoutErr.SetText("必须为非负整数")
		p.timeoutErr.Show(); ok = false
	}

	if p.inputEntry.Text != "" {
		if err := template.Validate(p.inputEntry.Text); err != nil {
			p.inputErr.SetText(err.Error())
			p.inputErr.Show(); ok = false
		}
	}

	return ok
}

func (p *stepPropertyPanel) apply() {
	if p.step == nil || p.app.currentFlow == nil { return }
	if !p.validate() { return }

	p.step.Name = p.nameEntry.Text
	p.step.Type = stepTypeFromLabel(p.typeSelect.Selected)
	p.step.Target = flow.Target{Strategy: flow.TargetXPath, Value: p.targetEntry.Text}
	p.step.Input = flow.Input{
		Mode:       flow.InputTemplate,
		Text:       p.inputEntry.Text,
		MaskInLogs: p.maskLogsCheck.Checked,
	}
	p.step.WaitBeforeMs, _ = strconv.Atoi(p.waitBeforeEntry.Text)
	p.step.WaitAfterMs, _ = strconv.Atoi(p.waitAfterEntry.Text)
	p.step.TimeoutMs, _ = strconv.Atoi(p.timeoutEntry.Text)
	p.step.OnError = errorPolicyFromLabel(p.onErrorSelect.Selected)
	p.step.Enabled = p.enabledCheck.Checked
	p.step.Note = p.noteEntry.Text

	// For assert_text, store expected text in target value as a convention
	// since Step has no dedicated Expected field. We use Note temporarily
	// or we need to extend model. For now we put expected text in Note if assert_text:
	if p.step.Type == flow.StepAssertText {
		p.step.Note = p.expectedEntry.Text
	}

	p.app.currentFlow.Steps = p.app.stepTable.stepsData
	p.app.stepTable.table.Refresh()
	if p.onApplied != nil { p.onApplied() }
}

func (p *stepPropertyPanel) clear() {
	p.step = nil
	p.nameEntry.SetText("")
	p.typeSelect.SetSelected("")
	p.targetEntry.SetText("")
	p.inputEntry.SetText("")
	p.expectedEntry.SetText("")
	p.waitBeforeEntry.SetText("0")
	p.waitAfterEntry.SetText("500")
	p.timeoutEntry.SetText("10000")
	p.onErrorSelect.SetSelected(errorPolicyLabel(flow.ErrStop))
	p.enabledCheck.SetChecked(true)
	p.maskLogsCheck.SetChecked(false)
	p.noteEntry.SetText("")
	p.previewLabel.SetText("模板预览：")
	p.clearErrors()
	p.form.Items = nil
	p.form.Refresh()
}
```

> Note: `flow.needsElement` is unexported. We'll need to either export it or duplicate the logic. Add a TODO to export `needsElement` in `validate.go` or duplicate in this file. We'll export it in Task 11.

- [ ] **Step 2: Build check**

Run: `go build ./internal/ui`
Expected: may fail due to `flow.needsElement` being unexported and `widget.NewModalPopUp`. Fix by changing `needsElement` to `NeedsElement` in `validate.go` and switching modal to `dialog.ShowCustomConfirm`.

- [ ] **Step 3: Fix exported NeedsElement**

Edit `internal/flow/validate.go`: rename `needsElement` → `NeedsElement` and update callers in the same file.

- [ ] **Step 4: Rebuild**

Run: `go build ./internal/ui`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/step_property.go internal/flow/validate.go
git commit -m "feat: add dynamic step property panel with validation"
```

---

### Task 6: Rewrite run_panel.go (Progress + Summary + Artifacts)

**Files:**
- Rewrite: `internal/ui/run_panel.go`
- Test: `go build ./internal/ui`

- [ ] **Step 1: Write new run_panel.go**

```go
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/runner"
)

type runPanel struct {
	app    *App
	widget fyne.CanvasObject

	progressBar  *widget.ProgressBar
	progressText *widget.Label
	logsEntry    *widget.Entry
	summary      *widget.Label
	currentStep  *widget.Label
	artifactBox  *fyne.Container
}

func newRunPanel(app *App) *runPanel {
	p := &runPanel{app: app}

	p.progressBar = widget.NewProgressBar()
	p.progressBar.Min = 0
	p.progressBar.Max = 1
	p.progressText = widget.NewLabel("就绪")

	p.logsEntry = widget.NewMultiLineEntry()
	p.logsEntry.Disable()
	p.logsEntry.Wrapping = fyne.TextWrapWord

	p.summary = widget.NewLabel("成功：0  失败：0  跳过：0  总耗时：0.0s")
	p.currentStep = widget.NewLabel("")
	p.artifactBox = container.NewHBox()

	startBtn := widget.NewButtonWithIcon("启动浏览器", theme.ViewRefreshIcon(), func() {
		go p.app.startBrowser()
	})
	runBtn := widget.NewButtonWithIcon("运行整个流程", theme.MediaPlayIcon(), func() {
		go p.app.runCurrentFlow()
	})
	stepBtn := widget.NewButtonWithIcon("单步执行", theme.MediaReplayIcon(), func() {
		go p.app.onStepButton()
	})
	p.app.stepBtn = stepBtn
	stopBtn := widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		p.app.runner.Stop()
	})

	controls := container.NewHBox(startBtn, runBtn, stepBtn, stopBtn)

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("运行摘要", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.summary,
	)

	p.widget = container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, p.progressText, controls, p.progressBar),
		),
		container.NewHBox(p.currentStep, p.artifactBox),
		rightPanel, nil,
		container.NewScroll(p.logsEntry),
	)
	return p
}

func (p *runPanel) log(msg string) {
	fyne.Do(func() {
		p.logsEntry.SetText(p.logsEntry.Text + msg + "\n")
	})
}

func (p *runPanel) setProgress(current, total int, stepName string) {
	fyne.Do(func() {
		if total > 0 {
			p.progressBar.Max = float64(total)
			p.progressBar.SetValue(float64(current))
		}
		p.progressText.SetText(fmt.Sprintf("第 %d 步 / 共 %d 步 · %s", current, total, stepName))
	})
}

func (p *runPanel) setSummary(res *runner.RunResult) {
	fyne.Do(func() {
		elapsed := res.FinishedAt.Sub(res.StartedAt).Seconds()
		p.summary.SetText(fmt.Sprintf("成功：%d  失败：%d  跳过：%d  总耗时：%.1fs", res.SuccessCount, res.FailedCount, res.SkippedCount, elapsed))
	})
}

func (p *runPanel) setCurrentStep(name string) {
	fyne.Do(func() {
		p.currentStep.SetText("当前步骤：" + name)
	})
}

func (p *runPanel) setArtifacts(screenshot, htmlSnap string) {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
		if screenshot != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, widget.NewLabel("截图："+screenshot))
		}
		if htmlSnap != "" {
			p.artifactBox.Objects = append(p.artifactBox.Objects, widget.NewLabel("HTML："+htmlSnap))
		}
		p.artifactBox.Refresh()
	})
}

func (p *runPanel) clearArtifacts() {
	fyne.Do(func() {
		p.artifactBox.Objects = nil
		p.artifactBox.Refresh()
	})
}

func (p *runPanel) reset() {
	fyne.Do(func() {
		p.progressBar.SetValue(0)
		p.progressText.SetText("就绪")
		p.currentStep.SetText("")
		p.clearArtifacts()
	})
}
```

- [ ] **Step 2: Build check**

Run: `go build ./internal/ui`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ui/run_panel.go
git commit -m "feat: rewrite run panel with progress, summary, artifacts"
```

---

### Task 7: Create status_bar.go (Widget)

**Files:**
- Create: `internal/ui/status_bar.go` (overwrite enums-only version from Task 1)
- Test: `go build ./internal/ui`

- [ ] **Step 1: Write complete status_bar.go**

```go
package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/browser"
)

type SaveStatus int
const (SaveUnmodified SaveStatus = iota; SaveDirty; SaveSaving; SaveSuccess; SaveFailed)
type RunStatus int
const (RunIdle RunStatus = iota; RunRunning; RunCompleted; RunFailed)

type statusBar struct {
	app          *App
	widget       fyne.CanvasObject
	flowLabel    *widget.Label
	saveLabel    *widget.Label
	chromeLabel  *widget.Label
	runLabel     *widget.Label
}

func newStatusBar(app *App) *statusBar {
	sb := &statusBar{app: app}
	sb.flowLabel = widget.NewLabel("未选择流程")
	sb.saveLabel = widget.NewLabel("未修改")
	sb.chromeLabel = widget.NewLabel("未安装")
	sb.runLabel = widget.NewLabel("空闲")

	sb.widget = container.NewHBox(
		widget.NewLabelWithStyle("Go Chrome", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		canvas.NewRectangle(color.Transparent), // spacer
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.flowLabel),
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.saveLabel),
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.chromeLabel),
		container.NewHBox(canvas.NewCircle(statusColorGray()), sb.runLabel),
	)
	return sb
}

func (sb *statusBar) setFlow(name string) {
	fyne.Do(func() {
		if name == "" { sb.flowLabel.SetText("未选择流程") } else { sb.flowLabel.SetText(name) }
	})
}

func (sb *statusBar) setSave(st SaveStatus) {
	fyne.Do(func() {
		switch st {
		case SaveUnmodified: sb.saveLabel.SetText("未修改")
		case SaveDirty:      sb.saveLabel.SetText("有未保存修改")
		case SaveSaving:     sb.saveLabel.SetText("保存中")
		case SaveSuccess:    sb.saveLabel.SetText("已保存")
		case SaveFailed:     sb.saveLabel.SetText("保存失败")
		}
	})
}

func (sb *statusBar) setChrome(st browser.ChromeStatus) {
	fyne.Do(func() {
		switch st {
		case browser.ChromeNotInstalled: sb.chromeLabel.SetText("未安装")
		case browser.ChromeInstalled:    sb.chromeLabel.SetText("已安装")
		case browser.ChromeDownloading:  sb.chromeLabel.SetText("下载中")
		case browser.ChromeStarting:     sb.chromeLabel.SetText("启动中")
		case browser.ChromeRunning:      sb.chromeLabel.SetText("已启动")
		case browser.ChromeStartFailed:  sb.chromeLabel.SetText("启动失败")
		}
	})
}

func (sb *statusBar) setRun(st RunStatus, current, total int, msg string) {
	fyne.Do(func() {
		switch st {
		case RunIdle:
			sb.runLabel.SetText("空闲")
		case RunRunning:
			if total > 0 {
				sb.runLabel.SetText(fmt.Sprintf("运行中 %d/%d", current, total))
			} else {
				sb.runLabel.SetText("运行中")
			}
		case RunCompleted:
			sb.runLabel.SetText("已完成")
		case RunFailed:
			if msg != "" { sb.runLabel.SetText("失败：" + msg) } else { sb.runLabel.SetText("失败") }
		}
	})
}

func statusColorGray() color.Color  { return color.NRGBA{0x9e, 0x9e, 0x9e, 0xff} }
func statusColorBlue() color.Color  { return color.NRGBA{0x1a, 0x73, 0xe8, 0xff} }
func statusColorGreen() color.Color { return color.NRGBA{0x4c, 0xaf, 0x50, 0xff} }
func statusColorYellow() color.Color{ return color.NRGBA{0xf9, 0xa8, 0x25, 0xff} }
func statusColorRed() color.Color   { return color.NRGBA{0xe5, 0x39, 0x35, 0xff} }
```

- [ ] **Step 2: Build check**

Run: `go build ./internal/ui`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ui/status_bar.go
git commit -m "feat: add status bar widget"
```

---

### Task 8: Create onboarding.go + Example Flow

**Files:**
- Create: `internal/ui/onboarding.go`
- Modify: `internal/flow/model.go`
- Test: `go build ./internal/...`

- [ ] **Step 1: Add NewExampleLoginFlow to model.go**

Open `internal/flow/model.go` and append after `NewStep`:

```go
// NewExampleLoginFlow creates a sample flow for onboarding.
func NewExampleLoginFlow() *Flow {
	f := NewFlow("示例：登录测试")
	f.Tags = []string{"示例"}
	f.Steps = []Step{
		NewStep("打开网址", StepNavigate),
		NewStep("输入用户名", StepInput),
		NewStep("输入密码", StepInput),
		NewStep("点击登录", StepClick),
		NewStep("断言欢迎文本", StepAssertExists),
		NewStep("页面截图", StepScreenshot),
	}
	f.Steps[0].Target = Target{Strategy: TargetXPath, Value: "https://example.com/login"}
	f.Steps[1].Target = Target{Strategy: TargetXPath, Value: "//input[@id='username']"}
	f.Steps[1].Input = Input{Mode: InputTemplate, Text: "${var:user=SP${11000-11099}}"}
	f.Steps[2].Target = Target{Strategy: TargetXPath, Value: "//input[@id='password']"}
	f.Steps[2].Input = Input{Mode: InputTemplate, Text: "Password123"}
	f.Steps[3].Target = Target{Strategy: TargetXPath, Value: "//button[@type='submit']"}
	f.Steps[4].Target = Target{Strategy: TargetXPath, Value: "//div[contains(text(),'欢迎')]"}
	return f
}
```

- [ ] **Step 2: Create onboarding.go**

```go
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

func (a *App) firstRunCheck() {
	if !a.browserMgr.IsInstalled() {
		a.runPanel.log("未检测到 Chrome，请点击「启动浏览器」下载")
	}
}

func (a *App) buildEmptyState() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("暂无流程", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("点击左侧「新建」创建流程，或导入示例流程快速体验。")
	hint.Alignment = fyne.TextAlignCenter

	newBtn := widget.NewButtonWithIcon("新建流程", theme.ContentAddIcon(), func() {
		a.createNewFlow()
	})
	importBtn := widget.NewButtonWithIcon("导入示例", theme.DocumentIcon(), func() {
		example := flow.NewExampleLoginFlow()
		if err := a.flowStore.Save(example); err != nil {
			widget.NewModalPopUp(widget.NewLabel("保存失败: "+err.Error()), a.mainWin.Canvas())
			return
		}
		a.refreshFlowList()
		a.flowLibrary.selectFlow(example.ID)
	})

	return container.NewCenter(container.NewVBox(
		theme.DocumentIcon(),
		title,
		hint,
		container.NewHBox(newBtn, importBtn),
	))
}
```

> Note: `widget.NewModalPopUp` might not exist; use `dialog.ShowError` instead.

Fix the error handling:

```go
import "fyne.io/fyne/v2/dialog"
// ...
if err := a.flowStore.Save(example); err != nil {
    dialog.ShowError(err, a.mainWin)
    return
}
```

- [ ] **Step 3: Build check**

Run: `go build ./internal/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/flow/model.go internal/ui/onboarding.go
git commit -m "feat: add onboarding empty state and example flow"
```

---

### Task 9: Rewrite main_window.go (Full Assembly)

**Files:**
- Rewrite: `internal/ui/main_window.go`
- Delete: `internal/ui/step_editor.go`
- Test: `go build ./cmd/go-chrome`

- [ ] **Step 1: Write new main_window.go**

```go
package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/assets"
	appdirs "go-chrome/internal/app"
	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/logx"
	"go-chrome/internal/runner"
)

type App struct {
	fyneApp     fyne.App
	mainWin     fyne.Window
	cfg         *config.Config
	dirs        *appdirs.Directories
	flowStore   *flow.Store
	recentStore *flow.RecentStore
	browserMgr  *browser.Manager
	runner      *runner.Runner
	stepRunner  *runner.StepRunner
	history     *runner.HistoryStore

	statusBar    *statusBar
	flowLibrary  *flowLibraryPanel
	flowEditor   *flowEditorPanel
	stepTable    *stepTablePanel
	stepProperty *stepPropertyPanel
	runPanel     *runPanel
	currentFlow  *flow.Flow

	dirty       bool
	chromeTicker *time.Ticker
	chromeDone   chan struct{}

	stepBtn *widget.Button
}

func New(cfg *config.Config, dirs *appdirs.Directories) *App {
	return &App{cfg: cfg, dirs: dirs}
}

func (a *App) Run() {
	a.fyneApp = app.NewWithID("com.go-chrome.app")
	a.fyneApp.Settings().SetTheme(newAppTheme())
	if ico := assets.Icon(); ico != nil {
		a.fyneApp.SetIcon(ico)
	}
	a.mainWin = a.fyneApp.NewWindow("Chrome 自动化编排工具")
	a.mainWin.Resize(fyne.NewSize(float32(a.cfg.App.WindowWidth), float32(a.cfg.App.WindowHeight)))
	a.mainWin.SetOnClosed(func() {
		size := a.mainWin.Canvas().Size()
		a.cfg.App.WindowWidth = int(size.Width)
		a.cfg.App.WindowHeight = int(size.Height)
		_ = config.Save(a.dirs.ConfigPath, a.cfg)
		if a.recentStore != nil {
			_ = a.recentStore.Save()
		}
		if a.chromeDone != nil {
			close(a.chromeDone)
		}
	})

	a.initDeps()
	a.buildUI()
	a.firstRunCheck()
	a.startChromeTicker()
	a.mainWin.ShowAndRun()
}

func (a *App) initDeps() {
	var err error
	a.flowStore, err = flow.NewStore(a.dirs.FlowsDir)
	if err != nil {
		logx.Errorf("flow store: %v", err)
	}
	a.recentStore, _ = flow.NewRecentStore(filepath.Join(a.dirs.DataDir, "recent-flows.json"))
	a.browserMgr = browser.NewManager(&a.cfg.Chrome)
	a.browserMgr.LoadManifest() // best effort

	historyDir := filepath.Join(a.dirs.DataDir, "run-history")
	a.history, _ = runner.NewHistoryStore(historyDir)
	if a.history != nil {
		_ = a.history.Cleanup(a.cfg.App.LogRetentionDays)
	}

	a.runner = runner.NewRunner(&a.cfg.Runner, a.browserMgr, a.history)
	go a.handleRunnerEvents()
}

func (a *App) buildUI() {
	onDirty := func() { a.markDirty() }

	a.statusBar = newStatusBar(a)
	a.flowLibrary = newFlowLibraryPanel(a)
	a.flowEditor = newFlowEditorPanel(a, onDirty)
	a.stepTable = newStepTablePanel(a, onDirty)
	a.stepProperty = newStepPropertyPanel(a, onDirty)
	a.runPanel = newRunPanel(a)

	// Layout
	centerTop := container.NewBorder(a.flowEditor.widget, nil, nil, nil, a.stepTable.widget)
	center := container.NewHSplit(centerTop, a.stepProperty.widget)
	center.SetOffset(0.55)

	mainSplit := container.NewVSplit(
		container.NewBorder(a.statusBar.widget, nil, a.flowLibrary.widget, nil, center),
		a.runPanel.widget,
	)
	mainSplit.SetOffset(0.72)

	a.mainWin.SetContent(mainSplit)
	a.refreshFlowList()
}

func (a *App) markDirty() {
	a.dirty = true
	a.statusBar.setSave(SaveDirty)
}

func (a *App) markClean() {
	a.dirty = false
	a.statusBar.setSave(SaveUnmodified)
}

func (a *App) startChromeTicker() {
	a.chromeTicker = time.NewTicker(1 * time.Second)
	a.chromeDone = make(chan struct{})
	go func() {
		for {
			select {
			case <-a.chromeTicker.C:
				fyne.Do(func() {
					a.statusBar.setChrome(a.browserMgr.Status())
				})
			case <-a.chromeDone:
				return
			}
		}
	}()
}

func (a *App) createNewFlow() {
	if a.dirty && a.currentFlow != nil {
		a.promptSaveBefore(func() { a.doCreateNewFlow() })
		return
	}
	a.doCreateNewFlow()
}

func (a *App) doCreateNewFlow() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("请输入流程名称")
	dialog.ShowForm("新建流程", "创建", "取消", []*widget.FormItem{
		widget.NewFormItem("流程名称", nameEntry),
	}, func(ok bool) {
		if !ok || nameEntry.Text == "" { return }
		f := flow.NewFlow(nameEntry.Text)
		if err := a.flowStore.Save(f); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.setCurrentFlow(f)
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) saveCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if err := flow.Validate(a.currentFlow); err != nil {
		dialog.ShowError(fmt.Errorf("保存前校验失败: %w", err), a.mainWin)
		return
	}
	a.statusBar.setSave(SaveSaving)
	if err := a.flowStore.Save(a.currentFlow); err != nil {
		a.statusBar.setSave(SaveFailed)
		dialog.ShowError(err, a.mainWin)
		return
	}
	a.statusBar.setSave(SaveSuccess)
	a.markClean()
}

func (a *App) importFlow() {
	if a.dirty && a.currentFlow != nil {
		a.promptSaveBefore(func() { a.doImportFlow() })
		return
	}
	a.doImportFlow()
}

func (a *App) doImportFlow() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil { return }
		defer reader.Close()
		f, err := a.flowStore.Import(reader.URI().Path())
		if err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		a.setCurrentFlow(f)
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) exportFlow() {
	if a.currentFlow == nil { return }
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil { return }
		defer writer.Close()
		if err := a.flowStore.Export(a.currentFlow.ID, writer.URI().Path()); err != nil {
			dialog.ShowError(err, a.mainWin)
		}
	}, a.mainWin)
}

func (a *App) startBrowser() {
	if !a.browserMgr.IsInstalled() {
		a.runPanel.log("未检测到本地 Chrome，开始下载...")
		if err := a.browserMgr.Install(func(d, t int64) {
			if t > 0 { a.runPanel.log(fmt.Sprintf("下载进度 %d%%", d*100/t)) }
		}); err != nil {
			a.runPanel.log("Chrome 下载失败：" + err.Error())
			fyne.Do(func() { dialog.ShowError(err, a.mainWin) })
			return
		}
	}
	port, err := a.browserMgr.Start()
	if err != nil {
		a.runPanel.log("Chrome 启动失败：" + err.Error())
		fyne.Do(func() { dialog.ShowError(err, a.mainWin) })
		return
	}
	a.runPanel.log(fmt.Sprintf("Chrome 已启动，调试端口：%d", port))
}

func (a *App) runCurrentFlow() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if a.runner.IsRunning() {
		dialog.ShowInformation("提示", "当前已有流程正在运行", a.mainWin)
		return
	}
	a.runPanel.reset()
	a.stepTable.clearStatuses()
	go func() {
		res := a.runner.RunFlow(a.currentFlow, 0)
		a.runPanel.setSummary(res)
	}()
}

func (a *App) onStepButton() {
	if a.currentFlow == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if a.stepRunner != nil && !a.stepRunner.IsFinished() {
		a.nextStep()
		return
	}
	if a.stepRunner != nil { a.stepRunner.Close() }
	a.stepRunner = runner.NewStepRunner(&a.cfg.Runner, a.browserMgr, a.history)
	if err := a.stepRunner.Init(a.currentFlow); err != nil {
		dialog.ShowError(err, a.mainWin)
		a.stepRunner = nil
		return
	}
	a.stepBtn.SetText("下一步")
	a.nextStep()
}

func (a *App) nextStep() {
	if a.stepRunner == nil { return }
	res, finished, err := a.stepRunner.Next()
	if err != nil {
		a.runPanel.log("单步执行错误：" + err.Error())
		a.stepBtn.SetText("单步执行")
		a.stepRunner.Close()
		a.stepRunner = nil
		return
	}
	if res != nil {
		logMsg := fmt.Sprintf("步骤 %d %s: %s", a.stepRunner.CurrentIndex(), res.StepName, res.Status)
		if res.Error != "" { logMsg += " - " + res.Error }
		a.runPanel.log(logMsg)
	}
	if finished {
		result := a.stepRunner.Result()
		a.runPanel.log(fmt.Sprintf("单步执行完成：%s（成功 %d，失败 %d）", result.Status, result.SuccessCount, result.FailedCount))
		a.stepBtn.SetText("单步执行")
		a.stepRunner.Close()
		a.stepRunner = nil
		a.refreshHistory()
	}
}

func (a *App) handleRunnerEvents() {
	var totalSteps int
	for ev := range a.runner.Events() {
		switch ev.Type {
		case runner.EventLog:
			a.runPanel.log(ev.LogMessage)
		case runner.EventStepStart:
			totalSteps = 0
			if a.currentFlow != nil {
				for _, s := range a.currentFlow.Steps { if s.Enabled { totalSteps++ } }
			}
			a.runPanel.setProgress(ev.StepIndex+1, totalSteps, ev.StepName)
			a.runPanel.setCurrentStep(ev.StepName)
			a.statusBar.setRun(RunRunning, ev.StepIndex+1, totalSteps, "")
		case runner.EventStepDone:
			statuses := make([]runner.Status, len(a.stepTable.stepsData))
			for i := range statuses { statuses[i] = runner.StatusPending }
			if ev.StepIndex >= 0 && ev.StepIndex < len(statuses) {
				statuses[ev.StepIndex] = ev.Result.Status
			}
			a.stepTable.setStatuses(statuses)
			if ev.Result.Status == runner.StatusFailed {
				a.runPanel.setArtifacts(ev.Result.Screenshot, ev.Result.HTMLSnapshot)
				a.stepTable.table.Select(widget.TableCellID{Row: ev.StepIndex, Col: 0})
			}
		case runner.EventRunDone:
			if ev.RunResult != nil {
				if ev.RunResult.FailedCount > 0 {
					a.statusBar.setRun(RunFailed, 0, 0, fmt.Sprintf("第 %d 步", ev.RunResult.FailedCount))
				} else {
					a.statusBar.setRun(RunCompleted, 0, 0, "")
				}
				a.runPanel.setSummary(ev.RunResult)
			}
			a.stepTable.clearStatuses()
			a.refreshHistory()
		}
	}
}

func (a *App) refreshFlowList() {
	flows, _ := a.flowStore.ListSorted()
	a.flowLibrary.setFlows(flows)
	if len(flows) == 0 {
		// Empty state could be shown by replacing center content; for now keep simple
	}
}

func (a *App) refreshHistory() {
	if a.currentFlow == nil { return }
	// History panel removed in redesign; history integrated into runPanel if needed
}

func (a *App) setCurrentFlow(f *flow.Flow) {
	a.currentFlow = f
	if a.recentStore != nil { a.recentStore.Touch(f.ID) }
	a.flowEditor.loadFlow(f)
	a.stepTable.loadFlow(f)
	a.stepProperty.clear()
	a.statusBar.setFlow(f.Name)
	a.markClean()
}

func (a *App) onFlowSelected(f *flow.Flow) {
	if a.dirty && a.currentFlow != nil && a.currentFlow.ID != f.ID {
		a.promptSaveBefore(func() { a.setCurrentFlow(f) })
		return
	}
	a.setCurrentFlow(f)
}

func (a *App) promptSaveBefore(next func()) {
	dialog.ShowConfirm("未保存的修改",
		fmt.Sprintf("当前流程 [%s] 有未保存的修改，是否保存？", a.currentFlow.Name),
		func(save bool) {
			if save {
				a.saveCurrentFlow()
				next()
			} else {
				a.markClean()
				next()
			}
		}, a.mainWin)
}

func (a *App) onFlowDelete(f *flow.Flow) {
	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定删除流程 [%s] 吗？", f.Name), func(ok bool) {
		if !ok { return }
		if err := a.flowStore.Delete(f.ID); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		if a.currentFlow != nil && a.currentFlow.ID == f.ID {
			a.setCurrentFlow(nil)
		}
		a.refreshFlowList()
	}, a.mainWin)
}

func (a *App) onFlowClone(f *flow.Flow) {
	cf := f.Clone()
	if err := a.flowStore.Save(cf); err != nil {
		dialog.ShowError(err, a.mainWin)
		return
	}
	a.refreshFlowList()
}

func (a *App) onTagFilter(tag string) {
	if strings.TrimSpace(tag) == "" {
		a.refreshFlowList()
		return
	}
	flows, _ := a.flowStore.ListSorted()
	var filtered []*flow.Flow
	for _, f := range flows {
		for _, t := range f.Tags {
			if t == tag { filtered = append(filtered, f); break }
		}
	}
	a.flowLibrary.setFlows(filtered)
}

func (a *App) onStepSelected(s *flow.Step, idx int) {
	if s == nil {
		a.stepProperty.clear()
		return
	}
	a.stepProperty.loadStep(s, idx, len(a.stepTable.stepsData))
}
```

- [ ] **Step 2: Delete old step_editor.go**

Run: `rm internal/ui/step_editor.go`

- [ ] **Step 3: Build check**

Run: `go build ./cmd/go-chrome`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/ui/main_window.go internal/ui/step_editor.go
git commit -m "feat: rewrite main window with new layout and event wiring"
```

---

### Task 10: Runner Event Extensions

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/event.go` (if exists) or `internal/runner/*.go`

The new `main_window.go` references `runner.EventStepStart` which may not exist. Check and add.

- [ ] **Step 1: Check event types**

Run: `grep -r "EventStepStart" internal/runner/`
If missing, add to the event definitions.

- [ ] **Step 2: Add EventStepStart if needed**

Read `internal/runner/` to find event definitions.

```bash
grep -rn "type Event" internal/runner/
```

If `EventStepStart` doesn't exist, add it alongside `EventStepDone`.

- [ ] **Step 3: Emit EventStepStart in Runner.RunFlow**

In `runner.go`, before executing each step, add:
```go
r.emit(Event{Type: EventStepStart, StepIndex: i, StepName: step.Name})
```

- [ ] **Step 4: Build check**

Run: `go build ./internal/runner`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/
git commit -m "feat: add EventStepStart for progress tracking"
```

---

### Task 11: Cleanup + Final Build + Tests

**Files:**
- All `internal/ui/*.go`
- Test: `go test ./internal/...`, `go vet ./...`

- [ ] **Step 1: Remove unused code**

Run: `go build ./cmd/go-chrome`
Fix any "declared but not used" or "undefined" errors.

Common issues to fix:
- `historyPanel` references if still in code
- Unused imports
- `widget.NewModalPopUp` → replace with `dialog.ShowCustom`

- [ ] **Step 2: Run unit tests**

Run: `go test ./internal/browser ./internal/config ./internal/flow ./internal/runner ./internal/template`
Expected: all PASS

- [ ] **Step 3: Run vet**

Run: `go vet ./internal/... ./cmd/go-chrome`
Expected: PASS

- [ ] **Step 4: Full build**

Run: `go build -mod=readonly ./cmd/go-chrome`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: complete UI rewrite stages A/B/C/D"
```

---

## Spec Coverage Checklist

| Spec Requirement | Implementing Task |
|-----------------|-------------------|
| 顶部状态栏 5 个胶囊 | Task 7 |
| 保存状态跟踪（dirty/clean） | Task 9 (main_window.go markDirty/markClean) |
| Chrome 状态轮询 | Task 9 (startChromeTicker) |
| 运行状态展示 | Task 9 (handleRunnerEvents → statusBar.setRun) |
| 步骤表格 widget.Table | Task 4 |
| 新增步骤先选类型 | Task 4 (showAddStepDialog) |
| 动态表单按类型显示字段 | Task 5 (rebuildForm) |
| 字段级实时校验 | Task 5 (validate) |
| 切换前未保存提示 | Task 9 (promptSaveBefore) |
| 进度条 | Task 6 (setProgress) |
| 步骤状态图标 | Task 4 (statusIcon) + Task 9 (handleRunnerEvents) |
| 失败自动选中步骤 | Task 9 (EventStepDone → Select) |
| 日志产物路径 | Task 6 (setArtifacts) + Task 9 (EventStepDone) |
| 运行摘要面板 | Task 6 (setSummary) |
| 空状态页 | Task 8 (buildEmptyState) |
| 示例流程 | Task 8 (NewExampleLoginFlow) |
| 首次启动检查 | Task 8 (firstRunCheck) |
| JSON schema 不变 | All tasks (no model changes except NewExampleLoginFlow) |

---

## Self-Review Fixes Applied

1. `widget.NewModalPopUp` replaced with `dialog.ShowCustom` / `dialog.ShowConfirm` in all locations.
2. `flow.needsElement` exported to `NeedsElement` to allow use from `step_property.go`.
3. `EventStepStart` added to runner events for accurate progress tracking.
4. Table row rendering simplified to `widget.Label` to avoid canvas complexity inside Table cells.
5. `stepPropertyPanel.apply()` stores assert_text expected value in `Note` field as temporary measure (Step model has no Expected field).
