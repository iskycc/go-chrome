package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

// HistorySaver persists run results.
type HistorySaver interface {
	Save(result *RunResult) error
}

// Runner executes flows via CDP.
type Runner struct {
	cfg          *config.RunnerConfig
	browser      *browser.Manager
	cdp          *browser.CDPClient
	history      HistorySaver
	events       chan Event
	stopCh       chan struct{}
	running      bool
}

// NewRunner creates a new runner.
func NewRunner(cfg *config.RunnerConfig, bm *browser.Manager, history HistorySaver) *Runner {
	return &Runner{
		cfg:     cfg,
		browser: bm,
		history: history,
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
		ID:        uuid.New().String(),
		FlowID:    f.ID,
		FlowName:  f.Name,
		StartedAt: time.Now(),
		Steps:     make([]StepResult, 0, len(f.Steps)),
		Status:    StatusRunning,
	}
	defer func() {
		result.TotalSteps = len(f.Steps)
		result.FinishedAt = time.Now()
		if r.history != nil {
			_ = r.history.Save(result)
		}
		r.emit(Event{Type: EventRunDone, RunResult: result})
	}()

	if !r.browser.IsInstalled() {
		if err := r.browser.Install(nil); err != nil {
			result.Status = StatusFailed
			r.emit(Event{Type: EventLog, LogMessage: "Install Chrome failed: " + err.Error()})
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
		r.emit(Event{Type: EventLog, LogMessage: "Start Chrome failed: " + err.Error()})
		return result
	}
	cdp, err := browser.Connect(port)
	if err != nil {
		result.Status = StatusFailed
		r.emit(Event{Type: EventLog, LogMessage: "CDP connect failed: " + err.Error()})
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
			r.emit(Event{Type: EventLog, LogMessage: "用户停止执行"})
			return result
		default:
		}

		step := f.Steps[i]
		r.emit(Event{Type: EventStepStart, StepIndex: i, StepName: step.Name})
		res := r.executeStepWithRetry(actExec, step, eng)

		result.Steps = append(result.Steps, *res)
		if res.Status == StatusSuccess {
			result.SuccessCount++
		} else if res.Status == StatusFailed {
			result.FailedCount++
		} else if res.Status == StatusSkipped {
			result.SkippedCount++
		}

		logMsg := fmt.Sprintf("步骤 %d %s: %s", i+1, step.Name, res.Status)
		if res.Error != "" {
			logMsg += " - " + res.Error
		}
		if res.GeneratedInput != "" {
			if step.Input.MaskInLogs || r.cfg.MaskInputValueInLogs {
				logMsg += " [输入值已脱敏]"
			} else {
				logMsg += " 输入: " + maskIfNeeded(res.GeneratedInput, 4)
			}
		}
		r.emit(Event{Type: EventLog, LogMessage: logMsg})
		r.emit(Event{Type: EventStepDone, StepIndex: i, Result: res})

		if res.Status == StatusFailed {
			if step.OnError == flow.ErrStop {
				result.Status = StatusFailed
				return result
			}
		}
	}

	if result.FailedCount > 0 {
		result.Status = StatusFailed
	} else {
		result.Status = StatusSuccess
	}
	return result
}

func (r *Runner) executeStepWithRetry(actExec *ActionExecutor, step flow.Step, eng *template.Engine) *StepResult {
	retries := 0
	if step.OnError == flow.ErrRetry {
		retries = 2 // default retry count; could be configurable
	}

	var res *StepResult
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			r.emit(Event{Type: EventLog, LogMessage: fmt.Sprintf("重试步骤 %s (第 %d 次)", step.Name, attempt)})
		}
		timeout := time.Duration(step.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = time.Duration(r.cfg.DefaultTimeoutMs) * time.Millisecond
		}
		ctx, cancel := context.WithTimeout(r.cdp.Context(), timeout)
		res = actExec.ExecuteStep(ctx, step, eng)
		cancel()
		if res.Status == StatusSuccess || res.Status == StatusSkipped {
			break
		}
		if attempt < retries {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return res
}

func maskIfNeeded(s string, visible int) string {
	if len(s) <= visible*2 {
		return strings.Repeat("*", len(s))
	}
	return s[:visible] + strings.Repeat("*", len(s)-visible*2) + s[len(s)-visible:]
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
