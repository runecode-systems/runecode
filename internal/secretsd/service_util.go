package secretsd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

func validateBinding(secretRef, consumerID, roleKind, scope string) error {
	if secretRef != "placeholder" && strings.TrimSpace(secretRef) == "" {
		return fmt.Errorf("secret_ref is required")
	}
	if strings.TrimSpace(consumerID) == "" {
		return fmt.Errorf("consumer_id is required")
	}
	if strings.TrimSpace(roleKind) == "" {
		return fmt.Errorf("role_kind is required")
	}
	if strings.TrimSpace(scope) == "" {
		return fmt.Errorf("scope is required")
	}
	return nil
}

func effectiveTTL(requested int) int {
	if requested <= 0 {
		return defaultTTLSeconds
	}
	if requested > hardCapTTLSeconds {
		return hardCapTTLSeconds
	}
	return requested
}

func randomID(r io.Reader) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return "lease_" + hex.EncodeToString(b), nil
}

func digestHex(b []byte) string {
	d := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(d[:])
}
