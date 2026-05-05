package launcherbackend

import "strings"

func hasIsolateAttestationEvidence(input *PostHandshakeRuntimeAttestationInput) bool {
	if input == nil {
		return false
	}
	return input.RunID != "" && input.IsolateID != "" && input.SessionID != "" &&
		input.RuntimeEvidenceCollected && input.SessionNonce != "" && input.LaunchContextDigest != "" && input.HandshakeTranscriptHash != "" &&
		input.IsolateSessionKeyIDValue != "" && input.RuntimeImageDescriptorDigest != "" &&
		input.RuntimeImageBootProfile != "" && input.AttestationSourceKind != "" && input.AttestationSourceKind != AttestationSourceKindUnknown &&
		input.MeasurementProfile != "" && input.MeasurementProfile != MeasurementProfileUnknown
}

func isolateAttestationEvidenceFromPostHandshakeInput(input PostHandshakeRuntimeAttestationInput, launch LaunchRuntimeEvidence) *IsolateAttestationEvidence {
	return &IsolateAttestationEvidence{
		RunID:                        input.RunID,
		IsolateID:                    input.IsolateID,
		SessionID:                    input.SessionID,
		SessionNonce:                 input.SessionNonce,
		LaunchContextDigest:          input.LaunchContextDigest,
		HandshakeTranscriptHash:      input.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     input.IsolateSessionKeyIDValue,
		LaunchRuntimeEvidenceDigest:  launch.EvidenceDigest,
		RuntimeImageDescriptorDigest: input.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      input.RuntimeImageBootProfile,
		BootComponentDigestByName:    cloneStringMap(input.BootComponentDigestByName),
		BootComponentDigests:         uniqueSortedStrings(input.BootComponentDigests),
		AttestationSourceKind:        input.AttestationSourceKind,
		MeasurementProfile:           input.MeasurementProfile,
		FreshnessMaterial:            uniqueSortedStrings(input.FreshnessMaterial),
		FreshnessBindingClaims:       uniqueSortedStrings(input.FreshnessBindingClaims),
		EvidenceClaimsDigest:         input.EvidenceClaimsDigest,
	}
}

func isolateAttestationEvidenceDigestInput(evidence IsolateAttestationEvidence) isolateAttestationEvidenceDigestFields {
	return isolateAttestationEvidenceDigestFields{
		RunID:                        evidence.RunID,
		IsolateID:                    evidence.IsolateID,
		SessionID:                    evidence.SessionID,
		SessionNonce:                 evidence.SessionNonce,
		LaunchContextDigest:          evidence.LaunchContextDigest,
		HandshakeTranscriptHash:      evidence.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     evidence.IsolateSessionKeyIDValue,
		LaunchRuntimeEvidenceDigest:  evidence.LaunchRuntimeEvidenceDigest,
		RuntimeImageDescriptorDigest: evidence.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      evidence.RuntimeImageBootProfile,
		BootComponentDigestByName:    cloneStringMap(evidence.BootComponentDigestByName),
		BootComponentDigests:         evidence.BootComponentDigests,
		AttestationSourceKind:        evidence.AttestationSourceKind,
		MeasurementProfile:           evidence.MeasurementProfile,
		FreshnessMaterial:            evidence.FreshnessMaterial,
		FreshnessBindingClaims:       evidence.FreshnessBindingClaims,
		EvidenceClaimsDigest:         evidence.EvidenceClaimsDigest,
	}
}

func isolateAttestationVerificationFromPostHandshakeInput(input PostHandshakeRuntimeAttestationInput, launchEvidenceDigest, attestationEvidenceDigest string) (*IsolateAttestationVerificationRecord, error) {
	input = withTrustedAttestationVerificationDefaults(input)
	replayIdentityDigest, err := canonicalSHA256Digest(isolateAttestationReplayIdentityInput(input, launchEvidenceDigest, attestationEvidenceDigest), "isolate attestation replay identity")
	if err != nil {
		return nil, err
	}
	verification := &IsolateAttestationVerificationRecord{
		AttestationEvidenceDigest:       attestationEvidenceDigest,
		ReplayIdentityDigest:            replayIdentityDigest,
		VerifierPolicyID:                input.VerifierPolicyID,
		VerifierPolicyDigest:            input.VerifierPolicyDigest,
		VerificationRulesProfileVersion: input.VerificationRulesProfileVersion,
		VerificationTimestamp:           input.VerificationTimestamp,
		VerificationResult:              input.VerificationResult,
		ReasonCodes:                     uniqueSortedStrings(input.VerificationReasonCodes),
		ReplayVerdict:                   input.ReplayVerdict,
		DerivedMeasurementDigests:       []string{input.EvidenceClaimsDigest},
	}
	if verification.DerivedMeasurementDigests[0] == "" {
		verification.DerivedMeasurementDigests = nil
	}
	digest, err := canonicalSHA256Digest(isolateAttestationVerificationDigestInput(*verification), "isolate attestation verification")
	if err != nil {
		return nil, err
	}
	verification.VerificationDigest = digest
	return verification, nil
}

func trustedVerificationPolicyDigest(input PostHandshakeRuntimeAttestationInput) string {
	if looksLikeDigest(input.VerifierPolicyDigest) {
		return strings.TrimSpace(input.VerifierPolicyDigest)
	}
	if looksLikeDigest(input.AuthorityStateDigest) {
		return strings.TrimSpace(input.AuthorityStateDigest)
	}
	if looksLikeDigest(input.RuntimeImageVerifierRef) {
		return strings.TrimSpace(input.RuntimeImageVerifierRef)
	}
	return ""
}

