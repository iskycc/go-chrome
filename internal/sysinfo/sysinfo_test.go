package sysinfo

import (
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type fakeProcess struct {
	name       string
	nameErr    error
	cpu        float64
	cpuErr     error
	mem        *process.MemoryInfoStat
	memErr     error
	createTime int64
	createErr  error
}

func (f *fakeProcess) Name() (string, error) {
	return f.name, f.nameErr
}

func (f *fakeProcess) Percent(interval time.Duration) (float64, error) {
	return f.cpu, f.cpuErr
}

func (f *fakeProcess) MemoryInfo() (*process.MemoryInfoStat, error) {
	return f.mem, f.memErr
}

func (f *fakeProcess) CreateTime() (int64, error) {
	return f.createTime, f.createErr
}

func TestSelfInfo(t *testing.T) {
	info, err := SelfInfo()
	if err != nil {
		t.Fatalf("SelfInfo failed: %v", err)
	}
	if !info.Exists {
		t.Fatal("expected current process to exist")
	}
	if info.PID <= 0 {
		t.Fatalf("expected positive pid, got %d", info.PID)
	}
	if info.MemoryMB < 0 {
		t.Fatalf("expected non-negative memory, got %f", info.MemoryMB)
	}
}

func TestChromeInfo_InvalidPID(t *testing.T) {
	info, err := ChromeInfo(0)
	if err != nil {
		t.Fatalf("ChromeInfo(0) should not error: %v", err)
	}
	if info.Exists {
		t.Fatal("expected ChromeInfo(0) to report non-existent process")
	}
}

func TestChromeInfo_NonExistentPID(t *testing.T) {
	info, err := ChromeInfo(999999)
	if err != nil {
		t.Fatalf("ChromeInfo(nonexistent) should not error: %v", err)
	}
	if info.Exists {
		t.Fatal("expected non-existent process")
	}
}

func TestInfoForPID_NameError(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return &fakeProcess{nameErr: errors.New("name error")}, nil
	}
	defer func() { processProvider = old }()

	info, err := infoForPID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.Exists {
		t.Fatal("expected exists")
	}
	if info.Name != "" {
		t.Fatalf("expected empty name on error, got %q", info.Name)
	}
}

func TestInfoForPID_CPUError(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return &fakeProcess{cpuErr: errors.New("cpu error")}, nil
	}
	defer func() { processProvider = old }()

	info, err := infoForPID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CPU != 0 {
		t.Fatalf("expected zero cpu on error, got %f", info.CPU)
	}
}

func TestInfoForPID_MemoryError(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return &fakeProcess{memErr: errors.New("mem error")}, nil
	}
	defer func() { processProvider = old }()

	info, err := infoForPID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.MemoryMB != 0 {
		t.Fatalf("expected zero memory on error, got %f", info.MemoryMB)
	}
}

func TestInfoForPID_MemoryNil(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return &fakeProcess{mem: nil}, nil
	}
	defer func() { processProvider = old }()

	info, err := infoForPID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.MemoryMB != 0 {
		t.Fatalf("expected zero memory on nil meminfo, got %f", info.MemoryMB)
	}
}

func TestInfoForPID_FullValues(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return &fakeProcess{
			name: "chrome.exe",
			cpu:  12.5,
			mem:  &process.MemoryInfoStat{RSS: 256 * 1024 * 1024},
		}, nil
	}
	defer func() { processProvider = old }()

	info, err := infoForPID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.Exists {
		t.Fatal("expected exists")
	}
	if info.PID != 42 {
		t.Fatalf("expected pid 42, got %d", info.PID)
	}
	if info.Name != "chrome.exe" {
		t.Fatalf("expected chrome.exe, got %q", info.Name)
	}
	if info.CPU != 12.5 {
		t.Fatalf("expected cpu 12.5, got %f", info.CPU)
	}
	if info.MemoryMB != 256 {
		t.Fatalf("expected memory 256 MB, got %f", info.MemoryMB)
	}
}

