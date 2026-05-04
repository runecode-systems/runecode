package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

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

type AuditAnchorPresenceGetRequest struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	SealDigest    trustpolicy.Digest `json:"seal_digest"`
}

type AuditAnchorPreflightGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type AuditAnchorPreflightGetResponse struct {
	SchemaID             string                          `json:"schema_id"`
	SchemaVersion        string                          `json:"schema_version"`
	RequestID            string                          `json:"request_id"`
	LatestAnchorableSeal *AuditAnchorableSealRef         `json:"latest_anchorable_seal,omitempty"`
	SignerReadiness      AuditAnchorSignerReadiness      `json:"signer_readiness"`
	VerifierReadiness    AuditAnchorVerifierReadiness    `json:"verifier_readiness"`
	PresenceRequirements AuditAnchorPresenceRequirements `json:"presence_requirements"`
	ApprovalRequirements AuditAnchorApprovalRequirements `json:"approval_requirements"`
}

type AuditAnchorableSealRef struct {
	SegmentID  string             `json:"segment_id"`
	SealDigest trustpolicy.Digest `json:"seal_digest"`
}

type AuditAnchorSignerReadiness struct {
	Ready              bool   `json:"ready"`
	PresenceMode       string `json:"presence_mode,omitempty"`
	SignerLogicalScope string `json:"signer_logical_scope,omitempty"`
	ReasonCode         string `json:"reason_code,omitempty"`
	Message            string `json:"message,omitempty"`
}

type AuditAnchorVerifierReadiness struct {
	Ready      bool   `json:"ready"`
	ReasonCode string `json:"reason_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type AuditAnchorPresenceRequirements struct {
	Required         bool   `json:"required"`
	AttestationMode  string `json:"attestation_mode,omitempty"`
	AttestationReady bool   `json:"attestation_ready,omitempty"`
	ReasonCode       string `json:"reason_code,omitempty"`
	Message          string `json:"message,omitempty"`
}

type AuditAnchorApprovalRequirements struct {
	Required               bool   `json:"required"`
	RequiredAssuranceLevel string `json:"required_assurance_level,omitempty"`
	PolicyDecisionRef      string `json:"policy_decision_ref,omitempty"`
	ReasonCode             string `json:"reason_code,omitempty"`
	Message                string `json:"message,omitempty"`
}

type AuditAnchorPresenceGetResponse struct {
	SchemaID            string                          `json:"schema_id"`
	SchemaVersion       string                          `json:"schema_version"`
	RequestID           string                          `json:"request_id"`
	SealDigest          trustpolicy.Digest              `json:"seal_digest"`
	PresenceMode        string                          `json:"presence_mode"`
	PresenceAttestation *AuditAnchorPresenceAttestation `json:"presence_attestation,omitempty"`
}

type AuditAnchorPresenceAttestation struct {
	Challenge           string `json:"challenge"`
	AcknowledgmentToken string `json:"acknowledgment_token"`
}

type AuditAnchorSegmentResponse struct {
	SchemaID                 string              `json:"schema_id"`
	SchemaVersion            string              `json:"schema_version"`
	RequestID                string              `json:"request_id"`
	ProjectContextID         string              `json:"project_context_identity_digest,omitempty"`
	SealDigest               trustpolicy.Digest  `json:"seal_digest"`
	ReceiptDigest            *trustpolicy.Digest `json:"receipt_digest,omitempty"`
	VerificationReportDigest *trustpolicy.Digest `json:"verification_report_digest,omitempty"`
	AnchoringStatus          string              `json:"anchoring_status"`
	ExportedReceiptRef       string              `json:"exported_receipt_ref,omitempty"`
	FailureCode              string              `json:"failure_code,omitempty"`
	FailureMessage           string              `json:"failure_message,omitempty"`
}
