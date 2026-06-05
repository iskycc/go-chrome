package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"

	"go-chrome/internal/flow"
	"go-chrome/internal/logx"
	"go-chrome/internal/template"
)

// ActionExecutor performs CDP actions.
type ActionExecutor struct {
	cdp context.Context // chromedp context from browser.CDPClient
	dir string          // snapshot directory
}

// NewActionExecutor creates an executor.
func NewActionExecutor(cdpCtx context.Context, snapshotDir string) *ActionExecutor {
	return &ActionExecutor{cdp: cdpCtx, dir: snapshotDir}
}

// ExecuteStep runs a single step.
func (a *ActionExecutor) ExecuteStep(ctx context.Context, step flow.Step, eng *template.Engine) *StepResult {
	res := &StepResult{
		StepID:   step.ID,
		StepName: step.Name,
		Type:     string(step.Type),
		Status:   StatusRunning,
	}
	start := time.Now()
	defer func() {
		res.DurationMs = time.Since(start).Milliseconds()
	}()

	if !step.Enabled {
		res.Status = StatusSkipped
		return res
	}

	if step.WaitBeforeMs > 0 {
		select {
		case <-time.After(time.Duration(step.WaitBeforeMs) * time.Millisecond):
		case <-ctx.Done():
			res.Status = StatusFailed
			res.Error = "cancelled during pre-wait"
			return res
		}
	}

	var err error
	switch step.Type {
	case flow.StepNavigate:
		err = a.navigate(ctx, step.Target.Value)
	case flow.StepClick:
		err = a.click(ctx, step.Target.Value)
	case flow.StepInput:
		err = a.input(ctx, step, eng, res)
	case flow.StepClearAndInput:
		err = a.clearAndInput(ctx, step, eng, res)
	case flow.StepWaitPresent:
		err = a.waitPresent(ctx, step.Target.Value)
	case flow.StepWaitVisible:
		err = a.waitVisible(ctx, step.Target.Value)
	case flow.StepWaitFixed:
		err = a.waitFixed(ctx, step.Target.Value)
	case flow.StepGetText:
		err = a.getText(ctx, step.Target.Value)
	case flow.StepAssertExists:
		err = a.assertExists(ctx, step.Target.Value)
	case flow.StepAssertText:
		err = a.assertText(ctx, step.Target.Value, step.Input.Text)
	case flow.StepScreenshot:
		err = a.screenshot(ctx, res)
	default:
		err = fmt.Errorf("unsupported step type: %s", step.Type)
	}

	if err != nil {
		res.Status = StatusFailed
		res.Error = err.Error()
		// Capture failure artifacts
		if ss, serr := a.captureScreenshot(ctx); serr == nil && ss != "" {
			res.Screenshot = ss
		}
		if html, herr := a.captureHTML(ctx); herr == nil && html != "" {
			res.HTMLSnapshot = html
		}
		return res
	}

	if step.WaitAfterMs > 0 {
		select {
		case <-time.After(time.Duration(step.WaitAfterMs) * time.Millisecond):
		case <-ctx.Done():
			res.Status = StatusFailed
			res.Error = "cancelled during post-wait"
			return res
		}
	}

	res.Status = StatusSuccess
	return res
}

func (a *ActionExecutor) navigate(ctx context.Context, url string) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	return chromedp.Run(ctx, chromedp.Navigate(url))
}

func (a *ActionExecutor) click(ctx context.Context, xpath string) error {
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx, chromedp.Nodes(xpath, &nodes, chromedp.NodeVisible)); err != nil {
		return fmt.Errorf("find element: %w", err)
	}
	if len(nodes) == 0 {
		return fmt.Errorf("no element found for xpath: %s", xpath)
	}
	return chromedp.Run(ctx, chromedp.MouseClickNode(nodes[0]))
}

