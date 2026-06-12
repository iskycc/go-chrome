package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

type fakeBrowserController struct {
	installed  bool
	installErr error
	startErr   error
	port       int

	installCalls int
	startCalls   int
}

func (b *fakeBrowserController) IsInstalled() bool {
	return b.installed
}

func (b *fakeBrowserController) Install(browser.DownloadProgress) error {
	b.installCalls++
	if b.installErr == nil {
		b.installed = true
	}
	return b.installErr
}

func (b *fakeBrowserController) StartReplay(string) (int, error) {
	b.startCalls++
	if b.startErr != nil {
		return 0, b.startErr
	}
	if b.port == 0 {
		return 9222, nil
	}
	return b.port, nil
}

type fakeCDPSession struct {
	ctx    context.Context
	closed bool
}

func (c *fakeCDPSession) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *fakeCDPSession) Close() {
	c.closed = true
}

type fakeStepExecutor struct {
	results []Status
	errors  []string
	calls   int
	delay   time.Duration
}

func (e *fakeStepExecutor) ExecuteStep(_ context.Context, step flow.Step, _ *template.Engine) *StepResult {
	if e.delay > 0 {
		time.Sleep(e.delay)
	}
	idx := e.calls
	e.calls++
	status := StatusSuccess
	if idx < len(e.results) {
		status = e.results[idx]
	}
	errText := ""
	if idx < len(e.errors) {
		errText = e.errors[idx]
	}
	return &StepResult{
		StepID:     step.ID,
		StepName:   step.Name,
		Type:       string(step.Type),
		Status:     status,
		Error:      errText,
		DurationMs: 1,
	}
}

type fakeHistorySaver struct {
	saved []*RunResult
}

func (s *fakeHistorySaver) Save(result *RunResult) error {
	s.saved = append(s.saved, result)
	return nil
}

func newInjectedRunner(b *fakeBrowserController, exec stepExecutor, history HistorySaver) *Runner {
	r := NewRunner(&config.RunnerConfig{DefaultTimeoutMs: 100, MaskInputValueInLogs: true}, nil, history)
	r.browser = b
	r.connectCDP = func(int) (cdpSession, error) {
		return &fakeCDPSession{}, nil
	}
	r.newActionExecutor = func(context.Context, string, bool) stepExecutor {
		return exec
	}
	r.retrySleep = func(time.Duration) {}
	return r
}

func testFlow(steps ...flow.Step) *flow.Flow {
	f := flow.NewFlow("test flow")
	f.Steps = steps
	return f
}

func testStep(name string, policy flow.ErrorPolicy) flow.Step {
	step := flow.NewStep(name, flow.StepWaitFixed)
	step.WaitAfterMs = 0
	step.TimeoutMs = 10
	step.OnError = policy
	return step
}

func drainRunnerEvents(r *Runner) []Event {
	var events []Event
	for {
		select {
		case ev := <-r.events:
			events = append(events, ev)
		default:
			return events
		}
	}
}

