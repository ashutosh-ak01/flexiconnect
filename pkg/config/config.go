package config

import "time"

// APIConfig defines the outer wrapper containing name, versioning state, and detail config
type APIConfig struct {
	ID        int          `json:"id,omitempty"`
	APIName   string       `json:"api_name"`
	Version   string       `json:"version"`
	IsActive  bool         `json:"is_active"`
	Config    ConfigDetail `json:"config"`
	CreatedAt time.Time    `json:"created_at,omitempty"`
	UpdatedAt time.Time    `json:"updated_at,omitempty"`
}

// ConfigDetail contains the actual operational endpoints, auth, and resiliency configurations
type ConfigDetail struct {
	BaseURL        string            `json:"base_url"`
	TimeoutMs      int               `json:"timeout_ms"`
	RetryPolicy    *RetryConfig      `json:"retry_policy,omitempty"`
	CircuitBreaker *CBConfig         `json:"circuit_breaker,omitempty"`
	RateLimiting   *RateLimitConfig  `json:"rate_limiting,omitempty"`
	Auth           *AuthConfig       `json:"auth,omitempty"`
	Tracking       *TrackingConfig   `json:"tracking,omitempty"`
	Endpoints      []EndpointConfig  `json:"endpoints"`
}

// RetryConfig outlines backoff and retry limitations
type RetryConfig struct {
	MaxAttempts          int   `json:"max_attempts"`
	BackoffMs            int   `json:"backoff_ms"`
	RetryableStatusCodes []int `json:"retryable_status_codes,omitempty"` // Status codes to retry (e.g. 502, 503, 504, 429)
}

// CBConfig specifies local circuit breaker thresholds
type CBConfig struct {
	MaxRequests         uint32 `json:"max_requests"`         // Max requests allowed in half-open state
	IntervalSec         int    `json:"interval_sec"`         // Time window in seconds to clear failures in closed state
	TimeoutSec          int    `json:"timeout_sec"`          // Duration in seconds in open state before transitioning to half-open
	ConsecutiveFailures uint32 `json:"consecutive_failures"` // Number of consecutive failures to trip the breaker
}

// RateLimitConfig represents client-side rate limits (P1)
type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	Burst             int     `json:"burst"`
}

// AuthConfig holds authorization credential specs (P1)
type AuthConfig struct {
	Type   string       `json:"type"`             // "oauth2", "static_token", "basic"
	OAuth2 *OAuthConfig `json:"oauth2,omitempty"` // Configuration if OAuth2 is chosen
}

// OAuthConfig defines properties for OAuth2 client credentials flows (P1)
type OAuthConfig struct {
	TokenURL     string   `json:"token_url"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes,omitempty"`
}

// TrackingConfig enables and configures auditing with sensitive data masking
type TrackingConfig struct {
	Enabled          bool     `json:"enabled"`
	MaskHeaders      []string `json:"mask_headers,omitempty"`
	MaskRequestKeys  []string `json:"mask_request_keys,omitempty"`
	MaskResponseKeys []string `json:"mask_response_keys,omitempty"`
}

// EndpointConfig defines individual endpoints under an API
type EndpointConfig struct {
	Name                   string            `json:"name"`
	Method                 string            `json:"method"`
	Path                   string            `json:"path"`
	Headers                map[string]string `json:"headers,omitempty"`
	BodyTemplate           string            `json:"body_template,omitempty"`
	SuccessCondition       *SuccessCondition `json:"success_condition,omitempty"`
	ResponseTransformation map[string]string `json:"response_transformation,omitempty"` // Map ClientKey -> JSONPath
}

// SuccessCondition defines the criteria used to assert if an execution was successful
type SuccessCondition struct {
	StatusCodes        []int               `json:"status_codes,omitempty"`        // Acceptable HTTP statuses (e.g. [200, 201])
	JSONPathAssertions []JSONPathAssertion `json:"jsonpath_assertions,omitempty"` // Body JSON value check assertions
}

// JSONPathAssertion defines expectations on response JSON keys
type JSONPathAssertion struct {
	Path          string      `json:"path"`                     // JSONPath to retrieve value
	ExpectedValue interface{} `json:"expected_value,omitempty"` // Value to assert equality (optional)
	ExpectedEmpty bool        `json:"expected_empty,omitempty"` // Assert value is empty/null/absent (optional)
}
