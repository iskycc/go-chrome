package ui

import (
	"fmt"

	"fyne.io/fyne/v2/dialog"
)

// showEnvManager selects the environment configuration tab.
// The environment manager is now a primary module tab; this function is kept
// as a compatibility entry point for existing callers.
func (a *App) showEnvManager() {
	if a.envRepo == nil {
		dialog.ShowError(fmt.Errorf("环境管理不可用"), a.mainWin)
		return
	}
	if a.moduleTabs == nil {
		return
	}
	for i, tab := range a.moduleTabs.Items {
		if tab.Text == "环境配置" {
			a.moduleTabs.SelectIndex(i)
			return
		}
	}
}
