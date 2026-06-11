package flow

import (
	"testing"
)

func TestValidateEmptyName(t *testing.T) {
	f := NewFlow("")
	if err := Validate(f); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateStepWithoutTarget(t *testing.T) {
	f := NewFlow("Test")
	f.Steps = []Step{NewStep("Click", StepClick)}
	// StepClick requires target
	if err := Validate(f); err == nil {
		t.Fatal("expected error for click without target")
	}
}

func TestValidateStepNegativeWait(t *testing.T) {
	f := NewFlow("Test")
	s := NewStep("Click", StepClick)
	s.Target.Value = "//button"
	s.WaitBeforeMs = -1
	f.Steps = []Step{s}
	if err := Validate(f); err == nil {
		t.Fatal("expected error for negative wait")
	}
}

func TestValidateStepNegativeTimeout(t *testing.T) {
	f := NewFlow("Test")
	s := NewStep("Click", StepClick)
	s.Target.Value = "//button"
	s.TimeoutMs = -1
	f.Steps = []Step{s}
	if err := Validate(f); err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

func TestValidateUnknownStepType(t *testing.T) {
	f := NewFlow("Test")
	s := NewStep("Bad", StepType("unknown_type"))
	f.Steps = []Step{s}
	if err := Validate(f); err == nil {
		t.Fatal("expected error for unknown step type")
	}
}

func TestValidateNavigateWithoutTargetOK(t *testing.T) {
	f := NewFlow("Test")
	f.Steps = []Step{NewStep("Open", StepNavigate)}
	// Navigate does not require element target
	if err := Validate(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
