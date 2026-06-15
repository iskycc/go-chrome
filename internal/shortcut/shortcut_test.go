package shortcut

import (
	"path/filepath"
	"testing"
)

func TestOptionsFields(t *testing.T) {
	opts := Options{
		TargetPath:   `C:\Program Files\go-chrome\go-chrome.exe`,
		Arguments:    `--flow=abc --env=def`,
		WorkingDir:   `C:\Program Files\go-chrome`,
		IconPath:     `C:\Program Files\go-chrome\go-chrome.exe`,
		Description:  "Login flow with production env",
		ShortcutPath: filepath.Join(t.TempDir(), "test.lnk"),
	}
	if opts.TargetPath == "" {
		t.Fatal("target path empty")
	}
	if opts.Arguments == "" {
		t.Fatal("arguments empty")
	}
}