func TestRunFlowFailsBeforeBrowserWhenEnvMissing(t *testing.T) {
	history := &fakeHistorySaver{}
	r := newInjectedRunner(nil, nil, history)
	step := testStep("needs env", flow.ErrStop)
	step.Target.Value = "${env:BASE_URL}"

	res := r.RunFlow(testFlow(step), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	if len(history.saved) != 1 || history.saved[0] != res {
		t.Fatal("expected failed result to be saved")
	}
	events := drainRunnerEvents(r)
	if len(events) == 0 || events[len(events)-1].Type != EventRunDone {
		t.Fatalf("expected run done event, got %+v", events)
	}
	if !strings.Contains(events[0].LogMessage, "BASE_URL") {
		t.Fatalf("expected missing env log, got %+v", events[0])
	}
}

func TestRunFlowInstallFailure(t *testing.T) {
	b := &fakeBrowserController{installErr: errors.New("offline")}
	r := newInjectedRunner(b, nil, nil)

	res := r.RunFlow(testFlow(testStep("open", flow.ErrStop)), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	if b.installCalls != 1 || b.startCalls != 0 {
		t.Fatalf("unexpected browser calls: install=%d start=%d", b.installCalls, b.startCalls)
	}
}

func TestRunFlowFailsWhenBrowserManagerMissing(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{DefaultTimeoutMs: 100}, nil, nil)

	res := r.RunFlow(testFlow(testStep("open", flow.ErrStop)), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	events := drainRunnerEvents(r)
	if len(events) < 2 || !strings.Contains(events[0].LogMessage, "Browser manager") {
		t.Fatalf("expected browser manager log, got %+v", events)
	}
}

func TestRunFlowInstallsMissingBrowserThenRuns(t *testing.T) {
	b := &fakeBrowserController{installed: false}
	exec := &fakeStepExecutor{results: []Status{StatusSuccess}}
	r := newInjectedRunner(b, exec, nil)

	res := r.RunFlow(testFlow(testStep("open", flow.ErrStop)), RunOptions{})
	if res.Status != StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if b.installCalls != 1 || b.startCalls != 1 {
		t.Fatalf("unexpected browser calls: install=%d start=%d", b.installCalls, b.startCalls)
	}
}

func TestRunFlowStartReplayFailure(t *testing.T) {
	b := &fakeBrowserController{installed: true, startErr: errors.New("cannot start")}
	r := newInjectedRunner(b, nil, nil)

	res := r.RunFlow(testFlow(testStep("open", flow.ErrStop)), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	if b.startCalls != 1 {
		t.Fatalf("expected start replay once, got %d", b.startCalls)
	}
}

func TestRunFlowClosesExistingCDPBeforeRun(t *testing.T) {
	old := &fakeCDPSession{}
	r := newInjectedRunner(&fakeBrowserController{installed: true}, &fakeStepExecutor{results: []Status{StatusSuccess}}, nil)
	r.cdp = old

	res := r.RunFlow(testFlow(testStep("open", flow.ErrStop)), RunOptions{})
	if res.Status != StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if !old.closed {
		t.Fatal("expected old cdp session to be closed")
	}
}

func TestRunFlowCDPConnectFailure(t *testing.T) {
	b := &fakeBrowserController{installed: true}
	r := newInjectedRunner(b, nil, nil)
	r.connectCDP = func(int) (cdpSession, error) {
		return nil, errors.New("refused")
	}

	res := r.RunFlow(testFlow(testStep("open", flow.ErrStop)), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
}

func TestRunFlowCountsStepStatusesAndSavesHistory(t *testing.T) {
	history := &fakeHistorySaver{}
	exec := &fakeStepExecutor{
		results: []Status{StatusSuccess, StatusSkipped, StatusFailed},
		errors:  []string{"", "", "boom"},
	}
	r := newInjectedRunner(&fakeBrowserController{installed: true}, exec, history)

	res := r.RunFlow(testFlow(
		testStep("ok", flow.ErrStop),
		testStep("skip", flow.ErrStop),
		testStep("fail", flow.ErrContinue),
	), RunOptions{EnvironmentID: "env-1"})

	if res.Status != StatusFailed {
		t.Fatalf("expected final failed, got %s", res.Status)
	}
	if res.SuccessCount != 1 || res.SkippedCount != 1 || res.FailedCount != 1 {
		t.Fatalf("unexpected counts: success=%d skipped=%d failed=%d", res.SuccessCount, res.SkippedCount, res.FailedCount)
	}
	if res.EnvironmentID != "env-1" || len(history.saved) != 1 {
		t.Fatalf("expected saved env result, saved=%d env=%q", len(history.saved), res.EnvironmentID)
	}
	if exec.calls != 3 {
		t.Fatalf("expected 3 executor calls, got %d", exec.calls)
	}
}

func TestRunFlowStopsOnFailedErrStop(t *testing.T) {
	exec := &fakeStepExecutor{results: []Status{StatusFailed, StatusSuccess}, errors: []string{"boom", ""}}
	r := newInjectedRunner(&fakeBrowserController{installed: true}, exec, nil)

	res := r.RunFlow(testFlow(
		testStep("fail", flow.ErrStop),
		testStep("never", flow.ErrStop),
	), RunOptions{})

	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	if exec.calls != 1 || len(res.Steps) != 1 {
		t.Fatalf("expected stop after one call, calls=%d steps=%d", exec.calls, len(res.Steps))
	}
}

func TestRunFlowRetriesFailedStepUntilSuccess(t *testing.T) {
	exec := &fakeStepExecutor{
		results: []Status{StatusFailed, StatusFailed, StatusSuccess},
		errors:  []string{"first", "second", ""},
	}
	r := newInjectedRunner(&fakeBrowserController{installed: true}, exec, nil)

	res := r.RunFlow(testFlow(testStep("flaky", flow.ErrRetry)), RunOptions{})
	if res.Status != StatusSuccess {
		t.Fatalf("expected success after retry, got %s", res.Status)
	}
	if exec.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", exec.calls)
	}
}
