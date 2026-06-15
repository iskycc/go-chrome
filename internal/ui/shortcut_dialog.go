package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/shortcut"
)

// showCreateShortcutDialogForFlow opens the shortcut creation dialog for the
// currently selected flow and the currently selected environment.
func (a *App) showCreateShortcutDialogForFlow(f *flow.Flow) {
	if f == nil {
		dialog.ShowInformation("提示", "请先选择或新建一个流程", a.mainWin)
		return
	}
	if a.envRepo == nil {
		dialog.ShowError(fmt.Errorf("环境仓库未初始化"), a.mainWin)
		return
	}
	envID, _, err := a.currentEnvProvider()
	if err != nil {
		dialog.ShowError(fmt.Errorf("无法获取当前环境: %w", err), a.mainWin)
		return
	}
	env, err := a.envRepo.Get(envID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("无法获取当前环境: %w", err), a.mainWin)
		return
	}
	a.showCreateShortcutDialog(f.ID, env.ID, f.Name, env.Name)
}

// showCreateShortcutDialog lets the user edit and confirm the shortcut name
// before creating a desktop shortcut for the given flow and environment.
func (a *App) showCreateShortcutDialog(flowID, envID, flowName, envName string) {
	if a.mainWin == nil {
		return
	}

	desktop := desktopDir()
	if desktop == "" {
		dialog.ShowError(fmt.Errorf("无法定位桌面目录"), a.mainWin)
		return
	}

	defaultName := uniqueShortcutName(flowName, envName, desktop)

	nameEntry := widget.NewEntry()
	nameEntry.SetText(defaultName)
	nameEntry.SetPlaceHolder("快捷方式名称")

	form := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("流程：%s", flowName)),
		widget.NewLabel(fmt.Sprintf("环境：%s", envName)),
		widget.NewLabel("快捷方式名称"),
		nameEntry,
	)

	d := dialog.NewCustomConfirm("生成桌面快捷方式", "生成", "取消", form, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			dialog.ShowInformation("提示", "名称不能为空", a.mainWin)
			return
		}
		if !strings.HasSuffix(strings.ToLower(name), ".lnk") {
			name += ".lnk"
		}
		shortcutPath := filepath.Join(desktop, sanitizeShortcutName(name))
		if err := a.createShortcutFile(flowID, envID, shortcutPath); err != nil {
			dialog.ShowError(fmt.Errorf("生成快捷方式失败: %w", err), a.mainWin)
			return
		}
		a.runPanel.log("已生成桌面快捷方式: " + shortcutPath)
	}, a.mainWin)
	d.Resize(fyne.NewSize(440, 240))
	d.Show()
}

// createShortcutFile writes the .lnk file pointing to the current executable
// with --flow and --env arguments.
func (a *App) createShortcutFile(flowID, envID, shortcutPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exe)
	return shortcut.Create(shortcut.Options{
		TargetPath:   exe,
		Arguments:    fmt.Sprintf("--flow=%s --env=%s", flowID, envID),
		WorkingDir:   exeDir,
		IconPath:     exe,
		Description:  fmt.Sprintf("Chrome 自动化: %s", filepath.Base(shortcutPath)),
		ShortcutPath: shortcutPath,
	})
}

// uniqueShortcutName returns a file name in the form "flowName-envName.lnk".
// If the file already exists on the desktop, it appends "-1", "-2", etc.
func uniqueShortcutName(flowName, envName, dir string) string {
	base := fmt.Sprintf("%s-%s", flowName, envName)
	base = sanitizeShortcutName(base)
	candidate := base + ".lnk"
	if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
		return candidate
	}
	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s-%d.lnk", base, i)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

// sanitizeShortcutName removes characters that are illegal in Windows file names.
func sanitizeShortcutName(name string) string {
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", `"`, "", "/", "", "\\", "", "|", "", "?", "", "*", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

// desktopDir returns the user's Desktop directory. It uses the standard
// "Desktop" folder under the home directory.
func desktopDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Desktop")
}
