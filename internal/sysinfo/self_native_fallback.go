//go:build !linux && !windows

package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func currentProcessName() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Base(os.Args[0])
	}
	return filepath.Base(exe)
}

func currentProcessRSSMB() (float64, error) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.Sys) / 1024 / 1024, nil
}

func currentProcessCPUTime() (time.Duration, error) {
	return 0, nil
}
