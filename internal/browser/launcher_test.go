package browser

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBuildLaunchArgsIncludesReplayRequirements(t *testing.T) {
	userDataDir := filepath.Join("data", "chrome-profile")
	args := buildLaunchArgs(LaunchOptions{UserDataDir: userDataDir})
	assertContains(t, args, "--incognito")
	assertContains(t, args, "--start-maximized")
	assertContains(t, args, "--ignore-certificate-errors")
	assertContains(t, args, "--allow-insecure-localhost")
	assertContains(t, args, "--remote-debugging-port=0")
	assertContains(t, args, "--user-data-dir="+userDataDir)
}

func TestBuildLaunchArgsUsesConfiguredPort(t *testing.T) {
	args := buildLaunchArgs(LaunchOptions{RemotePort: 9333})
	assertContains(t, args, "--remote-debugging-port=9333")
}

func TestBuildLaunchArgsNormalizesNegativePortAndAppendsExtraArgs(t *testing.T) {
	args := buildLaunchArgs(LaunchOptions{RemotePort: -1, AdditionalArgs: []string{"--foo=bar"}})
	assertContains(t, args, "--remote-debugging-port=0")
	assertContains(t, args, "--foo=bar")
}

func TestReadDevToolsPortSupportsRootAndDefaultLocations(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadDevToolsPort(dir); err == nil {
		t.Fatal("expected missing port file error")
	}

	writePort(t, filepath.Join(dir, "DevToolsActivePort"), "9223\n/devtools/browser/test")
	port, err := ReadDevToolsPort(dir)
	if err != nil {
		t.Fatalf("read root port: %v", err)
	}
	if port != 9223 {
		t.Fatalf("unexpected root port: %d", port)
	}

	dir = t.TempDir()
	writePort(t, filepath.Join(dir, "Default", "DevToolsActivePort"), "9224\n/devtools/browser/test")
	port, err = ReadDevToolsPort(dir)
	if err != nil {
		t.Fatalf("read default port: %v", err)
	}
	if port != 9224 {
		t.Fatalf("unexpected default port: %d", port)
	}
}

func TestReadDevToolsPortInvalidPort(t *testing.T) {
	dir := t.TempDir()
	writePort(t, filepath.Join(dir, "DevToolsActivePort"), "not-a-port\n")
	if _, err := ReadDevToolsPort(dir); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestWaitForDevToolsPortReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	writePort(t, filepath.Join(dir, "DevToolsActivePort"), "9225\n")
	port, err := waitForDevToolsPort(dir, time.Second)
	if err != nil {
		t.Fatalf("wait for port: %v", err)
	}
	if port != 9225 {
		t.Fatalf("unexpected port: %d", port)
	}
}

func TestWaitForDevToolsPortTimeout(t *testing.T) {
	if _, err := waitForDevToolsPort(t.TempDir(), time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestIsChromeRunning(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"Browser":"Chrome"}`))
	}))
	defer okServer.Close()

	dir := t.TempDir()
	writePort(t, filepath.Join(dir, "DevToolsActivePort"), serverPort(t, okServer.URL)+"\n")
	if !IsChromeRunning(dir) {
		t.Fatal("expected chrome running")
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer failServer.Close()
	dir = t.TempDir()
	writePort(t, filepath.Join(dir, "DevToolsActivePort"), serverPort(t, failServer.URL)+"\n")
	if IsChromeRunning(dir) {
		t.Fatal("expected chrome not running for non-200 version endpoint")
	}
}

func TestLaunchRejectsEmptyAndMissingExe(t *testing.T) {
	if _, _, err := Launch(LaunchOptions{}); err == nil {
		t.Fatal("expected empty exe error")
	}
	if _, _, err := Launch(LaunchOptions{ExePath: filepath.Join(t.TempDir(), "chrome.exe")}); err == nil {
		t.Fatal("expected missing exe error")
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, v := range values {
		if v == want {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, values)
}

func writePort(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write port: %v", err)
	}
}

func serverPort(t *testing.T, rawURL string) string {
	t.Helper()
	idx := strings.LastIndex(rawURL, ":")
	if idx < 0 {
		t.Fatalf("server URL missing port: %s", rawURL)
	}
	port := rawURL[idx+1:]
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatalf("server URL has invalid port %q: %v", port, err)
	}
	return port
}
