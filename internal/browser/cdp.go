package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// CDPClient wraps a chromedp context.
type CDPClient struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type devtoolsTarget struct {
	ID   target.ID `json:"id"`
	URL  string    `json:"url"`
	Type string    `json:"type"`
}

// Connect connects to an existing Chrome debugging port.
func Connect(port int) (*CDPClient, error) {
	var lastErr error
	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	for i := 0; i < 8; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
		}
		targetID, err := ensurePageTarget(client, baseURL)
		if err != nil {
			lastErr = fmt.Errorf("cdp target discovery failed: %w", err)
			continue
		}
		allocatorCtx, allocatorCancel := chromedp.NewRemoteAllocator(context.Background(), baseURL)
		ctx, ctxCancel := chromedp.NewContext(allocatorCtx, chromedp.WithTargetID(targetID))
		var title string
		cdpErr := runWithTimeout(ctx, 10*time.Second, chromedp.Title(&title))
		if cdpErr == nil {
			cancel := func() {
				ctxCancel()
				allocatorCancel()
			}
			return &CDPClient{ctx: ctx, cancel: cancel}, nil
		}
		lastErr = fmt.Errorf("cdp connect failed: %w", cdpErr)
		ctxCancel()
		allocatorCancel()
		if cdpErr != nil && (cdpErr.Error() == "Failed to open new tab - no browser is open (-32000)" ||
			containsErrCode(cdpErr, -32000)) {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func ensurePageTarget(client *http.Client, baseURL string) (target.ID, error) {
	targets, err := listTargets(client, baseURL)
	if err != nil {
		return "", err
	}
	if id, ok := selectPageTarget(targets); ok {
		return id, nil
	}
	return createPageTarget(client, baseURL)
}

func listTargets(client *http.Client, baseURL string) ([]devtoolsTarget, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/json/list", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GET /json/list returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var targets []devtoolsTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("decode /json/list: %w", err)
	}
	return targets, nil
}

func selectPageTarget(targets []devtoolsTarget) (target.ID, bool) {
	for _, t := range targets {
		if t.Type == "page" && t.ID != "" {
			return t.ID, true
		}
	}
	return "", false
}

func createPageTarget(client *http.Client, baseURL string) (target.ID, error) {
	req, err := http.NewRequest(http.MethodPut, baseURL+"/json/new?about:blank", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("PUT /json/new returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var t devtoolsTarget
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", fmt.Errorf("decode /json/new: %w", err)
	}
	if t.ID == "" {
		return "", fmt.Errorf("PUT /json/new returned empty target id")
	}
	if t.Type != "" && t.Type != "page" {
		return "", fmt.Errorf("PUT /json/new returned non-page target %q", t.Type)
	}
	return t.ID, nil
}

func runWithTimeout(ctx context.Context, timeout time.Duration, actions ...chromedp.Action) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- chromedp.Run(ctx, actions...)
	}()
	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout after %s", timeout)
	}
}

func containsErrCode(err error, code int) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	codeStr := fmt.Sprintf("(%d)", code)
	return strings.Contains(s, codeStr)
}

// Context returns the chromedp context.
func (c *CDPClient) Context() context.Context {
	return c.ctx
}

// Close closes the CDP connection.
func (c *CDPClient) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

// WithTimeout returns a context with timeout derived from the CDP context.
func (c *CDPClient) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ctx, timeout)
}
