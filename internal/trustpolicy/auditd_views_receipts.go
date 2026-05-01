package trustpolicy

import (
	"encoding/json"
	"fmt"
)

type auditReceiptPayload struct {
	SchemaID             string          `json:"schema_id"`
	SchemaVersion        string          `json:"schema_version"`
	SubjectDigest        Digest          `json:"subject_digest"`
	AuditReceiptKind     string          `json:"audit_receipt_kind"`
	SubjectFamily        string          `json:"subject_family,omitempty"`
	Recorder             json.RawMessage `json:"recorder,omitempty"`
	RecordedAt           string          `json:"recorded_at"`
	ReceiptPayloadSchema string          `json:"receipt_payload_schema_id,omitempty"`
	ReceiptPayload       json.RawMessage `json:"receipt_payload,omitempty"`
}

type auditImportRestoreOperationalPayload struct {
	AuthorityContext json.RawMessage `json:"authority_context,omitempty"`
}

type auditAnchorOperationalPayload struct {
	AnchorKind           string                          `json:"anchor_kind"`
	KeyProtectionPosture string                          `json:"key_protection_posture,omitempty"`
	PresenceMode         string                          `json:"presence_mode,omitempty"`
	ApprovalAssurance    string                          `json:"approval_assurance_level,omitempty"`
	ApprovalDecision     *Digest                         `json:"approval_decision_digest,omitempty"`
	AnchorWitness        *auditAnchorWitnessOperational  `json:"anchor_witness,omitempty"`
	ExternalAnchor       *auditExternalAnchorOperational `json:"external_anchor,omitempty"`
}

type auditAnchorWitnessOperational struct {
	WitnessKind   string `json:"witness_kind"`
	WitnessDigest Digest `json:"witness_digest"`
}

type auditExternalAnchorOperational struct {
	TargetKind             string                              `json:"target_kind"`
	RuntimeAdapter         string                              `json:"runtime_adapter"`
	TargetDescriptorDigest Digest                              `json:"target_descriptor_digest"`
	Proof                  auditExternalAnchorProofOperational `json:"proof"`
}

type auditExternalAnchorProofOperational struct {
	ProofKind     string `json:"proof_kind"`
	ProofSchemaID string `json:"proof_schema_id"`
	ProofDigest   Digest `json:"proof_digest"`
}

func populateOperationalReceiptView(view *AuditOperationalView, envelope SignedObjectEnvelope) error {
	if envelope.PayloadSchemaVersion != AuditReceiptSchemaVersion {
		return fmt.Errorf("unsupported audit receipt schema version %q", envelope.PayloadSchemaVersion)
	}
	receipt, err := decodeAndValidateOperationalReceipt(envelope.Payload)
	if err != nil {
		return err
	}
	operationalReceipt, err := buildAuditReceiptOperationalView(receipt)
	if err != nil {
		return err
	}
	view.Receipt = operationalReceipt
	view.Redaction.RedactedFields = []string{"receipt_payload", "recorder"}
	return nil
}

func decodeAndValidateOperationalReceipt(payload json.RawMessage) (auditReceiptPayload, error) {
	receipt := auditReceiptPayload{}
	if err := json.Unmarshal(payload, &receipt); err != nil {
		return auditReceiptPayload{}, fmt.Errorf("decode audit receipt payload: %w", err)
	}
	if receipt.SchemaID != AuditReceiptSchemaID {
		return auditReceiptPayload{}, fmt.Errorf("unexpected audit receipt schema_id %q", receipt.SchemaID)
	}
	if receipt.SchemaVersion != AuditReceiptSchemaVersion {
		return auditReceiptPayload{}, fmt.Errorf("unexpected audit receipt schema_version %q", receipt.SchemaVersion)
	}
	if _, err := receipt.SubjectDigest.Identity(); err != nil {
		return auditReceiptPayload{}, fmt.Errorf("subject_digest: %w", err)
	}
	return receipt, nil
}

func buildAuditReceiptOperationalView(receipt auditReceiptPayload) (*AuditReceiptOperationalView, error) {
	anchorSummary, err := extractOperationalAnchorSummary(receipt)
	if err != nil {
		return nil, err
	}
	authorityContext, err := extractOperationalAuthorityContext(receipt)
	if err != nil {
		return nil, err
	}
	return &AuditReceiptOperationalView{
		SchemaID:                       receipt.SchemaID,
		SchemaVersion:                  receipt.SchemaVersion,
		SubjectDigest:                  receipt.SubjectDigest,
		AuditReceiptKind:               receipt.AuditReceiptKind,
		SubjectFamily:                  receipt.SubjectFamily,
		RecordedAt:                     receipt.RecordedAt,
		ReceiptPayloadSchema:           receipt.ReceiptPayloadSchema,
		AnchorKind:                     anchorSummary.AnchorKind,
		KeyProtectionPosture:           anchorSummary.KeyProtectionPosture,
		PresenceMode:                   anchorSummary.PresenceMode,
		ApprovalAssurance:              anchorSummary.ApprovalAssurance,
		ApprovalDecision:               anchorSummary.ApprovalDecision,
		AnchorWitnessDigest:            anchorSummary.AnchorWitnessDigest,
		ExternalTargetKind:             anchorSummary.ExternalTargetKind,
		ExternalRuntimeAdapter:         anchorSummary.ExternalRuntimeAdapter,
		ExternalTargetDescriptorDigest: anchorSummary.ExternalTargetDescriptorDigest,
		ExternalProofKind:              anchorSummary.ExternalProofKind,
		ExternalProofSchema:            anchorSummary.ExternalProofSchema,
		ExternalProofDigest:            anchorSummary.ExternalProofDigest,
		AuthorityContext:               authorityContext,
	}, nil
}

