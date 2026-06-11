package browser

import (
	"context"
	"fmt"
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
	allocatorCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), fmt.Sprintf("http://127.0.0.1:%d", port))
	ctx, _ := chromedp.NewContext(allocatorCtx)
	// Verify connection with a short timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
	defer timeoutCancel()
	var title string
	if err := chromedp.Run(timeoutCtx, chromedp.Title(&title)); err != nil {
		cancel()
		return nil, fmt.Errorf("cdp connect failed: %w", err)
	}
	return &CDPClient{ctx: ctx, cancel: cancel}, nil
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
