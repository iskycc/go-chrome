//go:build windows

package shortcut

import (
	"fmt"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func Create(opts Options) error {
	if opts.ShortcutPath == "" {
		return fmt.Errorf("shortcut path is required")
	}
	if opts.TargetPath == "" {
		return fmt.Errorf("target path is required")
	}

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if oleErr, ok := err.(*ole.OleError); !ok || oleErr.Code() != ole.S_FALSE {
			return fmt.Errorf("coinit: %w", err)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return fmt.Errorf("create WScript.Shell: %w", err)
	}
	defer unknown.Release()

	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query interface: %w", err)
	}
	defer shell.Release()

	shortcut, err := oleutil.CallMethod(shell, "CreateShortcut", opts.ShortcutPath)
	if err != nil {
		return fmt.Errorf("create shortcut: %w", err)
	}
	sc := shortcut.ToIDispatch()
	defer sc.Release()

	if _, err := oleutil.PutProperty(sc, "TargetPath", opts.TargetPath); err != nil {
		return fmt.Errorf("set target: %w", err)
	}
	if opts.Arguments != "" {
		if _, err := oleutil.PutProperty(sc, "Arguments", opts.Arguments); err != nil {
			return fmt.Errorf("set arguments: %w", err)
		}
	}
	if opts.WorkingDir != "" {
		if _, err := oleutil.PutProperty(sc, "WorkingDirectory", opts.WorkingDir); err != nil {
			return fmt.Errorf("set working dir: %w", err)
		}
	}
	if opts.IconPath != "" {
		icon := opts.IconPath + ",0"
		if _, err := oleutil.PutProperty(sc, "IconLocation", icon); err != nil {
			return fmt.Errorf("set icon: %w", err)
		}
	}
	if opts.Description != "" {
		if _, err := oleutil.PutProperty(sc, "Description", opts.Description); err != nil {
			return fmt.Errorf("set description: %w", err)
		}
	}
	if _, err := oleutil.CallMethod(sc, "Save"); err != nil {
		return fmt.Errorf("save shortcut: %w", err)
	}
	return nil
}
