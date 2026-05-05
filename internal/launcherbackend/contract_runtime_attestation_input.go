package launcherbackend

import "strings"

func NormalizePostHandshakeRuntimeAttestationInput(input *PostHandshakeRuntimeAttestationInput) *PostHandshakeRuntimeAttestationInput {
	if input == nil {
		return nil
	}
	out := *input
	out.RunID = strings.TrimSpace(out.RunID)
	out.IsolateID = strings.TrimSpace(out.IsolateID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.SessionNonce = strings.TrimSpace(out.SessionNonce)
	out.LaunchContextDigest = strings.TrimSpace(out.LaunchContextDigest)
	out.HandshakeTranscriptHash = strings.TrimSpace(out.HandshakeTranscriptHash)
	out.IsolateSessionKeyIDValue = strings.TrimSpace(out.IsolateSessionKeyIDValue)
	out.RuntimeImageDescriptorDigest = strings.TrimSpace(out.RuntimeImageDescriptorDigest)
	out.RuntimeImageBootProfile = normalizeBootProfile(out.RuntimeImageBootProfile)
	out.RuntimeImageVerifierRef = strings.TrimSpace(out.RuntimeImageVerifierRef)
	out.AuthorityStateDigest = strings.TrimSpace(out.AuthorityStateDigest)
	out.BootComponentDigestByName = cloneStringMap(out.BootComponentDigestByName)
	out.BootComponentDigests = uniqueSortedStrings(out.BootComponentDigests)
	if len(out.BootComponentDigests) == 0 && len(out.BootComponentDigestByName) > 0 {
		out.BootComponentDigests = make([]string, 0, len(out.BootComponentDigestByName))
		for _, digest := range out.BootComponentDigestByName {
			out.BootComponentDigests = append(out.BootComponentDigests, digest)
		}
		out.BootComponentDigests = uniqueSortedStrings(out.BootComponentDigests)
	}
	out.AttestationSourceKind = normalizeAttestationSourceKind(out.AttestationSourceKind)
	out.MeasurementProfile = normalizeMeasurementProfile(out.MeasurementProfile)
	out.FreshnessMaterial = uniqueSortedStrings(out.FreshnessMaterial)
	out.FreshnessBindingClaims = uniqueSortedStrings(out.FreshnessBindingClaims)
	out.EvidenceClaimsDigest = strings.TrimSpace(out.EvidenceClaimsDigest)
	out.VerifierPolicyID = strings.TrimSpace(out.VerifierPolicyID)
	out.VerifierPolicyDigest = strings.TrimSpace(out.VerifierPolicyDigest)
	out.VerificationRulesProfileVersion = strings.TrimSpace(out.VerificationRulesProfileVersion)
	out.VerificationTimestamp = strings.TrimSpace(out.VerificationTimestamp)
	out.VerificationResult = normalizeAttestationVerificationResult(out.VerificationResult)
	out.VerificationReasonCodes = uniqueSortedStrings(out.VerificationReasonCodes)
	out.ReplayVerdict = normalizeAttestationReplayVerdict(out.ReplayVerdict)
	return &out
}
