package transform

import (
	"encoding/json"
	"fmt"

	"github.com/yalp/jsonpath"
)

// ResponseTransformer parses raw JSON responses and maps fields using JSONPath queries
type ResponseTransformer struct{}

// NewResponseTransformer creates a new instance of ResponseTransformer
func NewResponseTransformer() *ResponseTransformer {
	return &ResponseTransformer{}
}

// Transform extracts fields from rawJSON using JSONPath expressions defined in mappings
func (rt *ResponseTransformer) Transform(rawJSON []byte, mappings map[string]string) (map[string]interface{}, error) {
	if len(rawJSON) == 0 || len(mappings) == 0 {
		// If no mappings defined, return the parsed raw JSON directly
		var rawMap map[string]interface{}
		if err := json.Unmarshal(rawJSON, &rawMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal raw response JSON: %w", err)
		}
		return rawMap, nil
	}

	// Parse JSON into generic interface structure for yalp/jsonpath
	var parsedJSON interface{}
	if err := json.Unmarshal(rawJSON, &parsedJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw response JSON: %w", err)
	}

	result := make(map[string]interface{})
	for targetKey, pathExpr := range mappings {
		val, err := jsonpath.Read(parsedJSON, pathExpr)
		if err != nil {
			// If JSONPath query fails (e.g. field doesn't exist), store nil or skip
			result[targetKey] = nil
			continue
		}
		result[targetKey] = val
	}

	return result, nil
}
