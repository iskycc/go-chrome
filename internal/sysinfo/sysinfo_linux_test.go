//go:build linux

package sysinfo

import (
	"errors"
	"runtime"
	"testing"
)

func TestLinuxMemInfoErrors(t *testing.T) {
	old := readFileFunc
	defer func() { readFileFunc = old }()

	readFileFunc = func(name string) ([]byte, error) { return nil, errors.New("mock") }
	if linuxMemInfo() != nil {
		t.Fatal("expected nil on read error")
	}

	readFileFunc = func(name string) ([]byte, error) {
		return []byte("MemTotal: 1024 kB\nMemAvailable: not-a-number kB\n"), nil
	}
	mem := linuxMemInfo()
	if mem["MemTotal"] != 1024 {
		t.Fatalf("expected MemTotal=1024, got %d", mem["MemTotal"])
	}
	if _, ok := mem["MemAvailable"]; ok {
		t.Fatal("expected MemAvailable skipped")
	}
}

func TestOsReleaseValueFallback(t *testing.T) {
	old := readFileFunc
	defer func() { readFileFunc = old }()

	readFileFunc = func(name string) ([]byte, error) { return []byte("NAME=\"TestOS\"\n"), nil }
	if got := osReleaseValue("PRETTY_NAME"); got != "" {
		t.Fatalf("expected empty for missing key, got %q", got)
	}
	if got := osReleaseValue("NAME"); got != "TestOS" {
		t.Fatalf("expected TestOS, got %q", got)
	}
}

func TestSplitColonLine(t *testing.T) {
	if k, v, ok := splitColonLine("key: value"); !ok || k != "key" || v != "value" {
		t.Fatalf("splitColonLine failed: %q %q %v", k, v, ok)
	}
	if _, _, ok := splitColonLine("no colon"); ok {
		t.Fatal("expected no colon to fail")
	}
}

func TestFillSystemMemoryEdgeCases(t *testing.T) {
	old := readFileFunc
	defer func() { readFileFunc = old }()

	readFileFunc = func(name string) ([]byte, error) { return []byte("MemTotal: 0 kB\n"), nil }
	snap := SystemSnapshot{}
	fillSystemMemory(&snap)
	if snap.MemoryTotalMB != 0 {
		t.Fatalf("expected zero memory when totalKB <= 0, got %f", snap.MemoryTotalMB)
	}
}

func TestSystemCPUTimesNowErrors(t *testing.T) {
	old := readFileFunc
	defer func() { readFileFunc = old }()

	readFileFunc = func(name string) ([]byte, error) { return nil, errors.New("mock") }
	if _, ok := systemCPUTimesNow(); ok {
		t.Fatal("expected false on read error")
	}

	readFileFunc = func(name string) ([]byte, error) { return []byte("cpu 1 2 3\n"), nil }
	if _, ok := systemCPUTimesNow(); ok {
		t.Fatal("expected false with too few fields")
	}

	readFileFunc = func(name string) ([]byte, error) { return []byte("cpu 1 two 3 4\n"), nil }
	if _, ok := systemCPUTimesNow(); ok {
		t.Fatal("expected false with invalid number")
	}
}

func TestSystemStaticInfoErrors(t *testing.T) {
	old := readFileFunc
	readFileFunc = func(name string) ([]byte, error) { return nil, errors.New("mock read error") }
	defer func() { readFileFunc = old }()

	snap := systemStaticInfo()
	if snap.OSName != runtime.GOOS {
		t.Fatalf("expected fallback OSName, got %q", snap.OSName)
	}
	if snap.Kernel != "" {
		t.Fatalf("expected empty kernel on error, got %q", snap.Kernel)
	}
	// Hostname uses os.Hostname() directly and cannot be mocked here.
}
