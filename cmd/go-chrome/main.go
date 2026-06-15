package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"go-chrome/internal/app"
	"go-chrome/internal/config"
	"go-chrome/internal/logx"
	"go-chrome/internal/singleinstance"
	"go-chrome/internal/ui"
)

type launchArgs struct {
	flowID string
	envID  string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	args := parseArgs()

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
	cfg.ResolvePaths(baseDir)
	config.SetInstance(cfg)

	// Init logger
	if err := logx.Init(dirs.LogsDir, cfg.App.LogRetentionDays, nil); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logx.Close()

	logx.Info("go-chrome starting")

	var autoRun *singleinstance.RunRequest
	if args.flowID != "" && args.envID != "" {
		autoRun = &singleinstance.RunRequest{FlowID: args.flowID, EnvID: args.envID}
	}

	uiApp := ui.New(cfg, dirs)

	if autoRun != nil {
		ctx := context.Background()
		res, inst, err := singleinstance.TryStart(ctx, *autoRun, func(req singleinstance.RunRequest) {
			uiApp.TriggerAutoRun(req.FlowID, req.EnvID)
		})
		if err != nil {
			logx.Warnf("single instance check failed: %v", err)
		}
		switch res {
		case singleinstance.ResultSent:
			logx.Info("forwarded auto-run to existing instance")
			if inst != nil {
				inst.Shutdown()
			}
			return nil
		case singleinstance.ResultFallback:
			logx.Warn("could not forward to existing instance; starting new instance")
		case singleinstance.ResultStarted:
			logx.Info("first instance, will auto-run after UI initializes")
		}
		if inst != nil {
			defer inst.Shutdown()
		}
		uiApp.SetAutoRun(autoRun.FlowID, autoRun.EnvID)
	}

	uiApp.Run()

	logx.Info("go-chrome exiting")
	return nil
}

func parseArgs() launchArgs {
	var a launchArgs
	flag.StringVar(&a.flowID, "flow", "", "flow ID to run automatically on startup")
	flag.StringVar(&a.envID, "env", "", "environment ID to use for automatic run")
	flag.Parse()
	return a
}
