package projectsubstrate

import "strings"

const (
	CompatibilityPostureMissing              = "missing"
	CompatibilityPostureInvalid              = "invalid"
	CompatibilityPostureNonVerified          = "non_verified"
	CompatibilityPostureSupportedCurrent     = "supported_current"
	CompatibilityPostureSupportedWithUpgrade = "supported_with_upgrade_available"
	CompatibilityPostureUnsupportedTooOld    = "unsupported_too_old"
	CompatibilityPostureUnsupportedTooNew    = "unsupported_too_new"
	compatibilityReasonMissing               = "project_substrate_missing"
	compatibilityReasonInvalid               = "project_substrate_invalid"
	compatibilityReasonNonVerified           = "project_substrate_non_verified"
	compatibilityReasonUnsupportedContract   = "project_substrate_contract_unsupported"
	compatibilityReasonUnsupportedTooOld     = "project_substrate_unsupported_too_old"
	compatibilityReasonUnsupportedTooNew     = "project_substrate_unsupported_too_new"
	compatibilityReasonUpgradeAvailable      = "project_substrate_upgrade_available"
	compatibilityReasonVersionUnparseable    = "project_substrate_version_unparseable"
	releaseSupportedRuneContextVersionMin    = "0.1.0-alpha.13"
	releaseSupportedRuneContextVersionMax    = "0.1.0-alpha.16"
	releaseRecommendedRuneContextVersion     = "0.1.0-alpha.14"
)

type CompatibilityPolicy struct {
	SupportedContractID             string `json:"supported_contract_id"`
	SupportedContractVersionMin     string `json:"supported_contract_version_min"`
	SupportedContractVersionMax     string `json:"supported_contract_version_max"`
	RecommendedContractVersion      string `json:"recommended_contract_version"`
	SupportedRuneContextVersionMin  string `json:"supported_runecontext_version_min"`
	SupportedRuneContextVersionMax  string `json:"supported_runecontext_version_max"`
	RecommendedRuneContextVersion   string `json:"recommended_runecontext_version"`
	DiagnosticsLocalRunecodeVersion string `json:"diagnostics_local_runecode_version,omitempty"`
	DiagnosticsLocalRunectxVersion  string `json:"diagnostics_local_runectx_version,omitempty"`
}

type CompatibilityAssessment struct {
	Posture                string              `json:"posture"`
	ReasonCodes            []string            `json:"reason_codes,omitempty"`
	BlockedReasonCodes     []string            `json:"blocked_reason_codes,omitempty"`
	NormalOperationAllowed bool                `json:"normal_operation_allowed"`
	Policy                 CompatibilityPolicy `json:"policy"`
}

func ReleaseCompatibilityPolicy() CompatibilityPolicy {
	return CompatibilityPolicy{
		SupportedContractID:            ContractIDV0,
		SupportedContractVersionMin:    ContractVersionV0,
		SupportedContractVersionMax:    ContractVersionV0,
		RecommendedContractVersion:     ContractVersionV0,
		SupportedRuneContextVersionMin: releaseSupportedRuneContextVersionMin,
		SupportedRuneContextVersionMax: releaseSupportedRuneContextVersionMax,
		RecommendedRuneContextVersion:  releaseRecommendedRuneContextVersion,
	}
}

func EvaluateCompatibility(snapshot ValidationSnapshot) CompatibilityAssessment {
	return EvaluateCompatibilityWithPolicy(snapshot, ReleaseCompatibilityPolicy())
}

func EvaluateCompatibilityWithPolicy(snapshot ValidationSnapshot, policy CompatibilityPolicy) CompatibilityAssessment {
	assessment := CompatibilityAssessment{Policy: policy}
	if posture, reasons, done := compatibilityForNonReadySnapshot(snapshot); done {
		assessment.Posture = posture
		return finalizeCompatibilityAssessment(assessment, reasons)
	}
	if unsupportedContract(snapshot, policy) {
		assessment.Posture = CompatibilityPostureInvalid
		return finalizeCompatibilityAssessment(assessment, []string{compatibilityReasonUnsupportedContract})
	}
	posture, reasons := compatibilityForVersion(snapshot.RuneContextVersion, policy)
	assessment.Posture = posture
	return finalizeCompatibilityAssessment(assessment, reasons)
}

func compatibilityForNonReadySnapshot(snapshot ValidationSnapshot) (string, []string, bool) {
	if snapshot.ValidationState == validationStateMissing {
		return CompatibilityPostureMissing, []string{compatibilityReasonMissing}, true
	}
	if containsString(snapshot.ReasonCodes, reasonNonVerifiedPosture) {
		return CompatibilityPostureNonVerified, []string{compatibilityReasonNonVerified}, true
	}
	if snapshot.ValidationState != validationStateValid {
		return CompatibilityPostureInvalid, []string{compatibilityReasonInvalid}, true
	}
	return "", nil, false
}

func unsupportedContract(snapshot ValidationSnapshot, policy CompatibilityPolicy) bool {
	return strings.TrimSpace(snapshot.Contract.ContractID) != policy.SupportedContractID || !isContractVersionSupported(snapshot.Contract.ContractVersion, policy)
}

func compatibilityForVersion(version string, policy CompatibilityPolicy) (string, []string) {
	actualVersion := strings.TrimSpace(version)
	cmpMin, minErr := compareVersion(actualVersion, policy.SupportedRuneContextVersionMin)
	cmpMax, maxErr := compareVersion(actualVersion, policy.SupportedRuneContextVersionMax)
	cmpRecommended, recErr := compareVersion(actualVersion, policy.RecommendedRuneContextVersion)
	if minErr != nil || maxErr != nil || recErr != nil {
		return CompatibilityPostureInvalid, []string{compatibilityReasonVersionUnparseable}
	}
	if cmpMin < 0 {
		return CompatibilityPostureUnsupportedTooOld, []string{compatibilityReasonUnsupportedTooOld}
	}
	if cmpMax > 0 {
		return CompatibilityPostureUnsupportedTooNew, []string{compatibilityReasonUnsupportedTooNew}
	}
	if cmpRecommended < 0 {
		return CompatibilityPostureSupportedWithUpgrade, []string{compatibilityReasonUpgradeAvailable}
	}
	return CompatibilityPostureSupportedCurrent, nil
}

func AllowsNormalOperation(posture string) bool {
	switch strings.TrimSpace(posture) {
	case CompatibilityPostureSupportedCurrent, CompatibilityPostureSupportedWithUpgrade:
		return true
	default:
		return false
	}
}

func isContractVersionSupported(contractVersion string, policy CompatibilityPolicy) bool {
	version := strings.TrimSpace(contractVersion)
	if version == "" {
		return false
	}
	min := strings.TrimSpace(policy.SupportedContractVersionMin)
	max := strings.TrimSpace(policy.SupportedContractVersionMax)
	if min == "" || max == "" {
		return version == strings.TrimSpace(policy.RecommendedContractVersion)
	}
	return version >= min && version <= max
}

func finalizeCompatibilityAssessment(assessment CompatibilityAssessment, reasons []string) CompatibilityAssessment {
	assessment.ReasonCodes = normalizeReasonCodes(reasons)
	assessment.NormalOperationAllowed = AllowsNormalOperation(assessment.Posture)
	if !assessment.NormalOperationAllowed {
		assessment.BlockedReasonCodes = append([]string{}, assessment.ReasonCodes...)
	}
	return assessment
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
