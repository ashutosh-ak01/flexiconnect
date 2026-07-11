package track

import (
	"context"
	"time"
)

// TrackRecord contains metadata, payloads, headers, latency, and errors for a specific request/response execution
type TrackRecord struct {
	APIName         string              `json:"api_name"`
	Version         string              `json:"version"`
	EndpointName    string              `json:"endpoint_name"`
	Method          string              `json:"method"`
	URL             string              `json:"url"`
	RequestHeaders  map[string][]string `json:"request_headers,omitempty"`
	RequestBody     string              `json:"request_body,omitempty"`
	ResponseStatus  int                 `json:"response_status"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	ResponseBody    string              `json:"response_body,omitempty"`
	DurationMs      int64               `json:"duration_ms"`
	Error           string              `json:"error,omitempty"`
	Timestamp       time.Time           `json:"timestamp"`
}

// RequestTracker defines the contract for persisting tracked request-response cycles
type RequestTracker interface {
	Track(ctx context.Context, record *TrackRecord) error
}

// NoOpTracker is a default implementation that drops all records (used when tracking is disabled)
type NoOpTracker struct{}

// NewNoOpTracker creates a new NoOpTracker
func NewNoOpTracker() *NoOpTracker {
	return &NoOpTracker{}
}

// Track drops the track record and returns nil
func (t *NoOpTracker) Track(ctx context.Context, record *TrackRecord) error {
	return nil
}
