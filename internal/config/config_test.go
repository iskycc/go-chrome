package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Chrome.DownloadSource != "official_stable" {
		t.Fatalf("unexpected download source: %s", cfg.Chrome.DownloadSource)
	}
	if cfg.Runner.DefaultTimeoutMs != 10000 {
		t.Fatalf("unexpected timeout: %d", cfg.Runner.DefaultTimeoutMs)
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load default: %v", err)
	}
	if cfg.Chrome.Channel != "Stable" {
		t.Fatalf("unexpected channel: %s", cfg.Chrome.Channel)
	}

	cfg.Chrome.Channel = "Beta"
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if cfg2.Chrome.Channel != "Beta" {
		t.Fatalf("channel not persisted: %s", cfg2.Chrome.Channel)
	}
}

func TestLoadExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"chrome":{"downloadSource":"custom_url","customDownloadURL":"http://example.com/chrome.zip"},"runner":{"defaultTimeoutMs":5000},"app":{"theme":"dark"}}`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Chrome.DownloadSource != "custom_url" {
		t.Fatalf("unexpected source: %s", cfg.Chrome.DownloadSource)
	}
	if cfg.Runner.DefaultTimeoutMs != 5000 {
		t.Fatalf("unexpected timeout: %d", cfg.Runner.DefaultTimeoutMs)
	}
	if !cfg.Chrome.FallbackToOfficial {
		t.Fatal("missing fallbackToOfficial should default to true")
	}
	if !cfg.Chrome.KeepDownloadCache {
		t.Fatal("missing keepDownloadCache should default to true")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadReadError(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected read error for directory path")
	}
}

func TestLoadBackfillsMissingDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"chrome":{},"runner":{},"app":{}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	defaults := Default()
	if cfg.Chrome.DownloadSource != defaults.Chrome.DownloadSource ||
		cfg.Chrome.Channel != defaults.Chrome.Channel ||
		cfg.Chrome.InstallDir != defaults.Chrome.InstallDir ||
		cfg.Chrome.UserDataDir != defaults.Chrome.UserDataDir ||
		cfg.Runner.DefaultTimeoutMs != defaults.Runner.DefaultTimeoutMs ||
		cfg.Runner.DefaultWaitAfterMs != defaults.Runner.DefaultWaitAfterMs ||
		cfg.App.LogRetentionDays != defaults.App.LogRetentionDays ||
		cfg.App.WindowWidth != defaults.App.WindowWidth ||
		cfg.App.WindowHeight != defaults.App.WindowHeight {
		t.Fatalf("defaults not backfilled: %+v", cfg)
	}
}

func TestSaveCreateDirError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if err := Save(filepath.Join(blocker, "config.json"), Default()); err == nil {
		t.Fatal("expected create dir error")
	}
}

func TestConfigInstance(t *testing.T) {
	cfg := Default()
	cfg.App.Theme = "custom"
	SetInstance(cfg)
	if Get().App.Theme != "custom" {
		t.Fatalf("unexpected singleton config: %v", Get())
	}
}

func TestResolvePaths(t *testing.T) {
	cfg := Default()
	cfg.Chrome.InstallDir = "./chrome"
	cfg.Chrome.UserDataDir = "./data/profile"

	base := `C:\app\base`
	cfg.ResolvePaths(base)

	wantInstall := filepath.Join(base, "chrome")
	wantData := filepath.Join(base, "data", "profile")
	if cfg.Chrome.InstallDir != wantInstall {
		t.Errorf("InstallDir: got %q, want %q", cfg.Chrome.InstallDir, wantInstall)
	}
	if cfg.Chrome.UserDataDir != wantData {
		t.Errorf("UserDataDir: got %q, want %q", cfg.Chrome.UserDataDir, wantData)
	}
}

func TestResolvePathsLeavesAbsoluteAlone(t *testing.T) {
	cfg := Default()
	cfg.Chrome.InstallDir = `C:\abs\chrome`
	cfg.Chrome.UserDataDir = `D:\abs\data`

	cfg.ResolvePaths(`C:\app\base`)

	if cfg.Chrome.InstallDir != `C:\abs\chrome` {
		t.Errorf("InstallDir changed: %q", cfg.Chrome.InstallDir)
	}
	if cfg.Chrome.UserDataDir != `D:\abs\data` {
		t.Errorf("UserDataDir changed: %q", cfg.Chrome.UserDataDir)
	}
}

func TestResolvePathsNilSafe(t *testing.T) {
	var cfg *Config
	cfg.ResolvePaths(`C:\app\base`)
}
