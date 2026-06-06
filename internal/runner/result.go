package runner

import (
	"time"
)

// Status represents step execution status.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// StepResult holds the outcome of a single step.
type StepResult struct {
	StepID         string `json:"stepId"`
	StepName       string `json:"stepName"`
	Type           string `json:"type"`
	Status         Status `json:"status"`
	DurationMs     int64  `json:"durationMs"`
	Error          string `json:"error,omitempty"`
	Screenshot     string `json:"screenshot,omitempty"`
	HTMLSnapshot   string `json:"htmlSnapshot,omitempty"`
	GeneratedInput string `json:"generatedInput,omitempty"`
	MaskInLogs     bool   `json:"maskInLogs,omitempty"`
}

// RunResult holds the outcome of a full flow run.
type RunResult struct {
	ID           string       `json:"id"`
	FlowID       string       `json:"flowId"`
	FlowName     string       `json:"flowName"`
	StartedAt    time.Time    `json:"startedAt"`
	FinishedAt   time.Time    `json:"finishedAt"`
	Steps        []StepResult `json:"steps"`
	TotalSteps   int          `json:"totalSteps"`
	SuccessCount int          `json:"successCount"`
	FailedCount  int          `json:"failedCount"`
	SkippedCount int          `json:"skippedCount"`
	Status       Status       `json:"status"`
}

// Event is sent during execution.
type Event struct {
	Type       EventType   `json:"type"` // step_start, step_done, run_done, log
	StepIndex  int         `json:"stepIndex,omitempty"`
	StepName   string      `json:"stepName,omitempty"`
	Result     *StepResult `json:"result,omitempty"`
	RunResult  *RunResult  `json:"runResult,omitempty"`
	LogMessage string      `json:"logMessage,omitempty"`
}

// EventType enumerates event types.
type EventType string

const (
	EventStepStart EventType = "step_start"
	EventStepDone  EventType = "step_done"
	EventRunDone   EventType = "run_done"
	EventLog       EventType = "log"
)
