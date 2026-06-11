package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config holds all application settings.
type Config struct {
	Chrome ChromeConfig `json:"chrome"`
	Runner RunnerConfig `json:"runner"`
	App    AppConfig    `json:"app"`
}

// ChromeConfig holds Chrome-related settings.
type ChromeConfig struct {
	DownloadSource       string `json:"downloadSource"` // official_stable, official_fixed_version, custom_url
	Channel              string `json:"channel"`        // Stable, Beta, Dev, Canary
	FixedVersion         string `json:"fixedVersion"`
	CustomDownloadURL    string `json:"customDownloadURL"`
	CustomDownloadSHA256 string `json:"customDownloadSHA256"`
	CustomVersionLabel   string `json:"customVersionLabel"`
	FallbackToOfficial   bool   `json:"fallbackToOfficial"`
	InstallDir           string `json:"installDir"`
	UserDataDir          string `json:"userDataDir"`
	KeepDownloadCache    bool   `json:"keepDownloadCache"`
}

// RunnerConfig holds automation runner settings.
type RunnerConfig struct {
	DefaultTimeoutMs       int    `json:"defaultTimeoutMs"`
	DefaultWaitAfterMs     int    `json:"defaultWaitAfterMs"`
	TemplateRandomSeedMode string `json:"templateRandomSeedMode"` // system, fixed
	TemplatePreviewCount   int    `json:"templatePreviewCount"`
	MaskInputValueInLogs   bool   `json:"maskInputValueInLogs"`
	AutoScreenshotOnFinish bool   `json:"autoScreenshotOnFinish"`
}

// AppConfig holds general application settings.
type AppConfig struct {
	CloseManagedChromeOnExit bool   `json:"closeManagedChromeOnExit"`
	LogRetentionDays         int    `json:"logRetentionDays"`
	Theme                    string `json:"theme"`
	WindowWidth              int    `json:"windowWidth"`
	WindowHeight             int    `json:"windowHeight"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Chrome: ChromeConfig{
			DownloadSource:     "official_stable",
			Channel:            "Stable",
			FallbackToOfficial: true,
			InstallDir:         "./chrome",
			UserDataDir:        "./data/chrome-profile",
			KeepDownloadCache:  true,
		},
		Runner: RunnerConfig{
			DefaultTimeoutMs:       10000,
			DefaultWaitAfterMs:     500,
			TemplateRandomSeedMode: "system",
			TemplatePreviewCount:   5,
			MaskInputValueInLogs:   true,
			AutoScreenshotOnFinish: false,
		},
		App: AppConfig{
			CloseManagedChromeOnExit: true,
			LogRetentionDays:         14,
			Theme:                    "default",
			WindowWidth:              1400,
			WindowHeight:             900,
		},
	}
}

var (
	instance *Config
	once     sync.Once
)

// Load reads configuration from the given path, creating a default if missing.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := Default()
			if err := Save(path, cfg); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Ensure defaults for missing fields
	defaultCfg := Default()
	var raw struct {
		Chrome map[string]json.RawMessage `json:"chrome"`
	}
	_ = json.Unmarshal(data, &raw)
	if cfg.Chrome.DownloadSource == "" {
		cfg.Chrome.DownloadSource = defaultCfg.Chrome.DownloadSource
	}
	if cfg.Chrome.Channel == "" {
		cfg.Chrome.Channel = defaultCfg.Chrome.Channel
	}
	if _, ok := raw.Chrome["fallbackToOfficial"]; !ok {
		cfg.Chrome.FallbackToOfficial = defaultCfg.Chrome.FallbackToOfficial
	}
	if _, ok := raw.Chrome["keepDownloadCache"]; !ok {
		cfg.Chrome.KeepDownloadCache = defaultCfg.Chrome.KeepDownloadCache
	}
	if cfg.Chrome.InstallDir == "" {
		cfg.Chrome.InstallDir = defaultCfg.Chrome.InstallDir
	}
	if cfg.Chrome.UserDataDir == "" {
		cfg.Chrome.UserDataDir = defaultCfg.Chrome.UserDataDir
	}
	if cfg.Runner.DefaultTimeoutMs == 0 {
		cfg.Runner.DefaultTimeoutMs = defaultCfg.Runner.DefaultTimeoutMs
	}
	if cfg.Runner.DefaultWaitAfterMs == 0 {
		cfg.Runner.DefaultWaitAfterMs = defaultCfg.Runner.DefaultWaitAfterMs
	}
	if cfg.App.LogRetentionDays == 0 {
		cfg.App.LogRetentionDays = defaultCfg.App.LogRetentionDays
	}
	if cfg.App.WindowWidth == 0 {
		cfg.App.WindowWidth = defaultCfg.App.WindowWidth
	}
	if cfg.App.WindowHeight == 0 {
		cfg.App.WindowHeight = defaultCfg.App.WindowHeight
	}
	return &cfg, nil
}

// Save writes configuration to the given path.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Get returns the singleton config instance (must call Load first).
func Get() *Config {
	return instance
}

// SetInstance sets the singleton instance (used after Load).
func SetInstance(cfg *Config) {
	instance = cfg
}
