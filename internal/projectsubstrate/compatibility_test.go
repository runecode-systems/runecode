package projectsubstrate

import "testing"

func TestEvaluateCompatibilityPostures(t *testing.T) {
	policy := CompatibilityPolicy{
		SupportedContractID:            ContractIDV0,
		SupportedContractVersionMin:    ContractVersionV0,
		SupportedContractVersionMax:    ContractVersionV0,
		RecommendedContractVersion:     ContractVersionV0,
		SupportedRuneContextVersionMin: "0.1.0-alpha.13",
		SupportedRuneContextVersionMax: "0.1.0-alpha.16",
		RecommendedRuneContextVersion:  "0.1.0-alpha.14",
	}
	for _, tt := range compatibilityPostureTests() {
		t.Run(tt.name, func(t *testing.T) {
			assertCompatibilityAssessment(t, EvaluateCompatibilityWithPolicy(tt.snapshot, policy), tt)
		})
	}
}

func assertCompatibilityAssessment(t *testing.T, assessment CompatibilityAssessment, tt compatibilityPostureTestCase) {
	t.Helper()
	if assessment.Posture != tt.wantPosture {
		t.Fatalf("posture = %q, want %q", assessment.Posture, tt.wantPosture)
	}
	assertCompatibilityBlocking(t, assessment, tt.wantBlocked)
	assertCompatibilityReason(t, assessment, tt.wantReason)
}

func assertCompatibilityBlocking(t *testing.T, assessment CompatibilityAssessment, wantBlocked bool) {
	t.Helper()
	if assessment.NormalOperationAllowed == wantBlocked {
		t.Fatalf("normal_operation_allowed = %t, expected blocked=%t", assessment.NormalOperationAllowed, wantBlocked)
	}
	if wantBlocked && len(assessment.BlockedReasonCodes) == 0 {
		t.Fatal("blocked_reason_codes empty for blocked posture")
	}
}

func assertCompatibilityReason(t *testing.T, assessment CompatibilityAssessment, wantReason string) {
	t.Helper()
	if wantReason != "" && !containsString(assessment.ReasonCodes, wantReason) {
		t.Fatalf("reason_codes = %v, want %q", assessment.ReasonCodes, wantReason)
	}
}

type compatibilityPostureTestCase struct {
	name        string
	snapshot    ValidationSnapshot
	wantPosture string
	wantBlocked bool
	wantReason  string
}

func compatibilityPostureTests() []compatibilityPostureTestCase {
	return append(compatibilityInvalidPostureTests(), compatibilityVersionPostureTests()...)
}

func compatibilityInvalidPostureTests() []compatibilityPostureTestCase {
	return []compatibilityPostureTestCase{
		{name: "missing posture", snapshot: ValidationSnapshot{ValidationState: validationStateMissing, Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureMissing, wantBlocked: true, wantReason: compatibilityReasonMissing},
		{name: "invalid posture", snapshot: ValidationSnapshot{ValidationState: validationStateInvalid, Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureInvalid, wantBlocked: true, wantReason: compatibilityReasonInvalid},
		{name: "non verified posture", snapshot: ValidationSnapshot{ValidationState: validationStateInvalid, ReasonCodes: []string{reasonNonVerifiedPosture}, Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureNonVerified, wantBlocked: true, wantReason: compatibilityReasonNonVerified},
	}
}

func compatibilityVersionPostureTests() []compatibilityPostureTestCase {
	return []compatibilityPostureTestCase{
		{name: "unsupported too old", snapshot: ValidationSnapshot{ValidationState: validationStateValid, RuneContextVersion: "0.1.0-alpha.12", Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureUnsupportedTooOld, wantBlocked: true, wantReason: compatibilityReasonUnsupportedTooOld},
		{name: "supported with upgrade available", snapshot: ValidationSnapshot{ValidationState: validationStateValid, RuneContextVersion: "0.1.0-alpha.13", Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureSupportedWithUpgrade, wantBlocked: false, wantReason: compatibilityReasonUpgradeAvailable},
		{name: "supported current", snapshot: ValidationSnapshot{ValidationState: validationStateValid, RuneContextVersion: releaseRecommendedRuneContextVersion, Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureSupportedCurrent, wantBlocked: false},
		{name: "unsupported too new", snapshot: ValidationSnapshot{ValidationState: validationStateValid, RuneContextVersion: "0.1.0-alpha.99", Contract: defaultContract(RepoRootAuthorityExplicitConfig)}, wantPosture: CompatibilityPostureUnsupportedTooNew, wantBlocked: true, wantReason: compatibilityReasonUnsupportedTooNew},
	}
}

func TestEvaluateCompatibilityTreatsUnparseableVersionInvalid(t *testing.T) {
	assessment := EvaluateCompatibilityWithPolicy(ValidationSnapshot{
		ValidationState:    validationStateValid,
		RuneContextVersion: "not-semver",
		Contract:           defaultContract(RepoRootAuthorityExplicitConfig),
	}, CompatibilityPolicy{
		SupportedContractID:            ContractIDV0,
		SupportedContractVersionMin:    ContractVersionV0,
		SupportedContractVersionMax:    ContractVersionV0,
		RecommendedContractVersion:     ContractVersionV0,
		SupportedRuneContextVersionMin: "0.1.0-alpha.13",
		SupportedRuneContextVersionMax: "0.1.0-alpha.16",
		RecommendedRuneContextVersion:  "0.1.0-alpha.14",
	})
	if assessment.Posture != CompatibilityPostureInvalid {
		t.Fatalf("posture = %q, want %q", assessment.Posture, CompatibilityPostureInvalid)
	}
	if !containsString(assessment.ReasonCodes, compatibilityReasonVersionUnparseable) {
		t.Fatalf("reason_codes = %v, want %q", assessment.ReasonCodes, compatibilityReasonVersionUnparseable)
	}
}

func TestAllowsNormalOperationOnlySupportedPostures(t *testing.T) {
	if !AllowsNormalOperation(CompatibilityPostureSupportedCurrent) {
		t.Fatal("supported_current should allow normal operation")
	}
	if !AllowsNormalOperation(CompatibilityPostureSupportedWithUpgrade) {
		t.Fatal("supported_with_upgrade_available should allow normal operation")
	}
	blocked := []string{
		CompatibilityPostureMissing,
		CompatibilityPostureInvalid,
		CompatibilityPostureNonVerified,
		CompatibilityPostureUnsupportedTooOld,
		CompatibilityPostureUnsupportedTooNew,
	}
	for _, posture := range blocked {
		if AllowsNormalOperation(posture) {
			t.Fatalf("posture %q unexpectedly allows normal operation", posture)
		}
	}
}
