package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func isJSONContentType(contentType string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(contentType))
	if trimmed == "application/json" {
		return true
	}
	return strings.HasPrefix(trimmed, "application/json;")
}

func canonicalizeJSONBytes(payload []byte) ([]byte, error) {
	trimmed := strings.TrimLeft(string(payload), " \t\r\n")
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("json payload must not be empty")
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return nil, fmt.Errorf("top-level JSON value must be an object or array")
	}
	return jsoncanonicalizer.Transform(payload)
}

func canonicalizeJSONValue(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal json value: %w", err)
	}
	return canonicalizeJSONBytes(payload)
}
