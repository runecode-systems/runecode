package artifacts

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func loadRestoredAttestationVerificationCache(next *StoreState, verificationCache map[string]launcherbackend.IsolateAttestationVerificationRecord) error {
	for key, record := range verificationCache {
		normalizedKey, err := normalizeAttestationVerificationCacheRestoreKey(key)
		if err != nil {
			return err
		}
		normalized := cloneAttestationVerificationRecord(record)
		if err := validateRestoredAttestationVerificationRecord(normalized); err != nil {
			return fmt.Errorf("attestation verification cache key %q: %w", normalizedKey, err)
		}
		if existing, ok := next.AttestationVerificationCache[normalizedKey]; ok {
			if !attestationVerificationRecordsEquivalent(existing, normalized) {
				return fmt.Errorf("attestation verification cache key collision for %q", normalizedKey)
			}
			continue
		}
		next.AttestationVerificationCache[normalizedKey] = normalized
	}
	return nil
}

func normalizeAttestationVerificationCacheRestoreKey(key string) (string, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", fmt.Errorf("attestation verification cache key is required")
	}
	if strings.Contains(trimmedKey, "|") {
		attestationEvidenceDigest, authorityStateDigest, measurementProfile, ok := parseLegacyAttestationVerificationCacheKey(trimmedKey)
		if !ok {
			return "", fmt.Errorf("attestation verification cache key must be structurally valid")
		}
		normalized := attestationVerificationCacheKeyFromFields(attestationEvidenceDigest, authorityStateDigest, measurementProfile)
		if normalized == "" {
			return "", fmt.Errorf("attestation verification cache key must be structurally valid")
		}
		return normalized, nil
	}
	if !isValidDigest(trimmedKey) {
		return "", fmt.Errorf("attestation verification cache key must be a sha256 digest")
	}
	return trimmedKey, nil
}

func validateRestoredAttestationVerificationRecord(record launcherbackend.IsolateAttestationVerificationRecord) error {
	return launcherbackend.ValidateIsolateAttestationVerificationRecord(record)
}

func attestationVerificationRecordsEquivalent(left, right launcherbackend.IsolateAttestationVerificationRecord) bool {
	return strings.TrimSpace(left.AttestationEvidenceDigest) == strings.TrimSpace(right.AttestationEvidenceDigest) &&
		strings.TrimSpace(left.ReplayIdentityDigest) == strings.TrimSpace(right.ReplayIdentityDigest) &&
		strings.TrimSpace(left.VerifierPolicyID) == strings.TrimSpace(right.VerifierPolicyID) &&
		strings.TrimSpace(left.VerifierPolicyDigest) == strings.TrimSpace(right.VerifierPolicyDigest) &&
		strings.TrimSpace(left.VerificationRulesProfileVersion) == strings.TrimSpace(right.VerificationRulesProfileVersion) &&
		strings.TrimSpace(left.VerificationTimestamp) == strings.TrimSpace(right.VerificationTimestamp) &&
		strings.TrimSpace(left.VerificationResult) == strings.TrimSpace(right.VerificationResult) &&
		strings.TrimSpace(left.ReplayVerdict) == strings.TrimSpace(right.ReplayVerdict) &&
		strings.TrimSpace(left.VerificationDigest) == strings.TrimSpace(right.VerificationDigest) &&
		stringSlicesEqual(left.ReasonCodes, right.ReasonCodes) &&
		stringSlicesEqual(left.DerivedMeasurementDigests, right.DerivedMeasurementDigests)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if strings.TrimSpace(left[i]) != strings.TrimSpace(right[i]) {
			return false
		}
	}
	return true
}
