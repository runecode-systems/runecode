package brokerapi

import (
	"fmt"
	"strings"
)

func hasGateBinding(gateID, gateKind, gateVersion, gateAttemptID, gateState string, normalized []string) bool {
	return strings.TrimSpace(gateID) != "" || strings.TrimSpace(gateKind) != "" || strings.TrimSpace(gateVersion) != "" || strings.TrimSpace(gateAttemptID) != "" || strings.TrimSpace(gateState) != "" || len(normalized) > 0
}

func isGateKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "build", "test", "lint", "format", "secret_scan", "policy":
		return true
	default:
		return false
	}
}

func isGateCheckpointState(state string) bool {
	switch strings.TrimSpace(state) {
	case "planned", "running", "passed", "failed", "overridden", "superseded":
		return true
	default:
		return false
	}
}

func isGateResultState(state string) bool {
	switch strings.TrimSpace(state) {
	case "passed", "failed", "overridden", "superseded":
		return true
	default:
		return false
	}
}

func gateStateForCheckpointCode(code string) (string, bool) {
	switch strings.TrimSpace(code) {
	case "gate_planned":
		return "planned", true
	case "gate_started":
		return "running", true
	case "gate_passed":
		return "passed", true
	case "gate_failed":
		return "failed", true
	case "gate_overridden":
		return "overridden", true
	case "gate_superseded":
		return "superseded", true
	default:
		return "", false
	}
}

func gateStateForResultCode(code string) (string, bool) {
	switch strings.TrimSpace(code) {
	case "gate_passed":
		return "passed", true
	case "gate_failed":
		return "failed", true
	case "gate_overridden":
		return "overridden", true
	case "gate_superseded":
		return "superseded", true
	default:
		return "", false
	}
}

func validateNormalizedInputDigests(digests []string) error {
	if len(digests) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, digest := range digests {
		d := strings.TrimSpace(digest)
		if !isValidDigestIdentity(d) {
			return fmt.Errorf("normalized_input_digests contains invalid digest %q", digest)
		}
		if _, ok := seen[d]; ok {
			return fmt.Errorf("normalized_input_digests contains duplicate digest %q", d)
		}
		seen[d] = struct{}{}
	}
	return nil
}

func isValidDigestIdentity(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, ch := range value[len("sha256:"):] {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}
