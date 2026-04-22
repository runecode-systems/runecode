package projectsubstrate

import (
	"sort"
	"strings"
)

const (
	ContractSchemaID               = "runecode.protocol.v0.ProjectSubstrateContractState"
	ContractSchemaVersion          = "0.1.0"
	SnapshotSchemaID               = "runecode.protocol.v0.ProjectSubstrateValidationSnapshot"
	SnapshotSchemaVersion          = "0.1.0"
	AdoptionSchemaID               = "runecode.protocol.v0.ProjectSubstrateAdoptionResult"
	AdoptionSchemaVersion          = "0.1.0"
	InitPreviewSchemaID            = "runecode.protocol.v0.ProjectSubstrateInitPreview"
	InitPreviewVersion             = "0.1.0"
	InitApplySchemaID              = "runecode.protocol.v0.ProjectSubstrateInitApplyResult"
	InitApplyVersion               = "0.1.0"
	UpgradePreviewSchemaID         = "runecode.protocol.v0.ProjectSubstrateUpgradePreview"
	UpgradePreviewVersion          = "0.1.0"
	UpgradeApplySchemaID           = "runecode.protocol.v0.ProjectSubstrateUpgradeApplyResult"
	UpgradeApplyVersion            = "0.1.0"
	ContractIDV0                   = "runecode.runecontext.project_substrate.v0"
	ContractVersionV0              = "v0"
	CanonicalConfigPath            = "runecontext.yaml"
	CanonicalSourcePath            = "runecontext"
	CanonicalAssurancePath         = "runecontext/assurance"
	canonicalAssuranceBaselinePath = "runecontext/assurance/baseline.yaml"

	validationStateValid   = "valid"
	validationStateInvalid = "invalid"
	validationStateMissing = "missing"

	reasonMissingConfigAnchor      = "anchor_missing_runecontext_yaml"
	reasonMissingSourceAnchor      = "anchor_missing_runecontext_source"
	reasonMissingAssuranceAnchor   = "anchor_missing_runecontext_assurance"
	reasonMissingAssuranceBaseline = "anchor_missing_runecontext_assurance_baseline"
	reasonConfigParseInvalid       = "config_parse_invalid"
	reasonAssuranceBaselineInvalid = "assurance_baseline_invalid"
	reasonNonVerifiedPosture       = "posture_non_verified"
	reasonNonCanonicalSourcePath   = "source_path_non_canonical"
	reasonPrivateMirrorDetected    = "runecode_private_mirror_detected"
	reasonDiscoveryRootInvalid     = "repository_root_invalid"
	reasonConfigMissingSourcePath  = "source_path_missing"
	reasonAdoptionNotCanonical     = "adoption_not_canonical"
	reasonAdoptionNotVerified      = "adoption_not_verified"

	reasonInitConflictDetected        = "init_conflict_detected"
	reasonInitConflictCanonicalExists = "init_conflict_canonical_exists"
	reasonInitConflictPrivateMirror   = "init_conflict_private_mirror_detected"
	reasonInitConflictCandidateState  = "init_conflict_candidate_state_detected"
	reasonInitPreviewTokenMismatch    = "init_preview_token_mismatch"
	reasonInitSnapshotChanged         = "init_snapshot_changed_since_preview"
	reasonInitPostValidationFailed    = "init_post_validation_failed"

	reasonRemediationFlowRequired            = "remediation_flow_required"
	reasonRemediationInitRequired            = "remediation_init_required"
	reasonRemediationConfigAnchorRequired    = "remediation_config_anchor_required"
	reasonRemediationConfigInvalid           = "remediation_config_invalid"
	reasonRemediationDeclaredSourceConflicts = "remediation_declared_source_conflicts"
	reasonUpgradeApplyExplicitRequired       = "upgrade_apply_explicit_required"
	reasonUpgradePreviewDigestRequired       = "upgrade_preview_digest_required"
	reasonUpgradeSnapshotChanged             = "upgrade_snapshot_changed_since_preview"
	reasonUpgradePreviewDigestMismatch       = "upgrade_preview_digest_mismatch"
	reasonUpgradePostValidationFailed        = "upgrade_post_validation_failed"
	reasonAuditAppendFailed                  = "audit_append_failed"

	upgradePreviewStatusNoop                 = "noop"
	upgradePreviewStatusReady                = "ready_for_apply"
	upgradePreviewStatusBlockedRemediation   = "blocked_remediation_only"
	upgradeApplyStatusNoop                   = "noop"
	upgradeApplyStatusApplied                = "applied"
	upgradeApplyStatusBlocked                = "blocked"
	upgradeApplyStatusAppliedValidationError = "applied_validation_failed"

	adoptionStatusAdopted = "adopted"
	adoptionStatusBlocked = "blocked"

	initPreviewStatusNoop         = "noop"
	initPreviewStatusReady        = "ready_for_apply"
	initPreviewStatusBlocked      = "blocked_conflict"
	initApplyStatusNoop           = "noop"
	initApplyStatusApplied        = "applied"
	initApplyStatusBlocked        = "blocked"
	initApplyStatusAppliedInvalid = "applied_validation_failed"
)

