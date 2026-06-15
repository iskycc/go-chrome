//go:build !windows

package shortcut

import "fmt"

func Create(opts Options) error {
	return fmt.Errorf("creating Windows shortcuts is only supported on Windows")
}
