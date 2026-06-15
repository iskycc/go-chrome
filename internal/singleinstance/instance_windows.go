//go:build windows

package singleinstance

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

const mutexName = `Local\go-chrome-single-instance`

type windowsInstance struct {
	listener    net.Listener
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	connWg      sync.WaitGroup
	portFile    string
	mutexHandle windows.Handle
}

func TryStart(ctx context.Context, req RunRequest, h Handler) (Result, *Instance, error) {
	handle, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			sent, forwardErr := forwardRequest(ctx, req)
			if sent {
				return ResultSent, nil, nil
			}
			return ResultFallback, nil, forwardErr
		}
		return ResultFallback, nil, fmt.Errorf("create mutex: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		windows.CloseHandle(handle)
		return ResultFallback, nil, fmt.Errorf("listen: %w", err)
	}

	portFile := portFilePath()
	if err := os.MkdirAll(filepath.Dir(portFile), 0755); err != nil {
		listener.Close()
		windows.CloseHandle(handle)
		return ResultFallback, nil, fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)), 0644); err != nil {
		listener.Close()
		windows.CloseHandle(handle)
		return ResultFallback, nil, fmt.Errorf("write port file: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	wi := &windowsInstance{listener: listener, cancel: cancel, portFile: portFile, mutexHandle: handle}
	wi.wg.Add(1)
	go wi.serve(ctx, h)

	return ResultStarted, &Instance{shutdown: func() { wi.shutdown() }}, nil
}

func (wi *windowsInstance) serve(ctx context.Context, h Handler) {
	defer wi.wg.Done()
	for {
		conn, err := wi.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		wi.connWg.Add(1)
		go wi.handleConn(conn, h)
	}
}

func (wi *windowsInstance) handleConn(conn net.Conn, h Handler) {
	defer wi.connWg.Done()
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return
	}
	var req RunRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return
	}
	if h != nil {
		h(req)
	}
}

func (wi *windowsInstance) shutdown() {
	wi.cancel()
	_ = wi.listener.Close()
	wi.wg.Wait()
	wi.connWg.Wait()
	_ = os.Remove(wi.portFile)
	if wi.mutexHandle != 0 {
		windows.CloseHandle(wi.mutexHandle)
		wi.mutexHandle = 0
	}
}

func forwardRequest(ctx context.Context, req RunRequest) (bool, error) {
	port, err := readPortFile()
	if err != nil {
		return false, err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	data, err := json.Marshal(req)
	if err != nil {
		return false, err
	}
	_, err = fmt.Fprintf(conn, "%s\n", data)
	if err != nil {
		return false, err
	}
	return true, nil
}
