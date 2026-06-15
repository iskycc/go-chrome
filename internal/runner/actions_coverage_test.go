package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

func TestExecuteStepActionsSuccess(t *testing.T) {
	oldRun := chromedpRun
	oldNodes := queryNodesFunc
	oldText := queryTextFunc
	defer func() {
		chromedpRun = oldRun
		queryNodesFunc = oldNodes
		queryTextFunc = oldText
	}()

	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}
	queryNodesFunc = func(ctx context.Context, sel interface{}, nodes *[]*cdp.Node, opts ...chromedp.QueryOption) error {
		*nodes = append(*nodes, &cdp.Node{})
		return nil
	}
	queryTextFunc = func(ctx context.Context, sel interface{}, text *string, opts ...chromedp.QueryOption) error {
		*text = "expected text content"
		return nil
	}

	exec := NewActionExecutor(context.Background(), t.TempDir())
	eng := template.NewEngine()

	cases := []struct {
		name string
		step flow.Step
	}{
		{"navigate", flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: "https://example.test"}}},
		{"click", flow.Step{ID: "s", Name: "click", Enabled: true, Type: flow.StepClick, Target: flow.Target{Value: "//button"}}},
		{"input", flow.Step{ID: "s", Name: "input", Enabled: true, Type: flow.StepInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "value"}}},
		{"clearAndInput", flow.Step{ID: "s", Name: "clear", Enabled: true, Type: flow.StepClearAndInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "value"}}},
		{"waitPresent", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitPresent, Target: flow.Target{Value: "//div"}}},
		{"waitVisible", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitVisible, Target: flow.Target{Value: "//div"}}},
		{"waitFixed", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitFixed}},
		{"getText", flow.Step{ID: "s", Name: "get", Enabled: true, Type: flow.StepGetText, Target: flow.Target{Value: "//div"}}},
		{"assertExists", flow.Step{ID: "s", Name: "assert", Enabled: true, Type: flow.StepAssertExists, Target: flow.Target{Value: "//div"}}},
		{"assertText", flow.Step{ID: "s", Name: "assert", Enabled: true, Type: flow.StepAssertText, Target: flow.Target{Value: "//div"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "expected"}}},
		{"screenshot", flow.Step{ID: "s", Name: "ss", Enabled: true, Type: flow.StepScreenshot}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := exec.ExecuteStep(context.Background(), tc.step, eng)
			if res.Status != StatusSuccess {
				t.Fatalf("expected success, got %+v", res)
			}
		})
	}
}

func TestExecuteStepActionsFailure(t *testing.T) {
	old := chromedpRun
	defer func() { chromedpRun = old }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return errors.New("mock chromedp error")
	}

	exec := NewActionExecutor(context.Background(), t.TempDir())
	eng := template.NewEngine()

	cases := []struct {
		name string
		step flow.Step
	}{
		{"navigate", flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: "https://example.test"}}},
		{"click", flow.Step{ID: "s", Name: "click", Enabled: true, Type: flow.StepClick, Target: flow.Target{Value: "//button"}}},
		{"input", flow.Step{ID: "s", Name: "input", Enabled: true, Type: flow.StepInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "value"}}},
		{"clearAndInput", flow.Step{ID: "s", Name: "clear", Enabled: true, Type: flow.StepClearAndInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "value"}}},
		{"waitPresent", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitPresent, Target: flow.Target{Value: "//div"}}},
		{"waitVisible", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitVisible, Target: flow.Target{Value: "//div"}}},
		{"getText", flow.Step{ID: "s", Name: "get", Enabled: true, Type: flow.StepGetText, Target: flow.Target{Value: "//div"}}},
		{"assertExists", flow.Step{ID: "s", Name: "assert", Enabled: true, Type: flow.StepAssertExists, Target: flow.Target{Value: "//div"}}},
		{"assertText", flow.Step{ID: "s", Name: "assert", Enabled: true, Type: flow.StepAssertText, Target: flow.Target{Value: "//div"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "expected"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := exec.ExecuteStep(context.Background(), tc.step, eng)
			if res.Status != StatusFailed {
				t.Fatalf("expected failure, got %+v", res)
			}
		})
	}
}

