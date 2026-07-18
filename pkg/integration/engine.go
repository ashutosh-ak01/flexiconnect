package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
	"github.com/ashutosh-ak01/flexiconnect/pkg/track"
	"github.com/ashutosh-ak01/flexiconnect/pkg/transform"
	"github.com/sony/gobreaker"
	"github.com/yalp/jsonpath"
)

// Engine orchestrates versioned configuration resolution, templates interpolation, HTTP execution, and audits.
type Engine struct {
	registry            config.ConfigRegistry
	secretResolver      secret.SecretProvider
	tracker             track.RequestTracker
	requestTransformer  *transform.RequestTransformer
	responseTransformer *transform.ResponseTransformer
	httpClient          *Client

	breakersMu sync.RWMutex
	breakers   map[string]*gobreaker.CircuitBreaker // key format: api:version:endpoint
}

// NewEngine instantiates a new execution engine.
func NewEngine(registry config.ConfigRegistry, sp secret.SecretProvider, tracker track.RequestTracker) *Engine {
	if tracker == nil {
		tracker = track.NewNoOpTracker()
	}
	return &Engine{
		registry:            registry,
		secretResolver:      sp,
		tracker:             tracker,
		requestTransformer:  transform.NewRequestTransformer(sp),
		responseTransformer: transform.NewResponseTransformer(),
		httpClient:          NewClient(15 * time.Second),
		breakers:            make(map[string]*gobreaker.CircuitBreaker),
	}
}

// ExecuteRequest resolves config, applies template transformations, resolves secrets, and executes request through circuit breakers.
func (e *Engine) ExecuteRequest(ctx context.Context, apiName string, version string, endpointName string, input map[string]interface{}) (map[string]interface{}, error) {
	startTime := time.Now()

	apiCfg, err := e.registry.Get(ctx, apiName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}

	var epConfig *config.EndpointConfig
	for _, ep := range apiCfg.Config.Endpoints {
		if ep.Name == endpointName {
			epConfig = &ep
			break
		}
	}
	if epConfig == nil {
		return nil, fmt.Errorf("endpoint %s not defined for API %s", endpointName, apiName)
	}

	resolvedURL, err := e.requestTransformer.Transform(ctx, apiCfg.Config.BaseURL+epConfig.Path, input)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	httpHeaders := make(http.Header)
	for headerKey, headerValTemplate := range epConfig.Headers {
		resolvedHeaderVal, err := e.requestTransformer.Transform(ctx, headerValTemplate, input)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve header %s: %w", headerKey, err)
		}
		httpHeaders.Set(headerKey, resolvedHeaderVal)
	}

	var requestBodyStr string
	if epConfig.BodyTemplate != "" {
		requestBodyStr, err = e.requestTransformer.Transform(ctx, epConfig.BodyTemplate, input)
		if err != nil {
			return nil, fmt.Errorf("failed to transform request body: %w", err)
		}
	}

	var bodyReader io.Reader
	if requestBodyStr != "" {
		bodyReader = strings.NewReader(requestBodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, epConfig.Method, resolvedURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate HTTP request: %w", err)
	}
	req.Header = httpHeaders

	cb := e.getBreaker(apiCfg.APIName, apiCfg.Version, epConfig.Name, apiCfg.Config.CircuitBreaker)

	resp, httpErr := e.httpClient.Do(ctx, req, apiCfg.Config.RetryPolicy, cb)

	durationMs := time.Since(startTime).Milliseconds()

	var responseStatusCode int
	var responseBodyBytes []byte
	var responseHeaders map[string][]string
	var executionErr error

	if httpErr != nil {
		executionErr = httpErr
	} else {
		defer DrainAndClose(resp.Body)
		responseStatusCode = resp.StatusCode
		responseHeaders = resp.Header

		responseBodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			executionErr = fmt.Errorf("failed to read response body: %w", err)
		} else {
			if err := e.assertSuccess(responseStatusCode, responseBodyBytes, epConfig.SuccessCondition); err != nil {
				executionErr = err
			}
		}
	}

	var transformedResponse map[string]interface{}
	if executionErr == nil {
		transformedResponse, executionErr = e.responseTransformer.Transform(responseBodyBytes, epConfig.ResponseTransformation)
	}

	if apiCfg.Config.Tracking != nil && apiCfg.Config.Tracking.Enabled {
		record := &track.TrackRecord{
			APIName:         apiCfg.APIName,
			Version:         apiCfg.Version,
			EndpointName:    epConfig.Name,
			Method:          epConfig.Method,
			URL:             resolvedURL,
			RequestHeaders:  req.Header,
			RequestBody:     requestBodyStr,
			ResponseStatus:  responseStatusCode,
			ResponseHeaders: responseHeaders,
			ResponseBody:    string(responseBodyBytes),
			DurationMs:      durationMs,
			Timestamp:       startTime,
		}
		if executionErr != nil {
			record.Error = executionErr.Error()
		}

		// Mask sensitive fields synchronously in the main thread to prevent raw PII data from leaking into memory
		record.RequestHeaders = track.MaskHeaders(record.RequestHeaders, apiCfg.Config.Tracking.MaskHeaders)
		record.ResponseHeaders = track.MaskHeaders(record.ResponseHeaders, apiCfg.Config.Tracking.MaskHeaders)
		record.RequestBody = track.MaskJSONBody(record.RequestBody, apiCfg.Config.Tracking.MaskRequestKeys)
		record.ResponseBody = track.MaskJSONBody(record.ResponseBody, apiCfg.Config.Tracking.MaskResponseKeys)

		// Persist tracking asynchronously using a detached context so cancellation doesn't abort audits
		go func(r *track.TrackRecord) {
			_ = e.tracker.Track(context.Background(), r)
		}(record)
	}

	if executionErr != nil {
		return nil, executionErr
	}

	return transformedResponse, nil
}

