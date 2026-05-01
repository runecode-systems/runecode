package trustpolicy

import (
	"fmt"
	"strings"
	"time"
)

const (
	ExternalAnchorEvidenceSchemaID      = "runecode.protocol.v0.ExternalAnchorEvidence"
	ExternalAnchorEvidenceSchemaVersion = "0.1.0"

	ExternalAnchorTargetRequirementRequired = "required"
	ExternalAnchorTargetRequirementOptional = "optional"

	ExternalAnchorOutcomeCompleted   = "completed"
	ExternalAnchorOutcomeDeferred    = "deferred"
	ExternalAnchorOutcomeUnavailable = "unavailable"
	ExternalAnchorOutcomeInvalid     = "invalid"
	ExternalAnchorOutcomeFailed      = "failed"

	ExternalAnchorSidecarKindProofBytes        = "proof_bytes"
	ExternalAnchorSidecarKindProviderReceipt   = "provider_receipt"
	ExternalAnchorSidecarKindVerifyTranscript  = "verification_transcript"
	ExternalAnchorSidecarKindAttestationRef    = "attestation_ref"
	ExternalAnchorSidecarKindProjectContextRef = "project_context_ref"
)

type ExternalAnchorEvidenceSidecarRef struct {
	EvidenceKind string `json:"evidence_kind"`
	Digest       Digest `json:"digest"`
}

type ExternalAnchorEvidencePayload struct {
	SchemaID                string                             `json:"schema_id"`
	SchemaVersion           string                             `json:"schema_version"`
	RecordedAt              string                             `json:"recorded_at"`
	RunID                   string                             `json:"run_id,omitempty"`
	PreparedMutationID      string                             `json:"prepared_mutation_id,omitempty"`
	ExecutionAttemptID      string                             `json:"execution_attempt_id,omitempty"`
	CanonicalTargetKind     string                             `json:"canonical_target_kind"`
	CanonicalTargetDigest   Digest                             `json:"canonical_target_digest"`
	CanonicalTargetIdentity string                             `json:"canonical_target_identity,omitempty"`
	TargetRequirement       string                             `json:"target_requirement,omitempty"`
	AnchoringSubjectFamily  string                             `json:"anchoring_subject_family"`
	AnchoringSubjectDigest  Digest                             `json:"anchoring_subject_digest"`
	OutboundPayloadDigest   *Digest                            `json:"outbound_payload_digest,omitempty"`
	OutboundSubjectDigest   *Digest                            `json:"outbound_subject_digest,omitempty"`
	OutboundBytes           int64                              `json:"outbound_bytes,omitempty"`
	StartedAt               string                             `json:"started_at,omitempty"`
	CompletedAt             string                             `json:"completed_at,omitempty"`
	Outcome                 string                             `json:"outcome"`
	OutcomeReasonCode       string                             `json:"outcome_reason_code,omitempty"`
	TypedRequestHash        *Digest                            `json:"typed_request_hash,omitempty"`
	ActionRequestHash       *Digest                            `json:"action_request_hash,omitempty"`
	PolicyDecisionHash      *Digest                            `json:"policy_decision_hash,omitempty"`
	TargetAuthLeaseID       string                             `json:"target_auth_lease_id,omitempty"`
	RequiredApprovalID      string                             `json:"required_approval_id,omitempty"`
	ApprovalRequestHash     *Digest                            `json:"approval_request_hash,omitempty"`
	ApprovalDecisionHash    *Digest                            `json:"approval_decision_hash,omitempty"`
	ProofSchemaID           string                             `json:"proof_schema_id"`
	ProofKind               string                             `json:"proof_kind"`
	SidecarRefs             []ExternalAnchorEvidenceSidecarRef `json:"sidecar_refs"`
}

func ValidateExternalAnchorEvidencePayload(payload ExternalAnchorEvidencePayload) error {
	if err := validateExternalAnchorEvidenceEnvelope(payload); err != nil {
		return err
	}
	if err := validateExternalAnchorEvidenceBindings(payload); err != nil {
		return err
	}
	return validateExternalAnchorEvidenceSidecars(payload.SidecarRefs)
}

func validateExternalAnchorEvidenceEnvelope(payload ExternalAnchorEvidencePayload) error {
	if payload.SchemaID != ExternalAnchorEvidenceSchemaID {
		return fmt.Errorf("unexpected external anchor evidence schema_id %q", payload.SchemaID)
	}
	if payload.SchemaVersion != ExternalAnchorEvidenceSchemaVersion {
		return fmt.Errorf("unexpected external anchor evidence schema_version %q", payload.SchemaVersion)
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.RecordedAt)); err != nil {
		return fmt.Errorf("recorded_at: %w", err)
	}
	if strings.TrimSpace(payload.CanonicalTargetKind) == "" {
		return fmt.Errorf("canonical_target_kind is required")
	}
	if strings.TrimSpace(payload.ProofSchemaID) == "" {
		return fmt.Errorf("proof_schema_id is required")
	}
	if strings.TrimSpace(payload.ProofKind) == "" {
		return fmt.Errorf("proof_kind is required")
	}
	return validateExternalAnchorOutcome(payload.Outcome)
}

