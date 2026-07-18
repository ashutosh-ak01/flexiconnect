package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ashutosh-ak01/flexiconnect/pkg/integration"
)

// Server wraps the FlexiConnect engine and provides an HTTP API.
type Server struct {
	engine *integration.Engine
	mux    *http.ServeMux
}

// NewServer creates a new HTTP server instance with registered routes.
func NewServer(engine *integration.Engine) *Server {
	s := &Server{
		engine: engine,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ExecuteRequest defines the expected JSON body for the execution endpoint.
type ExecuteRequest struct {
	Service string                 `json:"service"`
	Version string                 `json:"version,omitempty"`
	Action  string                 `json:"action"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// registerRoutes sets up the HTTP multiplexer with the required endpoints.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /v1/execute", s.handleExecute)

	// Health check endpoint
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleExecute processes incoming API requests and delegates to the integration engine.
func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode JSON request body", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Service == "" || req.Action == "" {
		http.Error(w, "Missing required fields: service, action", http.StatusBadRequest)
		return
	}

	slog.Info("Received execute request", "service", req.Service, "version", req.Version, "action", req.Action)

	result, err := s.engine.ExecuteRequest(r.Context(), req.Service, req.Version, req.Action, req.Payload)
	if err != nil {
		slog.Error("Engine execution failed", "error", err, "service", req.Service, "action", req.Action)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("Failed to encode response payload", "error", err)
	}
}
