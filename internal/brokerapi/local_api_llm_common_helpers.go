package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func digestIdentityStrict(d trustpolicy.Digest) (string, error) {
	id, err := d.Identity()
	if err != nil {
		return "", fmt.Errorf("digest identity invalid: %w", err)
	}
	return id, nil
}

func digestFromIdentityOrNil(identity string) (*trustpolicy.Digest, error) {
	if strings.TrimSpace(identity) == "" {
		return nil, nil
	}
	d, err := digestFromIdentity(identity)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Service) llmExecutionUnavailable(requestID string) *ErrorResponse {
	errOut := s.makeError(requestID, "gateway_failure", "internal", false, llmExecutionUnavailableMessage)
	return &errOut
}

func stringField(value map[string]any, key string) string {
	raw, _ := value[key].(string)
	return raw
}

func intField(value map[string]any, key string) int64 {
	raw := value[key]
	switch typed := raw.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}
