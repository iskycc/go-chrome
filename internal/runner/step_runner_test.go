package runner

import (
	"testing"

	"go-chrome/internal/config"
)

func TestStepRunnerNotStarted(t *testing.T) {
	cfg := &config.RunnerConfig{DefaultTimeoutMs: 1000}
	sr := NewStepRunner(cfg, nil, nil)
	if sr.IsFinished() {
		t.Fatal("should not be finished before init")
	}
	_, _, err := sr.Next()
	if err == nil {
		t.Fatal("expected error when not initialized")
	}
}

func TestStepRunnerResultEmpty(t *testing.T) {
	cfg := &config.RunnerConfig{DefaultTimeoutMs: 1000}
	sr := NewStepRunner(cfg, nil, nil)
	if sr.Result() != nil {
		t.Fatal("result should be nil before init")
	}
}

func TestStepRunnerCloseNoPanic(t *testing.T) {
	cfg := &config.RunnerConfig{DefaultTimeoutMs: 1000}
	sr := NewStepRunner(cfg, nil, nil)
	sr.Close() // should not panic
}
