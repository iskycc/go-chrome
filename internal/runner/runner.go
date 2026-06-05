package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/logx"
	"go-chrome/internal/template"
)

// Runner executes flows via CDP.
type Runner struct {
	cfg     *config.RunnerConfig
	browser *browser.Manager
	cdp     *browser.CDPClient
	events  chan Event
	stopCh  chan struct{}
	running bool
}

// NewRunner creates a new runner.
func NewRunner(cfg *config.RunnerConfig, bm *browser.Manager) *Runner {
	return &Runner{
		cfg:     cfg,
		browser: bm,
		events:  make(chan Event, 100),
		stopCh:  make(chan struct{}),
	}
}

// Events returns the event channel.
func (r *Runner) Events() <-chan Event {
	return r.events
}

// IsRunning returns true if a flow is currently running.
func (r *Runner) IsRunning() bool {
	return r.running
}

// Stop signals the current run to stop.
func (r *Runner) Stop() {
	select {
	case r.stopCh <- struct{}{}:
	default:
	}
}

// RunFlow executes a full flow.
func (r *Runner) RunFlow(f *flow.Flow, startStep int) *RunResult {
	r.running = true
	defer func() { r.running = false }()

	result := &RunResult{
		FlowID:    f.ID,
		FlowName:  f.Name,
		StartedAt: time.Now(),
		Steps:     make([]StepResult, 0, len(f.Steps)),
		Status:    StatusRunning,
	}

	if !r.browser.IsInstalled() {
		if err := r.browser.Install(nil); err != nil {
			result.Status = StatusFailed
			result.FinishedAt = time.Now()
			r.emit(Event{Type: EventLog, LogMessage: "Install Chrome failed: " + err.Error()})
			r.emit(Event{Type: EventRunDone, RunResult: result})
			return result
		}
	}

	if r.cdp != nil {
		r.cdp.Close()
		r.cdp = nil
	}

	runID := result.StartedAt.Format("20060102-150405.000")
	port, err := r.browser.StartReplay(runID)
	if err != nil {
		result.Status = StatusFailed
		result.FinishedAt = time.Now()
		r.emit(Event{Type: EventRunDone, RunResult: result})
		return result
	}
	cdp, err := browser.Connect(port)
	if err != nil {
		result.Status = StatusFailed
		result.FinishedAt = time.Now()
		r.emit(Event{Type: EventRunDone, RunResult: result})
		return result
	}
	r.cdp = cdp

	eng := template.NewEngine()
	snapDir := filepath.Join("data", "run-history", f.ID, runID)
	os.MkdirAll(snapDir, 0755)
	actExec := NewActionExecutor(r.cdp.Context(), snapDir)

	for i := startStep; i < len(f.Steps); i++ {
		select {
		case <-r.stopCh:
			result.Status = StatusFailed
			result.FinishedAt = time.Now()
			r.emit(Event{Type: EventRunDone, RunResult: result})
			return result
		default:
		}

		step := f.Steps[i]
		r.emit(Event{Type: EventStepStart, StepIndex: i})
		logx.Infof("Running step %d: %s (%s)", i, step.Name, step.Type)

		timeout := time.Duration(step.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = time.Duration(r.cfg.DefaultTimeoutMs) * time.Millisecond
		}
		ctx, cancel := context.WithTimeout(r.cdp.Context(), timeout)
		res := actExec.ExecuteStep(ctx, step, eng)
		cancel()

		result.Steps = append(result.Steps, *res)
		if res.Status == StatusSuccess {
			result.SuccessCount++
		} else if res.Status == StatusFailed {
			result.FailedCount++
		} else if res.Status == StatusSkipped {
			result.SkippedCount++
		}

		logMsg := fmt.Sprintf("Step %d %s: %s", i, step.Name, res.Status)
		if res.Error != "" {
			logMsg += " - " + res.Error
		}
		r.emit(Event{Type: EventLog, LogMessage: logMsg})
		r.emit(Event{Type: EventStepDone, StepIndex: i, Result: res})

		if res.Status == StatusFailed {
			if step.OnError == flow.ErrStop {
				result.Status = StatusFailed
				break
			}
			// continue or retry not fully implemented in this version
		}
	}

	result.TotalSteps = len(f.Steps)
	if result.Status != StatusFailed {
		result.Status = StatusSuccess
	}
	result.FinishedAt = time.Now()
	r.emit(Event{Type: EventRunDone, RunResult: result})
	return result
}

func (r *Runner) emit(ev Event) {
	select {
	case r.events <- ev:
	default:
	}
}

// Close disconnects from CDP.
func (r *Runner) Close() {
	if r.cdp != nil {
		r.cdp.Close()
		r.cdp = nil
	}
}
