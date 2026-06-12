package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// CDPClient wraps a chromedp context.
type CDPClient struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Connect connects to an existing Chrome debugging port.
func Connect(port int) (*CDPClient, error) {
	var lastErr error
	for i := 0; i < 8; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
		}
		allocatorCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), fmt.Sprintf("http://127.0.0.1:%d", port))
		ctx, _ := chromedp.NewContext(allocatorCtx)
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 10*time.Second)
		var title string
		cdpErr := chromedp.Run(timeoutCtx, chromedp.Title(&title))
		if cdpErr == nil {
			timeoutCancel()
			return &CDPClient{ctx: ctx, cancel: cancel}, nil
		}
		lastErr = fmt.Errorf("cdp connect failed: %w", cdpErr)
		timeoutCancel()
		cancel()
		if cdpErr != nil && (cdpErr.Error() == "Failed to open new tab - no browser is open (-32000)" ||
			containsErrCode(cdpErr, -32000)) {
			return nil, lastErr
		}
	}
	return nil, lastErr
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
