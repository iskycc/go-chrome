//go:build linux

package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const linuxClockTicksPerSecond = 100

func currentProcessName() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Base(os.Args[0])
	}
	return filepath.Base(exe)
}

func currentProcessRSSMB() (float64, error) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, nil
		}
		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, err
		}
		return kb / 1024, nil
	}
	return 0, nil
}

func currentProcessCPUTime() (time.Duration, error) {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, err
	}
	stat := string(data)
	endName := strings.LastIndex(stat, ")")
	if endName < 0 || endName+2 >= len(stat) {
		return 0, nil
	}
	fields := strings.Fields(stat[endName+2:])
	if len(fields) <= 12 {
		return 0, nil
	}
	userTicks, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, err
	}
	systemTicks, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, err
	}
	totalTicks := userTicks + systemTicks
	return time.Duration(totalTicks) * time.Second / linuxClockTicksPerSecond, nil
}
