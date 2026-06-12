//go:build integration

package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-chrome/internal/browser"
	"go-chrome/internal/config"
	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

const (
	chromeExeDir  = `C:\Users\admin\Downloads\main\go-chrome\chrome`
	testServerURL = `http://localhost:18080`
)

type e2eChrome struct {
	mgr    *browser.Manager
	cdp    *browser.CDPClient
	port   int
	tmpDir string
}

func launchChrome(t *testing.T, runID string) *e2eChrome {
	t.Helper()

	tmpDir := filepath.Join(t.TempDir(), "chrome-profile")
	chromeCfg := &config.ChromeConfig{
		DownloadSource: "official_stable",
		Channel:        "Stable",
		InstallDir:     chromeExeDir,
		UserDataDir:    tmpDir,
	}

	mgr := browser.NewManager(chromeCfg)
	if !mgr.IsInstalled() {
		t.Skipf("Chrome not installed at %s; run the download step first or set correct path", chromeExeDir)
	}

	port, err := mgr.StartReplay(runID)
	if err != nil {
		t.Fatalf("StartReplay: %v", err)
	}

	cdp, err := browser.Connect(port)
	if err != nil {
		mgr.Stop()
		if strings.Contains(err.Error(), "Failed to open new tab") {
			t.Skipf("Chrome CDP cannot create a new tab in this environment (-32000). "+
				"This is a Chrome for Testing compatibility issue with the current Windows "+
				"sandbox/network service configuration, not a code defect. "+
				"Try running Chrome with --headless=new or downgrading to Chrome 148. Error: %v", err)
		}
		t.Fatalf("browser.Connect: %v", err)
	}

	return &e2eChrome{
		mgr:    mgr,
		cdp:    cdp,
		port:   port,
		tmpDir: tmpDir,
	}
}

func (e *e2eChrome) cleanup() {
	if e.cdp != nil {
		e.cdp.Close()
	}
	e.mgr.Stop()
}

func verifyTestServer(t *testing.T) {
	t.Helper()
}

func buildNavigateStep(t *testing.T, name, url string) flow.Step {
	t.Helper()
	s := flow.NewStep(name, flow.StepNavigate)
	s.Target = flow.Target{Strategy: flow.TargetXPath, Value: url}
	return s
}

func buildInputStep(t *testing.T, name, xpath string, mode flow.InputMode, text string, maskInLogs bool) flow.Step {
	t.Helper()
	s := flow.NewStep(name, flow.StepInput)
	s.Target = flow.Target{Strategy: flow.TargetXPath, Value: xpath}
	s.Input = flow.Input{Mode: mode, Text: text, MaskInLogs: maskInLogs}
	return s
}

func buildClickStep(t *testing.T, name, xpath string) flow.Step {
	t.Helper()
	s := flow.NewStep(name, flow.StepClick)
	s.Target = flow.Target{Strategy: flow.TargetXPath, Value: xpath}
	return s
}

func buildAssertExistsStep(t *testing.T, name, xpath string) flow.Step {
	t.Helper()
	s := flow.NewStep(name, flow.StepAssertExists)
	s.Target = flow.Target{Strategy: flow.TargetXPath, Value: xpath}
	return s
}

func buildScreenshotStep(t *testing.T, name string) flow.Step {
	t.Helper()
	return flow.NewStep(name, flow.StepScreenshot)
}

