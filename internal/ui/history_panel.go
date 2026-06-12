package ui

import (
	"fmt"
	"net/url"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/runner"
)

// historyPanel shows run history for the current flow.
type historyPanel struct {
	app          *App
	widget       fyne.CanvasObject
	list         *widget.List
	results      []*runner.RunResult
	envSelect    *widget.Select
	statusSelect *widget.Select
	envIDsByName map[string]string
}

func newHistoryPanel(app *App) *historyPanel {
	p := &historyPanel{app: app}
	p.envIDsByName = map[string]string{"全部环境": ""}
	p.envSelect = widget.NewSelect([]string{"全部环境"}, func(string) {
		p.app.refreshHistory()
	})
	p.envSelect.SetSelected("全部环境")
	p.statusSelect = widget.NewSelect([]string{"全部状态", "成功", "失败"}, func(string) {
		p.app.refreshHistory()
	})
	p.statusSelect.SetSelected("全部状态")
	p.list = widget.NewList(
		func() int { return len(p.results) },
		func() fyne.CanvasObject {
			return newContextMenuLabel("历史记录", nil)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(p.results) {
				return
			}
			r := p.results[id]
			label := fmt.Sprintf("%s | %s | 成功:%d 失败:%d",
				r.StartedAt.Format("01-02 15:04:05"),
				r.Status,
				r.SuccessCount, r.FailedCount)
			l := item.(*contextMenuLabel)
			l.SetText(label)
			l.onSecondaryTap = func(e *fyne.PointEvent) {
				p.showHistoryContextMenu(int(id), e)
			}
		},
	)
	p.widget = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("执行历史", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(widget.NewLabel("环境"), p.envSelect, widget.NewLabel("状态"), p.statusSelect),
		),
		nil, nil, nil,
		container.NewScroll(p.list),
	)
	return p
}

func (p *historyPanel) setResults(results []*runner.RunResult) {
	fyne.Do(func() {
		p.results = results
		p.list.Refresh()
	})
}

func (p *historyPanel) refreshFilters() {
	if p.app.envRepo == nil {
		return
	}
	envs, err := p.app.envRepo.List()
	if err != nil {
		return
	}
	options := []string{"全部环境"}
	ids := map[string]string{"全部环境": ""}
	for _, env := range envs {
		options = append(options, env.Name)
		ids[env.Name] = env.ID
	}
	current := p.envSelect.Selected
	if current == "" {
		current = "全部环境"
	}
	if _, ok := ids[current]; !ok {
		current = "全部环境"
	}
	fyne.Do(func() {
		p.envIDsByName = ids
		p.envSelect.Options = options
		p.envSelect.SetSelected(current)
	})
}

func (p *historyPanel) showHistoryContextMenu(idx int, e *fyne.PointEvent) {
	if idx < 0 || idx >= len(p.results) {
		return
	}
	p.list.Select(idx)
	r := p.results[idx]

	detailItem := fyne.NewMenuItem("查看详情", func() {
		elapsed := r.FinishedAt.Sub(r.StartedAt).Seconds()
		msg := fmt.Sprintf("运行 ID：%s\n状态：%s\n成功：%d  失败：%d  跳过：%d\n总耗时：%.1fs\n开始时间：%s",
			r.ID, r.Status, r.SuccessCount, r.FailedCount, r.SkippedCount, elapsed,
			r.StartedAt.Format("2006-01-02 15:04:05"))
		dialog.ShowInformation("运行详情", msg, p.app.mainWin)
	})
	copySummaryItem := fyne.NewMenuItem("复制运行摘要", func() {
		elapsed := r.FinishedAt.Sub(r.StartedAt).Seconds()
		summary := fmt.Sprintf("%s | %s | 成功:%d 失败:%d 跳过:%d | %.1fs",
			r.StartedAt.Format("2006-01-02 15:04:05"), r.Status,
			r.SuccessCount, r.FailedCount, r.SkippedCount, elapsed)
		p.app.fyneApp.Clipboard().SetContent(clipCopy(summary))
		p.app.runPanel.log("运行摘要已复制到剪贴板")
	})
	copyIDItem := fyne.NewMenuItem("复制运行 ID", func() {
		p.app.fyneApp.Clipboard().SetContent(clipCopy(r.ID))
		p.app.runPanel.log("运行 ID 已复制到剪贴板")
	})
	openDirItem := fyne.NewMenuItem("打开产物目录", func() {
		p.openHistoryArtifactDir(r)
	})
	openDirItem.Disabled = p.historyArtifactDir(r) == ""
	copyDirItem := fyne.NewMenuItem("复制产物目录路径", func() {
		if dir := p.historyArtifactDir(r); dir != "" {
			p.app.fyneApp.Clipboard().SetContent(clipCopy(dir))
			p.app.runPanel.log("产物目录路径已复制到剪贴板")
		}
	})
	copyDirItem.Disabled = p.historyArtifactDir(r) == ""
	filterEnvItem := fyne.NewMenuItem("筛选此环境", func() {
		p.filterByEnvID(r.EnvironmentID)
	})
	filterEnvItem.Disabled = r.EnvironmentID == ""
	rerunItem := fyne.NewMenuItem("按此环境重新运行", func() {
		p.rerunWithHistoryEnv(r)
	})
	rerunItem.Disabled = r.EnvironmentID == "" || p.app.currentFlow == nil

	menu := fyne.NewMenu("历史操作",
		detailItem,
		copySummaryItem,
		copyIDItem,
		fyne.NewMenuItemSeparator(),
		openDirItem,
		copyDirItem,
		fyne.NewMenuItemSeparator(),
		filterEnvItem,
		rerunItem,
	)
	showContextMenu(menu, p.app.mainWin.Canvas(), e.AbsolutePosition)
}