func (e *Engine) getBreaker(apiName, version, endpointName string, cbCfg *config.CBConfig) *gobreaker.CircuitBreaker {
	key := fmt.Sprintf("%s:%s:%s", apiName, version, endpointName)

	e.breakersMu.RLock()
	cb, exists := e.breakers[key]
	e.breakersMu.RUnlock()

	if exists {
		return cb
	}

	e.breakersMu.Lock()
	defer e.breakersMu.Unlock()

	if cb, exists = e.breakers[key]; exists {
		return cb
	}

	var settings gobreaker.Settings
	settings.Name = key
	if cbCfg != nil {
		settings.MaxRequests = cbCfg.MaxRequests
		settings.Interval = time.Duration(cbCfg.IntervalSec) * time.Second
		settings.Timeout = time.Duration(cbCfg.TimeoutSec) * time.Second
		settings.ReadyToTrip = func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbCfg.ConsecutiveFailures
		}
	} else {
		settings.ReadyToTrip = func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		}
	}

	cb = gobreaker.NewCircuitBreaker(settings)
	e.breakers[key] = cb
	return cb
}

func (e *Engine) assertSuccess(statusCode int, bodyBytes []byte, cond *config.SuccessCondition) error {
	if cond == nil {
		if statusCode < 200 || statusCode >= 300 {
			return fmt.Errorf("HTTP status code %d not in successful range [200-299]", statusCode)
		}
		return nil
	}

	if len(cond.StatusCodes) > 0 {
		matched := false
		for _, code := range cond.StatusCodes {
			if statusCode == code {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("HTTP status code %d not in allowed success list %v", statusCode, cond.StatusCodes)
		}
	} else {
		if statusCode < 200 || statusCode >= 300 {
			return fmt.Errorf("HTTP status code %d not in successful range [200-299]", statusCode)
		}
	}

	if len(cond.JSONPathAssertions) > 0 {
		if len(bodyBytes) == 0 {
			return fmt.Errorf("response body is empty; cannot run JSONPath assertions")
		}

		var parsedJSON interface{}
		if err := json.Unmarshal(bodyBytes, &parsedJSON); err != nil {
			return fmt.Errorf("failed to parse response JSON for assertions: %w", err)
		}

		for _, assertion := range cond.JSONPathAssertions {
			val, err := jsonpath.Read(parsedJSON, assertion.Path)
			if err != nil {
				if assertion.ExpectedEmpty {
					continue
				}
				return fmt.Errorf("failed to read JSONPath '%s' for success verification: %w", assertion.Path, err)
			}

			if assertion.ExpectedEmpty {
				if val != nil && val != "" {
					return fmt.Errorf("success condition verification failed: expected JSONPath '%s' to be empty, got: %v", assertion.Path, val)
				}
			} else {
				expectedStr := fmt.Sprintf("%v", assertion.ExpectedValue)
				gotStr := fmt.Sprintf("%v", val)
				if expectedStr != gotStr {
					return fmt.Errorf("success condition verification failed: JSONPath '%s' expected '%s', got '%s'", assertion.Path, expectedStr, gotStr)
				}
			}
		}
	}

	return nil
}