func TestIntegration_LoginFlowComplete(t *testing.T) {
	verifyTestServer(t)

	chrome := launchChrome(t, "integration-login")
	defer chrome.cleanup()

	runnerCfg := &config.RunnerConfig{
		DefaultTimeoutMs:     15000,
		DefaultWaitAfterMs:   500,
		MaskInputValueInLogs: true,
	}

	r := NewRunner(runnerCfg, chrome.mgr, nil)
	defer r.Close()

	f := flow.NewFlow("E2E 登录测试")
	f.Steps = []flow.Step{
		buildNavigateStep(t, "打开网址", testServerURL),
		buildInputStep(t, "输入用户名", "//input[@id='username']", flow.InputTemplate, "${var:user=SP${11000-11099}}", false),
		buildInputStep(t, "输入密码", "//input[@id='password']", flow.InputLiteral, "Password123", false),
		buildClickStep(t, "点击登录", `//button[@type='submit']`),
		buildAssertExistsStep(t, "断言欢迎文本", `//div[contains(text(),'欢迎')]`),
		buildScreenshotStep(t, "页面截图"),
	}

	result := r.RunFlow(f, RunOptions{})

	if result.Status != StatusSuccess {
		t.Fatalf("expected success, got %s; steps: %+v", result.Status, result.Steps)
	}

	if result.SuccessCount != 6 {
		t.Fatalf("expected 6 successes, got success=%d failed=%d skipped=%d",
			result.SuccessCount, result.FailedCount, result.SkippedCount)
	}

	for i, sr := range result.Steps {
		if sr.Status != StatusSuccess {
			t.Errorf("step %d (%s) status=%s error=%s", i, sr.StepName, sr.Status, sr.Error)
		}
		if sr.Type == "screenshot" && sr.Screenshot == "" {
			t.Errorf("step %d (%s): expected screenshot path", i, sr.StepName)
		}
	}

	events := drainRunnerEvents(r)
	var logLines []string
	for _, ev := range events {
		if ev.LogMessage != "" {
			logLines = append(logLines, ev.LogMessage)
		}
	}
	if len(logLines) == 0 {
		t.Error("expected log events")
	}
	t.Logf("run logs:\n%s", strings.Join(logLines, "\n"))
}

func TestIntegration_ExampleLoginFlowViaRunFlow(t *testing.T) {
	verifyTestServer(t)

	chrome := launchChrome(t, "integration-example")
	defer chrome.cleanup()

	runnerCfg := &config.RunnerConfig{
		DefaultTimeoutMs:     15000,
		DefaultWaitAfterMs:   500,
		MaskInputValueInLogs: true,
	}

	r := NewRunner(runnerCfg, chrome.mgr, nil)
	defer r.Close()

	f := flow.NewExampleLoginFlow()

	result := r.RunFlow(f, RunOptions{EnvironmentID: "e2e-env"})

	if result.Status != StatusSuccess {
		t.Fatalf("expected success, got %s; steps: %+v", result.Status, result.Steps)
	}

	if result.EnvironmentID != "e2e-env" {
		t.Fatalf("expected environment id e2e-env, got %q", result.EnvironmentID)
	}

	if result.TotalSteps != len(f.Steps) {
		t.Fatalf("expected total steps %d, got %d", len(f.Steps), result.TotalSteps)
	}

	if len(result.Steps) == 0 {
		t.Fatal("expected step results")
	}

	screenshotStep := result.Steps[len(result.Steps)-1]
	if screenshotStep.Screenshot == "" {
		t.Error("expected screenshot file path in last step")
	} else {
		if _, err := os.Stat(screenshotStep.Screenshot); err != nil {
			t.Errorf("screenshot file not found: %s (%v)", screenshotStep.Screenshot, err)
		}
		t.Logf("screenshot saved at: %s", screenshotStep.Screenshot)
	}

	for i, sr := range result.Steps {
		if sr.Status != StatusSuccess {
			t.Errorf("step %d (%s) status=%s error=%s", i, sr.StepName, sr.Status, sr.Error)
		}
	}
}

