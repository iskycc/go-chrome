package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-chrome/internal/config"
	"go-chrome/internal/logx"
)

// ChromeStatus enumerates Chrome lifecycle states.
type ChromeStatus int

const (
	ChromeNotInstalled ChromeStatus = iota
	ChromeInstalled
	ChromeDownloading
	ChromeStarting
	ChromeRunning
	ChromeStartFailed
)

// Manager handles Chrome lifecycle.
type Manager struct {
	cfg      *config.ChromeConfig
	manifest VersionManifest
	proc     *os.Process
}

// NewManager creates a new browser manager.
func NewManager(cfg *config.ChromeConfig) *Manager {
	return &Manager{cfg: cfg}
}

// ManifestPath returns the path to the version manifest.
func (m *Manager) ManifestPath() string {
	return filepath.Join(m.cfg.InstallDir, "chrome-version.json")
}

// LoadManifest reads the installed manifest if present.
func (m *Manager) LoadManifest() error {
	path := m.ManifestPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no manifest found")
		}
		return err
	}
	return json.Unmarshal(data, &m.manifest)
}

// SaveManifest writes the version manifest.
func (m *Manager) SaveManifest() error {
	path := m.ManifestPath()
	data, err := json.MarshalIndent(m.manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// IsInstalled checks if chrome.exe exists.
func (m *Manager) IsInstalled() bool {
	exe, err := m.ExePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(exe)
	return err == nil
}

// ExePath returns the resolved chrome.exe path.
func (m *Manager) ExePath() (string, error) {
	if m.manifest.ExePath != "" {
		if _, err := os.Stat(m.manifest.ExePath); err == nil {
			return m.manifest.ExePath, nil
		}
	}
	// Try common layout
	candidates := []string{
		filepath.Join(m.cfg.InstallDir, "chrome.exe"),
		filepath.Join(m.cfg.InstallDir, "chrome-win64", "chrome.exe"),
		filepath.Join(m.cfg.InstallDir, "chrome-win32", "chrome.exe"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	// Fallback to search
	return FindChromeExe(m.cfg.InstallDir)
}

// Install downloads and installs Chrome.
func (m *Manager) Install(progress DownloadProgress) error {
	if m.IsInstalled() {
		exe, _ := m.ExePath()
		logx.Infof("Local Chrome already exists, skip download: %s", exe)
		return nil
	}

	var version, url, hash, source, label string

	switch m.cfg.DownloadSource {
	case "custom_url":
		if m.cfg.CustomDownloadURL == "" {
			return fmt.Errorf("custom download URL is empty")
		}
		url = m.cfg.CustomDownloadURL
		hash = m.cfg.CustomDownloadSHA256
		version = m.cfg.CustomVersionLabel
		if version == "" {
			version = "custom"
		}
		source = "custom_url"
		label = m.cfg.CustomVersionLabel
	case "official_fixed_version":
		// TODO: fetch specific version
		return fmt.Errorf("official_fixed_version not implemented")
	default:
		// official_stable
		v, u, h, err := FetchStableInfo()
		if err != nil {
			return fmt.Errorf("fetch stable info: %w", err)
		}
		version, url, hash = v, u, h
		source = "official"
		label = ""
	}

	// Download
	dlDir := filepath.Join(filepath.Dir(m.cfg.InstallDir), "data", "downloads")
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return err
	}
	zipPath := filepath.Join(dlDir, fmt.Sprintf("chrome-%s.zip", version))
	logx.Infof("Downloading Chrome from %s", url)
	if err := DownloadFile(url, zipPath, progress); err != nil {
		// Fallback
		if m.cfg.FallbackToOfficial && source == "custom_url" {
			logx.Warn("Custom download failed, falling back to official stable")
			v, u, h, err2 := FetchStableInfo()
			if err2 != nil {
				return fmt.Errorf("custom download failed: %w; fallback also failed: %v", err, err2)
			}
			version, url, hash = v, u, h
			source = "official"
			label = ""
			zipPath = filepath.Join(dlDir, fmt.Sprintf("chrome-%s.zip", version))
			if err2 := DownloadFile(url, zipPath, progress); err2 != nil {
				return fmt.Errorf("custom download failed: %w; fallback download also failed: %v", err, err2)
			}
		} else {
			return fmt.Errorf("download failed: %w", err)
		}
	}

	// Verify
	if hash != "" {
		logx.Info("Verifying SHA256...")
		if err := VerifySHA256(zipPath, hash); err != nil {
			return fmt.Errorf("verify: %w", err)
		}
	}

	// Backup existing
	if m.IsInstalled() {
		bak := m.cfg.InstallDir + ".bak"
		os.RemoveAll(bak)
		if err := os.Rename(m.cfg.InstallDir, bak); err != nil {
			return fmt.Errorf("backup existing chrome: %w", err)
		}
		logx.Info("Backed up existing Chrome")
	}

	// Extract to temp
	tmpDir := m.cfg.InstallDir + ".tmp"
	os.RemoveAll(tmpDir)
	logx.Info("Extracting...")
	if err := ExtractZIP(zipPath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		m.rollback()
		return fmt.Errorf("extract: %w", err)
	}

	// Find chrome.exe
	exe, err := FindChromeExe(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		m.rollback()
		return fmt.Errorf("find chrome.exe after extract: %w", err)
	}

	// Atomic move
	if err := os.Rename(tmpDir, m.cfg.InstallDir); err != nil {
		os.RemoveAll(tmpDir)
		m.rollback()
		return fmt.Errorf("move chrome dir: %w", err)
	}

	m.manifest = VersionManifest{
		Version:     version,
		Source:      source,
		Channel:     m.cfg.Channel,
		DownloadURL: url,
		CustomLabel: label,
		InstalledAt: time.Now(),
		ExePath:     exe,
	}
	if err := m.SaveManifest(); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	// Cleanup tmp and old backup
	os.RemoveAll(tmpDir)
	os.RemoveAll(m.cfg.InstallDir + ".bak")
	if !m.cfg.KeepDownloadCache {
		os.Remove(zipPath)
	}
	logx.Info("Chrome installed successfully")
	return nil
}

func (m *Manager) rollback() {
	bak := m.cfg.InstallDir + ".bak"
	if _, err := os.Stat(bak); err == nil {
		os.RemoveAll(m.cfg.InstallDir)
		os.Rename(bak, m.cfg.InstallDir)
		logx.Info("Rolled back to previous Chrome")
	}
}

// Start launches Chrome if not already running.
func (m *Manager) Start() (int, error) {
	if IsChromeRunning(m.cfg.UserDataDir) {
		logx.Info("Chrome already running, reusing")
		return ReadDevToolsPort(m.cfg.UserDataDir)
	}
	exe, err := m.ExePath()
	if err != nil {
		return 0, fmt.Errorf("chrome not installed: %w", err)
	}
	proc, port, err := Launch(LaunchOptions{
		ExePath:     exe,
		UserDataDir: m.cfg.UserDataDir,
	})
	if err != nil {
		return 0, err
	}
	m.proc = proc
	logx.Infof("Chrome started on port %d", port)
	return port, nil
}

// StartReplay launches a fresh Chrome instance for a replay run.
func (m *Manager) StartReplay(runID string) (int, error) {
	exe, err := m.ExePath()
	if err != nil {
		return 0, fmt.Errorf("chrome not installed: %w", err)
	}
	if err := m.Stop(); err != nil {
		logx.Warnf("stop previous managed Chrome failed: %v", err)
	}
	userDataDir := filepath.Join(m.cfg.UserDataDir, "replay", runID)
	os.RemoveAll(userDataDir)
	proc, port, err := Launch(LaunchOptions{
		ExePath:     exe,
		UserDataDir: userDataDir,
	})
	if err != nil {
		return 0, err
	}
	m.proc = proc
	logx.Infof("Replay Chrome started on port %d", port)
	return port, nil
}

// Stop terminates the Chrome process managed by this manager.
func (m *Manager) Stop() error {
	if m.proc == nil {
		return nil
	}
	logx.Info("Stopping managed Chrome")
	err := m.proc.Kill()
	_, _ = m.proc.Wait()
	m.proc = nil
	return err
}

// Status returns the current Chrome installation/launch status.
func (m *Manager) Status() ChromeStatus {
	if m.proc != nil {
		if port, err := ReadDevToolsPort(m.cfg.UserDataDir); err == nil && port > 0 {
			return ChromeRunning
		}
		return ChromeStarting
	}
	if m.IsInstalled() {
		return ChromeInstalled
	}
	return ChromeNotInstalled
}
