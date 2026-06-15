package singleinstance

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRequestSerialization(t *testing.T) {
	req := RunRequest{FlowID: "flow-1", EnvID: "env-2"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got RunRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != req {
		t.Fatalf("got %+v, want %+v", got, req)
	}
}

func TestPortFileReadWrite(t *testing.T) {
	dir := t.TempDir()
	orig := portFilePath
	portFilePath = func() string { return filepath.Join(dir, "instance-port") }
	defer func() { portFilePath = orig }()

	if err := os.WriteFile(portFilePath(), []byte("12345"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	port, err := readPortFile()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if port != 12345 {
		t.Fatalf("port = %d, want 12345", port)
	}

	// Missing file returns error.
	if err := os.Remove(portFilePath()); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := readPortFile(); err == nil {
		t.Fatal("expected error for missing port file")
	}

	// Invalid content returns error.
	if err := os.WriteFile(portFilePath(), []byte("not-a-port"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := readPortFile(); err == nil {
		t.Fatal("expected error for invalid port file content")
	}
}

func TestDefaultPortFilePath(t *testing.T) {
	path := defaultPortFilePath()
	if path == "" {
		t.Fatal("expected non-empty default path")
	}
}

func TestInstanceShutdown(t *testing.T) {
	var called bool
	inst := &Instance{shutdown: func() { called = true }}
	inst.Shutdown()
	if !called {
		t.Fatal("expected shutdown callback called")
	}

	// Nil receiver and nil shutdown are safe.
	var nilInst *Instance
	nilInst.Shutdown()
	(&Instance{}).Shutdown()
}

func TestTryStartOnNonWindows(t *testing.T) {
	res, inst, err := TryStart(context.Background(), RunRequest{FlowID: "f", EnvID: "e"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != ResultStarted {
		t.Fatalf("expected ResultStarted, got %d", res)
	}
	if inst == nil {
		t.Fatal("expected non-nil instance")
	}
	inst.Shutdown()
}