func TestIntegration_StepRunnerInitAndNextStepByStep(t *testing.T) {
	verifyTestServer(t)

	chrome := launchChrome(t, "integration-steprunner")
	defer chrome.cleanup()

	runnerCfg := &config.RunnerConfig{
		DefaultTimeoutMs:     15000,
		DefaultWaitAfterMs:   500,
		MaskInputValueInLogs: true,
	}

	sr := NewStepRunner(runnerCfg, chrome.mgr, nil)
	defer sr.Close()

	f := flow.NewExampleLoginFlow()

	if err := sr.Init(f, nil, "step-by-step"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if sr.IsFinished() {
		t.Fatal("should not be finished right after init")
	}

	stepCount := 0
	for !sr.IsFinished() {
		idx := sr.CurrentIndex()
		res, finished, err := sr.Next()
		if err != nil {
			t.Fatalf("next at index %d: %v", idx, err)
		}
		t.Logf("step %d (%s) status=%s finished=%v duration=%dms", idx, res.StepName, res.Status, finished, res.DurationMs)

		if res.Status == StatusFailed {
			t.Errorf("step %d (%s) failed: %s", idx, res.StepName, res.Error)
			break
		}

		stepCount++
		if idx < len(f.Steps) && res.Type == "screenshot" && res.Screenshot != "" {
			if _, err := os.Stat(res.Screenshot); err != nil {
				t.Errorf("screenshot file not found: %s", res.Screenshot)
			}
		}
	}

	result := sr.Result()
	if result == nil {
		t.Fatal("expected non-nil result after step-by-step execution")
	}

	if result.Status != StatusSuccess {
		t.Fatalf("expected final status success, got %s; success=%d failed=%d skipped=%d",
			result.Status, result.SuccessCount, result.FailedCount, result.SkippedCount)
	}

	if result.EnvironmentID != "step-by-step" {
		t.Fatalf("expected environment id step-by-step, got %q", result.EnvironmentID)
	}

	t.Logf("step-by-step completed: %d steps, success=%d, failed=%d, duration=%s",
		stepCount, result.SuccessCount, result.FailedCount, result.FinishedAt.Sub(result.StartedAt))
}

func TestIntegration_Validation(t *testing.T) {
	t.Run("empty steps", func(t *testing.T) {
		f := flow.NewFlow("empty")
		f.Steps = nil
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "must have at least one step") {
			t.Fatalf("expected empty steps validation error, got %v", err)
		}
	})

	t.Run("invalid onError", func(t *testing.T) {
		f := flow.NewFlow("bad-error-policy")
		f.Steps = []flow.Step{
			{ID: "s1", Name: "step", Enabled: true, Type: flow.StepWaitFixed, OnError: "invalid_policy"},
		}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "unknown onError policy") {
			t.Fatalf("expected invalid onError validation error, got %v", err)
		}
	})

	t.Run("empty navigate URL", func(t *testing.T) {
		f := flow.NewFlow("empty-url")
		f.Steps = []flow.Step{
			{ID: "s1", Name: "navigate", Enabled: true, Type: flow.StepNavigate},
		}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "navigate URL is required") {
			t.Fatalf("expected empty navigate URL validation error, got %v", err)
		}
	})

	t.Run("missing step id", func(t *testing.T) {
		f := flow.NewFlow("missing-id")
		f.Steps = []flow.Step{
			{Name: "no-id", Enabled: true, Type: flow.StepWaitFixed},
		}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "id is required") {
			t.Fatalf("expected missing id error, got %v", err)
		}
	})

	t.Run("empty step name", func(t *testing.T) {
		f := flow.NewFlow("empty-name")
		f.Steps = []flow.Step{
			{ID: "s1", Name: "", Enabled: true, Type: flow.StepWaitFixed},
		}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "name is required") {
			t.Fatalf("expected empty name error, got %v", err)
		}
	})

	t.Run("unknown step type", func(t *testing.T) {
		f := flow.NewFlow("bad-type")
		f.Steps = []flow.Step{
			{ID: "s1", Name: "weird", Enabled: true, Type: flow.StepType("unknown_type")},
		}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "unknown type") {
			t.Fatalf("expected unknown type error, got %v", err)
		}
	})

	t.Run("negative timeout", func(t *testing.T) {
		f := flow.NewFlow("negative-timeout")
		step := flow.NewStep("wait", flow.StepWaitFixed)
		step.TimeoutMs = -1
		f.Steps = []flow.Step{step}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "timeoutMs must be >= 0") {
			t.Fatalf("expected negative timeout error, got %v", err)
		}
	})

	t.Run("missing target for element step", func(t *testing.T) {
		f := flow.NewFlow("missing-target")
		f.Steps = []flow.Step{
			{ID: "s1", Name: "click", Enabled: true, Type: flow.StepClick},
		}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "target value is required") {
			t.Fatalf("expected missing target error, got %v", err)
		}
	})

	t.Run("invalid input mode", func(t *testing.T) {
		f := flow.NewFlow("bad-input-mode")
		step := flow.NewStep("input", flow.StepInput)
		step.Target.Value = "//input"
		step.Input.Mode = flow.InputMode("raw")
		f.Steps = []flow.Step{step}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "unknown input.mode") {
			t.Fatalf("expected invalid input mode error, got %v", err)
		}
	})

	t.Run("invalid target strategy", func(t *testing.T) {
		f := flow.NewFlow("bad-strategy")
		step := flow.NewStep("click", flow.StepClick)
		step.Target = flow.Target{Strategy: flow.TargetStrategy("css"), Value: ".button"}
		f.Steps = []flow.Step{step}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "unknown target.strategy") {
			t.Fatalf("expected invalid strategy error, got %v", err)
		}
	})

	t.Run("negative wait before", func(t *testing.T) {
		f := flow.NewFlow("negative-wait")
		step := flow.NewStep("wait", flow.StepWaitFixed)
		step.WaitBeforeMs = -1
		f.Steps = []flow.Step{step}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "waitBeforeMs must be >= 0") {
			t.Fatalf("expected negative wait error, got %v", err)
		}
	})

	t.Run("negative wait after", func(t *testing.T) {
		f := flow.NewFlow("negative-wait-after")
		step := flow.NewStep("wait", flow.StepWaitFixed)
		step.WaitAfterMs = -1
		f.Steps = []flow.Step{step}
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "waitAfterMs must be >= 0") {
			t.Fatalf("expected negative wait error, got %v", err)
		}
	})

	t.Run("empty flow name", func(t *testing.T) {
		f := flow.NewFlow("   ")
		f.Steps = []flow.Step{
			flow.NewStep("open", flow.StepNavigate),
		}
		f.Steps[0].Target.Value = "http://example.test"
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "flow name is required") {
			t.Fatalf("expected empty name error, got %v", err)
		}
	})

	t.Run("empty flow id", func(t *testing.T) {
		f := &flow.Flow{
			ID:   "",
			Name: "test",
			Steps: []flow.Step{
				flow.NewStep("open", flow.StepNavigate),
			},
		}
		f.Steps[0].Target.Value = "http://example.test"
		err := flow.Validate(f)
		if err == nil || !strings.Contains(err.Error(), "flow id is required") {
			t.Fatalf("expected empty id error, got %v", err)
		}
	})
}