func TestFormatters(t *testing.T) {
	if got := FormatCPU(12.345); got != "12.3%" {
		t.Fatalf("FormatCPU: got %q", got)
	}
	if got := FormatMemory(512); got != "512.0 MB" {
		t.Fatalf("FormatMemory(MB): got %q", got)
	}
	if got := FormatMemory(2048); got != "2.00 GB" {
		t.Fatalf("FormatMemory(GB): got %q", got)
	}
	if got := FormatMemory(1024); got != "1.00 GB" {
		t.Fatalf("FormatMemory(1024): got %q", got)
	}
}

func TestUptime(t *testing.T) {
	uptime, err := Uptime()
	if err != nil {
		t.Fatalf("Uptime failed: %v", err)
	}
	if uptime <= 0 {
		t.Fatalf("expected positive uptime, got %v", uptime)
	}
}

func TestStartTime(t *testing.T) {
	start, err := StartTime()
	if err != nil {
		t.Fatalf("StartTime failed: %v", err)
	}
	if start.IsZero() {
		t.Fatal("expected non-zero start time")
	}
	if time.Since(start) < 0 {
		t.Fatal("start time should not be in the future")
	}
}

func TestFormatStartTime(t *testing.T) {
	ts := time.Date(2026, 6, 14, 12, 30, 45, 0, time.UTC)
	got := FormatStartTime(ts)
	want := "2026-06-14 12:30:45"
	if got != want {
		t.Fatalf("FormatStartTime: got %q, want %q", got, want)
	}
}

func TestFormatUptime(t *testing.T) {
	if got := FormatUptime(90*time.Second + 250*time.Millisecond); got != "1m30s" {
		t.Fatalf("FormatUptime without hours: got %q", got)
	}
	if got := FormatUptime(2*time.Hour + 3*time.Minute + 4*time.Second + 999*time.Millisecond); got != "2h03m04s" {
		t.Fatalf("FormatUptime with hours: got %q", got)
	}
	if got := FormatUptime(-time.Second); got != "0m00s" {
		t.Fatalf("FormatUptime negative: got %q", got)
	}
}

func TestGoMemStats(t *testing.T) {
	heapAlloc, heapSys := GoMemStats()
	if heapAlloc < 0 {
		t.Fatalf("expected non-negative heapAlloc, got %f", heapAlloc)
	}
	if heapSys < 0 {
		t.Fatalf("expected non-negative heapSys, got %f", heapSys)
	}
	if heapAlloc > heapSys {
		t.Fatalf("expected heapAlloc <= heapSys, got alloc=%f sys=%f", heapAlloc, heapSys)
	}
}

func TestUptime_CreateTimeError(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return &fakeProcess{createErr: errors.New("create time error")}, nil
	}
	defer func() { processProvider = old }()

	_, err := Uptime()
	if err == nil {
		t.Fatal("expected error from CreateTime")
	}
}

func TestUptime_ProcessError(t *testing.T) {
	old := processProvider
	processProvider = func(pid int32) (processHandle, error) {
		return nil, errors.New("process error")
	}
	defer func() { processProvider = old }()

	_, err := Uptime()
	if err == nil {
		t.Fatal("expected error from process provider")
	}
}

func TestSamplerReusesProcessHandles(t *testing.T) {
	old := processProvider
	calls := 0
	processProvider = func(pid int32) (processHandle, error) {
		calls++
		return &fakeProcess{
			name:       "sampled",
			mem:        &process.MemoryInfoStat{RSS: 64 * 1024 * 1024},
			createTime: time.Now().Add(-time.Minute).UnixMilli(),
		}, nil
	}
	defer func() { processProvider = old }()

	sampler := NewSampler()
	if _, _, _, err := sampler.SelfInfoWithUptime(); err != nil {
		t.Fatalf("first self sample: %v", err)
	}
	if _, _, _, err := sampler.SelfInfoWithUptime(); err != nil {
		t.Fatalf("second self sample: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected self samples to avoid processProvider, got %d calls", calls)
	}

	if _, err := sampler.ChromeInfo(42); err != nil {
		t.Fatalf("first chrome sample: %v", err)
	}
	if _, err := sampler.ChromeInfo(42); err != nil {
		t.Fatalf("second chrome sample: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one processProvider call for same chrome PID, got %d", calls)
	}

	if _, err := sampler.ChromeInfo(43); err != nil {
		t.Fatalf("new chrome PID sample: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected processProvider call after chrome PID changed, got %d", calls)
	}
}
