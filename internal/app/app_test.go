package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirsCreatesRuntimeDirsWithoutLegacyFlowsDir(t *testing.T) {
	base := t.TempDir()
	dirs, err := EnsureDirs(base)
	if err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}

	for _, dir := range []string{dirs.DataDir, dirs.LogsDir, dirs.ChromeDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}

	if _, err := os.Stat(filepath.Join(base, "data", "flows")); !os.IsNotExist(err) {
		t.Fatalf("legacy data/flows dir should not be created, err=%v", err)
	}
}