type RepoRootAuthority string

const (
	RepoRootAuthorityProcessWorkingDirectory RepoRootAuthority = "process_working_directory"
	RepoRootAuthorityExplicitConfig          RepoRootAuthority = "explicit_config"
)

type ContractState struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	ContractID             string `json:"contract_id"`
	ContractVersion        string `json:"contract_version"`
	RequiredConfigPath     string `json:"required_config_path"`
	RequiredSourcePath     string `json:"required_source_path"`
	RequiredAssurancePath  string `json:"required_assurance_path"`
	RequiredAssuranceTier  string `json:"required_assurance_tier"`
	RepoRootAuthority      string `json:"repo_root_authority"`
	ForbidPrivateTruthCopy bool   `json:"forbid_private_truth_copy"`
}

type ValidationSnapshot struct {
	SchemaID                     string        `json:"schema_id"`
	SchemaVersion                string        `json:"schema_version"`
	Contract                     ContractState `json:"contract"`
	ValidationState              string        `json:"validation_state"`
	ReasonCodes                  []string      `json:"reason_codes,omitempty"`
	RuneContextVersion           string        `json:"runecontext_version,omitempty"`
	DeclaredAssuranceTier        string        `json:"declared_assurance_tier,omitempty"`
	DeclaredSourceType           string        `json:"declared_source_type,omitempty"`
	DeclaredSourcePath           string        `json:"declared_source_path,omitempty"`
	CanonicalCandidatePaths      []string      `json:"canonical_candidate_paths,omitempty"`
	SnapshotDigest               string        `json:"snapshot_digest"`
	ValidatedSnapshotDigest      string        `json:"validated_snapshot_digest,omitempty"`
	ProjectContextIdentityDigest string        `json:"project_context_identity_digest,omitempty"`
	Anchors                      AnchorStatus  `json:"anchors"`
}

type AnchorStatus struct {
	HasConfigAnchor      bool `json:"has_config_anchor"`
	HasSourceAnchor      bool `json:"has_source_anchor"`
	HasAssuranceAnchor   bool `json:"has_assurance_anchor"`
	HasAssuranceBaseline bool `json:"has_assurance_baseline"`
	HasVerifiedPosture   bool `json:"has_verified_posture"`
	HasCanonicalSource   bool `json:"has_canonical_source"`
	HasPrivateTruthCopy  bool `json:"has_private_truth_copy"`
}

func defaultContract(authority RepoRootAuthority) ContractState {
	return ContractState{
		SchemaID:               ContractSchemaID,
		SchemaVersion:          ContractSchemaVersion,
		ContractID:             ContractIDV0,
		ContractVersion:        ContractVersionV0,
		RequiredConfigPath:     CanonicalConfigPath,
		RequiredSourcePath:     CanonicalSourcePath,
		RequiredAssurancePath:  CanonicalAssurancePath,
		RequiredAssuranceTier:  "verified",
		RepoRootAuthority:      normalizeAuthority(authority),
		ForbidPrivateTruthCopy: true,
	}
}

func normalizeAuthority(authority RepoRootAuthority) string {
	switch authority {
	case RepoRootAuthorityExplicitConfig:
		return string(RepoRootAuthorityExplicitConfig)
	default:
		return string(RepoRootAuthorityProcessWorkingDirectory)
	}
}

func normalizeReasonCodes(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) > 1 {
		sort.Strings(out)
	}
	return out
}
