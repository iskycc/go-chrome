//go:build !windows

package singleinstance

import "context"

func TryStart(ctx context.Context, req RunRequest, h Handler) (Result, *Instance, error) {
	return ResultStarted, &Instance{}, nil
}