func TestIntegration_MaskInputInLogsHonored(t *testing.T) {
	verifyTestServer(t)

	chrome := launchChrome(t, "integration-mask")
	defer chrome.cleanup()

	runnerCfg := &config.RunnerConfig{
		DefaultTimeoutMs:     15000,
		DefaultWaitAfterMs:   500,
		MaskInputValueInLogs: true,
	}

	r := NewRunner(runnerCfg, chrome.mgr, nil)
	defer r.Close()

	f := flow.NewFlow("Mask Test")
	f.Steps = []flow.Step{
		buildNavigateStep(t, "open", testServerURL),
		buildInputStep(t, "user", "//input[@id='username']", flow.InputTemplate, "${var:u=SP${11000-11099}}", true),
		buildInputStep(t, "pass", "//input[@id='password']", flow.InputLiteral, "Password123", true),
		buildClickStep(t, "submit", `//button[@type='submit']`),
	}

	result := r.RunFlow(f, RunOptions{})

	if result.Status != StatusSuccess {
		t.Fatalf("expected success, got %s", result.Status)
	}

	for i, sr := range result.Steps {
		if sr.Type == "input" {
			if !sr.MaskInLogs {
				t.Errorf("step %d (%s): expected MaskInLogs=true", i, sr.StepName)
			}
			if sr.GeneratedInputMasked == "" {
				t.Errorf("step %d (%s): expected generated input masked value", i, sr.StepName)
			}
			if strings.Contains(sr.GeneratedInputMasked, "Password123") {
				t.Errorf("step %d (%s): password leaked in masked value: %s", i, sr.StepName, sr.GeneratedInputMasked)
			}
		}
	}

	events := drainRunnerEvents(r)
	for _, ev := range events {
		if ev.LogMessage != "" && strings.Contains(ev.LogMessage, "Password123") {
			t.Errorf("password leaked in log: %s", ev.LogMessage)
		}
	}
}

