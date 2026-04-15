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
	AnchorKind           string                        `json:"anchor_kind"`
	KeyProtectionPosture string                        `json:"key_protection_posture"`
	PresenceMode         string                        `json:"presence_mode"`
	ApprovalAssurance    string                        `json:"approval_assurance_level,omitempty"`
	ApprovalDecision     *Digest                       `json:"approval_decision_digest,omitempty"`
	AnchorWitness        auditAnchorWitnessOperational `json:"anchor_witness"`
}

type auditAnchorWitnessOperational struct {
	WitnessKind   string `json:"witness_kind"`
	WitnessDigest Digest `json:"witness_digest"`
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
		SchemaID:             receipt.SchemaID,
		SchemaVersion:        receipt.SchemaVersion,
		SubjectDigest:        receipt.SubjectDigest,
		AuditReceiptKind:     receipt.AuditReceiptKind,
		SubjectFamily:        receipt.SubjectFamily,
		RecordedAt:           receipt.RecordedAt,
		ReceiptPayloadSchema: receipt.ReceiptPayloadSchema,
		AnchorKind:           anchorSummary.AnchorKind,
		KeyProtectionPosture: anchorSummary.KeyProtectionPosture,
		PresenceMode:         anchorSummary.PresenceMode,
		ApprovalAssurance:    anchorSummary.ApprovalAssurance,
		ApprovalDecision:     anchorSummary.ApprovalDecision,
		AnchorWitnessDigest:  anchorSummary.AnchorWitnessDigest,
		AuthorityContext:     authorityContext,
	}, nil
}

type auditAnchorOperationalSummary struct {
	AnchorKind           string
	KeyProtectionPosture string
	PresenceMode         string
	ApprovalAssurance    string
	ApprovalDecision     *Digest
	AnchorWitnessDigest  *Digest
}

func extractOperationalAnchorSummary(receipt auditReceiptPayload) (auditAnchorOperationalSummary, error) {
	if receipt.AuditReceiptKind != "anchor" || len(receipt.ReceiptPayload) == 0 {
		return auditAnchorOperationalSummary{}, nil
	}
	payload := auditAnchorOperationalPayload{}
	if err := json.Unmarshal(receipt.ReceiptPayload, &payload); err != nil {
		return auditAnchorOperationalSummary{}, fmt.Errorf("decode anchor receipt_payload for operational view: %w", err)
	}
	if _, err := payload.AnchorWitness.WitnessDigest.Identity(); err != nil {
		return auditAnchorOperationalSummary{}, fmt.Errorf("anchor_witness.witness_digest: %w", err)
	}
	summary := auditAnchorOperationalSummary{
		AnchorKind:           payload.AnchorKind,
		KeyProtectionPosture: payload.KeyProtectionPosture,
		PresenceMode:         payload.PresenceMode,
		ApprovalAssurance:    payload.ApprovalAssurance,
		AnchorWitnessDigest:  &payload.AnchorWitness.WitnessDigest,
	}
	if payload.ApprovalDecision != nil {
		if _, err := payload.ApprovalDecision.Identity(); err != nil {
			return auditAnchorOperationalSummary{}, fmt.Errorf("approval_decision_digest: %w", err)
		}
		summary.ApprovalDecision = payload.ApprovalDecision
	}
	return summary, nil
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
