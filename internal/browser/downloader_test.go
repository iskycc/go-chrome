package browser

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
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

func TestVerifySHA256EmptyExpectedAndMissingFile(t *testing.T) {
	if err := VerifySHA256(filepath.Join(t.TempDir(), "missing.txt"), ""); err != nil {
		t.Fatalf("empty expected hash should skip verification: %v", err)
	}
	if err := VerifySHA256(filepath.Join(t.TempDir(), "missing.txt"), "abc"); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestFindChromeExeNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindChromeExe(dir)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestFindChromeExeCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "nested", "Chrome.EXE")
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(exe, []byte("fake"), 0644); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	got, err := FindChromeExe(dir)
	if err != nil {
		t.Fatalf("find chrome: %v", err)
	}
	if got != exe {
		t.Fatalf("unexpected exe: %s", got)
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

func TestDownloadFileCreateDestDirFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	base := t.TempDir()
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if err := DownloadFile(server.URL, filepath.Join(blocker, "file.txt"), nil); err == nil {
		t.Fatal("expected mkdir failure")
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

func TestExtractZIPRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "slip.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(file)
	w, err := zw.Create("../evil.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write([]byte("evil")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	if err := ExtractZIP(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected zip slip rejection")
	}
}

func TestExtractZIPInvalidArchive(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "bad.zip")
	if err := os.WriteFile(zipPath, []byte("not zip"), 0644); err != nil {
		t.Fatalf("write bad zip: %v", err)
	}
	if err := ExtractZIP(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected invalid zip error")
	}
}

func TestManagerInstallCustomURLWithHashAndCleanup(t *testing.T) {
	zipBytes := makeChromeZip(t)
	sum := sha256.Sum256(zipBytes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	dir := t.TempDir()
	installDir := filepath.Join(dir, "chrome")
	cfg := &config.ChromeConfig{
		DownloadSource:       "custom_url",
		CustomDownloadURL:    server.URL,
		CustomDownloadSHA256: hex.EncodeToString(sum[:]),
		CustomVersionLabel:   "mirror-build",
		InstallDir:           installDir,
		UserDataDir:          filepath.Join(dir, "profile"),
		KeepDownloadCache:    false,
	}
	mgr := NewManager(cfg)
	if err := mgr.Install(nil); err != nil {
		t.Fatalf("install custom: %v", err)
	}
	if !mgr.IsInstalled() {
		t.Fatal("expected installed chrome")
	}
	if mgr.manifest.Source != "custom_url" || mgr.manifest.CustomLabel != "mirror-build" {
		t.Fatalf("unexpected manifest: %+v", mgr.manifest)
	}
	if _, err := os.Stat(mgr.manifest.ExePath); err != nil {
		t.Fatalf("manifest exe path should point to installed chrome: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "downloads", "chrome-mirror-build.zip")); !os.IsNotExist(err) {
		t.Fatalf("download cache should be removed, err=%v", err)
	}
}

func TestManagerInstallCustomURLRequiresURL(t *testing.T) {
	mgr := NewManager(&config.ChromeConfig{
		DownloadSource: "custom_url",
		InstallDir:     filepath.Join(t.TempDir(), "chrome"),
	})
	if err := mgr.Install(nil); err == nil {
		t.Fatal("expected custom url error")
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

func makeChromeZip(t *testing.T) []byte {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "chrome.zip")
	file, err := os.Create(tmp)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(file)
	w, err := zw.Create("chrome-win64/chrome.exe")
	if err != nil {
		t.Fatalf("create chrome entry: %v", err)
	}
	if _, err := w.Write([]byte("fake chrome")); err != nil {
		t.Fatalf("write chrome entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	return data
}
