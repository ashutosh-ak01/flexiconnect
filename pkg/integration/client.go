package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/sony/gobreaker"
)

// Client wraps an http.Client with retry policies and circuit breaker support.
type Client struct {
	httpClient *http.Client
}

// NewClient initializes a Client with the specified HTTP request timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Do dispatches the request, wrapping it in a circuit breaker and/or retry policy if configured.
func (c *Client) Do(ctx context.Context, req *http.Request, retryCfg *config.RetryConfig, cb *gobreaker.CircuitBreaker) (*http.Response, error) {
	if cb != nil {
		respVal, err := cb.Execute(func() (interface{}, error) {
			return c.executeWithRetry(ctx, req, retryCfg)
		})
		if err != nil {
			return nil, err
		}
		return respVal.(*http.Response), nil
	}

	return c.executeWithRetry(ctx, req, retryCfg)
}

func (c *Client) executeWithRetry(ctx context.Context, req *http.Request, retryCfg *config.RetryConfig) (*http.Response, error) {
	maxAttempts := 1
	backoff := 100 * time.Millisecond

	if retryCfg != nil {
		if retryCfg.MaxAttempts > 0 {
			maxAttempts = retryCfg.MaxAttempts
		}
		if retryCfg.BackoffMs > 0 {
			backoff = time.Duration(retryCfg.BackoffMs) * time.Millisecond
		}
	}

	var lastErr error
	var resp *http.Response

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Clone request to support dispatch retries with pristine body readers
		clonedReq := req.Clone(ctx)
		if req.Body != nil && req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				clonedReq.Body = body
			}
		}

		resp, lastErr = c.httpClient.Do(clonedReq)
		if lastErr == nil {
			if !c.isRetryable(resp.StatusCode, retryCfg) {
				return resp, nil
			}

			// Close the body on retryable failures to prevent TCP connection leaks
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned retryable error status: %d", resp.StatusCode)
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxAttempts, lastErr)
}

func (c *Client) isRetryable(statusCode int, retryCfg *config.RetryConfig) bool {
	if retryCfg == nil || len(retryCfg.RetryableStatusCodes) == 0 {
		return statusCode >= 500
	}
	for _, code := range retryCfg.RetryableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// DrainAndClose exhausts the response body reader to allow connection reuse.
func DrainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
