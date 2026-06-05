package browser

import (
	"os"
	"path/filepath"
	"testing"

	"go-chrome/internal/config"
)

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	// SHA256 of "hello"
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := VerifySHA256(path, expected); err != nil {
		t.Fatalf("verify correct hash: %v", err)
	}
	if err := VerifySHA256(path, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestFindChromeExeNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindChromeExe(dir)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestManagerInstallSkipsWhenLocalChromeExists(t *testing.T) {
	dir := t.TempDir()
	installDir := filepath.Join(dir, "chrome")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "chrome.exe"), []byte("fake"), 0644); err != nil {
		t.Fatalf("write fake chrome: %v", err)
	}

	cfg := &config.ChromeConfig{
		DownloadSource:     "custom_url",
		CustomDownloadURL:  "",
		InstallDir:         installDir,
		UserDataDir:        filepath.Join(dir, "profile"),
		FallbackToOfficial: false,
	}
	mgr := NewManager(cfg)
	if err := mgr.Install(nil); err != nil {
		t.Fatalf("install should skip existing chrome: %v", err)
	}
}
