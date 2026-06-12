package flow

import (
	"time"

	"github.com/google/uuid"
)

// Flow represents an automation flow.
type Flow struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Tags          []string  `json:"tags"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Steps         []Step    `json:"steps"`
}

// Step represents a single automation step.
type Step struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Enabled      bool        `json:"enabled"`
	Type         StepType    `json:"type"`
	Target       Target      `json:"target"`
	Input        Input       `json:"input"`
	WaitBeforeMs int         `json:"waitBeforeMs"`
	WaitAfterMs  int         `json:"waitAfterMs"`
	TimeoutMs    int         `json:"timeoutMs"`
	OnError      ErrorPolicy `json:"onError"`
	Note         string      `json:"note"`
}

// StepType enumerates supported step types.
type StepType string

const (
	StepNavigate      StepType = "navigate"
	StepClick         StepType = "click"
	StepInput         StepType = "input"
	StepClearAndInput StepType = "clear_and_input"
	StepWaitPresent   StepType = "wait_present"
	StepWaitVisible   StepType = "wait_visible"
	StepWaitFixed     StepType = "wait_fixed"
	StepGetText       StepType = "get_text"
	StepAssertExists  StepType = "assert_exists"
	StepAssertText    StepType = "assert_text"
	StepScreenshot    StepType = "screenshot"
)

// Target defines how to locate an element.
type Target struct {
	Strategy TargetStrategy `json:"strategy"`
	Value    string         `json:"value"`
}

// TargetStrategy enumerates target strategies.
type TargetStrategy string

const (
	TargetXPath TargetStrategy = "xpath"
)

// Input holds input data for a step.
type Input struct {
	Mode       InputMode `json:"mode"`       // template, literal
	Text       string    `json:"text"`       // raw template or literal text
	MaskInLogs bool      `json:"maskInLogs"` // if true, mask generated value in logs
}

// InputMode enumerates input modes.
type InputMode string

const (
	InputTemplate InputMode = "template"
	InputLiteral  InputMode = "literal"
)

// ErrorPolicy defines behavior on step failure.
type ErrorPolicy string

const (
	ErrStop     ErrorPolicy = "stop"
	ErrContinue ErrorPolicy = "continue"
	ErrRetry    ErrorPolicy = "retry"
)

// NewFlow creates a new flow with defaults.
func NewFlow(name string) *Flow {
	now := time.Now()
	return &Flow{
		SchemaVersion: 1,
		ID:            uuid.New().String(),
		Name:          name,
		Tags:          []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
		Steps:         []Step{},
	}
}

// NewStep creates a new step with defaults.
func NewStep(name string, t StepType) Step {
	return Step{
		ID:           uuid.New().String(),
		Name:         name,
		Enabled:      true,
		Type:         t,
		WaitBeforeMs: 0,
		WaitAfterMs:  500,
		TimeoutMs:    10000,
		OnError:      ErrStop,
	}
}

// NewExampleLoginFlow creates a sample flow for onboarding.
func NewExampleLoginFlow() *Flow {
	f := NewFlow("示例：登录测试")
	f.Tags = []string{"示例"}
	f.Steps = []Step{
		NewStep("打开网址", StepNavigate),
		NewStep("输入用户名", StepInput),
		NewStep("输入密码", StepInput),
		NewStep("点击登录", StepClick),
		NewStep("断言欢迎文本", StepAssertExists),
		NewStep("页面截图", StepScreenshot),
	}
	f.Steps[0].Target = Target{Strategy: TargetXPath, Value: "http://localhost:18080"}
	f.Steps[1].Target = Target{Strategy: TargetXPath, Value: "//input[@id='username']"}
	f.Steps[1].Input = Input{Mode: InputTemplate, Text: "${var:user=SP${11000-11099}}"}
	f.Steps[2].Target = Target{Strategy: TargetXPath, Value: "//input[@id='password']"}
	f.Steps[2].Input = Input{Mode: InputLiteral, Text: "Password123"}
	f.Steps[3].Target = Target{Strategy: TargetXPath, Value: "//button[@type='submit']"}
	f.Steps[4].Target = Target{Strategy: TargetXPath, Value: "//div[contains(text(),'欢迎')]"}
	return f
}

// Clone creates a deep copy of the flow with new IDs.
func (f *Flow) Clone() *Flow {
	now := time.Now()
	cf := &Flow{
		SchemaVersion: f.SchemaVersion,
		ID:            uuid.New().String(),
		Name:          f.Name + " (Copy)",
		Description:   f.Description,
		Tags:          append([]string{}, f.Tags...),
		CreatedAt:     now,
		UpdatedAt:     now,
		Steps:         make([]Step, len(f.Steps)),
	}
	for i, s := range f.Steps {
		cf.Steps[i] = s
		cf.Steps[i].ID = uuid.New().String()
	}
	return cf
}
