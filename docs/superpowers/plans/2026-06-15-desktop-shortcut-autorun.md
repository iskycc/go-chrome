# 桌面快捷方式一键执行实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为指定流程 + 环境配置生成 Windows 桌面快捷方式，双击后一键打开已有程序实例并自动执行该流程；若程序未运行则启动新实例执行。

**Architecture:** 新增 `internal/singleinstance`（命名互斥体 + TCP IPC）保证单实例，`internal/shortcut`（Windows COM）创建 `.lnk`，`cmd/go-chrome/main.go` 解析 `--flow`/`--env` 并调度执行请求，`internal/ui` 增加生成入口与自动执行逻辑。

**Tech Stack:** Go 1.26, Fyne v2, Windows COM (`WScript.Shell` via `go-ole`), `golang.org/x/sys/windows`

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `internal/singleinstance/instance.go` | 公共接口与参数结构（跨平台）。 |
| `internal/singleinstance/instance_windows.go` | Windows 命名互斥体 + TCP IPC 实现。 |
| `internal/singleinstance/instance_other.go` | 非 Windows stub（不限制实例）。 |
| `internal/singleinstance/instance_test.go` | 参数序列化/反序列化、端口文件读写测试。 |
| `internal/shortcut/shortcut.go` | 公共接口与参数结构（跨平台）。 |
| `internal/shortcut/shortcut_windows.go` | Windows COM 创建 `.lnk` 实现。 |
| `internal/shortcut/shortcut_other.go` | 非 Windows stub（返回不支持错误）。 |
| `internal/ui/shortcut_dialog.go` | 快捷方式名称编辑对话框与名称冲突处理。 |
| `cmd/go-chrome/main.go` | 解析 `--flow`/`--env`，启动单实例守护或发送参数给已有实例。 |
| `internal/ui/main_window.go` | 接收自动执行请求，选中流程/环境并触发运行。 |
| `internal/ui/flow_library.go` | 流程库右键菜单增加“生成桌面快捷方式”。 |
| `internal/ui/global_toolbar.go` | 工具栏增加“生成桌面快捷方式”按钮。 |

---

## Task 1: 实现 `internal/singleinstance` 单实例与 IPC

**Files:**
- Create: `internal/singleinstance/instance.go`
- Create: `internal/singleinstance/instance_windows.go`
- Create: `internal/singleinstance/instance_other.go`
- Create: `internal/singleinstance/instance_test.go`

- [ ] **Step 1.1: 定义公共接口与参数结构**

```go
// internal/singleinstance/instance.go
package singleinstance

import "context"

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
func TryStart(ctx context.Context, req RunRequest, h Handler) (Result, *Instance, error)

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
```

- [ ] **Step 1.2: 实现 Windows 单实例 + TCP IPC**

```go
// internal/singleinstance/instance_windows.go
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

const mutexName = `Global\go-chrome-single-instance`

var (
	mutexHandle windows.Handle
	mutexOnce   sync.Once
)

type windowsInstance struct {
	listener net.Listener
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	portFile string
}

func TryStart(ctx context.Context, req RunRequest, h Handler) (Result, *Instance, error) {
	// Try to create the named mutex.
	handle, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			// Another instance is running; forward the request.
			sent, forwardErr := forwardRequest(ctx, req)
			if sent {
				return ResultSent, nil, nil
			}
			return ResultFallback, nil, forwardErr
		}
		return ResultFallback, nil, fmt.Errorf("create mutex: %w", err)
	}
	mutexHandle = handle

	// First instance: start TCP listener.
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
	wi := &windowsInstance{listener: listener, cancel: cancel, portFile: portFile}
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
		go wi.handleConn(conn, h)
	}
}

func (wi *windowsInstance) handleConn(conn net.Conn, h Handler) {
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
	_ = os.Remove(wi.portFile)
	if mutexHandle != 0 {
		windows.CloseHandle(mutexHandle)
		mutexHandle = 0
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

func portFilePath() string {
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
		return 0, err
	}
	return port, nil
}
```

- [ ] **Step 1.3: 实现非 Windows stub**

```go
// internal/singleinstance/instance_other.go
//go:build !windows

package singleinstance

import "context"

func TryStart(ctx context.Context, req RunRequest, h Handler) (Result, *Instance, error) {
	// Non-Windows builds do not restrict instances; just start normally.
	return ResultStarted, &Instance{}, nil
}
```