func validateExternalAnchorEvidenceBindings(payload ExternalAnchorEvidencePayload) error {
	requirement := externalAnchorTargetRequirementOrDefault(payload.TargetRequirement)
	if err := validateExternalAnchorTargetRequirement(requirement); err != nil {
		return err
	}
	if _, err := payload.CanonicalTargetDigest.Identity(); err != nil {
		return fmt.Errorf("canonical_target_digest: %w", err)
	}
	if strings.TrimSpace(payload.AnchoringSubjectFamily) != AuditSegmentAnchoringSubjectSeal {
		return fmt.Errorf("anchoring_subject_family must be %q", AuditSegmentAnchoringSubjectSeal)
	}
	if _, err := payload.AnchoringSubjectDigest.Identity(); err != nil {
		return fmt.Errorf("anchoring_subject_digest: %w", err)
	}
	if err := validateExternalAnchorOutboundBindings(payload.OutboundPayloadDigest, payload.OutboundSubjectDigest); err != nil {
		return err
	}
	if payload.OutboundBytes < 0 {
		return fmt.Errorf("outbound_bytes must be non-negative")
	}
	return nil
}

func validateExternalAnchorOutboundBindings(outboundPayloadDigest, outboundSubjectDigest *Digest) error {
	if outboundPayloadDigest == nil && outboundSubjectDigest == nil {
		return fmt.Errorf("either outbound_payload_digest or outbound_subject_digest is required")
	}
	if outboundPayloadDigest != nil {
		if _, err := outboundPayloadDigest.Identity(); err != nil {
			return fmt.Errorf("outbound_payload_digest: %w", err)
		}
	}
	if outboundSubjectDigest != nil {
		if _, err := outboundSubjectDigest.Identity(); err != nil {
			return fmt.Errorf("outbound_subject_digest: %w", err)
		}
	}
	return nil
}

func validateExternalAnchorEvidenceSidecars(sidecarRefs []ExternalAnchorEvidenceSidecarRef) error {
	if len(sidecarRefs) == 0 {
		return fmt.Errorf("sidecar_refs must include at least proof_bytes")
	}
	seen := map[string]struct{}{}
	for i := range sidecarRefs {
		ref := sidecarRefs[i]
		if err := validateExternalAnchorSidecarRef(ref, i, seen); err != nil {
			return err
		}
	}
	if _, ok := seen[ExternalAnchorSidecarKindProofBytes]; !ok {
		return fmt.Errorf("sidecar_refs must include evidence_kind %q", ExternalAnchorSidecarKindProofBytes)
	}
	return nil
}

func validateExternalAnchorSidecarRef(ref ExternalAnchorEvidenceSidecarRef, index int, seen map[string]struct{}) error {
	if err := validateExternalAnchorSidecarKind(ref.EvidenceKind); err != nil {
		return fmt.Errorf("sidecar_refs[%d]: %w", index, err)
	}
	if _, err := ref.Digest.Identity(); err != nil {
		return fmt.Errorf("sidecar_refs[%d].digest: %w", index, err)
	}
	if _, dup := seen[ref.EvidenceKind]; dup {
		return fmt.Errorf("sidecar_refs[%d] duplicates evidence_kind %q", index, ref.EvidenceKind)
	}
	seen[ref.EvidenceKind] = struct{}{}
	return nil
}

func externalAnchorTargetRequirementOrDefault(requirement string) string {
	if strings.TrimSpace(requirement) == "" {
		return ExternalAnchorTargetRequirementRequired
	}
	return strings.TrimSpace(requirement)
}

func validateExternalAnchorTargetRequirement(requirement string) error {
	switch strings.TrimSpace(requirement) {
	case ExternalAnchorTargetRequirementRequired, ExternalAnchorTargetRequirementOptional:
		return nil
	default:
		return fmt.Errorf("unsupported external anchor target_requirement %q", requirement)
	}
}

func NormalizeExternalAnchorTargetRequirement(requirement string) string {
	return externalAnchorTargetRequirementOrDefault(requirement)
}

func ValidateExternalAnchorTargetRequirement(requirement string) error {
	return validateExternalAnchorTargetRequirement(requirement)
}

func validateExternalAnchorOutcome(outcome string) error {
	switch strings.TrimSpace(outcome) {
	case ExternalAnchorOutcomeCompleted, ExternalAnchorOutcomeDeferred, ExternalAnchorOutcomeUnavailable, ExternalAnchorOutcomeInvalid, ExternalAnchorOutcomeFailed:
		return nil
	default:
		return fmt.Errorf("unsupported external anchor outcome %q", outcome)
	}
}

func validateExternalAnchorSidecarKind(kind string) error {
	switch strings.TrimSpace(kind) {
	case ExternalAnchorSidecarKindProofBytes,
		ExternalAnchorSidecarKindProviderReceipt,
		ExternalAnchorSidecarKindVerifyTranscript,
		ExternalAnchorSidecarKindAttestationRef,
		ExternalAnchorSidecarKindProjectContextRef:
		return nil
	default:
		return fmt.Errorf("unsupported external anchor sidecar evidence_kind %q", kind)
	}
}

func ValidateExternalAnchorSidecarKind(kind string) error {
	return validateExternalAnchorSidecarKind(kind)
}
