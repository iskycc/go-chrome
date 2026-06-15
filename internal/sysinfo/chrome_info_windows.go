//go:build windows

package sysinfo

import (
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (s *Sampler) chromeInfoLocked(pid int32) ProcessSnapshot {
	snap, cpuTime, wall, err := processSnapshotWindows(pid, s.lastChromeCPUTime, s.lastChromeCPUWall)
	if err != nil {
		s.resetChromeLocked()
		return ProcessSnapshot{Exists: false}
	}
	s.chromePID = pid
	s.chrome = nil
	s.lastChromeCPUTime = cpuTime
	s.lastChromeCPUWall = wall
	return snap
}

func processSnapshotWindows(pid int32, lastCPUTime time.Duration, lastCPUWall time.Time) (ProcessSnapshot, time.Duration, time.Time, error) {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ,
		false,
		uint32(pid),
	)
	if err != nil {
		return ProcessSnapshot{}, 0, time.Time{}, err
	}
	defer windows.CloseHandle(handle)

	name := processNameWindows(handle)
	memMB, err := processRSSMBWindows(handle)
	if err != nil {
		return ProcessSnapshot{}, 0, time.Time{}, err
	}
	cpuTime, err := processCPUTimeWindows(handle)
	if err != nil {
		return ProcessSnapshot{}, 0, time.Time{}, err
	}
	wall := time.Now()
	cpu := cpuPercent(cpuTime, wall, lastCPUTime, lastCPUWall)

	return ProcessSnapshot{
		PID:      pid,
		Name:     name,
		CPU:      cpu,
		MemoryMB: memMB,
		Exists:   true,
	}, cpuTime, wall, nil
}

func processNameWindows(handle windows.Handle) string {
	buf := make([]uint16, syscall.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil || size == 0 {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buf[:size]))
}

func processRSSMBWindows(handle windows.Handle) (float64, error) {
	var counters processMemoryCounters
	counters.cb = uint32(unsafe.Sizeof(counters))
	r1, _, e1 := syscall.SyscallN(
		procGetProcessMemoryInfo.Addr(),
		uintptr(handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.cb),
	)
	if r1 == 0 {
		if e1 != 0 {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	return float64(counters.workingSetSize) / 1024 / 1024, nil
}

func processCPUTimeWindows(handle windows.Handle) (time.Duration, error) {
	var creationTime, exitTime, kernelTime, userTime windows.Filetime
	if err := windows.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime); err != nil {
		return 0, err
	}
	return filetimeDuration(kernelTime) + filetimeDuration(userTime), nil
}
