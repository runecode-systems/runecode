package artifacts

import (
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
	key := attestationVerificationCacheKey(evidence)
	if key == "" {
		return evidence
	}
	cached, ok := s.state.AttestationVerificationCache[key]
	if !ok {
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
	if strings.TrimSpace(verification.VerifierPolicyDigest) == "" && strings.TrimSpace(verification.VerifierPolicyID) == "" &&
		strings.TrimSpace(verification.VerificationResult) == "" &&
		strings.TrimSpace(verification.ReplayVerdict) == "" &&
		len(verification.ReasonCodes) == 0 {
		return true
	}
	return false
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
	if attestationEvidenceDigest == "" || authorityStateDigest == "" || measurementProfile == "" {
		return ""
	}
	return attestationEvidenceDigest + "|" + authorityStateDigest + "|" + measurementProfile
}

func cloneAttestationVerificationRecord(in launcherbackend.IsolateAttestationVerificationRecord) launcherbackend.IsolateAttestationVerificationRecord {
	out := in
	out.ReasonCodes = uniqueSortedStrings(in.ReasonCodes)
	out.DerivedMeasurementDigests = uniqueSortedStrings(in.DerivedMeasurementDigests)
	return out
}