func (p *historyPanel) historyArtifactDir(r *runner.RunResult) string {
	for _, s := range r.Steps {
		if s.Screenshot != "" {
			return filepath.Dir(s.Screenshot)
		}
		if s.HTMLSnapshot != "" {
			return filepath.Dir(s.HTMLSnapshot)
		}
	}
	return ""
}

func (p *historyPanel) openHistoryArtifactDir(r *runner.RunResult) {
	dir := p.historyArtifactDir(r)
	if dir == "" {
		p.app.runPanel.log("该历史记录没有产物目录")
		return
	}
	uri, err := url.Parse(storage.NewFileURI(dir).String())
	if err != nil {
		p.app.runPanel.log("打开产物目录失败：" + err.Error())
		return
	}
	if err := p.app.fyneApp.OpenURL(uri); err != nil {
		p.app.runPanel.log("打开产物目录失败：" + err.Error())
	}
}

func (p *historyPanel) filterByEnvID(envID string) {
	if envID == "" || p.app.envRepo == nil {
		return
	}
	env, err := p.app.envRepo.Get(envID)
	if err != nil {
		return
	}
	if _, ok := p.envIDsByName[env.Name]; ok {
		p.envSelect.SetSelected(env.Name)
	}
}

func (p *historyPanel) rerunWithHistoryEnv(r *runner.RunResult) {
	if r.EnvironmentID == "" || p.app.currentFlow == nil {
		return
	}
	env, err := p.app.envRepo.Get(r.EnvironmentID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("无法切换到历史环境: %w", err), p.app.mainWin)
		return
	}
	if p.app.currentFlow.ID != r.FlowID {
		showWrappedConfirm("流程不一致", "当前流程与历史记录所属流程不同，是否仍用当前流程按该环境运行？", "继续运行", "取消", fyne.NewSize(520, 180), func(ok bool) {
			if ok {
				p.doRerunWithEnv(env.Name)
			}
		}, p.app.mainWin)
		return
	}
	p.doRerunWithEnv(env.Name)
}

func (p *historyPanel) doRerunWithEnv(envName string) {
	if _, ok := p.envIDsByName[envName]; ok {
		p.envSelect.SetSelected(envName)
	}
	p.app.refreshEnvironmentSelectors()
	p.app.runCurrentFlow()
}

func (p *historyPanel) environmentFilter() string {
	if p == nil || p.envSelect == nil {
		return ""
	}
	return p.envIDsByName[p.envSelect.Selected]
}

func (p *historyPanel) statusFilter() runner.Status {
	if p == nil || p.statusSelect == nil {
		return ""
	}
	switch p.statusSelect.Selected {
	case "成功":
		return runner.StatusSuccess
	case "失败":
		return runner.StatusFailed
	default:
		return ""
	}
}
