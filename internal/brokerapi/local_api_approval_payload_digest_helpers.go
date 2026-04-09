package brokerapi

import (
	"fmt"
	"strings"
)

func digestIdentityFromPayloadObject(payload map[string]any, key string) (string, error) {
	raw, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("missing key %q", key)
	}
	digest, ok := raw.(map[string]any)
	if !ok {
		return "", fmt.Errorf("key %q has type %T, want digest object", key, raw)
	}
	hashAlg, ok := digest["hash_alg"].(string)
	if !ok || strings.TrimSpace(hashAlg) == "" {
		return "", fmt.Errorf("key %q hash_alg missing or invalid", key)
	}
	hash, ok := digest["hash"].(string)
	if !ok || strings.TrimSpace(hash) == "" {
		return "", fmt.Errorf("key %q hash missing or invalid", key)
	}
	identity := strings.TrimSpace(hashAlg) + ":" + strings.TrimSpace(hash)
	if !isSHA256Digest(identity) {
		return "", fmt.Errorf("key %q has invalid digest identity %q", key, identity)
	}
	return identity, nil
}

func digestIdentitiesFromPayloadArray(payload map[string]any, key string) ([]string, error) {
	raw, ok := payload[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q", key)
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want []any", key, raw)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		digest, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("key %q contains non-object digest", key)
		}
		hashAlg, _ := digest["hash_alg"].(string)
		hash, _ := digest["hash"].(string)
		identity := strings.TrimSpace(hashAlg) + ":" + strings.TrimSpace(hash)
		if !isSHA256Digest(identity) {
			return nil, fmt.Errorf("key %q contains invalid digest identity %q", key, identity)
		}
		out = append(out, identity)
	}
	return uniqueSortedDigests(out), nil
}