func isolateAttestationReplayIdentityInput(input PostHandshakeRuntimeAttestationInput, launchEvidenceDigest, attestationEvidenceDigest string) isolateAttestationReplayIdentityFields {
	return isolateAttestationReplayIdentityFields{
		RunID:                     input.RunID,
		IsolateID:                 input.IsolateID,
		SessionID:                 input.SessionID,
		SessionNonce:              input.SessionNonce,
		LaunchContextDigest:       input.LaunchContextDigest,
		HandshakeTranscriptHash:   input.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:  input.IsolateSessionKeyIDValue,
		LaunchEvidenceDigest:      launchEvidenceDigest,
		AttestationEvidenceDigest: attestationEvidenceDigest,
		MeasurementProfile:        input.MeasurementProfile,
	}
}

func isolateAttestationVerificationDigestInput(verification IsolateAttestationVerificationRecord) isolateAttestationVerificationRecordDigestFields {
	return isolateAttestationVerificationRecordDigestFields{
		AttestationEvidenceDigest:       verification.AttestationEvidenceDigest,
		ReplayIdentityDigest:            verification.ReplayIdentityDigest,
		VerifierPolicyID:                verification.VerifierPolicyID,
		VerifierPolicyDigest:            verification.VerifierPolicyDigest,
		VerificationRulesProfileVersion: verification.VerificationRulesProfileVersion,
		VerificationTimestamp:           verification.VerificationTimestamp,
		VerificationResult:              verification.VerificationResult,
		ReasonCodes:                     verification.ReasonCodes,
		ReplayVerdict:                   verification.ReplayVerdict,
		DerivedMeasurementDigests:       verification.DerivedMeasurementDigests,
	}
}

func attestationIdentityMatchesLaunchEvidence(launch LaunchRuntimeEvidence, attestation *IsolateAttestationEvidence) bool {
	if attestation == nil {
		return false
	}
	if attestation.RunID != launch.RunID || attestation.IsolateID != launch.IsolateID || attestation.SessionID != launch.SessionID || attestation.SessionNonce != launch.SessionNonce {
		return false
	}
	if attestation.LaunchContextDigest != launch.LaunchContextDigest || attestation.HandshakeTranscriptHash != launch.HandshakeTranscriptHash || attestation.IsolateSessionKeyIDValue != launch.IsolateSessionKeyIDValue {
		return false
	}
	if attestation.RuntimeImageDescriptorDigest != launch.RuntimeImageDescriptorDigest || attestation.RuntimeImageBootProfile != launch.RuntimeImageBootProfile {
		return false
	}
	if (len(launch.BootComponentDigestByName) > 0 || len(attestation.BootComponentDigestByName) > 0) && !stringMapsEqual(attestation.BootComponentDigestByName, launch.BootComponentDigestByName) {
		return false
	}
	return true
}

func stringMapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if rightValue, ok := right[key]; !ok || rightValue != leftValue {
			return false
		}
	}
	return true
}

func finalizeAttestationVerificationRecord(evidence *RuntimeEvidenceSnapshot) error {
	if evidence == nil || evidence.AttestationVerification == nil {
		return nil
	}
	verification := evidence.AttestationVerification
	if verification.AttestationEvidenceDigest == "" && evidence.Attestation != nil {
		verification.AttestationEvidenceDigest = evidence.Attestation.EvidenceDigest
	}
	if verification.ReplayIdentityDigest == "" && evidence.Attestation != nil {
		replayIdentityDigest, err := canonicalSHA256Digest(isolateAttestationReplayIdentityFields{
			RunID:                     evidence.Attestation.RunID,
			IsolateID:                 evidence.Attestation.IsolateID,
			SessionID:                 evidence.Attestation.SessionID,
			SessionNonce:              evidence.Attestation.SessionNonce,
			LaunchContextDigest:       evidence.Attestation.LaunchContextDigest,
			HandshakeTranscriptHash:   evidence.Attestation.HandshakeTranscriptHash,
			IsolateSessionKeyIDValue:  evidence.Attestation.IsolateSessionKeyIDValue,
			LaunchEvidenceDigest:      evidence.Attestation.LaunchRuntimeEvidenceDigest,
			AttestationEvidenceDigest: evidence.Attestation.EvidenceDigest,
			MeasurementProfile:        evidence.Attestation.MeasurementProfile,
		}, "isolate attestation replay identity")
		if err != nil {
			return err
		}
		verification.ReplayIdentityDigest = replayIdentityDigest
	}
	return FinalizeIsolateAttestationVerificationRecord(verification)
}

func FinalizeIsolateAttestationVerificationRecord(record *IsolateAttestationVerificationRecord) error {
	if record == nil {
		return nil
	}
	record.ReasonCodes = uniqueSortedStrings(record.ReasonCodes)
	record.DerivedMeasurementDigests = uniqueSortedStrings(record.DerivedMeasurementDigests)
	return assignAttestationVerificationDigest(record)
}

func assignAttestationVerificationDigest(record *IsolateAttestationVerificationRecord) error {
	digest, err := canonicalSHA256Digest(isolateAttestationVerificationDigestInput(*record), "isolate attestation verification")
	if err != nil {
		return err
	}
	record.VerificationDigest = digest
	return nil
}
