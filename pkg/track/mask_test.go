package track

import (
	"encoding/json"
	"testing"
)

func TestMaskHeaders(t *testing.T) {
	headers := map[string][]string{
		"Authorization":   {"Bearer token"},
		"X-Api-Key":       {"my-secret-key"},
		"Content-Type":    {"application/json"},
		"X-Custom-Header": {"normal-val"},
	}

	keysToMask := []string{"authorization", "x-api-key", "cookie"}
	masked := MaskHeaders(headers, keysToMask)

	if masked["Authorization"][0] != "[MASKED]" {
		t.Errorf("Expected Authorization header to be masked, got: %s", masked["Authorization"][0])
	}
	if masked["X-Api-Key"][0] != "[MASKED]" {
		t.Errorf("Expected X-Api-Key header to be masked, got: %s", masked["X-Api-Key"][0])
	}
	if masked["Content-Type"][0] != "application/json" {
		t.Errorf("Expected Content-Type to remain untouched, got: %s", masked["Content-Type"][0])
	}
}

func TestMaskJSONBody(t *testing.T) {
	rawJSON := `{
		"username": "alex",
		"password": "mysecretpassword",
		"payment": {
			"card_number": "1234-5678-9012-3456",
			"cvv": "123"
		},
		"tags": ["personal", "private_ssn"],
		"nested_array": [
			{"ssn": "000-11-2222", "public_id": "pub_1"}
		]
	}`

	keysToMask := []string{"password", "card_number", "cvv", "ssn"}
	maskedStr := MaskJSONBody(rawJSON, keysToMask)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(maskedStr), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse masked JSON string: %v", err)
	}

	if parsed["password"] != "[MASKED]" {
		t.Errorf("Expected password to be masked, got: %v", parsed["password"])
	}

	payment := parsed["payment"].(map[string]interface{})
	if payment["card_number"] != "[MASKED]" {
		t.Errorf("Expected card_number to be masked, got: %v", payment["card_number"])
	}
	if payment["cvv"] != "[MASKED]" {
		t.Errorf("Expected cvv to be masked, got: %v", payment["cvv"])
	}

	nestedArray := parsed["nested_array"].([]interface{})
	item := nestedArray[0].(map[string]interface{})
	if item["ssn"] != "[MASKED]" {
		t.Errorf("Expected ssn in nested array to be masked, got: %v", item["ssn"])
	}
	if item["public_id"] != "pub_1" {
		t.Errorf("Expected public_id to remain unchanged, got: %v", item["public_id"])
	}
}