func TestExecuteStepPostWaitCancelled(t *testing.T) {
	old := chromedpRun
	defer func() { chromedpRun = old }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}

	exec := NewActionExecutor(context.Background(), "")
	step := flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: "https://example.test"}, WaitAfterMs: 100}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := exec.ExecuteStep(ctx, step, template.NewEngine())
	if res.Status != StatusFailed || res.Error != "cancelled during post-wait" {
		t.Fatalf("expected post-wait cancel, got %+v", res)
	}
}

func TestNavigateResolveError(t *testing.T) {
	exec := NewActionExecutor(context.Background(), "")
	step := flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: "${env:MISSING}"}}
	res := exec.ExecuteStep(context.Background(), step, template.NewEngine())
	if res.Status != StatusFailed {
		t.Fatalf("expected failure for unresolved env var, got %+v", res)
	}
}

func TestAssertTextMismatch(t *testing.T) {
	old := chromedpRun
	defer func() { chromedpRun = old }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}

	exec := NewActionExecutor(context.Background(), "")
	step := flow.Step{ID: "s", Name: "assert", Enabled: true, Type: flow.StepAssertText, Target: flow.Target{Value: "//div"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "expected"}}
	res := exec.ExecuteStep(context.Background(), step, template.NewEngine())
	if res.Status != StatusFailed {
		t.Fatalf("expected text mismatch failure, got %+v", res)
	}
}

func TestCaptureArtifactsError(t *testing.T) {
	old := chromedpRun
	defer func() { chromedpRun = old }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return errors.New("mock")
	}

	exec := NewActionExecutor(context.Background(), t.TempDir())
	step := flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: "https://example.test"}}
	res := exec.ExecuteStep(context.Background(), step, template.NewEngine())
	if res.Status != StatusFailed {
		t.Fatalf("expected failure, got %+v", res)
	}
	// Screenshot and HTML artifacts should be empty on error.
	if res.Screenshot != "" || res.HTMLSnapshot != "" {
		t.Fatalf("expected no artifacts on capture error, got screenshot=%q html=%q", res.Screenshot, res.HTMLSnapshot)
	}
}

func TestActionExecutorResolveErrors(t *testing.T) {
	exec := NewActionExecutor(context.Background(), "")
	eng := template.NewEngine()

	cases := []flow.StepType{
		flow.StepClick,
		flow.StepInput,
		flow.StepClearAndInput,
		flow.StepWaitPresent,
		flow.StepWaitVisible,
		flow.StepGetText,
		flow.StepAssertExists,
		flow.StepAssertText,
	}
	for _, typ := range cases {
		step := flow.Step{ID: "s", Name: "x", Enabled: true, Type: typ, Target: flow.Target{Value: "${env:MISSING}"}}
		res := exec.ExecuteStep(context.Background(), step, eng)
		if res.Status != StatusFailed {
			t.Fatalf("expected failure for %s unresolved target", typ)
		}
	}
}

func TestExecuteStepMiscPaths(t *testing.T) {
	oldRun := chromedpRun
	defer func() { chromedpRun = oldRun }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}

	exec := NewActionExecutor(context.Background(), "")
	eng := template.NewEngine()

	// disabled step
	res := exec.ExecuteStep(context.Background(), flow.Step{ID: "s", Name: "x", Enabled: false, Type: flow.StepNavigate}, eng)
	if res.Status != StatusSkipped {
		t.Fatalf("expected skipped, got %+v", res)
	}

	// unsupported step type
	res = exec.ExecuteStep(context.Background(), flow.Step{ID: "s", Name: "x", Enabled: true, Type: "unknown"}, eng)
	if res.Status != StatusFailed {
		t.Fatalf("expected failed for unsupported type, got %+v", res)
	}

	// empty url
	res = exec.ExecuteStep(context.Background(), flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: ""}}, eng)
	if res.Status != StatusFailed || res.Error != "url is empty" {
		t.Fatalf("expected empty url failure, got %+v", res)
	}

	// pre-wait cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res = exec.ExecuteStep(ctx, flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: "https://x"}, WaitBeforeMs: 100}, eng)
	if res.Status != StatusFailed || res.Error != "cancelled during pre-wait" {
		t.Fatalf("expected pre-wait cancel, got %+v", res)
	}
}

