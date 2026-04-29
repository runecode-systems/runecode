package launcherbackend

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	unique := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		unique[trimmed] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	out := make([]string, 0, len(unique))
	for value := range unique {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func validateSingleRoleToken(field string, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !roleTokenPattern.MatchString(trimmed) {
		return fmt.Errorf("%s must identify exactly one role token", field)
	}
	return nil
}

func validateLaunchIdentityToken(field string, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", field)
	}
	if trimmed == "." || trimmed == ".." || !roleTokenPattern.MatchString(trimmed) {
		return fmt.Errorf("%s must match token pattern and must not contain path traversal material", field)
	}
	return nil
}

func looksLikeHostPath(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if hasAnyPrefix(trimmed, []string{"/", `\\`}) {
		return true
	}
	if hasAnyPrefix(trimmed, []string{"./", "../"}) {
		return true
	}
	if len(trimmed) > 2 && trimmed[1] == ':' && (trimmed[2] == '\\' || trimmed[2] == '/') {
		return true
	}
	if hasAnyPrefix(trimmed, []string{"~/", "~\\"}) {
		return true
	}
	if hasAnyPrefix(trimmed, []string{"$HOME/", "$HOME\\", "${HOME}/", "${HOME}\\"}) {
		return true
	}
	if hasAnyPrefix(trimmed, []string{"$USERPROFILE/", "$USERPROFILE\\", "${USERPROFILE}/", "${USERPROFILE}\\"}) {
		return true
	}
	if isWindowsEnvRootPath(trimmed) {
		return true
	}
	return false
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func isWindowsEnvRootPath(value string) bool {
	if !strings.HasPrefix(value, "%") {
		return false
	}
	end := strings.Index(value[1:], "%")
	if end < 0 {
		return false
	}
	suffix := value[end+2:]
	return strings.HasPrefix(suffix, "/") || strings.HasPrefix(suffix, "\\")
}

func canonicalSHA256Digest(value any, context string) (string, error) {
	canonical, err := canonicalJSONBytes(value, context)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func canonicalJSONBytes(value any, context string) ([]byte, error) {
	blob, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%s serialization failed: %w", context, err)
	}
	canonical, err := jsoncanonicalizer.Transform(blob)
	if err != nil {
		return nil, fmt.Errorf("%s canonicalization failed: %w", context, err)
	}
	return canonical, nil
}

func looksLikeDeviceNumberingMaterial(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return false
	}
	if trimmed == AttachmentChannelVirtualDisk || trimmed == AttachmentChannelReadOnlyChannel || trimmed == AttachmentChannelEphemeralDisk {
		return false
	}
	if strings.Contains(trimmed, "/dev/") {
		return true
	}
	if deviceNumberingPattern.MatchString(trimmed) {
		return true
	}
	if strings.Contains(trimmed, "slot-") || strings.Contains(trimmed, "lun-") {
		return true
	}
	return false
}

func looksLikeDigest(value string) bool {
	return digestPattern.MatchString(strings.TrimSpace(value))
}

func looksLikeHexKeyIDValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 64 {
		return false
	}
	for _, ch := range trimmed {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}
