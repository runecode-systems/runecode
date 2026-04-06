package trustpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	AuditReceiptSchemaID      = "runecode.protocol.v0.AuditReceipt"
	AuditReceiptSchemaVersion = "0.4.0"

	AuditOperationalViewPolicyID      = "audit_default_operational_view"
	AuditOperationalViewPolicyVersion = "0.1.0"
)

type AuditOperationalView struct {
	ViewPolicyID      string                        `json:"view_policy_id"`
	ViewPolicyVersion string                        `json:"view_policy_version"`
	RecordDigest      Digest                        `json:"record_digest"`
	PayloadSchemaID   string                        `json:"payload_schema_id"`
	PayloadVersion    string                        `json:"payload_schema_version"`
	Signature         SignatureBlock                `json:"signature"`
	Redaction         AuditOperationalRedaction     `json:"redaction"`
	Event             *AuditEventOperationalPayload `json:"event,omitempty"`
	Receipt           *AuditReceiptOperationalView  `json:"receipt,omitempty"`
}

type AuditOperationalRedaction struct {
	ExcludedDataClasses []string `json:"excluded_data_classes"`
	RedactedFields      []string `json:"redacted_fields"`
}

type AuditEventOperationalPayload struct {
	SchemaID                   string                `json:"schema_id"`
	SchemaVersion              string                `json:"schema_version"`
	AuditEventType             string                `json:"audit_event_type"`
	EmitterStreamID            string                `json:"emitter_stream_id"`
	Seq                        int64                 `json:"seq"`
	OccurredAt                 string                `json:"occurred_at"`
	PreviousEventHash          *Digest               `json:"previous_event_hash,omitempty"`
	ActiveRoleManifestHash     *Digest               `json:"active_role_manifest_hash,omitempty"`
	ActiveCapabilityHash       *Digest               `json:"active_capability_manifest_hash,omitempty"`
	EventPayloadSchemaID       string                `json:"event_payload_schema_id"`
	EventPayloadHash           Digest                `json:"event_payload_hash"`
	ProtocolBundleManifestHash Digest                `json:"protocol_bundle_manifest_hash"`
	SubjectRef                 *AuditTypedReference  `json:"subject_ref,omitempty"`
	CauseRefs                  []AuditTypedReference `json:"cause_refs,omitempty"`
	RelatedRefs                []AuditTypedReference `json:"related_refs,omitempty"`
	SignerEvidenceRefs         []AuditTypedReference `json:"signer_evidence_refs,omitempty"`
	Scope                      map[string]string     `json:"scope,omitempty"`
	Correlation                map[string]string     `json:"correlation,omitempty"`
	GatewayContext             *AuditGatewayContext  `json:"gateway_context,omitempty"`
}

type AuditReceiptOperationalView struct {
	SchemaID             string          `json:"schema_id"`
	SchemaVersion        string          `json:"schema_version"`
	SubjectDigest        Digest          `json:"subject_digest"`
	AuditReceiptKind     string          `json:"audit_receipt_kind"`
	SubjectFamily        string          `json:"subject_family,omitempty"`
	RecordedAt           string          `json:"recorded_at"`
	ReceiptPayloadSchema string          `json:"receipt_payload_schema_id,omitempty"`
	AuthorityContext     json.RawMessage `json:"authority_context,omitempty"`
}

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

func ComputeSignedEnvelopeAuditRecordDigest(envelope SignedObjectEnvelope) (Digest, error) {
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return Digest{}, fmt.Errorf("marshal signed envelope: %w", err)
	}
	canonicalEnvelopeBytes, err := jsoncanonicalizer.Transform(envelopeBytes)
	if err != nil {
		return Digest{}, fmt.Errorf("canonicalize signed envelope: %w", err)
	}
	sum := sha256.Sum256(canonicalEnvelopeBytes)
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func BuildDefaultOperationalAuditView(envelope SignedObjectEnvelope) (AuditOperationalView, error) {
	if envelope.PayloadSchemaID == "" || envelope.PayloadSchemaVersion == "" {
		return AuditOperationalView{}, fmt.Errorf("payload schema identity is required")
	}

	recordDigest, err := ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return AuditOperationalView{}, err
	}

	view := newBaseOperationalView(envelope, recordDigest)

	if err := populateOperationalViewByPayloadSchema(&view, envelope); err != nil {
		return AuditOperationalView{}, err
	}

	sort.Strings(view.Redaction.ExcludedDataClasses)
	sort.Strings(view.Redaction.RedactedFields)
	return view, nil
}

func newBaseOperationalView(envelope SignedObjectEnvelope, recordDigest Digest) AuditOperationalView {
	return AuditOperationalView{
		ViewPolicyID:      AuditOperationalViewPolicyID,
		ViewPolicyVersion: AuditOperationalViewPolicyVersion,
		RecordDigest:      recordDigest,
		PayloadSchemaID:   envelope.PayloadSchemaID,
		PayloadVersion:    envelope.PayloadSchemaVersion,
		Signature:         envelope.Signature,
		Redaction: AuditOperationalRedaction{
			ExcludedDataClasses: []string{"sensitive", "secret"},
		},
	}
}

func populateOperationalViewByPayloadSchema(view *AuditOperationalView, envelope SignedObjectEnvelope) error {
	switch envelope.PayloadSchemaID {
	case AuditEventSchemaID:
		if err := populateOperationalEventView(view, envelope); err != nil {
			return err
		}

	case AuditReceiptSchemaID:
		if err := populateOperationalReceiptView(view, envelope); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported payload_schema_id %q for default operational view", envelope.PayloadSchemaID)
	}
	return nil
}

func populateOperationalEventView(view *AuditOperationalView, envelope SignedObjectEnvelope) error {
	if envelope.PayloadSchemaVersion != AuditEventSchemaVersion {
		return fmt.Errorf("unsupported audit event schema version %q", envelope.PayloadSchemaVersion)
	}
	event, err := decodeAuditEventPayload(envelope.Payload)
	if err != nil {
		return err
	}
	if err := validateAuditEventPayloadShape(event); err != nil {
		return err
	}
	view.Event = buildAuditEventOperationalPayload(event)
	view.Redaction.RedactedFields = []string{"event_payload", "principal"}
	return nil
}

func buildAuditEventOperationalPayload(event AuditEventPayload) *AuditEventOperationalPayload {
	return &AuditEventOperationalPayload{
		SchemaID:                   event.SchemaID,
		SchemaVersion:              event.SchemaVersion,
		AuditEventType:             event.AuditEventType,
		EmitterStreamID:            event.EmitterStreamID,
		Seq:                        event.Seq,
		OccurredAt:                 event.OccurredAt,
		PreviousEventHash:          event.PreviousEventHash,
		ActiveRoleManifestHash:     event.ActiveRoleManifestHash,
		ActiveCapabilityHash:       event.ActiveCapabilityManifestHash,
		EventPayloadSchemaID:       event.EventPayloadSchemaID,
		EventPayloadHash:           event.EventPayloadHash,
		ProtocolBundleManifestHash: event.ProtocolBundleManifestHash,
		SubjectRef:                 event.SubjectRef,
		CauseRefs:                  event.CauseRefs,
		RelatedRefs:                event.RelatedRefs,
		SignerEvidenceRefs:         event.SignerEvidenceRefs,
		Scope:                      event.Scope,
		Correlation:                event.Correlation,
		GatewayContext:             event.GatewayContext,
	}
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
		AuthorityContext:     authorityContext,
	}, nil
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
