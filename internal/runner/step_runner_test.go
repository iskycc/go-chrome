package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/template"
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

func TestStepRunnerInitErrors(t *testing.T) {
	cfg := &config.RunnerConfig{DefaultTimeoutMs: 100}
	step := testStep("needs env", flow.ErrStop)
	step.Input.Text = "${env:USER}"
	f := testFlow(step)

	sr := NewStepRunner(cfg, nil, nil)
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "USER") {
		t.Fatalf("expected missing env error, got %v", err)
	}

	f = testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "browser manager") {
		t.Fatalf("expected missing browser error, got %v", err)
	}

	sr = NewStepRunner(cfg, nil, nil)
	sr.browser = &fakeBrowserController{installErr: errors.New("offline")}
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "install chrome") {
		t.Fatalf("expected install error, got %v", err)
	}

	sr = NewStepRunner(cfg, nil, nil)
	sr.browser = &fakeBrowserController{installed: true, startErr: errors.New("start failed")}
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "start chrome") {
		t.Fatalf("expected start error, got %v", err)
	}

	sr = NewStepRunner(cfg, nil, nil)
	sr.browser = &fakeBrowserController{installed: true}
	sr.connectCDP = func(int) (cdpSession, error) {
		return nil, errors.New("connect failed")
	}
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "cdp connect") {
		t.Fatalf("expected cdp connect error, got %v", err)
	}
}

func TestStepRunnerInitSuccessWithInjectedDependencies(t *testing.T) {
	cfg := &config.RunnerConfig{DefaultTimeoutMs: 100, MaskInputValueInLogs: true}
	exec := &fakeStepExecutor{results: []Status{StatusSuccess}}
	sr := NewStepRunner(cfg, nil, nil)
	sr.browser = &fakeBrowserController{installed: true}
	sr.connectCDP = func(int) (cdpSession, error) {
		return &fakeCDPSession{ctx: context.Background()}, nil
	}
	sr.newActionExecutor = func(context.Context, string, bool) stepExecutor {
		return exec
	}

	f := testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, "env-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if !sr.started || sr.result.EnvironmentID != "env-1" || sr.actExec != exec {
		t.Fatalf("unexpected initialized state: started=%v result=%+v", sr.started, sr.result)
	}
	sr.Close()
	if sr.started {
		t.Fatal("close should mark runner not started")
	}
}

func initializedStepRunner(exec stepExecutor, history HistorySaver, steps ...flow.Step) *StepRunner {
	f := testFlow(steps...)
	return &StepRunner{
		cfg:          &config.RunnerConfig{DefaultTimeoutMs: 100},
		browser:      &fakeBrowserController{installed: true},
		cdp:          &fakeCDPSession{ctx: context.Background()},
		history:      history,
		flow:         f,
		currentIndex: 0,
		eng:          template.NewEngine(),
		actExec:      exec,
		result: &RunResult{
			ID:        "run-1",
			FlowID:    f.ID,
			FlowName:  f.Name,
			StartedAt: time.Now(),
			Steps:     make([]StepResult, 0, len(f.Steps)),
			Status:    StatusRunning,
		},
		started:    true,
		retrySleep: func(time.Duration) {},
	}
}

func TestStepRunnerNextCountsAndFinishes(t *testing.T) {
	history := &fakeHistorySaver{}
	exec := &fakeStepExecutor{results: []Status{StatusSuccess, StatusFailed}, errors: []string{"", "boom"}}
	sr := initializedStepRunner(exec, history,
		testStep("ok", flow.ErrStop),
		testStep("fail", flow.ErrContinue),
	)

	res, finished, err := sr.Next()
	if err != nil || finished || res.Status != StatusSuccess {
		t.Fatalf("first next: res=%+v finished=%v err=%v", res, finished, err)
	}
	res, finished, err = sr.Next()
	if err != nil || !finished || res.Status != StatusFailed {
		t.Fatalf("second next: res=%+v finished=%v err=%v", res, finished, err)
	}
	if sr.result.Status != StatusFailed || sr.result.SuccessCount != 1 || sr.result.FailedCount != 1 {
		t.Fatalf("unexpected result: %+v", sr.result)
	}
	if len(history.saved) != 1 || history.saved[0] != sr.result {
		t.Fatal("expected finished result to be saved")
	}
}

func TestStepRunnerStopsOnFailedErrStop(t *testing.T) {
	exec := &fakeStepExecutor{results: []Status{StatusFailed}, errors: []string{"boom"}}
	sr := initializedStepRunner(exec, nil, testStep("fail", flow.ErrStop), testStep("never", flow.ErrStop))

	res, finished, err := sr.Next()
	if err != nil || !finished || res.Status != StatusFailed {
		t.Fatalf("next: res=%+v finished=%v err=%v", res, finished, err)
	}
	if sr.CurrentIndex() != 1 || exec.calls != 1 {
		t.Fatalf("expected stop after first step, index=%d calls=%d", sr.CurrentIndex(), exec.calls)
	}
	if !sr.IsFinished() {
		t.Fatal("expected runner to be finished after stop policy failure")
	}
}

func TestStepRunnerRetriesFailedStepUntilSuccess(t *testing.T) {
	exec := &fakeStepExecutor{results: []Status{StatusFailed, StatusSuccess}, errors: []string{"boom", ""}}
	sr := initializedStepRunner(exec, nil, testStep("flaky", flow.ErrRetry))

	res, finished, err := sr.Next()
	if err != nil || !finished || res.Status != StatusSuccess {
		t.Fatalf("next: res=%+v finished=%v err=%v", res, finished, err)
	}
	if exec.calls != 2 || sr.result.SuccessCount != 1 {
		t.Fatalf("unexpected retry result: calls=%d result=%+v", exec.calls, sr.result)
	}
}

func TestStepRunnerInitCdpConnectFailure(t *testing.T) {
	cfg := &config.RunnerConfig{DefaultTimeoutMs: 100, MaskInputValueInLogs: true}
	sr := NewStepRunner(cfg, nil, nil)
	sr.browser = &fakeBrowserController{installed: true}
	sr.connectCDP = func(int) (cdpSession, error) {
		return nil, errors.New("connection refused")
	}
	f := testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "cdp connect") {
		t.Fatalf("expected cdp connect error, got %v", err)
	}
	if sr.started {
		t.Fatal("started should be false after cdp failure")
	}
}

func TestStepRunnerResultAfterFinish(t *testing.T) {
	exec := &fakeStepExecutor{results: []Status{StatusFailed}, errors: []string{"some error"}}
	sr := initializedStepRunner(exec, &fakeHistorySaver{}, testStep("fail", flow.ErrStop))
	_, finished, _ := sr.Next()
	if !finished {
		t.Fatal("expected finished")
	}
	res := sr.Result()
	if res == nil {
		t.Fatal("expected non-nil result after finish")
	}
	if res.Status != StatusFailed || res.FailedCount != 1 {
		t.Fatalf("unexpected result status: %+v", res)
	}
}
