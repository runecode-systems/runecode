package launcherbackend

func DeriveAttestationPosture(receipt BackendLaunchReceipt) (string, []string) {
	normalized := receipt.Normalized()
	coarse := normalizeProvisioningPosture(normalized.ProvisioningPosture)
	switch coarse {
	case ProvisioningPostureTOFU:
		return AttestationPostureTOFUOnly, nil
	case ProvisioningPostureNotApplicable:
		return AttestationPostureNotApplicable, nil
	case ProvisioningPostureUnknown:
		return AttestationPostureUnknown, nil
	case ProvisioningPostureAttested:
		return deriveRequiredAttestationPosture(normalized)
	default:
		return AttestationPostureUnknown, nil
	}
}

func deriveRequiredAttestationPosture(receipt BackendLaunchReceipt) (string, []string) {
	if receipt.AttestationVerificationResult == AttestationVerificationResultValid && receipt.AttestationReplayVerdict == AttestationReplayVerdictOriginal {
		return AttestationPostureValid, nil
	}
	reasons := attestationPostureReasonCodes(receipt)
	if receipt.AttestationEvidenceSourceKind == AttestationSourceKindUnknown || receipt.AttestationMeasurementProfile == "" {
		return AttestationPostureUnavailable, append(reasons, "attestation_evidence_unavailable")
	}
	if receipt.AttestationVerificationResult == AttestationVerificationResultUnknown {
		return AttestationPostureUnavailable, append(reasons, "attestation_verification_unavailable")
	}
	return AttestationPostureInvalid, reasons
}

func attestationPostureReasonCodes(receipt BackendLaunchReceipt) []string {
	reasons := sanitizedAttestationReasonCodes(receipt.AttestationVerificationReasonCodes)
	if receipt.AttestationReplayVerdict == AttestationReplayVerdictReplay {
		reasons = append(reasons, "attestation_replay_detected")
	}
	return uniqueSortedStrings(reasons)
}

func DeriveAttestationPostureFromEvidence(evidence RuntimeEvidenceSnapshot) (string, []string) {
	receipt := BackendLaunchReceipt{
		ProvisioningPosture:                evidence.Launch.ProvisioningPosture,
		AttestationEvidenceSourceKind:      AttestationSourceKindUnknown,
		AttestationMeasurementProfile:      "",
		AttestationVerificationResult:      AttestationVerificationResultUnknown,
		AttestationVerificationReasonCodes: nil,
		AttestationReplayVerdict:           AttestationReplayVerdictUnknown,
	}
	if evidence.Attestation != nil {
		receipt.AttestationEvidenceSourceKind = evidence.Attestation.AttestationSourceKind
		receipt.AttestationMeasurementProfile = evidence.Attestation.MeasurementProfile
	}
	if evidence.AttestationVerification != nil {
		receipt.AttestationVerificationResult = evidence.AttestationVerification.VerificationResult
		receipt.AttestationVerificationReasonCodes = evidence.AttestationVerification.ReasonCodes
		receipt.AttestationReplayVerdict = evidence.AttestationVerification.ReplayVerdict
	}
	return DeriveAttestationPosture(receipt)
}

func sanitizedAttestationReasonCodes(reasonCodes []string) []string {
	if len(reasonCodes) == 0 {
		return nil
	}
	allowed := map[string]struct{}{
		"attestation_replay_detected":            {},
		"attestation_source_kind_invalid":        {},
		"attestation_measurement_digest_invalid": {},
		"attestation_freshness_material_missing": {},
		"attestation_freshness_binding_missing":  {},
		"attestation_freshness_stale":            {},
		"attestation_evidence_required":          {},
		"attestation_verification_required":      {},
		"attestation_verification_not_valid":     {},
		"attestation_evidence_unavailable":       {},
		"attestation_verification_unavailable":   {},
	}
	sanitized := make([]string, 0, len(reasonCodes))
	for _, reason := range reasonCodes {
		if _, ok := allowed[reason]; ok {
			sanitized = append(sanitized, reason)
			continue
		}
		sanitized = append(sanitized, "attestation_verification_failed")
	}
	return uniqueSortedStrings(sanitized)
}
