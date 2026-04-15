package trustpolicy

import (
	"regexp"
	"time"
)

const (
	AuditVerificationReportSchemaID      = "runecode.protocol.v0.AuditVerificationReport"
	AuditVerificationReportSchemaVersion = "0.1.0"

	AuditVerificationStatusOK       = "ok"
	AuditVerificationStatusDegraded = "degraded"
	AuditVerificationStatusFailed   = "failed"

	AuditVerificationSeverityInfo    = "info"
	AuditVerificationSeverityWarning = "warning"
	AuditVerificationSeverityError   = "error"

	AuditVerificationDimensionIntegrity        = "integrity"
	AuditVerificationDimensionAnchoring        = "anchoring"
	AuditVerificationDimensionStoragePosture   = "storage_posture"
	AuditVerificationDimensionSegmentLifecycle = "segment_lifecycle"

	AuditVerificationScopeInstance     = "instance"
	AuditVerificationScopeSegmentRange = "segment_range"
	AuditVerificationScopeSegment      = "segment"

	AuditVerificationReasonSegmentFrameDigestMismatch          = "segment_frame_digest_mismatch"
	AuditVerificationReasonSegmentFrameByteLengthMismatch      = "segment_frame_byte_length_mismatch"
	AuditVerificationReasonSegmentFileHashMismatch             = "segment_file_hash_mismatch"
	AuditVerificationReasonSegmentMerkleRootMismatch           = "segment_merkle_root_mismatch"
	AuditVerificationReasonSegmentSealInvalid                  = "segment_seal_invalid"
	AuditVerificationReasonSegmentSealChainMismatch            = "segment_seal_chain_mismatch"
	AuditVerificationReasonStreamSequenceGap                   = "stream_sequence_gap"
	AuditVerificationReasonStreamSequenceRollbackOrDuplicate   = "stream_sequence_rollback_or_duplicate"
	AuditVerificationReasonStreamPreviousHashMismatch          = "stream_previous_hash_mismatch"
	AuditVerificationReasonDetachedSignatureInvalid            = "detached_signature_invalid"
	AuditVerificationReasonSignerEvidenceMissing               = "signer_evidence_missing"
	AuditVerificationReasonSignerEvidenceInvalid               = "signer_evidence_invalid"
	AuditVerificationReasonSignerHistoricallyInadmissible      = "signer_historically_inadmissible"
	AuditVerificationReasonSignerCurrentlyRevokedOrCompromised = "signer_currently_revoked_or_compromised"
	AuditVerificationReasonEventContractMismatch               = "event_contract_mismatch"
	AuditVerificationReasonEventContractMissing                = "event_contract_missing"
	AuditVerificationReasonImportRestoreProvenanceInconsistent = "import_restore_provenance_inconsistent"
	AuditVerificationReasonReceiptInvalid                      = "receipt_invalid"
	AuditVerificationReasonAnchorReceiptMissing                = "anchor_receipt_missing"
	AuditVerificationReasonAnchorReceiptInvalid                = "anchor_receipt_invalid"
	AuditVerificationReasonAnchorPassphrasePresenceDegraded    = "anchor_passphrase_presence_degraded"
	AuditVerificationReasonSegmentLifecycleInconsistent        = "segment_lifecycle_inconsistent"
	AuditVerificationReasonStoragePostureDegraded              = "storage_posture_degraded"
	AuditVerificationReasonStoragePostureInvalid               = "storage_posture_invalid"
)

var (
	auditVerificationCodePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	auditVerificationAllowedCodes = map[string]struct{}{
		AuditVerificationReasonSegmentFrameDigestMismatch:          {},
		AuditVerificationReasonSegmentFrameByteLengthMismatch:      {},
		AuditVerificationReasonSegmentFileHashMismatch:             {},
		AuditVerificationReasonSegmentMerkleRootMismatch:           {},
		AuditVerificationReasonSegmentSealInvalid:                  {},
		AuditVerificationReasonSegmentSealChainMismatch:            {},
		AuditVerificationReasonStreamSequenceGap:                   {},
		AuditVerificationReasonStreamSequenceRollbackOrDuplicate:   {},
		AuditVerificationReasonStreamPreviousHashMismatch:          {},
		AuditVerificationReasonDetachedSignatureInvalid:            {},
		AuditVerificationReasonSignerEvidenceMissing:               {},
		AuditVerificationReasonSignerEvidenceInvalid:               {},
		AuditVerificationReasonSignerHistoricallyInadmissible:      {},
		AuditVerificationReasonSignerCurrentlyRevokedOrCompromised: {},
		AuditVerificationReasonEventContractMismatch:               {},
		AuditVerificationReasonEventContractMissing:                {},
		AuditVerificationReasonImportRestoreProvenanceInconsistent: {},
		AuditVerificationReasonReceiptInvalid:                      {},
		AuditVerificationReasonAnchorReceiptMissing:                {},
		AuditVerificationReasonAnchorReceiptInvalid:                {},
		AuditVerificationReasonAnchorPassphrasePresenceDegraded:    {},
		AuditVerificationReasonSegmentLifecycleInconsistent:        {},
		AuditVerificationReasonStoragePostureDegraded:              {},
		AuditVerificationReasonStoragePostureInvalid:               {},
	}
)

