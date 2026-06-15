//go:build linux

package sysinfo

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

func systemStaticInfo() SystemSnapshot {
	snap := SystemSnapshot{
		OSName:      osReleaseValue("PRETTY_NAME"),
		Kernel:      readTrimmed("/proc/sys/kernel/osrelease"),
		Arch:        runtime.GOARCH,
		Hostname:    hostname(),
		LogicalCPUs: runtime.NumCPU(),
	}
	if snap.OSName == "" {
		snap.OSName = runtime.GOOS
	}
	fillLinuxCPUInfo(&snap)
	return snap
}

func systemCPUTimesNow() (systemCPUTimes, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return systemCPUTimes{}, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return systemCPUTimes{}, false
		}
		var total uint64
		values := make([]uint64, 0, len(fields)-1)
		for _, field := range fields[1:] {
			v, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return systemCPUTimes{}, false
			}
			values = append(values, v)
			total += v
		}
		idle := values[3]
		if len(values) > 4 {
			idle += values[4]
		}
		return systemCPUTimes{idle: idle, total: total}, true
	}
	return systemCPUTimes{}, false
}

func fillSystemMemory(snap *SystemSnapshot) {
	mem := linuxMemInfo()
	totalKB := mem["MemTotal"]
	availableKB := mem["MemAvailable"]
	if totalKB <= 0 {
		return
	}
	if availableKB < 0 {
		availableKB = 0
	}
	usedKB := totalKB - availableKB
	if usedKB < 0 {
		usedKB = 0
	}
	snap.MemoryTotalMB = float64(totalKB) / 1024
	snap.MemoryAvailableMB = float64(availableKB) / 1024
	snap.MemoryUsedMB = float64(usedKB) / 1024
	snap.MemoryUsagePercent = float64(usedKB) / float64(totalKB) * 100
}

func fillLinuxCPUInfo(snap *SystemSnapshot) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return
	}
	physicalCores := 0
	seenCores := map[string]struct{}{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := splitColonLine(line)
		if !ok {
			continue
		}
		switch key {
		case "model name":
			if snap.CPUModel == "" {
				snap.CPUModel = value
			}
		case "vendor_id":
			if snap.CPUVendor == "" {
				snap.CPUVendor = value
			}
		case "cpu MHz":
			if snap.CPUMHz == 0 {
				if mhz, err := strconv.ParseFloat(value, 64); err == nil {
					snap.CPUMHz = int(mhz + 0.5)
				}
			}
		case "cpu cores":
			if physicalCores == 0 {
				physicalCores, _ = strconv.Atoi(value)
			}
		}
	}
	physicalID, coreID := "", ""
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := splitColonLine(line)
		if !ok {
			if physicalID != "" && coreID != "" {
				seenCores[physicalID+":"+coreID] = struct{}{}
			}
			physicalID, coreID = "", ""
			continue
		}
		switch key {
		case "physical id":
			physicalID = value
		case "core id":
			coreID = value
		}
	}
	if physicalID != "" && coreID != "" {
		seenCores[physicalID+":"+coreID] = struct{}{}
	}
	if len(seenCores) > 0 {
		snap.PhysicalCores = len(seenCores)
	} else {
		snap.PhysicalCores = physicalCores
	}
}

func linuxMemInfo() map[string]int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil
	}
	out := make(map[string]int64)
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := splitColonLine(line)
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		kb, err := strconv.ParseInt(fields[0], 10, 64)
		if err == nil {
			out[key] = kb
		}
	}
	return out
}

func osReleaseValue(key string) string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok || k != key {
			continue
		}
		return strings.Trim(v, `"`)
	}
	return ""
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func splitColonLine(line string) (string, string, bool) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}
