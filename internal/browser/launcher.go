package browser

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LaunchOptions controls Chrome startup.
type LaunchOptions struct {
	ExePath        string
	UserDataDir    string
	RemotePort     int // 0 = auto
	AdditionalArgs []string
}

// Launch starts Chrome and returns the process and actual port.
func Launch(opts LaunchOptions) (*os.Process, int, error) {
	if opts.ExePath == "" {
		return nil, 0, fmt.Errorf("exe path is empty")
	}
	if _, err := os.Stat(opts.ExePath); err != nil {
		return nil, 0, fmt.Errorf("chrome exe not found: %w", err)
	}

	if opts.UserDataDir != "" {
		if err := os.MkdirAll(opts.UserDataDir, 0755); err != nil {
			return nil, 0, fmt.Errorf("create user data dir: %w", err)
		}
	}

	args := buildLaunchArgs(opts)

	cmd := exec.Command(opts.ExePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, 0, fmt.Errorf("start chrome: %w", err)
	}

	// Wait for DevToolsActivePort file
	port, err := waitForDevToolsPort(opts.UserDataDir, 30*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return nil, 0, fmt.Errorf("read devtools port: %w", err)
	}
	return cmd.Process, port, nil
}

func buildLaunchArgs(opts LaunchOptions) []string {
	port := opts.RemotePort
	if port < 0 {
		port = 0
	}
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-sandbox",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--disable-features=Translate,RendererCodeIntegrity",
		"--disable-sync",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-popup-blocking",
		"--disable-background-networking",
		"--disable-renderer-backgrounding",
		"--disable-extensions",
		"--disable-dev-shm-usage",
		"--disable-component-update",
		"--disable-default-apps",
		"--incognito",
		"--start-maximized",
		"--ignore-certificate-errors",
		"--allow-insecure-localhost",
	}
	if opts.UserDataDir != "" {
		args = append(args, "--user-data-dir="+opts.UserDataDir)
	}
	args = append(args, opts.AdditionalArgs...)
	args = append(args, "about:blank")
	return args
}

func waitForDevToolsPort(userDataDir string, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		port, err := ReadDevToolsPort(userDataDir)
		if err == nil && port > 0 {
			return port, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return 0, fmt.Errorf("timeout waiting for DevToolsActivePort")
}

// ReadDevToolsPort tries to read the active port from an existing profile.
func ReadDevToolsPort(userDataDir string) (int, error) {
	var lastErr error
	for _, portFile := range devToolsPortFiles(userDataDir) {
		data, err := os.ReadFile(portFile)
		if err != nil {
			lastErr = err
			continue
		}
		lines := strings.Split(string(data), "\n")
		if len(lines) == 0 {
			return 0, fmt.Errorf("empty DevToolsActivePort")
		}
		return strconv.Atoi(strings.TrimSpace(lines[0]))
	}
	if lastErr != nil {
		return 0, lastErr
	}
	return 0, fmt.Errorf("DevToolsActivePort not found")
}

func devToolsPortFiles(userDataDir string) []string {
	return []string{
		filepath.Join(userDataDir, "DevToolsActivePort"),
		filepath.Join(userDataDir, "Default", "DevToolsActivePort"),
	}
}

// IsChromeRunning checks if a process with the given user data dir is alive.
func IsChromeRunning(userDataDir string) bool {
	port, err := ReadDevToolsPort(userDataDir)
	if err != nil || port <= 0 {
		return false
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