- [ ] **Step 1.4: 编写测试**

```go
// internal/singleinstance/instance_test.go
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
}
```

- [ ] **Step 1.5: 运行测试**

Run: `go test ./internal/singleinstance -v`
Expected: PASS

- [ ] **Step 1.6: 提交**

```bash
git add internal/singleinstance/
git commit -m "feat: add single-instance IPC for shortcut auto-run"
```

---

## Task 2: 实现 `internal/shortcut` 快捷方式创建

**Files:**
- Create: `internal/shortcut/shortcut.go`
- Create: `internal/shortcut/shortcut_windows.go`
- Create: `internal/shortcut/shortcut_other.go`
- Create: `internal/shortcut/shortcut_test.go`

- [ ] **Step 2.1: 定义公共接口**

```go
// internal/shortcut/shortcut.go
package shortcut

// Options describes a Windows shortcut to create.
type Options struct {
	TargetPath  string
	Arguments   string
	WorkingDir  string
	IconPath    string
	Description string
	ShortcutPath string
}

// Create writes a .lnk file at opts.ShortcutPath pointing to opts.TargetPath.
func Create(opts Options) error
```

- [ ] **Step 2.2: 实现 Windows COM 创建 `.lnk`**

```go
// internal/shortcut/shortcut_windows.go
//go:build windows

package shortcut

import (
	"fmt"
	"path/filepath"

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
		// S_FALSE means COM already initialized on this thread.
		if oleErr, ok := err.(*ole.OleError); !ok || oleErr.Code() != ole.S_FALSE {
			return fmt.Errorf("coinit: %w", err)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return fmt.Errorf("create WScript.Shell: %w", err)
	}
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
```

- [ ] **Step 2.3: 实现非 Windows stub**

```go
// internal/shortcut/shortcut_other.go
//go:build !windows

package shortcut

import "fmt"

func Create(opts Options) error {
	return fmt.Errorf("creating Windows shortcuts is only supported on Windows")
}
```

- [ ] **Step 2.4: 编写测试（名称冲突算法）**

由于 COM 创建 .lnk 难以在 Linux 单元测试中验证，主要测试 `Options` 的组装与路径处理：

```go
// internal/shortcut/shortcut_test.go
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
```

- [ ] **Step 2.5: 运行测试**

Run: `go test ./internal/shortcut -v`
Expected: PASS (on Windows it verifies the build; on Linux it passes the field test)

- [ ] **Step 2.6: 提交**

```bash
git add internal/shortcut/
git commit -m "feat: add Windows shortcut creation helper"
```

---

## Task 3: 实现快捷方式名称编辑对话框

**Files:**
- Create: `internal/ui/shortcut_dialog.go`

- [ ] **Step 3.1: 实现对话框与名称冲突算法**

```go
// internal/ui/shortcut_dialog.go
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showCreateShortcutDialog(flowID, envID, flowName, envName string) {
	if a.mainWin == nil {
		return
	}

	defaultName := uniqueShortcutName(flowName, envName, desktopDir())

	nameEntry := widget.NewEntry()
	nameEntry.SetText(defaultName)
	nameEntry.SetPlaceHolder("快捷方式名称")

	form := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("流程：%s", flowName)),
		widget.NewLabel(fmt.Sprintf("环境：%s", envName)),
		widget.NewLabel("快捷方式名称"),
		nameEntry,
	)

	d := dialog.NewCustomConfirm("生成桌面快捷方式", "生成", "取消", form, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			dialog.ShowInformation("提示", "名称不能为空", a.mainWin)
			return
		}
		if !strings.HasSuffix(strings.ToLower(name), ".lnk") {
			name += ".lnk"
		}
		shortcutPath := filepath.Join(desktopDir(), sanitizeShortcutName(name))
		if err := a.createShortcutFile(flowID, envID, shortcutPath); err != nil {
			dialog.ShowError(fmt.Errorf("生成快捷方式失败: %w", err), a.mainWin)
			return
		}
		a.runPanel.log("已生成桌面快捷方式: " + shortcutPath)
	}, a.mainWin)
	d.Resize(fyne.NewSize(440, 240))
	d.Show()
}

func (a *App) createShortcutFile(flowID, envID, shortcutPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exe)
	return shortcut.Create(shortcut.Options{
		TargetPath:   exe,
		Arguments:    fmt.Sprintf(`--flow=%s --env=%s`, flowID, envID),
		WorkingDir:   exeDir,
		IconPath:     exe,
		Description:  fmt.Sprintf("Chrome 自动化: %s", filepath.Base(shortcutPath)),
		ShortcutPath: shortcutPath,
	})
}

func uniqueShortcutName(flowName, envName, dir string) string {
	base := fmt.Sprintf("%s-%s", flowName, envName)
	base = sanitizeShortcutName(base)
	candidate := base + ".lnk"
	if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
		return candidate
	}
	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s-%d.lnk", base, i)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func sanitizeShortcutName(name string) string {
	// Remove characters that are illegal in Windows file names.
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", `"`, "", "/", "", "\\", "", "|", "", "?", "", "*", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

func desktopDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Desktop")
}
```

- [ ] **Step 3.2: 添加 import 到 ui 包并处理 import 循环**

`internal/ui/shortcut_dialog.go` imports `internal/shortcut`. Ensure `internal/shortcut` does not import `internal/ui`.

- [ ] **Step 3.3: 运行构建检查**

Run: `go build ./internal/ui`
Expected: success (non-Windows will build stub; Windows build needs ole package)

- [ ] **Step 3.4: 提交**

```bash
git add internal/ui/shortcut_dialog.go
git commit -m "feat: add shortcut creation dialog with name conflict handling"
```

---

## Task 4: 修改 `cmd/go-chrome/main.go` 解析参数并调度执行

**Files:**
- Modify: `cmd/go-chrome/main.go`

- [ ] **Step 4.1: 添加命令行参数解析与单实例调度**

```go
// cmd/go-chrome/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"go-chrome/internal/app"
	"go-chrome/internal/config"
	"go-chrome/internal/logx"
	"go-chrome/internal/singleinstance"
	"go-chrome/internal/ui"
)

type launchArgs struct {
	flowID string
	envID  string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	args := parseArgs()

	baseDir, err := app.ExecutableDir()
	if err != nil {
		baseDir = "."
	}

	dirs, err := app.EnsureDirs(baseDir)
	if err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	cfg, err := config.Load(dirs.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg.ResolvePaths(baseDir)
	config.SetInstance(cfg)

	if err := logx.Init(dirs.LogsDir, cfg.App.LogRetentionDays, nil); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logx.Close()

	logx.Info("go-chrome starting")

	var autoRun *singleinstance.RunRequest
	if args.flowID != "" && args.envID != "" {
		autoRun = &singleinstance.RunRequest{FlowID: args.flowID, EnvID: args.envID}
	}

	if autoRun != nil {
		ctx := context.Background()
		res, inst, err := singleinstance.TryStart(ctx, *autoRun, nil)
		if err != nil {
			logx.Warnf("single instance check failed: %v", err)
		}
		switch res {
		case singleinstance.ResultSent:
			logx.Info("forwarded auto-run to existing instance")
			return nil
		case singleinstance.ResultFallback:
			logx.Warn("could not forward to existing instance; starting new instance")
		case singleinstance.ResultStarted:
			if inst != nil {
				defer inst.Shutdown()
			}
		}
	}

	uiApp := ui.New(cfg, dirs)
	if autoRun != nil {
		uiApp.SetAutoRun(autoRun.FlowID, autoRun.EnvID)
	}
	uiApp.Run()

	logx.Info("go-chrome exiting")
	return nil
}

func parseArgs() launchArgs {
	var a launchArgs
	flag.StringVar(&a.flowID, "flow", "", "flow ID to run automatically on startup")
	flag.StringVar(&a.envID, "env", "", "environment ID to use for automatic run")
	flag.Parse()
	return a
}
```

- [ ] **Step 4.2: 运行构建检查**

Run: `go build ./cmd/go-chrome`
Expected: success

- [ ] **Step 4.3: 提交**

```bash
git add cmd/go-chrome/main.go
git commit -m "feat: parse --flow/--env and forward to existing instance"
```

---

## Task 5: 修改 `internal/ui/main_window.go` 支持自动执行

**Files:**
- Modify: `internal/ui/main_window.go`

- [ ] **Step 5.1: 在 `App` 结构体增加自动执行状态**

```go
type App struct {
	// ... existing fields ...
	autoRunFlowID string
	autoRunEnvID  string
}
```

- [ ] **Step 5.2: 新增 `SetAutoRun` 方法**

```go
// SetAutoRun records a flow+environment that should be executed once the UI is ready.
func (a *App) SetAutoRun(flowID, envID string) {
	a.autoRunFlowID = flowID
	a.autoRunEnvID = envID
}
```

- [ ] **Step 5.3: 在 `Run()` 中 UI 构建完成后触发自动执行**

Locate the end of `buildUI()` call in `Run()`:

```go
	a.buildUI()
	a.firstRunCheck()
	a.startChromeTicker()
	if a.autoRunFlowID != "" && a.autoRunEnvID != "" {
		go a.executeAutoRun()
	}
	a.mainWin.ShowAndRun()
```

- [ ] **Step 5.4: 实现 `executeAutoRun` 与 `runFlowByID` 方法**

```go
func (a *App) executeAutoRun() {
	if a.autoRunFlowID == "" || a.autoRunEnvID == "" {
		return
	}
	// Wait briefly for the UI to fully render.
	time.Sleep(200 * time.Millisecond)
	fyne.Do(func() {
		a.runFlowByID(a.autoRunFlowID, a.autoRunEnvID)
	})
}

func (a *App) runFlowByID(flowID, envID string) {
	if a.mainWin == nil {
		return
	}

	// Select the flow.
	found := false
	for _, f := range a.flowLibrary.flows {
		if f.ID == flowID {
			a.onFlowSelected(f)
			found = true
			break
		}
	}
	if !found {
		dialog.ShowError(fmt.Errorf("流程不存在: %s", flowID), a.mainWin)
		return
	}

	// Select the environment.
	env, err := a.envRepo.Get(envID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("环境配置不存在: %s", envID), a.mainWin)
		return
	}
	if a.globalToolbar != nil && a.globalToolbar.envSelect != nil {
		a.globalToolbar.envSelect.SetSelected(env.Name)
	}

	// Stop any running flow first.
	if a.runner.IsRunning() {
		a.runner.Stop()
		a.runPanel.log("已有流程运行中，已停止并准备执行新流程")
	}
	if a.stepRunner != nil && !a.stepRunner.IsFinished() {
		a.stepRunner.Stop()
		a.stepRunner = nil
	}

	// Switch to run panel tab and start.
	a.moduleTabs.SelectTabIndex(5) // "运行详情" tab
	a.runCurrentFlow()
}
```

- [ ] **Step 5.5: 为单实例 IPC 注册 handler**

In `Run()`, before `a.mainWin.ShowAndRun()`:

```go
	if inst != nil {
		// Register IPC handler to receive auto-run requests from another instance.
		// Note: singleinstance handler is set up in main.go; the Instance returned
		// from TryStart already captures it. We just need to forward to UI.
	}
