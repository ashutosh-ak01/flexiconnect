package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/ashutosh-ak01/flexiconnect/pkg/integration"
	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
	"github.com/ashutosh-ak01/flexiconnect/pkg/track"
)

func setupTestServer(t *testing.T) (*Server, *httptest.Server) {
	vendorMux := http.NewServeMux()
	vendorMux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	vendorMux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	})
	vendorServer := httptest.NewServer(vendorMux)

	registry := config.NewInMemoryRegistry()
	secretsProvider := secret.NewEnvSecretProvider()
	tracker := track.NewNoOpTracker()

	apiCfg := &config.APIConfig{
		APIName:  "test_vendor",
		Version:  "v1",
		IsActive: true,
		Config: config.ConfigDetail{
			BaseURL: vendorServer.URL,
			Endpoints: []config.EndpointConfig{
				{
					Name:   "do_success",
					Method: "POST",
					Path:   "/success",
				},
				{
					Name:   "do_fail",
					Method: "POST",
					Path:   "/fail",
				},
			},
		},
	}
	_ = registry.Register(context.Background(), apiCfg)

	engine := integration.NewEngine(registry, secretsProvider, tracker)
	return NewServer(engine), vendorServer
}

func TestServerExecute(t *testing.T) {
	srv, vendorServer := setupTestServer(t)
	defer vendorServer.Close()

	tests := []struct {
		name           string
		method         string
		url            string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name:   "Health Check",
			method: "GET",
			url:    "/health",
			reqBody: nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Successful Execution",
			method: "POST",
			url:    "/v1/execute",
			reqBody: ExecuteRequest{
				Service: "test_vendor",
				Version: "v1",
				Action:  "do_success",
				Payload: map[string]interface{}{"key": "value"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Engine Execution Failure (Vendor 500)",
			method: "POST",
			url:    "/v1/execute",
			reqBody: ExecuteRequest{
				Service: "test_vendor",
				Version: "v1",
				Action:  "do_fail",
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "Missing Required Fields (Service/Action)",
			method: "POST",
			url:    "/v1/execute",
			reqBody: ExecuteRequest{
				Service: "",
				Version: "v1",
				Action:  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Invalid JSON Body",
			method: "POST",
			url:    "/v1/execute",
			reqBody: "this is not json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyBytes []byte
			if tc.reqBody != nil {
				if s, ok := tc.reqBody.(string); ok {
					bodyBytes = []byte(s)
				} else {
					bodyBytes, _ = json.Marshal(tc.reqBody)
				}
			}

			req, _ := http.NewRequest(tc.method, tc.url, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
