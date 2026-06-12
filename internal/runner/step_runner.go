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

// StepRunner executes one step at a time.
type StepRunner struct {
	cfg           *config.RunnerConfig
	browser       browserController
	cdp           cdpSession
	history       HistorySaver
	envProvider   template.EnvProvider
	environmentID string

	flow         *flow.Flow
	currentIndex int
	eng          *template.Engine
	actExec      stepExecutor
	snapDir      string
	result       *RunResult
	started      bool

	connectCDP        func(int) (cdpSession, error)
	newActionExecutor func(context.Context, string, bool) stepExecutor
	retrySleep        func(time.Duration)
}

// NewStepRunner creates a new step runner.
func NewStepRunner(cfg *config.RunnerConfig, bm *browser.Manager, history HistorySaver) *StepRunner {
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
	return &StepRunner{
		cfg:        cfg,
		browser:    controller,
		history:    history,
		retrySleep: time.Sleep,
		connectCDP: func(port int) (cdpSession, error) {
			return browser.Connect(port)
		},
		newActionExecutor: func(ctx context.Context, snapDir string, maskInputs bool) stepExecutor {
			return NewActionExecutor(ctx, snapDir, maskInputs)
		},
	}
}

// Init initializes Chrome and CDP for the given flow.
func (sr *StepRunner) Init(f *flow.Flow, envProvider template.EnvProvider, environmentID string) error {
	if missing := MissingEnvVars(f, 0, envProvider); len(missing) > 0 {
		return fmt.Errorf("运行前检查失败，缺少环境变量: %s", strings.Join(missing, ", "))
	}

	if sr.browser == nil {
		return fmt.Errorf("browser manager is not initialized")
	}

	if !sr.browser.IsInstalled() {
		if err := sr.browser.Install(nil); err != nil {
			return fmt.Errorf("install chrome: %w", err)
		}
	}

	if sr.cdp != nil {
		sr.cdp.Close()
		sr.cdp = nil
	}

	sr.flow = f
	sr.currentIndex = 0
	if envProvider != nil {
		sr.eng = template.NewEngineWithEnv(envProvider)
	} else {
		sr.eng = template.NewEngine()
	}
	sr.envProvider = envProvider
	sr.environmentID = environmentID
	sr.result = &RunResult{
		ID:            uuid.New().String(),
		FlowID:        f.ID,
		FlowName:      f.Name,
		EnvironmentID: environmentID,
		StartedAt:     time.Now(),
		Steps:         make([]StepResult, 0, len(f.Steps)),
		Status:        StatusRunning,
	}

	runID := sr.result.StartedAt.Format("20060102-150405") + "-" + uuid.New().String()[:8]
	port, err := sr.browser.StartReplay(runID)
	if err != nil {
		return fmt.Errorf("start chrome: %w", err)
	}
	cdp, err := sr.connectCDP(port)
	if err != nil {
		return fmt.Errorf("cdp connect: %w", err)
	}
	sr.cdp = cdp

	sr.snapDir = filepath.Join("data", "run-history", f.ID, runID)
	if err := os.MkdirAll(sr.snapDir, 0755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}
	sr.actExec = sr.newActionExecutor(sr.cdp.Context(), sr.snapDir, sr.cfg.MaskInputValueInLogs)
	sr.started = true
	return nil
}

// Next executes the next step and returns its result.
// Returns nil, true when all steps are finished.
func (sr *StepRunner) Next() (*StepResult, bool, error) {
	if !sr.started {
		return nil, false, fmt.Errorf("not initialized")
	}
	if sr.currentIndex >= len(sr.flow.Steps) {
		sr.finish()
		return nil, true, nil
	}

	step := sr.flow.Steps[sr.currentIndex]
	res := sr.executeStepWithRetry(step)

	sr.result.Steps = append(sr.result.Steps, *res)
	if res.Status == StatusSuccess {
		sr.result.SuccessCount++
	} else if res.Status == StatusFailed {
		sr.result.FailedCount++
	} else if res.Status == StatusSkipped {
		sr.result.SkippedCount++
	}

	sr.currentIndex++

	if res.Status == StatusFailed && step.OnError == flow.ErrStop {
		sr.result.Status = StatusFailed
		sr.finish()
		return res, true, nil
	}

	if sr.currentIndex >= len(sr.flow.Steps) {
		sr.finish()
		return res, true, nil
	}

	return res, false, nil
}

func (sr *StepRunner) executeStepWithRetry(step flow.Step) *StepResult {
	retries := 0
	if step.OnError == flow.ErrRetry {
		retries = 2
	}
	var res *StepResult
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			sr.retrySleep(500 * time.Millisecond)
		}
		timeout := time.Duration(step.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = time.Duration(sr.cfg.DefaultTimeoutMs) * time.Millisecond
		}
		ctx, cancel := context.WithTimeout(sr.cdp.Context(), timeout)
		res = sr.actExec.ExecuteStep(ctx, step, sr.eng)
		cancel()
		if res.Status == StatusSuccess || res.Status == StatusSkipped {
			break
		}
	}
	return res
}

func (sr *StepRunner) finish() {
	sr.result.TotalSteps = len(sr.flow.Steps)
	sr.result.FinishedAt = time.Now()
	if sr.result.Status != StatusFailed {
		if sr.result.FailedCount > 0 {
			sr.result.Status = StatusFailed
		} else {
			sr.result.Status = StatusSuccess
		}
	}
	if sr.history != nil {
		_ = sr.history.Save(sr.result)
	}
}

// CurrentIndex returns the index of the next step to execute.
func (sr *StepRunner) CurrentIndex() int {
	return sr.currentIndex
}

// IsFinished returns true if all steps are done.
func (sr *StepRunner) IsFinished() bool {
	if !sr.started {
		return false
	}
	return sr.currentIndex >= len(sr.flow.Steps) || sr.result.Status == StatusFailed
}

// Result returns the run result.
func (sr *StepRunner) Result() *RunResult {
	return sr.result
}

// Close disconnects from CDP.
func (sr *StepRunner) Close() {
	if sr.cdp != nil {
		sr.cdp.Close()
		sr.cdp = nil
	}
	sr.started = false
}
