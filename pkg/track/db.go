package track

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// DBTracker handles asynchronous audit log writes to a database using a buffered worker pool
type DBTracker struct {
	db          *sql.DB
	recordsChan chan *TrackRecord
	wg          sync.WaitGroup
	quit        chan struct{}
	once        sync.Once
}

// NewDBTracker creates and starts a new DBTracker background worker pool
func NewDBTracker(db *sql.DB, queueSize int, workers int) *DBTracker {
	if queueSize <= 0 {
		queueSize = 1000
	}
	if workers <= 0 {
		workers = 5
	}

	tracker := &DBTracker{
		db:          db,
		recordsChan: make(chan *TrackRecord, queueSize),
		quit:        make(chan struct{}),
	}

	tracker.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go tracker.worker()
	}

	return tracker
}

// Track queues the record for writing. If queue is full, drops the record to prevent client blockage
func (t *DBTracker) Track(ctx context.Context, record *TrackRecord) error {
	select {
	case t.recordsChan <- record:
		return nil
	default:
		slog.Warn("DB audit logger queue is full; audit record dropped", "api", record.APIName, "endpoint", record.EndpointName)
		return nil
	}
}

// Close stops the worker pool gracefully, draining any remaining queued records
func (t *DBTracker) Close() {
	t.once.Do(func() {
		close(t.quit)
		close(t.recordsChan)
		t.wg.Wait()
	})
}

func (t *DBTracker) worker() {
	defer t.wg.Done()

	for {
		select {
		case record, ok := <-t.recordsChan:
			if !ok {
				return
			}
			t.save(record)
		case <-t.quit:
			// Drain anything left in recordsChan before exiting
			for record := range t.recordsChan {
				t.save(record)
			}
			return
		}
	}
}

func (t *DBTracker) save(record *TrackRecord) {
	reqHeadersJSON, _ := json.Marshal(record.RequestHeaders)
	respHeadersJSON, _ := json.Marshal(record.ResponseHeaders)

	query := `
		INSERT INTO audit_logs (
			api_name, version, endpoint_name, method, url, 
			request_headers, request_body, response_status, 
			response_headers, response_body, duration_ms, error, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := t.db.ExecContext(ctx, query,
		record.APIName,
		record.Version,
		record.EndpointName,
		record.Method,
		record.URL,
		reqHeadersJSON,
		record.RequestBody,
		record.ResponseStatus,
		respHeadersJSON,
		record.ResponseBody,
		record.DurationMs,
		record.Error,
		record.Timestamp,
	)
	if err != nil {
		slog.Error("failed to write audit log to DB", "api", record.APIName, "endpoint", record.EndpointName, "error", err)
	}
}
