package auditd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type ExternalAnchorEvidenceRequest struct {
	SchemaID                 string
	SchemaVersion            string
	RecordedAtRFC3339        string
	RunID                    string
	PreparedMutationID       string
	ExecutionAttemptID       string
	CanonicalTargetKind      string
	CanonicalTargetDigest    trustpolicy.Digest
	CanonicalTargetIdentity  string
	TargetRequirement        string
	AnchoringSubjectFamily   string
	AnchoringSubjectDigest   trustpolicy.Digest
	OutboundPayloadDigest    *trustpolicy.Digest
	OutboundSubjectDigest    *trustpolicy.Digest
	OutboundBytes            int64
	StartedAtRFC3339         string
	CompletedAtRFC3339       string
	Outcome                  string
	OutcomeReasonCode        string
	TypedRequestHash         *trustpolicy.Digest
	ActionRequestHash        *trustpolicy.Digest
	PolicyDecisionHash       *trustpolicy.Digest
	TargetAuthLeaseID        string
	RequiredApprovalID       string
	ApprovalRequestHash      *trustpolicy.Digest
	ApprovalDecisionHash     *trustpolicy.Digest
	AttestationEvidenceRef   *trustpolicy.Digest
	ProjectContextIdentity   *trustpolicy.Digest
	ProofDigest              trustpolicy.Digest
	ProofSchemaID            string
	ProofKind                string
	ProviderReceiptDigest    *trustpolicy.Digest
	VerificationTranscriptID *trustpolicy.Digest
}

func (l *Ledger) PersistExternalAnchorEvidence(req ExternalAnchorEvidenceRequest) (trustpolicy.Digest, trustpolicy.ExternalAnchorEvidencePayload, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	payload, err := buildExternalAnchorEvidencePayload(req, l.nowFn())
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ExternalAnchorEvidencePayload{}, err
	}
	digest, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ExternalAnchorEvidencePayload{}, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, externalAnchorEvidenceDir, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, payload); err != nil {
		return trustpolicy.Digest{}, trustpolicy.ExternalAnchorEvidencePayload{}, err
	}
	return digest, payload, nil
}

func buildExternalAnchorEvidencePayload(req ExternalAnchorEvidenceRequest, now time.Time) (trustpolicy.ExternalAnchorEvidencePayload, error) {
	if err := validateExternalAnchorEvidenceRequest(req); err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, err
	}
	payload := externalAnchorEvidencePayloadCore(req, now)
	payload.SidecarRefs = externalAnchorEvidenceSidecarRefs(req)
	if err := trustpolicy.ValidateExternalAnchorEvidencePayload(payload); err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, err
	}
	return payload, nil
}

func externalAnchorEvidencePayloadCore(req ExternalAnchorEvidenceRequest, now time.Time) trustpolicy.ExternalAnchorEvidencePayload {
	return trustpolicy.ExternalAnchorEvidencePayload{
		SchemaID:                trustpolicy.ExternalAnchorEvidenceSchemaID,
		SchemaVersion:           trustpolicy.ExternalAnchorEvidenceSchemaVersion,
		RecordedAt:              externalAnchorEvidenceRecordedAt(req.RecordedAtRFC3339, now),
		RunID:                   strings.TrimSpace(req.RunID),
		PreparedMutationID:      strings.TrimSpace(req.PreparedMutationID),
		ExecutionAttemptID:      strings.TrimSpace(req.ExecutionAttemptID),
		CanonicalTargetKind:     strings.TrimSpace(req.CanonicalTargetKind),
		CanonicalTargetDigest:   req.CanonicalTargetDigest,
		CanonicalTargetIdentity: strings.TrimSpace(req.CanonicalTargetIdentity),
		TargetRequirement:       trustpolicy.NormalizeExternalAnchorTargetRequirement(req.TargetRequirement),
		AnchoringSubjectFamily:  strings.TrimSpace(req.AnchoringSubjectFamily),
		AnchoringSubjectDigest:  req.AnchoringSubjectDigest,
		OutboundPayloadDigest:   cloneDigestPointer(req.OutboundPayloadDigest),
		OutboundSubjectDigest:   cloneDigestPointer(req.OutboundSubjectDigest),
		OutboundBytes:           req.OutboundBytes,
		StartedAt:               strings.TrimSpace(req.StartedAtRFC3339),
		CompletedAt:             strings.TrimSpace(req.CompletedAtRFC3339),
		Outcome:                 strings.TrimSpace(req.Outcome),
		OutcomeReasonCode:       strings.TrimSpace(req.OutcomeReasonCode),
		TypedRequestHash:        cloneDigestPointer(req.TypedRequestHash),
		ActionRequestHash:       cloneDigestPointer(req.ActionRequestHash),
		PolicyDecisionHash:      cloneDigestPointer(req.PolicyDecisionHash),
		TargetAuthLeaseID:       strings.TrimSpace(req.TargetAuthLeaseID),
		RequiredApprovalID:      strings.TrimSpace(req.RequiredApprovalID),
		ApprovalRequestHash:     cloneDigestPointer(req.ApprovalRequestHash),
		ApprovalDecisionHash:    cloneDigestPointer(req.ApprovalDecisionHash),
		ProofSchemaID:           strings.TrimSpace(req.ProofSchemaID),
		ProofKind:               strings.TrimSpace(req.ProofKind),
	}
}

func externalAnchorEvidenceRecordedAt(recordedAt string, now time.Time) string {
	trimmed := strings.TrimSpace(recordedAt)
	if trimmed != "" {
		return trimmed
	}
	return now.UTC().Format(time.RFC3339)
}

