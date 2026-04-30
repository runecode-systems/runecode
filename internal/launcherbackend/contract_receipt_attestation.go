package launcherbackend

import "strings"

const (
	AttestationSourceKindUnknown            = "unknown"
	AttestationSourceKindTrustedRuntime     = "trusted_runtime"
	AttestationSourceKindTPMQuote           = "tpm_quote"
	AttestationSourceKindSEVSNPReport       = "sev_snp_report"
	AttestationSourceKindTDXQuote           = "tdx_quote"
	AttestationSourceKindContainerImage     = "container_image"
	legacyAttestationSourceKindContainerSig = "container_signature"

	AttestationVerificationResultUnknown = "unknown"
	AttestationVerificationResultValid   = "valid"
	AttestationVerificationResultInvalid = "invalid"

	AttestationReplayVerdictUnknown  = "unknown"
	AttestationReplayVerdictOriginal = "original"
	AttestationReplayVerdictReplay   = "replay"
)

func normalizeReceiptAttestationFields(receipt *BackendLaunchReceipt) {
	receipt.AttestationEvidenceSourceKind = normalizeAttestationSourceKind(receipt.AttestationEvidenceSourceKind)
	receipt.AttestationMeasurementProfile = strings.TrimSpace(receipt.AttestationMeasurementProfile)
	receipt.AttestationFreshnessMaterial = uniqueSortedStrings(receipt.AttestationFreshnessMaterial)
	receipt.AttestationFreshnessBindingClaims = uniqueSortedStrings(receipt.AttestationFreshnessBindingClaims)
	receipt.AttestationEvidenceClaimsDigest = strings.TrimSpace(receipt.AttestationEvidenceClaimsDigest)
	receipt.AttestationEvidenceDigest = strings.TrimSpace(receipt.AttestationEvidenceDigest)
	receipt.AttestationVerifierPolicyID = strings.TrimSpace(receipt.AttestationVerifierPolicyID)
	receipt.AttestationVerifierPolicyDigest = strings.TrimSpace(receipt.AttestationVerifierPolicyDigest)
	receipt.AttestationVerificationRulesVersion = strings.TrimSpace(receipt.AttestationVerificationRulesVersion)
	receipt.AttestationVerificationResult = normalizeAttestationVerificationResult(receipt.AttestationVerificationResult)
	receipt.AttestationVerificationReasonCodes = uniqueSortedStrings(receipt.AttestationVerificationReasonCodes)
	receipt.AttestationReplayVerdict = normalizeAttestationReplayVerdict(receipt.AttestationReplayVerdict)
	receipt.AttestationVerificationTimestamp = strings.TrimSpace(receipt.AttestationVerificationTimestamp)
	receipt.AttestationVerificationDigest = strings.TrimSpace(receipt.AttestationVerificationDigest)
}

func normalizeAttestationSourceKind(sourceKind string) string {
	switch strings.ToLower(strings.TrimSpace(sourceKind)) {
	case AttestationSourceKindTrustedRuntime:
		return AttestationSourceKindTrustedRuntime
	case AttestationSourceKindTPMQuote:
		return AttestationSourceKindTPMQuote
	case AttestationSourceKindSEVSNPReport:
		return AttestationSourceKindSEVSNPReport
	case AttestationSourceKindTDXQuote:
		return AttestationSourceKindTDXQuote
	case AttestationSourceKindContainerImage, legacyAttestationSourceKindContainerSig:
		return AttestationSourceKindContainerImage
	default:
		return AttestationSourceKindUnknown
	}
}

func normalizeAttestationVerificationResult(result string) string {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case AttestationVerificationResultValid:
		return AttestationVerificationResultValid
	case AttestationVerificationResultInvalid:
		return AttestationVerificationResultInvalid
	default:
		return AttestationVerificationResultUnknown
	}
}

func normalizeAttestationReplayVerdict(verdict string) string {
	switch strings.ToLower(strings.TrimSpace(verdict)) {
	case AttestationReplayVerdictOriginal:
		return AttestationReplayVerdictOriginal
	case AttestationReplayVerdictReplay:
		return AttestationReplayVerdictReplay
	default:
		return AttestationReplayVerdictUnknown
	}
}
