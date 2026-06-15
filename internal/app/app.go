package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// Overridable for testing.
var (
	executableFunc = os.Executable
	mkdirAllFunc   = os.MkdirAll
)

// Directories holds application directory paths.
type Directories struct {
	DataDir    string
	FlowsDir   string
	LogsDir    string
	ChromeDir  string
	ConfigPath string
}

// EnsureDirs creates all required application directories.
func EnsureDirs(base string) (*Directories, error) {
	dirs := &Directories{
		DataDir:    filepath.Join(base, "data"),
		FlowsDir:   filepath.Join(base, "data", "flows"),
		LogsDir:    filepath.Join(base, "logs"),
		ChromeDir:  filepath.Join(base, "chrome"),
		ConfigPath: filepath.Join(base, "data", "app-config.json"),
	}
	for _, d := range []string{dirs.DataDir, dirs.LogsDir, dirs.ChromeDir} {
		if err := mkdirAllFunc(d, 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	return dirs, nil
}

// ExecutableDir returns the directory of the current executable.
func ExecutableDir() (string, error) {
	ex, err := executableFunc()
	if err != nil {
		return "", err
	}
	return filepath.Dir(ex), nil
}
