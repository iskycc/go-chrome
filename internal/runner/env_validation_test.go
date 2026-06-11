package runner

import (
	"testing"

	"go-chrome/internal/flow"
)

type envValidationProvider map[string]string

func (p envValidationProvider) GetEnvValue(key string) (string, bool, bool) {
	v, ok := p[key]
	return v, ok, false
}

func TestMissingEnvVarsScansEnabledStepsFromStart(t *testing.T) {
	f := flow.NewFlow("env")
	disabled := flow.NewStep("disabled", flow.StepNavigate)
	disabled.Enabled = false
	disabled.Target.Value = "${env:IGNORED}"
	first := flow.NewStep("open", flow.StepNavigate)
	first.Target.Value = "${env:BASE_URL}/login"
	second := flow.NewStep("input", flow.StepInput)
	second.Input.Text = "${env:USERNAME}"
	second.Note = "${env:NOTE_VAR}"
	f.Steps = []flow.Step{disabled, first, second}

	missing := MissingEnvVars(f, 1, envValidationProvider{"BASE_URL": "https://example.test"})
	if len(missing) != 2 {
		t.Fatalf("expected two missing vars, got %v", missing)
	}
	if missing[0] != "NOTE_VAR" || missing[1] != "USERNAME" {
		t.Fatalf("unexpected missing vars: %v", missing)
	}
}

func TestMissingEnvVarsNoProviderOnlyWhenReferencesExist(t *testing.T) {
	f := flow.NewFlow("env")
	noEnv := flow.NewStep("wait", flow.StepWaitFixed)
	noEnv.Target.Value = "100"
	f.Steps = []flow.Step{noEnv}
	if missing := MissingEnvVars(f, 0, nil); len(missing) != 0 {
		t.Fatalf("expected no missing vars, got %v", missing)
	}

	withEnv := flow.NewStep("open", flow.StepNavigate)
	withEnv.Target.Value = "${env:BASE_URL}"
	f.Steps = append(f.Steps, withEnv)
	missing := MissingEnvVars(f, 0, nil)
	if len(missing) != 1 || missing[0] != "BASE_URL" {
		t.Fatalf("unexpected missing vars: %v", missing)
	}
}