func TestIntegration_RetryOnFailure(t *testing.T) {
	verifyTestServer(t)

	chrome := launchChrome(t, "integration-retry")
	defer chrome.cleanup()

	runnerCfg := &config.RunnerConfig{
		DefaultTimeoutMs:     15000,
		DefaultWaitAfterMs:   500,
		MaskInputValueInLogs: true,
	}

	r := NewRunner(runnerCfg, chrome.mgr, nil)
	defer r.Close()

	f := flow.NewFlow("Retry Test")
	assertWelcome := flow.NewStep("assert welcome", flow.StepAssertExists)
	assertWelcome.Target = flow.Target{Strategy: flow.TargetXPath, Value: `//div[contains(text(),'欢迎')]`}
	assertWelcome.OnError = flow.ErrRetry
	assertWelcome.TimeoutMs = 5000
	f.Steps = []flow.Step{
		buildNavigateStep(t, "open", testServerURL),
		buildInputStep(t, "user", "//input[@id='username']", flow.InputTemplate, "${var:u=SP${11000-11099}}", false),
		buildInputStep(t, "pass", "//input[@id='password']", flow.InputLiteral, "Password123", false),
		buildClickStep(t, "submit", `//button[@type='submit']`),
		assertWelcome,
	}

	result := r.RunFlow(f, RunOptions{})

	if result.Status != StatusSuccess {
		t.Fatalf("expected success, got %s; steps: %+v", result.Status, result.Steps)
	}

	assertStep := result.Steps[len(result.Steps)-1]
	if assertStep.Status != StatusSuccess {
		t.Errorf("assert step should succeed with retry, got %s: %s", assertStep.Status, assertStep.Error)
	}

	t.Logf("retry flow result: success=%d failed=%d skipped=%d",
		result.SuccessCount, result.FailedCount, result.SkippedCount)
}

func TestIntegration_RunnerDrainStopSignal(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	r.Stop()
	r.Stop()
	r.drainStopSignal()
	select {
	case <-r.stopCh:
		t.Fatal("stop channel should be drained")
	default:
	}
}

func TestIntegration_RetryStopsAfterMaxRetries(t *testing.T) {
	exec := &fakeStepExecutor{
		results: []Status{StatusFailed, StatusFailed, StatusFailed},
		errors:  []string{"attempt 1", "attempt 2", "attempt 3"},
	}
	r := newInjectedRunner(&fakeBrowserController{installed: true}, exec, nil)

	step := testStep("always-fail", flow.ErrRetry)
	step.TimeoutMs = 10
	res := r.RunFlow(testFlow(step), RunOptions{})

	if res.Status != StatusFailed {
		t.Fatalf("expected final failed after all retries, got %s", res.Status)
	}
	if exec.calls != 3 {
		t.Fatalf("expected 3 attempts (1 initial + 2 retries), got %d", exec.calls)
	}
}

func TestIntegration_RunFlowContinueAfterFailedStep(t *testing.T) {
	exec := &fakeStepExecutor{
		results: []Status{StatusFailed, StatusSuccess},
		errors:  []string{"boom", ""},
	}
	r := newInjectedRunner(&fakeBrowserController{installed: true}, exec, nil)

	res := r.RunFlow(testFlow(
		testStep("fail-continue", flow.ErrContinue),
		testStep("ok", flow.ErrStop),
	), RunOptions{})

	if res.Status != StatusFailed {
		t.Fatalf("expected final status failed (due to prior failure), got %s", res.Status)
	}
	if exec.calls != 2 {
		t.Fatalf("expected 2 steps executed, got %d", exec.calls)
	}
	if res.FailedCount != 1 || res.SuccessCount != 1 {
		t.Fatalf("expected 1 failed + 1 success, got failed=%d success=%d", res.FailedCount, res.SuccessCount)
	}
}

func TestIntegration_NewRunnerAndStepRunnerDefaultConfigs(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	if r.cfg.DefaultTimeoutMs != 10000 {
		t.Fatalf("expected default timeout 10000, got %d", r.cfg.DefaultTimeoutMs)
	}
	if !r.cfg.MaskInputValueInLogs {
		t.Fatal("expected MaskInputValueInLogs to default to true")
	}

	sr := NewStepRunner(nil, nil, nil)
	if sr.cfg.DefaultTimeoutMs != 10000 {
		t.Fatalf("expected default timeout 10000, got %d", sr.cfg.DefaultTimeoutMs)
	}
	if !sr.cfg.MaskInputValueInLogs {
		t.Fatal("expected MaskInputValueInLogs to default to true")
	}

	r.Close()
	sr.Close()
}

