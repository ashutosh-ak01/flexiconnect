package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
	"github.com/ashutosh-ak01/flexiconnect/pkg/track"
)

type mockTracker struct {
	mu      sync.Mutex
	records []*track.TrackRecord
}

func (m *mockTracker) Track(ctx context.Context, record *track.TrackRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, record)
	return nil
}

func (m *mockTracker) GetRecords() []*track.TrackRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]*track.TrackRecord, len(m.records))
	copy(copied, m.records)
	return copied
}

func TestEngineExecution(t *testing.T) {
	// Create mock backend server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify dynamic headers were interpolated and matched
		if r.Header.Get("Authorization") != "Bearer secret-api-key-999" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var body map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &body)

		// Assert body fields were rendered properly
		if body["user"] != "tester_bob" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Sensitive-Response", "StripePlainSecret")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"id": "txn_abc123",
			"details": {
				"amount": 99.9,
				"token": "oauth-resp-token-555"
			}
		}`))
	}))
	defer server.Close()

	ctx := context.Background()
	registry := config.NewInMemoryRegistry()

	// 1. Create API Integration Configuration
	apiCfg := &config.APIConfig{
		APIName:  "payment_gateway",
		Version:  "v2",
		IsActive: true,
		Config: config.ConfigDetail{
			BaseURL:   server.URL,
			TimeoutMs: 2000,
			Tracking: &config.TrackingConfig{
				Enabled:          true,
				MaskHeaders:      []string{"Authorization", "X-Sensitive-Response"},
				MaskRequestKeys:  []string{"password", "ssn"},
				MaskResponseKeys: []string{"token"},
			},
			Endpoints: []config.EndpointConfig{
				{
					Name:   "pay",
					Method: "POST",
					Path:   "/charge",
					Headers: map[string]string{
						"Authorization": "Bearer {{ secret \"GATEWAY_TOKEN\" }}",
						"Content-Type":  "application/json",
					},
					BodyTemplate: `{"user": "{{ .input.name }}", "password": "{{ .input.pass }}", "ssn": "000-11-2222"}`,
					ResponseTransformation: map[string]string{
						"transaction_id": "$.id",
						"charge_status":  "$.status",
						"amount_charged": "$.details.amount",
					},
				},
			},
		},
	}
	_ = registry.Register(ctx, apiCfg)

	// Set dynamic environment variables
	t.Setenv("GATEWAY_TOKEN", "secret-api-key-999")

	secretProvider := secret.NewEnvSecretProvider()
	tracker := &mockTracker{}

	// Initialize Engine
	engine := NewEngine(registry, secretProvider, tracker)

	input := map[string]interface{}{
		"name": "tester_bob",
		"pass": "supersecretpassword",
	}

	// 2. Trigger Request
	result, err := engine.ExecuteRequest(ctx, "payment_gateway", "v2", "pay", input)
	if err != nil {
		t.Fatalf("Engine failed executing request: %v", err)
	}

	// 3. Assert Response Transform mappings
	if result["transaction_id"] != "txn_abc123" {
		t.Errorf("Expected transaction_id = 'txn_abc123', got %v", result["transaction_id"])
	}
	if result["charge_status"] != "success" {
		t.Errorf("Expected charge_status = 'success', got %v", result["charge_status"])
	}
	if result["amount_charged"] != 99.9 {
		t.Errorf("Expected amount_charged = 99.9, got %v", result["amount_charged"])
	}

	// 4. Verify Asynchronous Auditing and Sensitive Data Masking
	time.Sleep(100 * time.Millisecond) // Give goroutine worker time to run

	records := tracker.GetRecords()
	if len(records) != 1 {
		t.Fatalf("Expected 1 audit record captured, got %d", len(records))
	}

	rec := records[0]

	authVal := rec.RequestHeaders["Authorization"]
	if len(authVal) == 0 || authVal[0] != "[MASKED]" {
		t.Errorf("Expected Request Header 'Authorization' to be masked, got '%v'", authVal)
	}
	respVal := rec.ResponseHeaders["X-Sensitive-Response"]
	if len(respVal) == 0 || respVal[0] != "[MASKED]" {
		t.Errorf("Expected Response Header 'X-Sensitive-Response' to be masked, got '%v'", respVal)
	}

	// Verify JSON Body Masking (Request)
	if strings.Contains(rec.RequestBody, "supersecretpassword") {
		t.Error("Sensitive Request JSON key 'password' value was leaked (not masked)")
	}
	if strings.Contains(rec.RequestBody, "000-11-2222") {
		t.Error("Sensitive Request JSON key 'ssn' value was leaked (not masked)")
	}
	if !strings.Contains(rec.RequestBody, `"[MASKED]"`) {
		t.Errorf("Masked request keys didn't transform to [MASKED], body: %s", rec.RequestBody)
	}

	// Verify JSON Body Masking (Response)
	if strings.Contains(rec.ResponseBody, "oauth-resp-token-555") {
		t.Error("Sensitive Response JSON key 'token' value was leaked (not masked)")
	}
}

func TestSuccessCondition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": false, "error": "Invalid API Key"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	registry := config.NewInMemoryRegistry()
	apiCfg := &config.APIConfig{
		APIName:  "failing_api",
		Version:  "v1",
		IsActive: true,
		Config: config.ConfigDetail{
			BaseURL: server.URL,
			Endpoints: []config.EndpointConfig{
				{
					Name:   "test_success",
					Method: "POST",
					Path:   "/test",
					SuccessCondition: &config.SuccessCondition{
						StatusCodes: []int{200},
						JSONPathAssertions: []config.JSONPathAssertion{
							{Path: "$.success", ExpectedValue: false}, // this passes
						},
					},
				},
				{
					Name:   "test_failure",
					Method: "POST",
					Path:   "/test",
					SuccessCondition: &config.SuccessCondition{
						StatusCodes: []int{200},
						JSONPathAssertions: []config.JSONPathAssertion{
							{Path: "$.success", ExpectedValue: true}, // this fails
						},
					},
				},
				{
					Name:   "test_empty_error",
					Method: "POST",
					Path:   "/test",
					SuccessCondition: &config.SuccessCondition{
						JSONPathAssertions: []config.JSONPathAssertion{
							{Path: "$.error", ExpectedEmpty: true}, // this fails
						},
					},
				},
			},
		},
	}
	_ = registry.Register(ctx, apiCfg)

	engine := NewEngine(registry, secret.NewEnvSecretProvider(), nil)

	// This passes because ExpectedValue matches actual JSON value
	_, err := engine.ExecuteRequest(ctx, "failing_api", "v1", "test_success", nil)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// This should fail because $.success is false, not true
	_, err = engine.ExecuteRequest(ctx, "failing_api", "v1", "test_failure", nil)
	if err == nil {
		t.Error("Expected error due to failed success condition assertion, but got nil")
	} else if !strings.Contains(err.Error(), "success condition verification failed") {
		t.Errorf("Expected success verification error message, got: %v", err)
	}

	// This should fail because $.error is "Invalid API Key", not empty
	_, err = engine.ExecuteRequest(ctx, "failing_api", "v1", "test_empty_error", nil)
	if err == nil {
		t.Error("Expected error due to non-empty error path, but got nil")
	}
}

func TestCustomRetryPolicies(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests) // 429
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	registry := config.NewInMemoryRegistry()
	apiCfg := &config.APIConfig{
		APIName:  "retry_api",
		Version:  "v1",
		IsActive: true,
		Config: config.ConfigDetail{
			BaseURL: server.URL,
			RetryPolicy: &config.RetryConfig{
				MaxAttempts:          3,
				BackoffMs:            10,
				RetryableStatusCodes: []int{429},
			},
			Endpoints: []config.EndpointConfig{
				{
					Name:   "test_retry",
					Method: "GET",
					Path:   "/retry",
				},
			},
		},
	}
	_ = registry.Register(ctx, apiCfg)

	engine := NewEngine(registry, secret.NewEnvSecretProvider(), nil)

	_, err := engine.ExecuteRequest(ctx, "retry_api", "v1", "test_retry", nil)
	if err != nil {
		t.Fatalf("Expected execution to succeed after 3 attempts, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 request attempts, got: %d", attempts)
	}
}
