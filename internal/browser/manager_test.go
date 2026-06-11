package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-chrome/internal/config"
)

func TestManagerManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.ChromeConfig{InstallDir: filepath.Join(dir, "chrome")}
	mgr := NewManager(cfg)
	mgr.manifest = VersionManifest{
		Version:     "1.2.3",
		Source:      "custom_url",
		Channel:     "Stable",
		DownloadURL: "https://example.test/chrome.zip",
		InstalledAt: time.Now(),
		ExePath:     filepath.Join(cfg.InstallDir, "chrome.exe"),
	}

	if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := mgr.SaveManifest(); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	loaded := NewManager(cfg)
	if err := loaded.LoadManifest(); err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if loaded.manifest.Version != "1.2.3" {
		t.Fatalf("version mismatch: %s", loaded.manifest.Version)
	}
}

func TestLoadManifestInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.ChromeConfig{InstallDir: filepath.Join(dir, "chrome")}
	if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.InstallDir, "chrome-version.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := NewManager(cfg).LoadManifest(); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadManifestClearsStaleManifestWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.ChromeConfig{InstallDir: filepath.Join(dir, "chrome")}
	mgr := NewManager(cfg)
	mgr.manifest = VersionManifest{
		Version: "stale",
		ExePath: filepath.Join(dir, "old", "chrome.exe"),
	}

	if err := mgr.LoadManifest(); err == nil {
		t.Fatal("expected missing manifest error")
	}
	if mgr.manifest.Version != "" || mgr.manifest.ExePath != "" {
		t.Fatalf("expected stale manifest to be cleared: %+v", mgr.manifest)
	}
}

func TestManagerExePathCandidates(t *testing.T) {
	dir := t.TempDir()
	installDir := filepath.Join(dir, "chrome")
	exePath := filepath.Join(installDir, "chrome-win64", "chrome.exe")
	if err := os.MkdirAll(filepath.Dir(exePath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(exePath, []byte("fake"), 0644); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	mgr := NewManager(&config.ChromeConfig{InstallDir: installDir})
	got, err := mgr.ExePath()
	if err != nil {
		t.Fatalf("exe path: %v", err)
	}
	if got != exePath {
		t.Fatalf("unexpected exe path: %s", got)
	}
	if !mgr.IsInstalled() {
		t.Fatal("expected installed")
	}
}

func TestManagerExePathUsesValidManifestPath(t *testing.T) {
	dir := t.TempDir()
	installDir := filepath.Join(dir, "chrome")
	manifestExe := filepath.Join(dir, "manifest", "chrome.exe")
	candidateExe := filepath.Join(installDir, "chrome.exe")
	for _, exe := range []string{manifestExe, candidateExe} {
		if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(exe, []byte("fake"), 0644); err != nil {
			t.Fatalf("write exe: %v", err)
		}
	}

	mgr := NewManager(&config.ChromeConfig{InstallDir: installDir})
	mgr.manifest.ExePath = manifestExe
	got, err := mgr.ExePath()
	if err != nil {
		t.Fatalf("exe path: %v", err)
	}
	if got != manifestExe {
		t.Fatalf("expected manifest exe, got %s", got)
	}
}

func TestManagerStatusInstalledAndNotInstalled(t *testing.T) {
	dir := t.TempDir()
	installDir := filepath.Join(dir, "chrome")
	mgr := NewManager(&config.ChromeConfig{
		InstallDir:  installDir,
		UserDataDir: filepath.Join(dir, "profile"),
	})
	if got := mgr.Status(); got != ChromeNotInstalled {
		t.Fatalf("expected not installed, got %v", got)
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "chrome.exe"), []byte("fake"), 0644); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	if got := mgr.Status(); got != ChromeInstalled {
		t.Fatalf("expected installed, got %v", got)
	}
}

func TestManagerStartAndStartReplayRequireInstalledChrome(t *testing.T) {
	mgr := NewManager(&config.ChromeConfig{
		InstallDir:  filepath.Join(t.TempDir(), "chrome"),
		UserDataDir: filepath.Join(t.TempDir(), "profile"),
	})
	if _, err := mgr.Start(); err == nil || !strings.Contains(err.Error(), "chrome not installed") {
		t.Fatalf("expected start missing chrome error, got %v", err)
	}
	if _, err := mgr.StartReplay("run-1"); err == nil || !strings.Contains(err.Error(), "chrome not installed") {
		t.Fatalf("expected replay missing chrome error, got %v", err)
	}
}

func TestManagerStopWithoutProcess(t *testing.T) {
	if err := NewManager(&config.ChromeConfig{}).Stop(); err != nil {
		t.Fatalf("stop without process: %v", err)
	}
}
