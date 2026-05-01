package launcherbackend

const (
	AttestationVerifierClassUnknown            = "unknown"
	AttestationVerifierClassTrustedDomainLocal = "trusted_domain_local"
	AttestationVerifierClassHardwareRooted     = "hardware_rooted"
	AttestationVerifierClassExternalVerifier   = "external_verifier"
)

func DeriveAttestationVerifierClass(receipt BackendLaunchReceipt) string {
	switch normalizeAttestationSourceKind(receipt.Normalized().AttestationEvidenceSourceKind) {
	case AttestationSourceKindTrustedRuntime:
		return AttestationVerifierClassTrustedDomainLocal
	case AttestationSourceKindTPMQuote, AttestationSourceKindSEVSNPReport, AttestationSourceKindTDXQuote:
		return AttestationVerifierClassHardwareRooted
	case AttestationSourceKindContainerImage:
		return AttestationVerifierClassExternalVerifier
	default:
		return AttestationVerifierClassUnknown
	}
}

func DeriveAttestationVerifierClassFromEvidence(evidence RuntimeEvidenceSnapshot) string {
	if evidence.Attestation == nil {
		return AttestationVerifierClassUnknown
	}
	return DeriveAttestationVerifierClass(BackendLaunchReceipt{AttestationEvidenceSourceKind: evidence.Attestation.AttestationSourceKind})
}