func TestWaitFixedBranches(t *testing.T) {
	oldRun := chromedpRun
	defer func() { chromedpRun = oldRun }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}

	exec := NewActionExecutor(context.Background(), "")
	eng := template.NewEngine()

	cases := []struct {
		name   string
		step   flow.Step
		status Status
	}{
		{"duration_ms", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitFixed, Target: flow.Target{Value: "50ms"}}, StatusSuccess},
		{"plain_number", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitFixed, Target: flow.Target{Value: "30"}}, StatusSuccess},
		{"default_wait", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitFixed, WaitAfterMs: 10}, StatusSuccess},
		{"invalid", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitFixed, Target: flow.Target{Value: "abc"}}, StatusFailed},
		{"negative", flow.Step{ID: "s", Name: "wait", Enabled: true, Type: flow.StepWaitFixed, Target: flow.Target{Value: "-5ms"}}, StatusFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := exec.ExecuteStep(context.Background(), tc.step, eng)
			if res.Status != tc.status {
				t.Fatalf("expected %s, got %+v", tc.status, res)
			}
		})
	}
}

func TestInputMaskAndTemplateError(t *testing.T) {
	oldRun := chromedpRun
	defer func() { chromedpRun = oldRun }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}

	exec := NewActionExecutor(context.Background(), "")
	eng := template.NewEngine()

	// mask via step flag
	step := flow.Step{ID: "s", Name: "in", Enabled: true, Type: flow.StepInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "secret", MaskInLogs: true}}
	res := exec.ExecuteStep(context.Background(), step, eng)
	if res.Status != StatusSuccess || !res.MaskInLogs || res.GeneratedInputMasked == "secret" {
		t.Fatalf("expected masked input, got %+v", res)
	}

	// mask via executor default
	execMasked := NewActionExecutor(context.Background(), "", true)
	res = execMasked.ExecuteStep(context.Background(), flow.Step{ID: "s", Name: "in", Enabled: true, Type: flow.StepInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputLiteral, Text: "value"}}, eng)
	if res.Status != StatusSuccess || !res.MaskInLogs {
		t.Fatalf("expected masked by default, got %+v", res)
	}

	// template mode with invalid expression
	res = exec.ExecuteStep(context.Background(), flow.Step{ID: "s", Name: "in", Enabled: true, Type: flow.StepInput, Target: flow.Target{Value: "//input"}, Input: flow.Input{Mode: flow.InputTemplate, Text: "${env:MISSING}"}}, eng)
	if res.Status != StatusFailed {
		t.Fatalf("expected template resolve failure, got %+v", res)
	}
}

func TestFailureArtifactsCaptured(t *testing.T) {
	oldRun := chromedpRun
	defer func() { chromedpRun = oldRun }()
	chromedpRun = func(ctx context.Context, actions ...chromedp.Action) error {
		return nil
	}

	dir := t.TempDir()
	exec := NewActionExecutor(context.Background(), dir)
	step := flow.Step{ID: "s", Name: "nav", Enabled: true, Type: flow.StepNavigate, Target: flow.Target{Value: ""}}
	res := exec.ExecuteStep(context.Background(), step, template.NewEngine())
	if res.Status != StatusFailed {
		t.Fatalf("expected failure, got %+v", res)
	}
	if res.Screenshot == "" || res.HTMLSnapshot == "" {
		t.Fatalf("expected screenshot and html artifacts, got %+v", res)
	}
}
