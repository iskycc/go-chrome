package flow

import (
	"errors"
	"fmt"
	"strings"
)

// Validate checks a flow for structural errors.
func Validate(f *Flow) error {
	if f.ID == "" {
		return errors.New("flow id is required")
	}
	if strings.TrimSpace(f.Name) == "" {
		return errors.New("flow name is required")
	}
	if len(f.Steps) == 0 {
		return errors.New("flow must have at least one step")
	}
	for i, s := range f.Steps {
		if err := validateStep(i, s); err != nil {
			return err
		}
	}
	return nil
}

var validErrorPolicies = map[ErrorPolicy]bool{
	ErrStop:     true,
	ErrContinue: true,
	ErrRetry:    true,
}

var validInputModes = map[InputMode]bool{
	InputTemplate: true,
	InputLiteral:  true,
}

var validTargetStrategies = map[TargetStrategy]bool{
	TargetXPath: true,
}

func validateStep(idx int, s Step) error {
	if s.ID == "" {
		return fmt.Errorf("step %d: id is required", idx)
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("step %d: name is required", idx)
	}
	switch s.Type {
	case StepNavigate, StepClick, StepInput, StepClearAndInput,
		StepWaitPresent, StepWaitVisible, StepWaitFixed,
		StepGetText, StepAssertExists, StepAssertText, StepScreenshot:
		// ok
	default:
		return fmt.Errorf("step %d: unknown type %s", idx, s.Type)
	}
	needsTarget := NeedsElement(s.Type)
	if needsTarget && strings.TrimSpace(s.Target.Value) == "" {
		return fmt.Errorf("step %d (%s): target value is required", idx, s.Name)
	}
	if s.Type == StepNavigate && strings.TrimSpace(s.Target.Value) == "" {
		return fmt.Errorf("step %d (%s): navigate URL is required", idx, s.Name)
	}
	if s.WaitBeforeMs < 0 {
		return fmt.Errorf("step %d: waitBeforeMs must be >= 0", idx)
	}
	if s.WaitAfterMs < 0 {
		return fmt.Errorf("step %d: waitAfterMs must be >= 0", idx)
	}
	if s.TimeoutMs < 0 {
		return fmt.Errorf("step %d: timeoutMs must be >= 0", idx)
	}
	if s.OnError != "" && !validErrorPolicies[s.OnError] {
		return fmt.Errorf("step %d: unknown onError policy %q (valid: stop, continue, retry)", idx, s.OnError)
	}
	if s.Input.Mode != "" && !validInputModes[s.Input.Mode] {
		return fmt.Errorf("step %d: unknown input.mode %q (valid: template, literal)", idx, s.Input.Mode)
	}
	if s.Target.Strategy != "" && !validTargetStrategies[s.Target.Strategy] {
		return fmt.Errorf("step %d: unknown target.strategy %q (valid: xpath)", idx, s.Target.Strategy)
	}
	return nil
}

func NeedsElement(t StepType) bool {
	switch t {
	case StepClick, StepInput, StepClearAndInput, StepWaitPresent,
		StepWaitVisible, StepGetText, StepAssertExists, StepAssertText:
		return true
	}
	return false
}