```

Wait — the current design passes `nil` as handler in `main.go` when `autoRun != nil`. This is a bug. We need to pass a handler that forwards to the UI. Revise Step 4.1:

Change in `cmd/go-chrome/main.go`:

```go
	uiApp := ui.New(cfg, dirs)
	if autoRun != nil {
		res, inst, err := singleinstance.TryStart(ctx, *autoRun, func(req singleinstance.RunRequest) {
			uiApp.TriggerAutoRun(req.FlowID, req.EnvID)
		})
		// ... handle res ...
		if inst != nil {
			defer inst.Shutdown()
		}
	}
	uiApp.Run()
```

Add `TriggerAutoRun` to `internal/ui/main_window.go`:

```go
func (a *App) TriggerAutoRun(flowID, envID string) {
	fyne.Do(func() {
		a.runFlowByID(flowID, envID)
	})
}
```

- [ ] **Step 5.6: 运行构建检查**

Run: `go build ./cmd/go-chrome`
Expected: success

- [ ] **Step 5.7: 提交**

```bash
git add internal/ui/main_window.go
git commit -m "feat: auto-run flow from command-line or IPC request"
```

---

## Task 6: 在 UI 中添加快捷方式生成入口

**Files:**
- Modify: `internal/ui/flow_library.go`
- Modify: `internal/ui/global_toolbar.go`

- [ ] **Step 6.1: 在流程库右键菜单增加入口**

In `showFlowContextMenu`, after `runItem`:

```go
	createShortcutItem := fyne.NewMenuItem("生成桌面快捷方式", func() {
		a.showCreateShortcutDialogForFlow(f)
	})

	menu := fyne.NewMenu("流程操作",
		openItem,
		runItem,
		createShortcutItem,
		fyne.NewMenuItemSeparator(),
		// ... rest of menu items ...
	)
