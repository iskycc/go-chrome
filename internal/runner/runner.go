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

type browserController interface {
	IsInstalled() bool
	Install(browser.DownloadProgress) error
	StartReplay(string) (int, error)
}

type cdpSession interface {
	Context() context.Context
	Close()
}

type stepExecutor interface {
	ExecuteStep(context.Context, flow.Step, *template.Engine) *StepResult
}

// Runner executes flows via CDP.
type Runner struct {
	cfg               *config.RunnerConfig
	browser           browserController
	cdp               cdpSession
	history           HistorySaver
	events            chan Event
	stopCh            chan struct{}
	connectCDP        func(int) (cdpSession, error)
	newActionExecutor func(context.Context, string, bool) stepExecutor
	retrySleep        func(time.Duration)
	running           bool
}

// NewRunner creates a new runner.
func NewRunner(cfg *config.RunnerConfig, bm *browser.Manager, history HistorySaver) *Runner {
	if cfg == nil {
		cfg = &config.RunnerConfig{
			DefaultTimeoutMs:     10000,
			DefaultWaitAfterMs:   500,
			MaskInputValueInLogs: true,
		}
	}
	var controller browserController
	if bm != nil {
		controller = bm
	}
	return &Runner{
		cfg:        cfg,
		browser:    controller,
		history:    history,
		events:     make(chan Event, 100),
		stopCh:     make(chan struct{}, 1),
		retrySleep: time.Sleep,
		connectCDP: func(port int) (cdpSession, error) {
			return browser.Connect(port)
		},
		newActionExecutor: func(ctx context.Context, snapDir string, maskInputs bool) stepExecutor {
			return NewActionExecutor(ctx, snapDir, maskInputs)
		},
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

// RunOptions configures a flow run.
type RunOptions struct {
	StartStep     int
	EnvironmentID string
	EnvProvider   template.EnvProvider
}

// RunFlow executes a full flow.
func (r *Runner) RunFlow(f *flow.Flow, opts RunOptions) *RunResult {
	r.drainStopSignal()
	r.running = true
	defer func() { r.running = false }()

	result := &RunResult{
		ID:            uuid.New().String(),
		FlowID:        f.ID,
		FlowName:      f.Name,
		EnvironmentID: opts.EnvironmentID,
		StartedAt:     time.Now(),
		Steps:         make([]StepResult, 0, len(f.Steps)),
		Status:        StatusRunning,
	}
	defer func() {
		result.TotalSteps = len(f.Steps)
		result.FinishedAt = time.Now()
		if r.history != nil {
			_ = r.history.Save(result)
		}
		r.emit(Event{Type: EventRunDone, RunResult: result})
	}()

	if missing := MissingEnvVars(f, opts.StartStep, opts.EnvProvider); len(missing) > 0 {
		result.Status = StatusFailed
		r.emit(Event{Type: EventLog, LogMessage: "运行前检查失败，缺少环境变量: " + strings.Join(missing, ", ")})
		return result
	}

	if r.browser == nil {
		result.Status = StatusFailed
		r.emit(Event{Type: EventLog, LogMessage: "Browser manager is not initialized"})
		return result
	}

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

	runID := result.StartedAt.Format("20060102-150405") + "-" + uuid.New().String()[:8]
	port, err := r.browser.StartReplay(runID)
	if err != nil {
		result.Status = StatusFailed
		r.emit(Event{Type: EventLog, LogMessage: "Start Chrome failed: " + err.Error()})
		return result
	}
	cdp, err := r.connectCDP(port)
	if err != nil {
		result.Status = StatusFailed
		r.emit(Event{Type: EventLog, LogMessage: "CDP connect failed: " + err.Error()})
		return result
	}
	r.cdp = cdp

	var eng *template.Engine
	if opts.EnvProvider != nil {
		eng = template.NewEngineWithEnv(opts.EnvProvider)
	} else {
		eng = template.NewEngine()
	}
	snapDir := filepath.Join("data", "run-history", f.ID, runID)
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		result.Status = StatusFailed
		r.emit(Event{Type: EventLog, LogMessage: "Create snapshot dir failed: " + err.Error()})
		return result
	}
	actExec := r.newActionExecutor(r.cdp.Context(), snapDir, r.cfg.MaskInputValueInLogs)

	for i := opts.StartStep; i < len(f.Steps); i++ {
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
		if res.GeneratedInputMasked != "" {
			if step.Input.MaskInLogs || r.cfg.MaskInputValueInLogs {
				logMsg += " [输入值已脱敏]"
			} else {
				logMsg += " 输入: " + maskIfNeeded(res.GeneratedInputMasked, 4)
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

func (r *Runner) drainStopSignal() {
	for {
		select {
		case <-r.stopCh:
		default:
			return
		}
	}
}

func (r *Runner) executeStepWithRetry(actExec stepExecutor, step flow.Step, eng *template.Engine) *StepResult {
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
			r.retrySleep(500 * time.Millisecond)
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
