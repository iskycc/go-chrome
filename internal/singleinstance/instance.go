package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunRequest carries the auto-run parameters received from a shortcut.
type RunRequest struct {
	FlowID string `json:"flowID"`
	EnvID  string `json:"envID"`
}

// Result indicates what the new process should do after TryStart.
type Result int

const (
	// ResultStarted means this is the first instance and should keep running.
	ResultStarted Result = iota
	// ResultSent means another instance is running and we forwarded the request.
	ResultSent
	// ResultFallback means we could not contact the other instance; caller may start normally.
	ResultFallback
)

// Handler is called by the first instance when it receives a RunRequest.
type Handler func(req RunRequest)

// TryStart attempts to become the first instance. On success it returns
// ResultStarted and the caller must later call Instance.Shutdown(). On failure
// it tries to forward req to the running instance and returns ResultSent or
// ResultFallback.
//
// The implementation is platform-specific and is defined in the corresponding
// instance_*.go file.

// Instance represents the first-instance listener.
type Instance struct {
	shutdown func()
}

// Shutdown stops the listener.
func (i *Instance) Shutdown() {
	if i != nil && i.shutdown != nil {
		i.shutdown()
	}
}

// portFilePath is overridable for tests.
var portFilePath = defaultPortFilePath

func defaultPortFilePath() string {
	base := "."
	if ex, err := os.Executable(); err == nil {
		base = filepath.Dir(ex)
	}
	return filepath.Join(base, "data", "instance-port")
}

func readPortFile() (int, error) {
	data, err := os.ReadFile(portFilePath())
	if err != nil {
		return 0, err
	}
	var port int
	if _, err := fmt.Sscanf(string(data), "%d", &port); err != nil {
		return 0, fmt.Errorf("parse port file: %w", err)
	}
	return port, nil
}
