package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func approvedImplementationWriteIntent(payload []byte) (string, string, []byte, error) {
	decoded, err := decodeApprovedImplementationMutationArtifactPayload(payload)
	if err != nil {
		return "", "", nil, err
	}
	if err := validateApprovedImplementationMutationArtifactShape(decoded); err != nil {
		return "", "", nil, err
	}
	targetPath := strings.TrimSpace(requiredStringFromMap(decoded, "target_path"))
	if targetPath == "" {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact missing target_path")
	}
	contentDigest, ok := digestIdentityFromApprovedImplementationField(decoded, "content_digest")
	if !ok {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact missing content_digest")
	}
	contentRef := strings.TrimSpace(optionalStringFromMap(decoded, "content_artifact_digest"))
	if contentRef != "" {
		return targetPath, "", nil, fmt.Errorf("approved implementation mutation artifact content_artifact_digest is not supported in local broker mutation path")
	}
	contentText := requiredStringFromMap(decoded, "content")
	if contentText == "" {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact missing content")
	}
	content := []byte(contentText)
	if artifacts.DigestBytes(content) != strings.TrimSpace(contentDigest) {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact content_digest drift for %q", targetPath)
	}
	writeMode := strings.TrimSpace(requiredStringFromMap(decoded, "write_mode"))
	if writeMode == "" {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact missing write_mode")
	}
	if writeMode != "update" && writeMode != "create" {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact write_mode %q is unsupported", writeMode)
	}
	return targetPath, writeMode, content, nil
}

func validateApprovedImplementationMutationArtifactShape(decoded map[string]any) error {
	allowed := map[string]struct{}{
		"target_path":             {},
		"content":                 {},
		"content_digest":          {},
		"content_artifact_digest": {},
		"write_mode":              {},
	}
	for key := range decoded {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("approved implementation mutation artifact field %q is unsupported", key)
		}
	}
	for _, key := range []string{"target_path", "content", "write_mode"} {
		if _, ok := decoded[key].(string); !ok {
			return fmt.Errorf("approved implementation mutation artifact %s must be a string", key)
		}
	}
	if raw, ok := decoded["content_artifact_digest"]; ok {
		if _, ok := raw.(string); !ok {
			return fmt.Errorf("approved implementation mutation artifact content_artifact_digest must be a string")
		}
	}
	return nil
}

func decodeApprovedImplementationMutationArtifactPayload(payload []byte) (map[string]any, error) {
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode approved implementation mutation artifact: %w", err)
	}
	return decoded, nil
}

func requiredStringFromMap(in map[string]any, key string) string {
	value, _ := in[key].(string)
	return value
}

func optionalStringFromMap(in map[string]any, key string) string {
	value, _ := in[key].(string)
	return value
}

func digestIdentityFromApprovedImplementationValue(value map[string]any) (string, error) {
	hashAlg, _ := value["hash_alg"].(string)
	hash, _ := value["hash"].(string)
	return (trustpolicy.Digest{HashAlg: hashAlg, Hash: hash}).Identity()
}