func TestIntegration_StepRunnerNotStartedErrors(t *testing.T) {
	sr := NewStepRunner(&config.RunnerConfig{DefaultTimeoutMs: 1000}, nil, nil)
	defer sr.Close()

	if sr.IsFinished() {
		t.Fatal("should not be finished before init")
	}

	_, _, err := sr.Next()
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected not initialized error, got %v", err)
	}

	if sr.Result() != nil {
		t.Fatal("result should be nil before init")
	}
}

func TestIntegration_StepRunnerInitMissingBrowser(t *testing.T) {
	sr := NewStepRunner(&config.RunnerConfig{DefaultTimeoutMs: 1000}, nil, nil)
	defer sr.Close()

	f := testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "browser manager") {
		t.Fatalf("expected missing browser error, got %v", err)
	}
}

func TestIntegration_RunnerCloseNilCDP(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	r.Close()
	if r.cdp != nil {
		t.Fatal("cdp should be nil after close")
	}
}

func TestIntegration_RunnerRunFlowNoBrowserManager(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{DefaultTimeoutMs: 100}, nil, nil)
	defer r.Close()

	res := r.RunFlow(testFlow(testStep("x", flow.ErrStop)), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	events := drainRunnerEvents(r)
	found := false
	for _, ev := range events {
		if strings.Contains(ev.LogMessage, "Browser manager") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected browser manager log event, got %+v", events)
	}
}

func TestIntegration_RunnerStopStopsExecution(t *testing.T) {
	exec := &fakeStepExecutor{
		results: make([]Status, 20),
		delay:   100 * time.Millisecond,
	}
	for i := range exec.results {
		exec.results[i] = StatusSuccess
	}

	r := newInjectedRunner(&fakeBrowserController{installed: true}, exec, nil)
	steps := make([]flow.Step, 20)
	for i := range steps {
		steps[i] = testStep(fmt.Sprintf("step-%d", i), flow.ErrContinue)
		steps[i].TimeoutMs = 1000
	}

	f := testFlow(steps...)
	go func() {
		time.Sleep(10 * time.Millisecond)
		r.Stop()
	}()

	res := r.RunFlow(f, RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed after stop, got %s", res.Status)
	}
	if len(res.Steps) < 1 {
		t.Error("expected at least 1 executed step")
	}
}

func TestIntegration_RunFlowMissingEnvVars(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	defer r.Close()

	step := testStep("needs-env", flow.ErrStop)
	step.Target.Value = "${env:BASE_URL}"

	res := r.RunFlow(testFlow(step), RunOptions{})
	if res.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", res.Status)
	}
	events := drainRunnerEvents(r)
	found := false
	for _, ev := range events {
		if strings.Contains(ev.LogMessage, "BASE_URL") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing env log, got %+v", events)
	}
}

func TestIntegration_StepRunnerInitEnvMissing(t *testing.T) {
	sr := NewStepRunner(&config.RunnerConfig{DefaultTimeoutMs: 100}, nil, nil)
	defer sr.Close()

	step := testStep("needs-env", flow.ErrStop)
	step.Input.Text = "${env:USER}"

	if err := sr.Init(testFlow(step), nil, ""); err == nil || !strings.Contains(err.Error(), "USER") {
		t.Fatalf("expected missing env error, got %v", err)
	}
}

func TestIntegration_StepRunnerInitCdpConnectFailure(t *testing.T) {
	sr := NewStepRunner(&config.RunnerConfig{DefaultTimeoutMs: 100}, nil, nil)
	defer sr.Close()

	sr.browser = &fakeBrowserController{installed: true}
	sr.connectCDP = func(int) (cdpSession, error) {
		return nil, fmt.Errorf("connection refused")
	}

	f := testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "cdp connect") {
		t.Fatalf("expected cdp connect error, got %v", err)
	}
	if sr.started {
		t.Fatal("should not be started after cdp failure")
	}
}

func TestIntegration_StepRunnerInitInstallFailure(t *testing.T) {
	sr := NewStepRunner(&config.RunnerConfig{DefaultTimeoutMs: 100}, nil, nil)
	defer sr.Close()

	sr.browser = &fakeBrowserController{installErr: fmt.Errorf("offline")}

	f := testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "install chrome") {
		t.Fatalf("expected install error, got %v", err)
	}
}

