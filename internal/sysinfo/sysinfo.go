// Package sysinfo provides lightweight, cross-platform process resource queries
// used by the UI information panel.
package sysinfo

import (
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessSnapshot holds resource usage for a single OS process.
type ProcessSnapshot struct {
	PID      int32
	Name     string
	CPU      float64 // percentage of a single core
	MemoryMB float64
	Exists   bool
}

// processHandle abstracts the parts of gopsutil/process we need so tests can
// inject fake processes.
type processHandle interface {
	Name() (string, error)
	Percent(interval time.Duration) (float64, error)
	MemoryInfo() (*process.MemoryInfoStat, error)
	CreateTime() (int64, error)
}

// processProvider creates a processHandle for a PID. Tests can replace it.
var processProvider = func(pid int32) (processHandle, error) {
	return process.NewProcess(pid)
}

// SelfInfo returns the current process resource usage.
func SelfInfo() (ProcessSnapshot, error) {
	return infoForPID(int32(os.Getpid()))
}

// ChromeInfo returns resource usage for the managed Chrome process.
// If pid <= 0 the result will have Exists=false and no error.
func ChromeInfo(pid int) (ProcessSnapshot, error) {
	if pid <= 0 {
		return ProcessSnapshot{Exists: false}, nil
	}
	return infoForPID(int32(pid))
}

func infoForPID(pid int32) (ProcessSnapshot, error) {
	p, err := processProvider(pid)
	if err != nil {
		return ProcessSnapshot{Exists: false}, nil
	}

	name, err := p.Name()
	if err != nil {
		name = ""
	}

	// Percent(0) returns average CPU usage since process start.
	cpu, err := p.Percent(0)
	if err != nil {
		cpu = 0
	}

	memMB := float64(0)
	memInfo, err := p.MemoryInfo()
	if err == nil && memInfo != nil {
		memMB = float64(memInfo.RSS) / 1024 / 1024
	}

	return ProcessSnapshot{
		PID:      pid,
		Name:     name,
		CPU:      cpu,
		MemoryMB: memMB,
		Exists:   true,
	}, nil
}

// FormatCPU returns a human-readable CPU percentage string.
func FormatCPU(cpu float64) string {
	return fmt.Sprintf("%.1f%%", cpu)
}

// FormatMemory returns a human-readable memory string in MB/GB.
func FormatMemory(mb float64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.2f GB", mb/1024)
	}
	return fmt.Sprintf("%.1f MB", mb)
}

// Uptime returns the duration since the current process started.
func Uptime() (time.Duration, error) {
	p, err := processProvider(int32(os.Getpid()))
	if err != nil {
		return 0, err
	}
	createTime, err := p.CreateTime()
	if err != nil {
		return 0, err
	}
	start := time.UnixMilli(createTime)
	return time.Since(start), nil
}
