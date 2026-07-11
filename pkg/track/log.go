package track

import (
	"context"
	"log/slog"
)

// LogTracker logs API transaction events as structured JSON using the standard log/slog library
type LogTracker struct{}

// NewLogTracker creates a new instance of LogTracker
func NewLogTracker() *LogTracker {
	return &LogTracker{}
}

// Track implements RequestTracker and writes the transaction details to the logger
func (l *LogTracker) Track(ctx context.Context, record *TrackRecord) error {
	slog.InfoContext(ctx, "API Request Audit Log",
		slog.String("api_name", record.APIName),
		slog.String("version", record.Version),
		slog.String("endpoint", record.EndpointName),
		slog.String("method", record.Method),
		slog.String("url", record.URL),
		slog.Any("request_headers", record.RequestHeaders),
		slog.String("request_body", record.RequestBody),
		slog.Int("response_status", record.ResponseStatus),
		slog.Any("response_headers", record.ResponseHeaders),
		slog.String("response_body", record.ResponseBody),
		slog.Int64("duration_ms", record.DurationMs),
		slog.String("error", record.Error),
		slog.Time("timestamp", record.Timestamp),
	)
	return nil
}
