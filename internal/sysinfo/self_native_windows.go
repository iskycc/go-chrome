//go:build windows

package sysinfo

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetProcessMemoryInfo = syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

func currentProcessName() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Base(os.Args[0])
	}
	return filepath.Base(exe)
}

func currentProcessRSSMB() (float64, error) {
	handle, err := windows.GetCurrentProcess()
	if err != nil {
		return 0, err
	}
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

func currentProcessCPUTime() (time.Duration, error) {
	handle, err := windows.GetCurrentProcess()
	if err != nil {
		return 0, err
	}
	var creationTime, exitTime, kernelTime, userTime windows.Filetime
	if err := windows.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime); err != nil {
		return 0, err
	}
	return filetimeDuration(kernelTime) + filetimeDuration(userTime), nil
}

func filetimeDuration(ft windows.Filetime) time.Duration {
	ticks := uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
	return time.Duration(ticks * 100)
}
