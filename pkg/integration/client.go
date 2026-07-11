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

type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Client wrapper
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Do executes the HTTP request wrapped with retries and a circuit breaker
func (c *Client) Do(ctx context.Context, req *http.Request, retryCfg *config.RetryConfig, cb *gobreaker.CircuitBreaker) (*http.Response, error) {
	// If a circuit breaker is provided, execute request through it
	if cb != nil {
		respVal, err := cb.Execute(func() (interface{}, error) {
			return c.executeWithRetry(ctx, req, retryCfg)
		})
		if err != nil {
			return nil, err
		}
		return respVal.(*http.Response), nil
	}

	// Otherwise execute normal request with retries
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
		// Clone request to allow multiple dispatch attempts
		clonedReq := req.Clone(ctx)
		if req.Body != nil {
			// If body is present, reset/seek the body reader if possible
			// For http.Request, req.GetBody is the standard way to clone body readers
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err == nil {
					clonedReq.Body = body
				}
			}
		}

		resp, lastErr = c.httpClient.Do(clonedReq)
		if lastErr == nil {
			// If request returns a status code we shouldn't retry, return it
			if !c.isRetryable(resp.StatusCode, retryCfg) {
				return resp, nil
			}
			
			// If it's a retryable error, we close the body so we don't leak connections, and retry
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned retryable error status: %d", resp.StatusCode)
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				// Exponential backoff
				backoff *= 2
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxAttempts, lastErr)
}

func (c *Client) isRetryable(statusCode int, retryCfg *config.RetryConfig) bool {
	if retryCfg == nil || len(retryCfg.RetryableStatusCodes) == 0 {
		// Default behavior: retry on standard server errors
		return statusCode >= 500
	}
	for _, code := range retryCfg.RetryableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// DrainAndClose helper to exhaust and close response body
func DrainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