func (a *ActionExecutor) input(ctx context.Context, step flow.Step, eng *template.Engine, res *StepResult) error {
	val, err := a.resolveInput(step, eng)
	if err != nil {
		return err
	}
	res.GeneratedInput = val
	if err := chromedp.Run(ctx, chromedp.SendKeys(step.Target.Value, val, chromedp.BySearch)); err != nil {
		return fmt.Errorf("input: %w", err)
	}
	return nil
}

func (a *ActionExecutor) clearAndInput(ctx context.Context, step flow.Step, eng *template.Engine, res *StepResult) error {
	val, err := a.resolveInput(step, eng)
	if err != nil {
		return err
	}
	res.GeneratedInput = val
	if err := chromedp.Run(ctx,
		chromedp.Clear(step.Target.Value, chromedp.BySearch),
		chromedp.SendKeys(step.Target.Value, val, chromedp.BySearch),
	); err != nil {
		return fmt.Errorf("clear and input: %w", err)
	}
	return nil
}

func (a *ActionExecutor) waitPresent(ctx context.Context, xpath string) error {
	return chromedp.Run(ctx, chromedp.WaitVisible(xpath, chromedp.BySearch))
}

func (a *ActionExecutor) waitVisible(ctx context.Context, xpath string) error {
	return chromedp.Run(ctx, chromedp.WaitVisible(xpath, chromedp.BySearch))
}

func (a *ActionExecutor) waitFixed(ctx context.Context, val string) error {
	ms, err := time.ParseDuration(val + "ms")
	if err != nil {
		// try plain ms
		var n int
		_, err2 := fmt.Sscanf(val, "%d", &n)
		if err2 != nil {
			return fmt.Errorf("invalid wait value: %s", val)
		}
		ms = time.Duration(n) * time.Millisecond
	}
	select {
	case <-time.After(ms):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *ActionExecutor) getText(ctx context.Context, xpath string) error {
	var text string
	if err := chromedp.Run(ctx, chromedp.Text(xpath, &text, chromedp.BySearch)); err != nil {
		return fmt.Errorf("get text: %w", err)
	}
	logx.Infof("Got text: %s", text)
	return nil
}

func (a *ActionExecutor) assertExists(ctx context.Context, xpath string) error {
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx, chromedp.Nodes(xpath, &nodes, chromedp.BySearch)); err != nil {
		return fmt.Errorf("assert exists: %w", err)
	}
	if len(nodes) == 0 {
		return fmt.Errorf("element does not exist: %s", xpath)
	}
	return nil
}

func (a *ActionExecutor) assertText(ctx context.Context, xpath, expected string) error {
	var text string
	if err := chromedp.Run(ctx, chromedp.Text(xpath, &text, chromedp.BySearch)); err != nil {
		return fmt.Errorf("assert text: %w", err)
	}
	if !strings.Contains(text, expected) {
		return fmt.Errorf("text mismatch: got %q, want to contain %q", text, expected)
	}
	return nil
}

func (a *ActionExecutor) screenshot(ctx context.Context, res *StepResult) error {
	ss, err := a.captureScreenshot(ctx)
	if err != nil {
		return err
	}
	res.Screenshot = ss
	return nil
}

func (a *ActionExecutor) captureScreenshot(ctx context.Context) (string, error) {
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return "", err
	}
	fname := filepath.Join(a.dir, fmt.Sprintf("screenshot-%d.png", time.Now().UnixMilli()))
	if err := os.WriteFile(fname, buf, 0644); err != nil {
		return "", err
	}
	return fname, nil
}

func (a *ActionExecutor) captureHTML(ctx context.Context) (string, error) {
	var html string
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html)); err != nil {
		return "", err
	}
	fname := filepath.Join(a.dir, fmt.Sprintf("page-%d.html", time.Now().UnixMilli()))
	if err := os.WriteFile(fname, []byte(html), 0644); err != nil {
		return "", err
	}
	return fname, nil
}

func (a *ActionExecutor) resolveInput(step flow.Step, eng *template.Engine) (string, error) {
	if step.Input.Mode == flow.InputLiteral {
		return step.Input.Text, nil
	}
	return eng.Evaluate(step.Input.Text)
}
