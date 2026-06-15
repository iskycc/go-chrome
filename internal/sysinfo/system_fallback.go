//go:build !linux && !windows

package sysinfo

import (
	"os"
	"runtime"
)

func systemStaticInfo() SystemSnapshot {
	name, _ := os.Hostname()
	return SystemSnapshot{
		OSName:      runtime.GOOS,
		Arch:        runtime.GOARCH,
		Hostname:    name,
		LogicalCPUs: runtime.NumCPU(),
	}
}

func systemCPUTimesNow() (systemCPUTimes, bool) {
	return systemCPUTimes{}, false
}

func fillSystemMemory(_ *SystemSnapshot) {}