type AuditVerificationScope struct {
	ScopeKind      string `json:"scope_kind"`
	FirstSegmentID string `json:"first_segment_id,omitempty"`
	LastSegmentID  string `json:"last_segment_id,omitempty"`
}

type AuditVerificationFinding struct {
	Code                 string         `json:"code"`
	Dimension            string         `json:"dimension"`
	Severity             string         `json:"severity"`
	Message              string         `json:"message"`
	SegmentID            string         `json:"segment_id,omitempty"`
	SubjectRecordDigest  *Digest        `json:"subject_record_digest,omitempty"`
	RelatedRecordDigests []Digest       `json:"related_record_digests,omitempty"`
	Details              map[string]any `json:"details,omitempty"`
}

type AuditVerificationReportPayload struct {
	SchemaID               string                     `json:"schema_id"`
	SchemaVersion          string                     `json:"schema_version"`
	VerifiedAt             string                     `json:"verified_at"`
	VerificationScope      AuditVerificationScope     `json:"verification_scope"`
	CryptographicallyValid bool                       `json:"cryptographically_valid"`
	HistoricallyAdmissible bool                       `json:"historically_admissible"`
	CurrentlyDegraded      bool                       `json:"currently_degraded"`
	IntegrityStatus        string                     `json:"integrity_status"`
	AnchoringStatus        string                     `json:"anchoring_status"`
	StoragePostureStatus   string                     `json:"storage_posture_status"`
	SegmentLifecycleStatus string                     `json:"segment_lifecycle_status"`
	DegradedReasons        []string                   `json:"degraded_reasons"`
	HardFailures           []string                   `json:"hard_failures"`
	Findings               []AuditVerificationFinding `json:"findings"`
	Summary                string                     `json:"summary,omitempty"`
}

type AuditSegmentHeader struct {
	Format       string `json:"format"`
	SegmentID    string `json:"segment_id"`
	SegmentState string `json:"segment_state"`
	CreatedAt    string `json:"created_at"`
	Writer       string `json:"writer,omitempty"`
}

type AuditSegmentRecordFrame struct {
	RecordDigest                 Digest `json:"record_digest"`
	ByteLength                   int64  `json:"byte_length"`
	CanonicalSignedEnvelopeBytes string `json:"canonical_signed_envelope_bytes"`
}

type AuditSegmentLifecycleMarker struct {
	State    string `json:"state"`
	MarkedAt string `json:"marked_at"`
	Reason   string `json:"reason,omitempty"`
}

type AuditSegmentFilePayload struct {
	SchemaID                  string                      `json:"schema_id"`
	SchemaVersion             string                      `json:"schema_version"`
	Header                    AuditSegmentHeader          `json:"header"`
	Frames                    []AuditSegmentRecordFrame   `json:"frames"`
	LifecycleMarker           AuditSegmentLifecycleMarker `json:"lifecycle_marker"`
	TrailingPartialFrameBytes int64                       `json:"trailing_partial_frame_bytes,omitempty"`
}

type AuditVerificationInput struct {
	Scope                    AuditVerificationScope
	Segment                  AuditSegmentFilePayload
	RawFramedSegmentBytes    []byte
	SegmentSealEnvelope      SignedObjectEnvelope
	PreviousSealEnvelopeHash *Digest
	KnownSealDigests         []Digest
	ReceiptEnvelopes         []SignedObjectEnvelope
	VerifierRecords          []VerifierRecord
	EventContractCatalog     AuditEventContractCatalog
	SignerEvidence           []AuditSignerEvidenceReference
	StoragePostureEvidence   *AuditStoragePostureEvidence
	Now                      time.Time
}

type streamState struct {
	seq    int64
	digest Digest
}
