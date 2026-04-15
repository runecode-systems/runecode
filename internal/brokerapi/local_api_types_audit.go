package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type AuditTimelineRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
}

type AuditTimelineResponse struct {
	SchemaID      string                   `json:"schema_id"`
	SchemaVersion string                   `json:"schema_version"`
	RequestID     string                   `json:"request_id"`
	Order         string                   `json:"order"`
	Views         []AuditTimelineViewEntry `json:"views"`
	NextCursor    string                   `json:"next_cursor,omitempty"`
}

type AuditTimelineViewEntry struct {
	RecordDigest        trustpolicy.Digest              `json:"record_digest"`
	EventType           string                          `json:"event_type,omitempty"`
	Summary             string                          `json:"summary,omitempty"`
	LinkedReferences    []AuditRecordLinkedReference    `json:"linked_references,omitempty"`
	VerificationPosture *AuditRecordVerificationPosture `json:"verification_posture,omitempty"`
}

type AuditVerificationGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	ViewLimit     int    `json:"view_limit,omitempty"`
}

type AuditVerificationGetResponse struct {
	SchemaID      string                                         `json:"schema_id"`
	SchemaVersion string                                         `json:"schema_version"`
	RequestID     string                                         `json:"request_id"`
	Summary       trustpolicy.DerivedRunAuditVerificationSummary `json:"summary"`
	Report        trustpolicy.AuditVerificationReportPayload     `json:"report"`
	Views         []trustpolicy.AuditOperationalView             `json:"views"`
}

type AuditRecordLinkedReference struct {
	ReferenceKind string `json:"reference_kind"`
	ReferenceID   string `json:"reference_id"`
	Label         string `json:"label,omitempty"`
	Relation      string `json:"relation,omitempty"`
}

type AuditRecordVerificationPosture struct {
	Status      string   `json:"status"`
	ReasonCodes []string `json:"reason_codes"`
}

type AuditRecordScope struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	StageID     string `json:"stage_id,omitempty"`
	StepID      string `json:"step_id,omitempty"`
}

type AuditRecordCorrelation struct {
	SessionID         string `json:"session_id,omitempty"`
	OperationID       string `json:"operation_id,omitempty"`
	ParentOperationID string `json:"parent_operation_id,omitempty"`
}

type AuditRecordDetail struct {
	SchemaID            string                          `json:"schema_id"`
	SchemaVersion       string                          `json:"schema_version"`
	RecordDigest        trustpolicy.Digest              `json:"record_digest"`
	RecordFamily        string                          `json:"record_family"`
	OccurredAt          string                          `json:"occurred_at"`
	EventType           string                          `json:"event_type,omitempty"`
	Summary             string                          `json:"summary"`
	LinkedReferences    []AuditRecordLinkedReference    `json:"linked_references"`
	VerificationPosture *AuditRecordVerificationPosture `json:"verification_posture,omitempty"`
	Scope               *AuditRecordScope               `json:"scope,omitempty"`
	Correlation         *AuditRecordCorrelation         `json:"correlation,omitempty"`
}

type AuditRecordGetRequest struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	RecordDigest  trustpolicy.Digest `json:"record_digest"`
}

type AuditRecordGetResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	Record        AuditRecordDetail `json:"record"`
}

type AuditAnchorSegmentRequest struct {
	SchemaID               string                          `json:"schema_id"`
	SchemaVersion          string                          `json:"schema_version"`
	RequestID              string                          `json:"request_id"`
	SealDigest             trustpolicy.Digest              `json:"seal_digest"`
	ApprovalDecisionDigest *trustpolicy.Digest             `json:"approval_decision_digest,omitempty"`
	ApprovalAssuranceLevel string                          `json:"approval_assurance_level,omitempty"`
	SignerLogicalScope     string                          `json:"signer_logical_scope,omitempty"`
	SignerInstanceID       string                          `json:"signer_instance_id,omitempty"`
	PresenceAttestation    *AuditAnchorPresenceAttestation `json:"presence_attestation,omitempty"`
	ExportReceiptCopy      bool                            `json:"export_receipt_copy,omitempty"`
}

type AuditAnchorPresenceAttestation struct {
	Challenge           string `json:"challenge"`
	AcknowledgmentToken string `json:"acknowledgment_token"`
}

type AuditAnchorSegmentResponse struct {
	SchemaID                 string              `json:"schema_id"`
	SchemaVersion            string              `json:"schema_version"`
	RequestID                string              `json:"request_id"`
	SealDigest               trustpolicy.Digest  `json:"seal_digest"`
	ReceiptDigest            *trustpolicy.Digest `json:"receipt_digest,omitempty"`
	VerificationReportDigest *trustpolicy.Digest `json:"verification_report_digest,omitempty"`
	AnchoringStatus          string              `json:"anchoring_status"`
	ExportedReceiptRef       string              `json:"exported_receipt_ref,omitempty"`
	FailureCode              string              `json:"failure_code,omitempty"`
	FailureMessage           string              `json:"failure_message,omitempty"`
}
