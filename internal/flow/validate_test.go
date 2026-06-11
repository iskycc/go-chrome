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

func TestValidateEmptyFlowID(t *testing.T) {
	f := NewFlow("Test")
	f.ID = ""
	if err := Validate(f); err == nil {
		t.Fatal("expected error for empty flow id")
	}
}

func TestValidateStepEmptyIDAndName(t *testing.T) {
	f := NewFlow("Test")
	s := NewStep("Step", StepWaitFixed)
	s.ID = ""
	f.Steps = []Step{s}
	if err := Validate(f); err == nil {
		t.Fatal("expected error for empty step id")
	}

	s = NewStep("   ", StepWaitFixed)
	f.Steps = []Step{s}
	if err := Validate(f); err == nil {
		t.Fatal("expected error for empty step name")
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

func TestValidateStepNegativeWaitAfter(t *testing.T) {
	f := NewFlow("Test")
	s := NewStep("Wait", StepWaitFixed)
	s.WaitAfterMs = -1
	f.Steps = []Step{s}
	if err := Validate(f); err == nil {
		t.Fatal("expected error for negative waitAfterMs")
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

func TestNeedsElement(t *testing.T) {
	needs := []StepType{StepClick, StepInput, StepClearAndInput, StepWaitPresent, StepWaitVisible, StepGetText, StepAssertExists, StepAssertText}
	for _, typ := range needs {
		if !NeedsElement(typ) {
			t.Fatalf("expected %s to need an element", typ)
		}
	}
	noTarget := []StepType{StepNavigate, StepWaitFixed, StepScreenshot}
	for _, typ := range noTarget {
		if NeedsElement(typ) {
			t.Fatalf("expected %s to not need an element", typ)
		}
	}
}