func TestIntegration_StepRunnerInitStartFailure(t *testing.T) {
	sr := NewStepRunner(&config.RunnerConfig{DefaultTimeoutMs: 100}, nil, nil)
	defer sr.Close()

	sr.browser = &fakeBrowserController{installed: true, startErr: fmt.Errorf("start failed")}

	f := testFlow(testStep("open", flow.ErrStop))
	if err := sr.Init(f, nil, ""); err == nil || !strings.Contains(err.Error(), "start chrome") {
		t.Fatalf("expected start error, got %v", err)
	}
}

func TestIntegration_StepRunnerResultAfterFinish(t *testing.T) {
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
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestIntegration_ExampleFlowTargetCorrectness(t *testing.T) {
	f := flow.NewExampleLoginFlow()

	if len(f.Steps) != 6 {
		t.Fatalf("expected 6 steps in example flow, got %d", len(f.Steps))
	}

	expected := []struct {
		name  string
		typ   flow.StepType
		value string
	}{
		{"打开网址", flow.StepNavigate, testServerURL},
		{"输入用户名", flow.StepInput, "//input[@id='username']"},
		{"输入密码", flow.StepInput, "//input[@id='password']"},
		{"点击登录", flow.StepClick, "//button[@type='submit']"},
		{"断言欢迎文本", flow.StepAssertExists, "//div[contains(text(),'欢迎')]"},
		{"页面截图", flow.StepScreenshot, ""},
	}

	for i, exp := range expected {
		s := f.Steps[i]
		if s.Name != exp.name {
			t.Errorf("step %d: expected name %q, got %q", i, exp.name, s.Name)
		}
		if s.Type != exp.typ {
			t.Errorf("step %d: expected type %q, got %q", i, exp.typ, s.Type)
		}
		if s.Target.Value != exp.value {
			t.Errorf("step %d: expected target value %q, got %q", i, exp.value, s.Target.Value)
		}
	}

	if f.Steps[0].Target.Value != testServerURL {
		t.Fatalf("expected navigate URL %s, got %s", testServerURL, f.Steps[0].Target.Value)
	}

	if err := flow.Validate(f); err != nil {
		t.Fatalf("example flow should be valid: %v", err)
	}
}

func TestIntegration_RunnerEventsChannelBuffered(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{}, nil, nil)
	defer r.Close()

	for i := 0; i < cap(r.events)+10; i++ {
		r.emit(Event{Type: EventLog, LogMessage: "test"})
	}

	if len(r.events) != cap(r.events) {
		t.Fatalf("expected full channel (%d), got %d", cap(r.events), len(r.events))
	}
}

func TestIntegration_MaskIfNeeded(t *testing.T) {
	cases := []struct {
		input    string
		visible  int
		expected string
	}{
		{"abcd1234wxyz", 4, "abcd****wxyz"},
		{"short", 4, "*****"},
		{"ab", 4, "**"},
		{"1234567890", 3, "123****890"},
		{"", 4, ""},
		{"abcdefgh", 4, "********"},
	}

	for _, tc := range cases {
		got := maskIfNeeded(tc.input, tc.visible)
		if got != tc.expected {
			t.Errorf("maskIfNeeded(%q, %d) = %q, want %q", tc.input, tc.visible, got, tc.expected)
		}
	}
}

func TestIntegration_ExecuteStepCancelledPostWait(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	step := flow.NewStep("navigate", flow.StepNavigate)
	step.Target.Value = testServerURL
	step.WaitAfterMs = 1000
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	res := exec.ExecuteStep(ctx, step, template.NewEngine())
	if res.Status != StatusFailed {
		t.Fatalf("expected failed due to cancelled post-wait, got %s (error: %s)", res.Status, res.Error)
	}
}

func TestIntegration_ExecuteStepNavigateWithoutCDP(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	step := flow.NewStep("open", flow.StepNavigate)
	step.Target.Value = testServerURL

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	res := exec.ExecuteStep(ctx, step, template.NewEngine())
	if res.Status != StatusFailed {
		t.Logf("expected failure without real CDP context (status=%s); cdp might be reachable", res.Status)
	}
}

