package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) approvedImplementationWriteIntent(payload []byte) (string, string, []byte, error) {
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
	contentText, contentRef, err := approvedImplementationMutationContentSource(decoded)
	if err != nil {
		return "", "", nil, err
	}
	content, err := s.resolveApprovedImplementationMutationContent(targetPath, contentText, contentRef, strings.TrimSpace(contentDigest))
	if err != nil {
		return "", "", nil, err
	}
	if artifacts.DigestBytes(content) != strings.TrimSpace(contentDigest) {
		return "", "", nil, fmt.Errorf("approved implementation mutation artifact content_digest drift for %q", targetPath)
	}
	writeMode := strings.TrimSpace(requiredStringFromMap(decoded, "write_mode"))
	if err := validateApprovedImplementationMutationWriteMode(writeMode); err != nil {
		return "", "", nil, err
	}
	return targetPath, writeMode, content, nil
}

func approvedImplementationMutationContentSource(decoded map[string]any) (string, string, error) {
	contentRef := strings.TrimSpace(optionalStringFromMap(decoded, "content_artifact_digest"))
	contentText := requiredStringFromMap(decoded, "content")
	if contentText == "" && contentRef == "" {
		return "", "", fmt.Errorf("approved implementation mutation artifact missing content")
	}
	if contentText != "" && contentRef != "" {
		return "", "", fmt.Errorf("approved implementation mutation artifact must not include both content and content_artifact_digest")
	}
	return contentText, contentRef, nil
}

func (s *Service) resolveApprovedImplementationMutationContent(targetPath, contentText, contentRef, contentDigest string) ([]byte, error) {
	if contentRef == "" {
		return []byte(contentText), nil
	}
	if s == nil {
		return nil, fmt.Errorf("approved implementation mutation artifact content_artifact_digest requires broker service")
	}
	payload, err := s.readArtifactPayloadVerified(contentRef)
	if err != nil {
		return nil, fmt.Errorf("read approved implementation content artifact %q: %w", contentRef, err)
	}
	content := append([]byte(nil), payload...)
	if artifacts.DigestBytes(content) != contentDigest {
		return nil, fmt.Errorf("approved implementation mutation artifact content_digest drift for %q", targetPath)
	}
	return content, nil
}

func validateApprovedImplementationMutationWriteMode(writeMode string) error {
	if writeMode == "" {
		return fmt.Errorf("approved implementation mutation artifact missing write_mode")
	}
	if writeMode != "update" && writeMode != "create" {
		return fmt.Errorf("approved implementation mutation artifact write_mode %q is unsupported", writeMode)
	}
	return nil
}

func validateApprovedImplementationMutationArtifactShape(decoded map[string]any) error {
	if err := validateApprovedImplementationMutationArtifactFields(decoded); err != nil {
		return err
	}
	if err := validateApprovedImplementationMutationArtifactRequiredStrings(decoded); err != nil {
		return err
	}
	return validateApprovedImplementationMutationArtifactOptionalStrings(decoded)
}

func validateApprovedImplementationMutationArtifactFields(decoded map[string]any) error {
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
	return nil
}

func validateApprovedImplementationMutationArtifactRequiredStrings(decoded map[string]any) error {
	for _, key := range []string{"target_path", "write_mode"} {
		if _, ok := decoded[key].(string); !ok {
			return fmt.Errorf("approved implementation mutation artifact %s must be a string", key)
		}
	}
	return nil
}

func validateApprovedImplementationMutationArtifactOptionalStrings(decoded map[string]any) error {
	if raw, ok := decoded["content"]; ok {
		if _, ok := raw.(string); !ok {
			return fmt.Errorf("approved implementation mutation artifact content must be a string")
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