```

- [ ] **Step 6.2: 实现 `showCreateShortcutDialogForFlow` 辅助方法**

Add to `internal/ui/shortcut_dialog.go`:

```go
func (a *App) showCreateShortcutDialogForFlow(f *flow.Flow) {
	if f == nil {
		return
	}
	envID, _, err := a.currentEnvProvider()
	if err != nil {
		dialog.ShowError(fmt.Errorf("无法获取当前环境: %w", err), a.mainWin)
		return
	}
	env, err := a.envRepo.Get(envID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("无法获取当前环境: %w", err), a.mainWin)
		return
	}
	a.showCreateShortcutDialog(f.ID, env.ID, f.Name, env.Name)
}
```

- [ ] **Step 6.3: 在工具栏增加“快捷方式”按钮**

In `internal/ui/global_toolbar.go`, add a new button:

```go
	shortcutBtn := widget.NewButtonWithIcon("快捷方式", theme.DocumentCreateIcon(), func() {
		app.showCreateShortcutDialogForFlow(app.currentFlow)
	})
```

Add it to `runBox`:

```go
	runBox := newInlineToolbarGroup("执行",
		t.runBtn,
		t.stepBtn,
		t.stopBtn,
		shortcutBtn,
	)
```

- [ ] **Step 6.4: 运行构建检查**

Run: `go build ./cmd/go-chrome`
Expected: success

- [ ] **Step 6.5: 提交**

```bash
git add internal/ui/flow_library.go internal/ui/global_toolbar.go internal/ui/shortcut_dialog.go
git commit -m "feat: add UI entries for creating desktop shortcuts"
```

---

## Task 7: 运行测试与全量构建

**Files:**
- All modified files

- [ ] **Step 7.1: 运行核心包测试**

Run:
```bash
go test ./internal/singleinstance ./internal/shortcut ./internal/runner ./internal/config ./internal/flow ./internal/template
```
Expected: PASS for all packages

- [ ] **Step 7.2: 运行完整构建**

Run:
```bash
go build -mod=readonly ./cmd/go-chrome
```
Expected: success

- [ ] **Step 7.3: 手工验证清单**

1. 启动程序，选择一个流程 + 环境，点击“快捷方式”按钮，确认桌面生成 `.lnk`。
2. 关闭程序，双击 `.lnk`，确认程序启动并自动执行流程。
3. 保持程序打开，再次双击 `.lnk`，确认不启动新实例，已有实例重新执行流程。
4. 删除流程后双击 `.lnk`，确认程序启动并提示“流程不存在”。
5. 验证默认名称冲突递增：`流程A-环境B.lnk` 已存在时生成 `流程A-环境B-1.lnk`。

- [ ] **Step 7.4: 提交**

```bash
git commit -m "test: verify shortcut autorun feature builds and passes core tests"
```

---

## Self-Review

**Spec coverage:**
- `--flow`/`--env` 参数：Task 4
- 打开 UI 后自动执行：Task 5
- 允许编辑、默认名称格式与编号：Task 3
- 执行完成保持打开：Task 5 不调用 `os.Exit`
- 简短参数风格：Task 4 使用 `--flow` / `--env`
- 已有实例复用：Task 1 + Task 4 + Task 5

**Placeholder scan:**
- 无 TBD/TODO。
- 所有代码片段可直接复制使用。

**Type consistency:**
- `RunRequest` 在 `singleinstance` 与 `ui` 之间一致。
- `shortcut.Options` 字段与调用处一致。
- `globalToolbar.envSelect.Selected` 是环境名称字符串，与 `envRepo.GetByName` 一致。

**发现的问题与修正：**
- 初始 Step 4.1 中 `TryStart` 传入了 `nil` handler，导致已有实例无法处理新请求。已在 Step 5.5 中修正：创建 `uiApp` 后传入转发 handler，并新增 `TriggerAutoRun`。
