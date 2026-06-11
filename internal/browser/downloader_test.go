package browser

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
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

func TestDownloadFileWithProgress(t *testing.T) {
	var progressCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "11")
		_, _ = w.Write([]byte("hello world"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloads", "file.txt")
	if err := DownloadFile(server.URL, dest, func(downloaded, total int64) {
		progressCalls++
		if total != 11 {
			t.Fatalf("unexpected total: %d", total)
		}
		if downloaded <= 0 {
			t.Fatalf("downloaded should be positive: %d", downloaded)
		}
	}); err != nil {
		t.Fatalf("download file: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("unexpected data: %s", data)
	}
	if progressCalls == 0 {
		t.Fatal("expected progress callback")
	}
}

func TestDownloadFileHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	if err := DownloadFile(server.URL, filepath.Join(t.TempDir(), "file.txt"), nil); err == nil {
		t.Fatal("expected http error")
	}
}

func TestExtractZIP(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(file)
	w, err := zw.Create("chrome-win64/chrome.exe")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write([]byte("fake chrome")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	dest := filepath.Join(dir, "out")
	if err := ExtractZIP(zipPath, dest); err != nil {
		t.Fatalf("extract zip: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "chrome-win64", "chrome.exe")); err != nil {
		t.Fatalf("expected extracted chrome: %v", err)
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
