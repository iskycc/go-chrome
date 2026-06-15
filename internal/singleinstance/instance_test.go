package singleinstance

import (
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
}
