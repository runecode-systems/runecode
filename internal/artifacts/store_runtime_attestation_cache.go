package artifacts

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Store) upsertAttestationVerificationCacheLocked(evidence launcherbackend.RuntimeEvidenceSnapshot) {
	if s == nil {
		return
	}
	if evidence.AttestationVerification == nil {
		return
	}
	key := attestationVerificationCacheKey(evidence)
	if key == "" {
		return
	}
	normalized := cloneAttestationVerificationRecord(*evidence.AttestationVerification)
	s.state.AttestationVerificationCache[key] = normalized
}

func (s *Store) applyCachedAttestationVerificationLocked(evidence launcherbackend.RuntimeEvidenceSnapshot) launcherbackend.RuntimeEvidenceSnapshot {
	if s == nil || !shouldApplyCachedAttestationVerification(evidence.AttestationVerification) {
		return evidence
	}
	currentEvidenceDigest := attestationVerificationEvidenceDigest(evidence)
	if !isValidDigest(currentEvidenceDigest) {
		return evidence
	}
	key := attestationVerificationCacheKey(evidence)
	if key == "" {
		return evidence
	}
	cached, ok := s.state.AttestationVerificationCache[key]
	if !ok {
		return evidence
	}
	if strings.TrimSpace(cached.AttestationEvidenceDigest) != currentEvidenceDigest {
		return evidence
	}
	normalized := cloneAttestationVerificationRecord(cached)
	evidence.AttestationVerification = &normalized
	return evidence
}

func shouldApplyCachedAttestationVerification(verification *launcherbackend.IsolateAttestationVerificationRecord) bool {
	if verification == nil {
		return true
	}
	return strings.TrimSpace(verification.AttestationEvidenceDigest) == "" &&
		strings.TrimSpace(verification.ReplayIdentityDigest) == "" &&
		strings.TrimSpace(verification.VerifierPolicyDigest) == "" &&
		strings.TrimSpace(verification.VerifierPolicyID) == "" &&
		strings.TrimSpace(verification.VerificationRulesProfileVersion) == "" &&
		strings.TrimSpace(verification.VerificationTimestamp) == "" &&
		strings.TrimSpace(verification.VerificationResult) == "" &&
		strings.TrimSpace(verification.ReplayVerdict) == "" &&
		strings.TrimSpace(verification.VerificationDigest) == "" &&
		len(verification.ReasonCodes) == 0 &&
		len(verification.DerivedMeasurementDigests) == 0
}

func attestationVerificationEvidenceDigest(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	if evidence.Attestation != nil {
		return strings.TrimSpace(evidence.Attestation.EvidenceDigest)
	}
	if evidence.AttestationVerification != nil {
		return strings.TrimSpace(evidence.AttestationVerification.AttestationEvidenceDigest)
	}
	return ""
}

func attestationVerificationCacheKey(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	attestationEvidenceDigest := ""
	measurementProfile := ""
	if evidence.Attestation != nil {
		attestationEvidenceDigest = strings.TrimSpace(evidence.Attestation.EvidenceDigest)
		measurementProfile = strings.TrimSpace(evidence.Attestation.MeasurementProfile)
	}
	if attestationEvidenceDigest == "" && evidence.AttestationVerification != nil {
		attestationEvidenceDigest = strings.TrimSpace(evidence.AttestationVerification.AttestationEvidenceDigest)
	}
	authorityStateDigest := strings.TrimSpace(evidence.Launch.AuthorityStateDigest)
	if !isValidDigest(attestationEvidenceDigest) || !isValidDigest(authorityStateDigest) || measurementProfile == "" {
		return ""
	}
	return attestationVerificationCacheKeyFromFields(attestationEvidenceDigest, authorityStateDigest, measurementProfile)
}

func attestationVerificationCacheKeyFromFields(attestationEvidenceDigest, authorityStateDigest, measurementProfile string) string {
	trimmedEvidenceDigest := strings.TrimSpace(attestationEvidenceDigest)
	trimmedAuthorityDigest := strings.TrimSpace(authorityStateDigest)
	trimmedMeasurementProfile := strings.TrimSpace(measurementProfile)
	if !isValidDigest(trimmedEvidenceDigest) || !isValidDigest(trimmedAuthorityDigest) || trimmedMeasurementProfile == "" {
		return ""
	}
	h := sha256.New()
	writeCacheKeyPart(h, trimmedEvidenceDigest)
	writeCacheKeyPart(h, trimmedAuthorityDigest)
	writeCacheKeyPart(h, trimmedMeasurementProfile)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func writeCacheKeyPart(h interface{ Write([]byte) (int, error) }, value string) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	_, _ = h.Write(size[:])
	_, _ = h.Write([]byte(value))
}

func parseLegacyAttestationVerificationCacheKey(key string) (attestationEvidenceDigest, authorityStateDigest, measurementProfile string, ok bool) {
	parts := strings.Split(strings.TrimSpace(key), "|")
	if len(parts) != 3 {
		return "", "", "", false
	}
	attestationEvidenceDigest = strings.TrimSpace(parts[0])
	authorityStateDigest = strings.TrimSpace(parts[1])
	measurementProfile = strings.TrimSpace(parts[2])
	if !isValidDigest(attestationEvidenceDigest) || !isValidDigest(authorityStateDigest) || measurementProfile == "" {
		return "", "", "", false
	}
	return attestationEvidenceDigest, authorityStateDigest, measurementProfile, true
}

func cloneAttestationVerificationRecord(in launcherbackend.IsolateAttestationVerificationRecord) launcherbackend.IsolateAttestationVerificationRecord {
	out := in
	out.ReasonCodes = uniqueSortedStrings(in.ReasonCodes)
	out.DerivedMeasurementDigests = uniqueSortedStrings(in.DerivedMeasurementDigests)
	return out
}
