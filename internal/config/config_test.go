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
}
