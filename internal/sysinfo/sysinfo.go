// Package sysinfo provides lightweight, cross-platform process resource queries
// used by the UI information panel.
package sysinfo

import (
	"fmt"
	"os"
	"runtime"
	"sync"
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

var appStartedAt = time.Now()

// Sampler reuses process handles across refreshes. gopsutil's CPU percentage
// calculation also depends on reusing the same Process instance, so creating a
// new process object on every UI refresh is both wasteful and less accurate.
type Sampler struct {
	mu sync.Mutex

	selfPID   int32
	selfStart time.Time
	selfName  string

	lastSelfCPUTime time.Duration
	lastSelfCPUWall time.Time

	chromePID int32
	chrome    processHandle
}

// NewSampler creates a reusable process sampler for UI refresh loops.
func NewSampler() *Sampler {
	return &Sampler{
		selfPID:   int32(os.Getpid()),
		selfStart: appStartedAt,
		selfName:  currentProcessName(),
	}
}

// SelfInfo returns the current process resource usage.
func SelfInfo() (ProcessSnapshot, error) {
	return infoForPID(int32(os.Getpid()))
}

// SelfInfoWithUptime returns the current process resource usage together with
// its start time and uptime using a single process handle.
func SelfInfoWithUptime() (ProcessSnapshot, time.Time, time.Duration, error) {
	return NewSampler().SelfInfoWithUptime()
}

// SelfInfoWithUptime returns current process usage together with start time and
// uptime, reusing the sampler's cached process handle.
func (s *Sampler) SelfInfoWithUptime() (ProcessSnapshot, time.Time, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	memMB, err := currentProcessRSSMB()
	if err != nil {
		memMB = 0
	}
	cpu := s.selfCPUPercentLocked()
	info := ProcessSnapshot{
		PID:      s.selfPID,
		Name:     s.selfName,
		CPU:      cpu,
		MemoryMB: memMB,
		Exists:   true,
	}
	return info, s.selfStart, time.Since(s.selfStart), nil
}

// ChromeInfo returns resource usage for the managed Chrome process.
// If pid <= 0 the result will have Exists=false and no error.
func ChromeInfo(pid int) (ProcessSnapshot, error) {
	if pid <= 0 {
		return ProcessSnapshot{Exists: false}, nil
	}
	return infoForPID(int32(pid))
}

// ChromeInfo returns resource usage for the managed Chrome process while
// reusing the process handle as long as the PID is unchanged.
func (s *Sampler) ChromeInfo(pid int) (ProcessSnapshot, error) {
	if pid <= 0 {
		s.mu.Lock()
		s.chromePID = 0
		s.chrome = nil
		s.mu.Unlock()
		return ProcessSnapshot{Exists: false}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.processForChromeLocked(int32(pid))
	if err != nil {
		s.chromePID = 0
		s.chrome = nil
		return ProcessSnapshot{Exists: false}, nil
	}
	return infoFromProcess(int32(pid), p), nil
}

func (s *Sampler) selfCPUPercentLocked() float64 {
	cpuTime, err := currentProcessCPUTime()
	if err != nil {
		return 0
	}
	now := time.Now()
	if s.lastSelfCPUWall.IsZero() {
		s.lastSelfCPUTime = cpuTime
		s.lastSelfCPUWall = now
		return 0
	}
	cpuDelta := cpuTime - s.lastSelfCPUTime
	wallDelta := now.Sub(s.lastSelfCPUWall)
	s.lastSelfCPUTime = cpuTime
	s.lastSelfCPUWall = now
	if cpuDelta <= 0 || wallDelta <= 0 {
		return 0
	}
	return cpuDelta.Seconds() / wallDelta.Seconds() * 100
}

func (s *Sampler) processForChromeLocked(pid int32) (processHandle, error) {
	if s.chrome != nil && s.chromePID == pid {
		return s.chrome, nil
	}
	p, err := processProvider(pid)
	if err != nil {
		return nil, err
	}
	s.chromePID = pid
	s.chrome = p
	return p, nil
}

func infoForPID(pid int32) (ProcessSnapshot, error) {
	p, err := processProvider(pid)
	if err != nil {
		return ProcessSnapshot{Exists: false}, nil
	}
	return infoFromProcess(pid, p), nil
}

func infoFromProcess(pid int32, p processHandle) ProcessSnapshot {
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
	}
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
	_, uptime, err := startTimeAndUptime()
	return uptime, err
}

// StartTime returns the local timestamp when the current process started.
func StartTime() (time.Time, error) {
	start, _, err := startTimeAndUptime()
	return start, err
}

func startTimeAndUptime() (time.Time, time.Duration, error) {
	p, err := processProvider(int32(os.Getpid()))
	if err != nil {
		return time.Time{}, 0, err
	}
	createTime, err := p.CreateTime()
	if err != nil {
		return time.Time{}, 0, err
	}
	start := time.UnixMilli(createTime)
	return start, time.Since(start), nil
}

// FormatStartTime returns a human-readable start timestamp string.
func FormatStartTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatUptime returns a stable uptime string that changes at most once per
// second, avoiding unnecessary text layout work during repeated refreshes.
func FormatUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Truncate(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}

// GoMemStats reports Go runtime heap memory in megabytes.
func GoMemStats() (heapAllocMB float64, heapSysMB float64) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapAlloc) / 1024 / 1024,
		float64(ms.HeapSys) / 1024 / 1024
}
