package artifacts

import "time"

type DataClass string

const (
	DataClassSpecText                DataClass = "spec_text"
	DataClassUnapprovedFileExcerpts  DataClass = "unapproved_file_excerpts"
	DataClassApprovedFileExcerpts    DataClass = "approved_file_excerpts"
	DataClassDiffs                   DataClass = "diffs"
	DataClassBuildLogs               DataClass = "build_logs"
	DataClassAuditEvents             DataClass = "audit_events"
	DataClassAuditVerificationReport DataClass = "audit_verification_report"
	DataClassGateEvidence            DataClass = "gate_evidence"
	DataClassWebQuery                DataClass = "web_query"
	DataClassWebCitations            DataClass = "web_citations"
)

var allDataClasses = map[DataClass]struct{}{
	DataClassSpecText:                {},
	DataClassUnapprovedFileExcerpts:  {},
	DataClassApprovedFileExcerpts:    {},
	DataClassDiffs:                   {},
	DataClassBuildLogs:               {},
	DataClassAuditEvents:             {},
	DataClassAuditVerificationReport: {},
	DataClassGateEvidence:            {},
	DataClassWebQuery:                {},
	DataClassWebCitations:            {},
}

type ArtifactReference struct {
	Digest                string    `json:"digest"`
	SizeBytes             int64     `json:"size_bytes"`
	ContentType           string    `json:"content_type"`
	DataClass             DataClass `json:"data_class"`
	ProvenanceReceiptHash string    `json:"provenance_receipt_hash"`
}

type ArtifactRecord struct {
	Reference             ArtifactReference `json:"reference"`
	BlobPath              string            `json:"blob_path"`
	CreatedAt             time.Time         `json:"created_at"`
	CreatedByRole         string            `json:"created_by_role"`
	RunID                 string            `json:"run_id,omitempty"`
	StepID                string            `json:"step_id,omitempty"`
	StorageProtection     string            `json:"storage_protection"`
	ApprovalOfDigest      string            `json:"approval_of_digest,omitempty"`
	ApprovalDecisionHash  string            `json:"approval_decision_hash,omitempty"`
	PromotionRequestHash  string            `json:"promotion_request_hash,omitempty"`
	PromotionApprovedBy   string            `json:"promotion_approved_by,omitempty"`
	PromotionApprovedAt   *time.Time        `json:"promotion_approved_at,omitempty"`
	RetentionProtectedRun []string          `json:"retention_protected_run,omitempty"`
}

type Quota struct {
	MaxArtifactCount      int   `json:"max_artifact_count"`
	MaxTotalBytes         int64 `json:"max_total_bytes"`
	MaxSingleArtifactSize int64 `json:"max_single_artifact_bytes"`
}

type FlowRule struct {
	ProducerRole       string      `json:"producer_role"`
	ConsumerRole       string      `json:"consumer_role"`
	AllowedDataClasses []DataClass `json:"allowed_data_classes"`
}

type Policy struct {
	HandOffReferenceMode                string           `json:"handoff_reference_mode"`
	ReservedClassesEnabled              bool             `json:"reserved_classes_enabled"`
	EncryptedAtRestDefault              bool             `json:"encrypted_at_rest_default"`
	DevPlaintextOverride                bool             `json:"dev_plaintext_override"`
	ExplicitHumanApprovalRequired       bool             `json:"explicit_human_approval_required"`
	PromotionMintsNewArtifactReference  bool             `json:"promotion_mints_new_artifact_reference"`
	MaxPromotionRequestBytes            int64            `json:"max_promotion_request_bytes"`
	MaxPromotionRequestsPerMinute       int              `json:"max_promotion_requests_per_minute"`
	BulkPromotionRequiresSeparateReview bool             `json:"bulk_promotion_requires_separate_approval"`
	FlowMatrix                          []FlowRule       `json:"flow_matrix"`
	RevokedApprovedExcerptHashes        map[string]bool  `json:"revoked_approved_excerpt_hashes"`
	PerRoleQuota                        map[string]Quota `json:"per_role_quota"`
	PerStepQuota                        map[string]Quota `json:"per_step_quota"`
	UnreferencedTTLSeconds              int64            `json:"unreferenced_ttl_seconds"`
	DeleteOnQuotaPressure               bool             `json:"delete_unreferenced_on_quota_pressure"`
	RequireOriginMetadata               []string         `json:"require_origin_metadata"`
	RequireFullContentVisibility        bool             `json:"require_full_content_visibility"`
	ApprovedExcerptEgressOptInOnly      bool             `json:"approved_excerpt_egress_opt_in_only"`
	UnapprovedExcerptEgressDenied       bool             `json:"unapproved_excerpt_egress_denied"`
}

type PutRequest struct {
	Payload               []byte
	ContentType           string
	DataClass             DataClass
	ProvenanceReceiptHash string
	CreatedByRole         string
	TrustedSource         bool
	RunID                 string
	StepID                string
}

type FlowCheckRequest struct {
	ProducerRole  string
	ConsumerRole  string
	DataClass     DataClass
	Digest        string
	IsEgress      bool
	ManifestOptIn bool
}

type ArtifactReadRequest struct {
	Digest        string
	ProducerRole  string
	ConsumerRole  string
	DataClass     DataClass
	IsEgress      bool
	ManifestOptIn bool
}

type GCResult struct {
	DeletedDigests []string `json:"deleted_digests"`
	FreedBytes     int64    `json:"freed_bytes"`
}

type AuditEvent struct {
	Seq        int64                  `json:"seq"`
	Type       string                 `json:"type"`
	OccurredAt time.Time              `json:"occurred_at"`
	Actor      string                 `json:"actor"`
	Details    map[string]interface{} `json:"details"`
}

type BackupSignature struct {
	Schema         string    `json:"schema"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	HMACSHA256     string    `json:"hmac_sha256"`
	KeyID          string    `json:"key_id"`
	ExportedAt     time.Time `json:"exported_at"`
}
