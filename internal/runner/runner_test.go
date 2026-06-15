package runner

import (
	"testing"

	"go-chrome/internal/config"
)

func TestNewRunnerBasics(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	if r.IsRunning() {
		t.Fatal("new runner should not be running")
	}
	if r.Events() == nil {
		t.Fatal("expected events channel")
	}
	r.Close()
}

func TestRunnerCloseClosesCDP(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	cdp := &fakeCDPSession{}
	r.cdp = cdp
	r.Close()
	if !cdp.closed {
		t.Fatal("expected cdp session closed")
	}
	if r.cdp != nil {
		t.Fatal("expected cdp cleared")
	}
}

func TestRunnerStopSignalIsBufferedAndDrained(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	r.Stop()
	r.Stop()

	select {
	case <-r.stopCh:
	default:
		t.Fatal("expected one buffered stop signal")
	}

	r.Stop()
	r.drainStopSignal()
	select {
	case <-r.stopCh:
		t.Fatal("expected drained stop signal")
	default:
	}
}

func TestRunnerEmitDropsWhenChannelFull(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	for i := 0; i < cap(r.events)+10; i++ {
		r.emit(Event{Type: EventLog, LogMessage: "test"})
	}
	if len(r.events) != cap(r.events) {
		t.Fatalf("expected full channel, got %d/%d", len(r.events), cap(r.events))
	}
}

func TestMaskIfNeeded(t *testing.T) {
	if got := maskIfNeeded("abcd1234wxyz", 4); got != "abcd****wxyz" {
		t.Fatalf("unexpected mask: %s", got)
	}
	if got := maskIfNeeded("short", 4); got != "*****" {
		t.Fatalf("unexpected short mask: %s", got)
	}
}

func TestNewRunnerNilConfig(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	if r.cfg == nil {
		t.Fatal("expected default config when nil passed")
	}
	if r.cfg.DefaultTimeoutMs != 10000 {
		t.Fatalf("expected default timeout 10000, got %d", r.cfg.DefaultTimeoutMs)
	}
}

func TestNewStepRunnerNilConfig(t *testing.T) {
	sr := NewStepRunner(nil, nil, nil)
	if sr.cfg == nil {
		t.Fatal("expected default config when nil passed")
	}
	if sr.cfg.DefaultTimeoutMs != 10000 {
		t.Fatalf("expected default timeout 10000, got %d", sr.cfg.DefaultTimeoutMs)
	}
}
