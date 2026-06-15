//go:build windows

package sysinfo

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// readFileFunc is overridable for tests. The Windows implementation does not
// read procfs, so the var exists only to keep cross-platform test code in
// sysinfo_test.go compilable.
var readFileFunc = os.ReadFile

var (
	procGetSystemTimes       = syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
)

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

func systemStaticInfo() SystemSnapshot {
	snap := SystemSnapshot{
		OSName:      "Windows",
		Arch:        runtime.GOARCH,
		Hostname:    hostnameWindows(),
		LogicalCPUs: runtime.NumCPU(),
	}
	fillWindowsVersion(&snap)
	fillWindowsCPUInfo(&snap)
	return snap
}

func systemCPUTimesNow() (systemCPUTimes, bool) {
	var idle, kernel, user windows.Filetime
	r1, _, _ := syscall.SyscallN(
		procGetSystemTimes.Addr(),
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		return systemCPUTimes{}, false
	}
	idleTicks := filetimeTicks(idle)
	kernelTicks := filetimeTicks(kernel)
	userTicks := filetimeTicks(user)
	return systemCPUTimes{idle: idleTicks, total: kernelTicks + userTicks}, true
}

func fillSystemMemory(snap *SystemSnapshot) {
	var mem memoryStatusEx
	mem.length = uint32(unsafe.Sizeof(mem))
	r1, _, _ := syscall.SyscallN(
		procGlobalMemoryStatusEx.Addr(),
		uintptr(unsafe.Pointer(&mem)),
	)
	if r1 == 0 || mem.totalPhys == 0 {
		return
	}
	used := mem.totalPhys - mem.availPhys
	snap.MemoryTotalMB = float64(mem.totalPhys) / 1024 / 1024
	snap.MemoryAvailableMB = float64(mem.availPhys) / 1024 / 1024
	snap.MemoryUsedMB = float64(used) / 1024 / 1024
	snap.MemoryUsagePercent = float64(used) / float64(mem.totalPhys) * 100
}

func fillWindowsVersion(snap *SystemSnapshot) {
	version := windows.RtlGetVersion()
	if version != nil {
		snap.OSVersion = fmt.Sprintf("%d.%d", version.MajorVersion, version.MinorVersion)
		snap.OSBuild = fmt.Sprintf("%d", version.BuildNumber)
		if servicePack := windows.UTF16ToString(version.CsdVersion[:]); servicePack != "" {
			snap.OSVersion += " " + servicePack
		}
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	if product, _, err := key.GetStringValue("ProductName"); err == nil && product != "" {
		snap.OSName = product
	}
	if display, _, err := key.GetStringValue("DisplayVersion"); err == nil && display != "" {
		snap.OSVersion = strings.TrimSpace(snap.OSVersion + " " + display)
	} else if release, _, err := key.GetStringValue("ReleaseId"); err == nil && release != "" {
		snap.OSVersion = strings.TrimSpace(snap.OSVersion + " " + release)
	}
	if build, _, err := key.GetStringValue("CurrentBuildNumber"); err == nil && build != "" {
		snap.OSBuild = build
	}
}

func fillWindowsCPUInfo(snap *SystemSnapshot) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	if model, _, err := key.GetStringValue("ProcessorNameString"); err == nil {
		snap.CPUModel = strings.TrimSpace(model)
	}
	if vendor, _, err := key.GetStringValue("VendorIdentifier"); err == nil {
		snap.CPUVendor = strings.TrimSpace(vendor)
	}
	if identifier, _, err := key.GetStringValue("Identifier"); err == nil {
		snap.CPUIdentifier = strings.TrimSpace(identifier)
	}
	if mhz, _, err := key.GetIntegerValue("~MHz"); err == nil {
		snap.CPUMHz = int(mhz)
	}
}

func hostnameWindows() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

func filetimeTicks(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}
