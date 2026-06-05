package browser

import (
	"os"
	"path/filepath"
	"testing"
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
