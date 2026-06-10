package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

type actionTestEnvProvider struct {
	data    map[string]string
	secrets map[string]bool
}

func (p *actionTestEnvProvider) GetEnvValue(key string) (string, bool, bool) {
	v, ok := p.data[key]
	return v, ok, p.secrets[key]
}

func TestResolveTextEvaluatesEnvironmentVariables(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	eng := template.NewEngineWithEnv(&actionTestEnvProvider{
		data: map[string]string{"BASE_URL": "https://example.test"},
	})

	got, err := exec.resolveText("${env:BASE_URL}/login", eng)
	if err != nil {
		t.Fatalf("resolve text: %v", err)
	}
	if got != "https://example.test/login" {
		t.Fatalf("unexpected resolved text: %s", got)
	}
}

func TestResolveInputMasksSecretEnvironmentVariables(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	eng := template.NewEngineWithEnv(&actionTestEnvProvider{
		data:    map[string]string{"PASSWORD": "super-secret-value"},
		secrets: map[string]bool{"PASSWORD": true},
	})
	step := flow.NewStep("password", flow.StepInput)
	step.Input = flow.Input{Mode: flow.InputTemplate, Text: "${env:PASSWORD}"}

	input, err := exec.resolveInput(step, eng)
	if err != nil {
		t.Fatalf("resolve input: %v", err)
	}
	res := &StepResult{}
	exec.setGeneratedInput(res, step, input)

	if !res.MaskInLogs {
		t.Fatal("expected mask flag")
	}
	if strings.Contains(res.GeneratedInputMasked, "super-secret-value") {
		t.Fatalf("generated input leaked secret: %s", res.GeneratedInputMasked)
	}
}

func TestResolveInputHonorsDefaultMaskConfig(t *testing.T) {
	exec := NewActionExecutor(nil, "", true)
	eng := template.NewEngine()
	step := flow.NewStep("username", flow.StepInput)
	step.Input = flow.Input{Mode: flow.InputTemplate, Text: "plain-user"}

	input, err := exec.resolveInput(step, eng)
	if err != nil {
		t.Fatalf("resolve input: %v", err)
	}
	res := &StepResult{}
	exec.setGeneratedInput(res, step, input)

	if !res.MaskInLogs {
		t.Fatal("expected default mask flag")
	}
	if res.GeneratedInputMasked == "plain-user" {
		t.Fatal("expected masked generated input")
	}
}

func TestWaitFixedUsesWaitAfterMsWhenTargetEmpty(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	step := flow.NewStep("wait", flow.StepWaitFixed)
	step.WaitAfterMs = 1

	start := time.Now()
	if err := exec.waitFixed(context.Background(), step, template.NewEngine()); err != nil {
		t.Fatalf("wait fixed: %v", err)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("wait fixed took unexpectedly long")
	}
}

func TestWaitFixedSupportsTemplateTargetOverride(t *testing.T) {
	exec := NewActionExecutor(nil, "")
	step := flow.NewStep("wait", flow.StepWaitFixed)
	step.Target.Value = "${env:WAIT_MS}"
	eng := template.NewEngineWithEnv(&actionTestEnvProvider{data: map[string]string{"WAIT_MS": "1"}})

	if err := exec.waitFixed(context.Background(), step, eng); err != nil {
		t.Fatalf("wait fixed with template target: %v", err)
	}
}
