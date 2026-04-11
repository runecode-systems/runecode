package policyengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func canonicalHashBytes(payload []byte) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func CanonicalHashBytes(payload []byte) (string, error) {
	return canonicalHashBytes(payload)
}

func canonicalHashValue(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal value for hashing: %w", err)
	}
	return canonicalHashBytes(b)
}

func normalizeHashIdentity(hash string) (string, error) {
	if !strings.HasPrefix(hash, "sha256:") {
		return "", fmt.Errorf("digest must use sha256 identity format")
	}
	raw := strings.TrimPrefix(hash, "sha256:")
	if len(raw) != 64 {
		return "", fmt.Errorf("sha256 hash must be 64 lowercase hex characters")
	}
	if strings.ToLower(raw) != raw {
		return "", fmt.Errorf("sha256 hash must be lowercase hex")
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return "", fmt.Errorf("invalid sha256 hash value: %w", err)
	}
	return hash, nil
}

func NormalizeHashIdentity(hash string) (string, error) {
	return normalizeHashIdentity(hash)
}
