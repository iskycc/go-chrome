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
	// Ticker should not start until the panel becomes visible.
	if panel.refreshTicker != nil {
		t.Fatal("expected no auto-refresh ticker before visible")
	}
	panel.SetVisible(true)
	if panel.refreshTicker == nil {
		t.Fatal("expected auto-refresh ticker after visible")
	}
	panel.SetVisible(false)
}

func TestInfoPanelAutoRefreshStartStop(t *testing.T) {
	app := &App{
		fyneApp: test.NewApp(),
	}
	panel := newInfoPanel(app)
	panel.SetVisible(true)

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

	// refresh uses fyne.Do; in test app this should execute synchronously.
	panel.refresh()
}

func TestInfoPanelSetVisible(t *testing.T) {
	app := &App{
		fyneApp: test.NewApp(),
	}
	panel := newInfoPanel(app)

	panel.SetVisible(true)
	if !panel.visible || panel.refreshTicker == nil {
		t.Fatal("expected panel visible and ticker running")
	}

	panel.SetVisible(false)
	if panel.visible || panel.refreshTicker != nil {
		t.Fatal("expected panel hidden and ticker stopped")
	}

	// Re-showing should restart the ticker safely.
	panel.SetVisible(true)
	if !panel.visible || panel.refreshTicker == nil {
		t.Fatal("expected panel visible and ticker running again")
	}
	panel.SetVisible(false)
}

func TestInfoPanelSetLabel(t *testing.T) {
	app := &App{
		fyneApp: test.NewApp(),
	}
	panel := newInfoPanel(app)
	label := panel.selfPID
	cache := ""

	panel.setLabel(label, &cache, "123")
	if label.Text != "123" || cache != "123" {
		t.Fatalf("expected label updated to '123', got %q / cache %q", label.Text, cache)
	}

	panel.setLabel(label, &cache, "123")
	if label.Text != "123" || cache != "123" {
		t.Fatalf("expected unchanged label, got %q / cache %q", label.Text, cache)
	}
}