type auditAnchorOperationalSummary struct {
	AnchorKind                     string
	KeyProtectionPosture           string
	PresenceMode                   string
	ApprovalAssurance              string
	ApprovalDecision               *Digest
	AnchorWitnessDigest            *Digest
	ExternalTargetKind             string
	ExternalRuntimeAdapter         string
	ExternalTargetDescriptorDigest *Digest
	ExternalProofKind              string
	ExternalProofSchema            string
	ExternalProofDigest            *Digest
}

func extractOperationalAnchorSummary(receipt auditReceiptPayload) (auditAnchorOperationalSummary, error) {
	if receipt.AuditReceiptKind != "anchor" || len(receipt.ReceiptPayload) == 0 {
		return auditAnchorOperationalSummary{}, nil
	}
	payload := auditAnchorOperationalPayload{}
	if err := json.Unmarshal(receipt.ReceiptPayload, &payload); err != nil {
		return auditAnchorOperationalSummary{}, fmt.Errorf("decode anchor receipt_payload for operational view: %w", err)
	}
	summary := operationalAnchorSummaryBase(payload)
	if err := applyOperationalAnchorWitness(&summary, payload.AnchorWitness); err != nil {
		return auditAnchorOperationalSummary{}, err
	}
	if err := applyOperationalAnchorApprovalDecision(&summary, payload.ApprovalDecision); err != nil {
		return auditAnchorOperationalSummary{}, err
	}
	if err := applyOperationalExternalAnchor(&summary, payload.ExternalAnchor); err != nil {
		return auditAnchorOperationalSummary{}, err
	}
	return summary, nil
}

func operationalAnchorSummaryBase(payload auditAnchorOperationalPayload) auditAnchorOperationalSummary {
	return auditAnchorOperationalSummary{
		AnchorKind:           payload.AnchorKind,
		KeyProtectionPosture: payload.KeyProtectionPosture,
		PresenceMode:         payload.PresenceMode,
		ApprovalAssurance:    payload.ApprovalAssurance,
	}
}

func applyOperationalAnchorWitness(summary *auditAnchorOperationalSummary, witness *auditAnchorWitnessOperational) error {
	if witness == nil {
		return nil
	}
	if _, err := witness.WitnessDigest.Identity(); err != nil {
		return fmt.Errorf("anchor_witness.witness_digest: %w", err)
	}
	summary.AnchorWitnessDigest = &witness.WitnessDigest
	return nil
}

func applyOperationalAnchorApprovalDecision(summary *auditAnchorOperationalSummary, decision *Digest) error {
	if decision == nil {
		return nil
	}
	if _, err := decision.Identity(); err != nil {
		return fmt.Errorf("approval_decision_digest: %w", err)
	}
	summary.ApprovalDecision = decision
	return nil
}

func applyOperationalExternalAnchor(summary *auditAnchorOperationalSummary, external *auditExternalAnchorOperational) error {
	if external == nil {
		return nil
	}
	if _, err := external.TargetDescriptorDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.target_descriptor_digest: %w", err)
	}
	if _, err := external.Proof.ProofDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.proof.proof_digest: %w", err)
	}
	summary.ExternalTargetKind = external.TargetKind
	summary.ExternalRuntimeAdapter = external.RuntimeAdapter
	summary.ExternalTargetDescriptorDigest = &external.TargetDescriptorDigest
	summary.ExternalProofKind = external.Proof.ProofKind
	summary.ExternalProofSchema = external.Proof.ProofSchemaID
	summary.ExternalProofDigest = &external.Proof.ProofDigest
	return nil
}

func extractOperationalAuthorityContext(receipt auditReceiptPayload) (json.RawMessage, error) {
	if (receipt.AuditReceiptKind != "import" && receipt.AuditReceiptKind != "restore") || len(receipt.ReceiptPayload) == 0 {
		return nil, nil
	}
	payload := auditImportRestoreOperationalPayload{}
	if err := json.Unmarshal(receipt.ReceiptPayload, &payload); err != nil {
		return nil, fmt.Errorf("decode import/restore receipt_payload for operational view: %w", err)
	}
	if len(payload.AuthorityContext) == 0 {
		return nil, nil
	}
	if !isJSONObject(payload.AuthorityContext) {
		return nil, fmt.Errorf("authority_context must be a JSON object")
	}
	return payload.AuthorityContext, nil
}

func isJSONObject(raw json.RawMessage) bool {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	_, ok := value.(map[string]any)
	return ok
}