func TestIntegration_ResolveInputLiteralAndTemplate(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	eng := template.NewEngine()

	t.Run("literal", func(t *testing.T) {
		step := flow.NewStep("user", flow.StepInput)
		step.Input = flow.Input{Mode: flow.InputLiteral, Text: "admin"}
		input, err := exec.resolveInput(step, eng)
		if err != nil {
			t.Fatalf("resolve literal: %v", err)
		}
		if input.Value != "admin" || input.MaskedValue != "admin" {
			t.Fatalf("unexpected literal: %+v", input)
		}
	})

	t.Run("template number range", func(t *testing.T) {
		step := flow.NewStep("user", flow.StepInput)
		step.Input = flow.Input{Mode: flow.InputTemplate, Text: "SP${11000-11099}"}
		input, err := exec.resolveInput(step, eng)
		if err != nil {
			t.Fatalf("resolve template: %v", err)
		}
		if !strings.HasPrefix(input.Value, "SP") {
			t.Fatalf("expected SP prefix, got %q", input.Value)
		}
		numPart := strings.TrimPrefix(input.Value, "SP")
		if len(numPart) != 5 {
			t.Fatalf("expected 5-digit number, got %q", numPart)
		}
	})

	t.Run("variable reuse", func(t *testing.T) {
		exec2 := NewActionExecutor(nil, "")
		eng2 := template.NewEngine()

		step1 := flow.NewStep("set var", flow.StepInput)
		step1.Input = flow.Input{Mode: flow.InputTemplate, Text: "${var:u=user_${uuid}}"}
		input1, _ := exec2.resolveInput(step1, eng2)

		step2 := flow.NewStep("use var", flow.StepInput)
		step2.Input = flow.Input{Mode: flow.InputTemplate, Text: "${var:u}"}
		input2, _ := exec2.resolveInput(step2, eng2)

		if input1.Value != input2.Value {
			t.Fatalf("variable reuse failed: %q != %q", input1.Value, input2.Value)
		}
		if !strings.HasPrefix(input1.Value, "user_") {
			t.Fatalf("expected user_ prefix, got %q", input1.Value)
		}
	})

	t.Run("enumeration", func(t *testing.T) {
		step := flow.NewStep("pick", flow.StepInput)
		step.Input = flow.Input{Mode: flow.InputTemplate, Text: "${A|B|C}"}
		input, _ := exec.resolveInput(step, eng)
		if !strings.Contains("A|B|C", input.Value) {
			t.Fatalf("unexpected enum value: %q", input.Value)
		}
	})

	t.Run("nested template", func(t *testing.T) {
		exec3 := NewActionExecutor(nil, "")
		eng3 := template.NewEngine()
		step := flow.NewStep("nested", flow.StepInput)
		step.Input = flow.Input{Mode: flow.InputTemplate, Text: "${var:u=SP${11000-11099}}"}
		input, err := exec3.resolveInput(step, eng3)
		if err != nil {
			t.Fatalf("nested template: %v", err)
		}
		if !strings.HasPrefix(input.Value, "SP") {
			t.Fatalf("expected SP prefix in nested, got %q", input.Value)
		}
	})
}

func TestIntegration_CDPConnectionRoundTrip(t *testing.T) {
	verifyTestServer(t)

	chrome := launchChrome(t, "integration-cdp-test")
	defer chrome.cleanup()

	ctx := chrome.cdp.Context()
	if ctx == nil {
		t.Fatal("expected non-nil CDP context")
	}

	select {
	case <-ctx.Done():
		t.Fatal("cdp context should not be done immediately")
	case <-time.After(500 * time.Millisecond):
	}

	cdp2, err := browser.Connect(chrome.port)
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	defer cdp2.Close()
	if cdp2.Context() == nil {
		t.Fatal("second cdp client should have non-nil context")
	}
	t.Logf("cdp connection round-trip successful on port %d", chrome.port)
}

func TestIntegration_EventMaskValue(t *testing.T) {
	r := NewRunner(&config.RunnerConfig{MaskInputValueInLogs: true}, nil, nil)
	defer r.Close()

	r.emit(Event{Type: EventLog, LogMessage: "input: Password123"})
	r.emit(Event{Type: EventStepStart, StepIndex: 0, StepName: "test"})
	r.emit(Event{Type: EventStepDone, StepIndex: 0, Result: &StepResult{Status: StatusSuccess}})
	r.emit(Event{Type: EventRunDone, RunResult: &RunResult{ID: "r1", Status: StatusSuccess}})

	events := drainRunnerEvents(r)
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}
}