func externalAnchorEvidenceSidecarRefs(req ExternalAnchorEvidenceRequest) []trustpolicy.ExternalAnchorEvidenceSidecarRef {
	sidecars := []trustpolicy.ExternalAnchorEvidenceSidecarRef{{EvidenceKind: trustpolicy.ExternalAnchorSidecarKindProofBytes, Digest: req.ProofDigest}}
	for _, ref := range []struct {
		kind   string
		digest *trustpolicy.Digest
	}{{kind: trustpolicy.ExternalAnchorSidecarKindProviderReceipt, digest: req.ProviderReceiptDigest}, {kind: trustpolicy.ExternalAnchorSidecarKindVerifyTranscript, digest: req.VerificationTranscriptID}, {kind: trustpolicy.ExternalAnchorSidecarKindAttestationRef, digest: req.AttestationEvidenceRef}, {kind: trustpolicy.ExternalAnchorSidecarKindProjectContextRef, digest: req.ProjectContextIdentity}} {
		if ref.digest != nil {
			sidecars = append(sidecars, trustpolicy.ExternalAnchorEvidenceSidecarRef{EvidenceKind: ref.kind, Digest: *ref.digest})
		}
	}
	return sidecars
}

func validateExternalAnchorEvidenceRequest(req ExternalAnchorEvidenceRequest) error {
	if err := validateExternalAnchorEvidenceRequestEnvelope(req); err != nil {
		return err
	}
	if err := validateExternalAnchorEvidenceRequestBindings(req); err != nil {
		return err
	}
	return validateExternalAnchorEvidenceRequestSidecars(req)
}

func validateExternalAnchorEvidenceRequestEnvelope(req ExternalAnchorEvidenceRequest) error {
	if req.SchemaID != "" && strings.TrimSpace(req.SchemaID) != trustpolicy.ExternalAnchorEvidenceSchemaID {
		return fmt.Errorf("external anchor evidence schema_id must be %q", trustpolicy.ExternalAnchorEvidenceSchemaID)
	}
	if req.SchemaVersion != "" && strings.TrimSpace(req.SchemaVersion) != trustpolicy.ExternalAnchorEvidenceSchemaVersion {
		return fmt.Errorf("external anchor evidence schema_version must be %q", trustpolicy.ExternalAnchorEvidenceSchemaVersion)
	}
	if strings.TrimSpace(req.CanonicalTargetKind) == "" {
		return fmt.Errorf("canonical_target_kind is required")
	}
	if strings.TrimSpace(req.Outcome) == "" {
		return fmt.Errorf("outcome is required")
	}
	if err := validateExternalAnchorOutcome(req.Outcome); err != nil {
		return err
	}
	if strings.TrimSpace(req.ProofSchemaID) == "" {
		return fmt.Errorf("proof_schema_id is required")
	}
	if strings.TrimSpace(req.ProofKind) == "" {
		return fmt.Errorf("proof_kind is required")
	}
	return nil
}

func validateExternalAnchorEvidenceRequestBindings(req ExternalAnchorEvidenceRequest) error {
	if err := trustpolicy.ValidateExternalAnchorTargetRequirement(trustpolicy.NormalizeExternalAnchorTargetRequirement(req.TargetRequirement)); err != nil {
		return err
	}
	if _, err := req.CanonicalTargetDigest.Identity(); err != nil {
		return fmt.Errorf("canonical_target_digest: %w", err)
	}
	if strings.TrimSpace(req.AnchoringSubjectFamily) != trustpolicy.AuditSegmentAnchoringSubjectSeal {
		return fmt.Errorf("anchoring_subject_family must be %q", trustpolicy.AuditSegmentAnchoringSubjectSeal)
	}
	if _, err := req.AnchoringSubjectDigest.Identity(); err != nil {
		return fmt.Errorf("anchoring_subject_digest: %w", err)
	}
	if err := validateExternalAnchorEvidenceRequestOutbound(req.OutboundPayloadDigest, req.OutboundSubjectDigest, req.OutboundBytes); err != nil {
		return err
	}
	if _, err := req.ProofDigest.Identity(); err != nil {
		return fmt.Errorf("proof_digest: %w", err)
	}
	return nil
}

func validateExternalAnchorEvidenceRequestOutbound(outboundPayloadDigest, outboundSubjectDigest *trustpolicy.Digest, outboundBytes int64) error {
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
	if outboundBytes < 0 {
		return fmt.Errorf("outbound_bytes must be non-negative")
	}
	return nil
}

func validateExternalAnchorEvidenceRequestSidecars(req ExternalAnchorEvidenceRequest) error {
	for _, check := range []struct {
		label  string
		digest *trustpolicy.Digest
	}{{label: "provider_receipt_digest", digest: req.ProviderReceiptDigest}, {label: "verification_transcript_digest", digest: req.VerificationTranscriptID}, {label: "attestation_evidence_digest", digest: req.AttestationEvidenceRef}, {label: "project_context_identity_digest", digest: req.ProjectContextIdentity}} {
		if check.digest == nil {
			continue
		}
		if _, err := check.digest.Identity(); err != nil {
			return fmt.Errorf("%s: %w", check.label, err)
		}
	}
	return nil
}

func cloneDigestPointer(d *trustpolicy.Digest) *trustpolicy.Digest {
	if d == nil {
		return nil
	}
	v := *d
	return &v
}

func validateExternalAnchorOutcome(outcome string) error {
	switch strings.TrimSpace(outcome) {
	case trustpolicy.ExternalAnchorOutcomeCompleted,
		trustpolicy.ExternalAnchorOutcomeDeferred,
		trustpolicy.ExternalAnchorOutcomeUnavailable,
		trustpolicy.ExternalAnchorOutcomeInvalid,
		trustpolicy.ExternalAnchorOutcomeFailed:
		return nil
	default:
		return fmt.Errorf("unsupported external anchor outcome %q", outcome)
	}
}
