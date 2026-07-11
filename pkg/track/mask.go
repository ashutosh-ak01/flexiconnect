package track

import (
	"encoding/json"
	"strings"
)

// MaskHeaders replaces header values with "[MASKED]" for case-insensitive matching keys
func MaskHeaders(headers map[string][]string, keysToMask []string) map[string][]string {
	if len(headers) == 0 || len(keysToMask) == 0 {
		return headers
	}

	masked := make(map[string][]string)
	maskSet := make(map[string]bool)
	for _, key := range keysToMask {
		maskSet[strings.ToLower(key)] = true
	}

	for k, values := range headers {
		if maskSet[strings.ToLower(k)] {
			masked[k] = []string{"[MASKED]"}
		} else {
			masked[k] = values
		}
	}
	return masked
}

// MaskJSONBody parses raw JSON, recursively traverses and masks values for matching keys, and returns the marshalled string
func MaskJSONBody(rawJSON string, keysToMask []string) string {
	if rawJSON == "" || len(keysToMask) == 0 {
		return rawJSON
	}

	maskSet := make(map[string]bool)
	for _, key := range keysToMask {
		maskSet[strings.ToLower(key)] = true
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		// If it's not valid JSON, we cannot parse it, so we return it unchanged (fallback)
		return rawJSON
	}

	maskValue(&parsed, maskSet)

	maskedBytes, err := json.Marshal(parsed)
	if err != nil {
		return rawJSON
	}
	return string(maskedBytes)
}

func maskValue(node *interface{}, maskSet map[string]bool) {
	if node == nil || *node == nil {
		return
	}

	switch val := (*node).(type) {
	case map[string]interface{}:
		for k, v := range val {
			if maskSet[strings.ToLower(k)] {
				val[k] = "[MASKED]"
			} else {
				// Recursively traverse child node
				maskValue(&v, maskSet)
				val[k] = v
			}
		}
	case []interface{}:
		for i := range val {
			maskValue(&val[i], maskSet)
		}
	}
}
