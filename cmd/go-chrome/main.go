package main

import (
	"fmt"
	"os"

	"go-chrome/internal/app"
	"go-chrome/internal/config"
	"go-chrome/internal/logx"
	"go-chrome/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	// Determine executable directory as base
	baseDir, err := app.ExecutableDir()
	if err != nil {
		baseDir = "."
	}

	// Ensure directories
	dirs, err := app.EnsureDirs(baseDir)
	if err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	// Load config
	cfg, err := config.Load(dirs.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	config.SetInstance(cfg)

	// Init logger
	if err := logx.Init(dirs.LogsDir, cfg.App.LogRetentionDays, nil); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logx.Close()

	logx.Info("go-chrome starting")

	// Start GUI
	uiApp := ui.New(cfg, dirs)
	uiApp.Run()

	logx.Info("go-chrome exiting")
	return nil
}
