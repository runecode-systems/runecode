package brokerapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) trustedLLMDestinationRefForRun(runID string) (string, error) {
	destinationRef, _, err := s.trustedLLMDestinationForRun(runID)
	if err != nil {
		return "", err
	}
	return destinationRef, nil
}

func (s *Service) trustedLLMDestinationForRun(runID string) (string, gatewayAllowlistMatch, error) {
	runtime := policyRuntime{service: s}
	compileInput, err := runtime.loadCompileInput(strings.TrimSpace(runID))
	if err != nil {
		return "", gatewayAllowlistMatch{}, err
	}
	return resolveLLMDestinationFromAllowlists(compileInput.Allowlists)
}

func resolveLLMDestinationRefFromAllowlists(allowlists []policyengine.ManifestInput) (string, error) {
	ref, _, err := resolveLLMDestinationFromAllowlists(allowlists)
	if err != nil {
		return "", err
	}
	return ref, nil
}

func resolveLLMDestinationFromAllowlists(allowlists []policyengine.ManifestInput) (string, gatewayAllowlistMatch, error) {
	for _, allowlistInput := range allowlists {
		expectedHash, err := validateTrustedAllowlistInputHash(allowlistInput)
		if err != nil {
			return "", gatewayAllowlistMatch{}, err
		}
		allowlist := policyengine.PolicyAllowlist{}
		if err := json.Unmarshal(allowlistInput.Payload, &allowlist); err != nil {
			return "", gatewayAllowlistMatch{}, fmt.Errorf("decode trusted allowlist payload: %w", err)
		}
		for _, entry := range allowlist.Entries {
			if !entrySupportsLLMInvoke(entry) {
				continue
			}
			return destinationRefFromDescriptor(entry.Destination), gatewayAllowlistMatch{AllowlistRef: expectedHash, EntryID: entry.EntryID}, nil
		}
	}
	return "", gatewayAllowlistMatch{}, fmt.Errorf("trusted model gateway destination unavailable")
}

func validateTrustedAllowlistInputHash(input policyengine.ManifestInput) (string, error) {
	expected := strings.TrimSpace(input.ExpectedHash)
	if expected == "" {
		return "", fmt.Errorf("trusted allowlist expected hash missing")
	}
	if artifacts.DigestBytes(input.Payload) != expected {
		return "", fmt.Errorf("trusted allowlist payload hash mismatch")
	}
	return expected, nil
}

func entrySupportsLLMInvoke(entry policyengine.GatewayScopeRule) bool {
	if entry.ScopeKind != "gateway_destination" {
		return false
	}
	if !isHardenedModelDestination(entry.Destination) {
		return false
	}
	roleKind := strings.TrimSpace(entry.GatewayRoleKind)
	if roleKind != "" && roleKind != "model-gateway" {
		return false
	}
	for _, operation := range entry.PermittedOperations {
		if operation == "invoke_model" {
			return true
		}
	}
	return false
}

func isHardenedModelDestination(destination policyengine.DestinationDescriptor) bool {
	if destination.DescriptorKind != "model_endpoint" {
		return false
	}
	if strings.TrimSpace(destination.CanonicalHost) == "" {
		return false
	}
	if !isSafeDestinationPathPrefix(destination.CanonicalPathPrefix) {
		return false
	}
	if !destination.TLSRequired {
		return false
	}
	if destination.PrivateRangeBlocking != "enforced" {
		return false
	}
	if destination.DNSRebindingProtection != "enforced" {
		return false
	}
	return true
}

func isSafeDestinationPathPrefix(rawPath string) bool {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return true
	}
	decoded, err := url.PathUnescape(trimmed)
	if err != nil {
		return false
	}
	if strings.Contains(decoded, "..") {
		return false
	}
	normalized := normalizeDestinationPathPrefix(decoded)
	return strings.HasPrefix(normalized, "/") && !strings.Contains(normalized, "..")
}

func destinationRefFromDescriptor(descriptor policyengine.DestinationDescriptor) string {
	ref := strings.TrimSpace(descriptor.CanonicalHost)
	if descriptor.CanonicalPort != nil && *descriptor.CanonicalPort != 443 {
		ref = fmt.Sprintf("%s:%d", ref, *descriptor.CanonicalPort)
	}
	return ref + normalizeDestinationPathPrefix(descriptor.CanonicalPathPrefix)
}

func normalizeDestinationPathPrefix(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	normalized := path.Clean(trimmed)
	if normalized == "." {
		return "/"
	}
	return normalized
}
