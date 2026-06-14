package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

func TestNewInfoPanel(t *testing.T) {
	app := &App{
		fyneApp: test.NewApp(),
	}
	panel := newInfoPanel(app)
	if panel == nil {
		t.Fatal("expected non-nil infoPanel")
	}
	if panel.widget == nil {
		t.Fatal("expected non-nil widget")
	}
	if panel.refreshTicker == nil {
		t.Fatal("expected auto-refresh ticker to be started")
	}
	panel.stopAutoRefresh()
}

func TestInfoPanelAutoRefreshStartStop(t *testing.T) {
	app := &App{
		fyneApp: test.NewApp(),
	}
	panel := newInfoPanel(app)

	// Starting again should be a no-op.
	panel.startAutoRefresh(2 * time.Second)

	panel.stopAutoRefresh()
	if panel.refreshTicker != nil {
		t.Fatal("expected ticker to be nil after stop")
	}

	// Stopping again should be safe.
	panel.stopAutoRefresh()
}

func TestInfoPanelRefreshWithoutPanic(t *testing.T) {
	app := &App{
		fyneApp: test.NewApp(),
	}
	panel := newInfoPanel(app)
	defer panel.stopAutoRefresh()

	// refresh uses fyne.Do; in test app this should execute synchronously.
	panel.refresh()
}
