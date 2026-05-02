package auditd

import (
	"errors"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var (
	ErrAnchorReceiptInvalid = errors.New("anchor receipt invalid")
)

type AppendResult struct {
	SegmentID    string             `json:"segment_id"`
	RecordDigest trustpolicy.Digest `json:"record_digest"`
	ByteLength   int64              `json:"byte_length"`
	FrameCount   int                `json:"frame_count"`
}

type SealResult struct {
	SegmentID          string             `json:"segment_id"`
	SealEnvelopeDigest trustpolicy.Digest `json:"seal_envelope_digest"`
	NextOpenSegmentID  string             `json:"next_open_segment_id"`
}

type VerificationResult struct {
	SegmentID    string                                     `json:"segment_id"`
	ReportDigest trustpolicy.Digest                         `json:"report_digest"`
	Report       trustpolicy.AuditVerificationReportPayload `json:"report"`
}

type AnchorSegmentRequest struct {
	SealDigest             trustpolicy.Digest
	ApprovalDecisionDigest *trustpolicy.Digest
	ApprovalAssuranceLevel string
	AnchorKind             string
	KeyProtectionPosture   string
	PresenceMode           string
	AnchorWitnessKind      string
	AnchorWitnessDigest    trustpolicy.Digest
	Signature              trustpolicy.SignatureBlock
	Recorder               trustpolicy.PrincipalIdentity
	SignerPublicKeyBase64  string
	SignerKeyIDValue       string
	SignerLogicalScope     string
	SignerInstanceID       string
	RecordedAtRFC3339      string
}

type AnchorSegmentResult struct {
	SealDigest           trustpolicy.Digest `json:"seal_digest"`
	ReceiptDigest        trustpolicy.Digest `json:"receipt_digest"`
	VerificationDigest   trustpolicy.Digest `json:"verification_digest"`
	AnchorStatus         string             `json:"anchor_status"`
	FailureReasonCode    string             `json:"failure_reason_code,omitempty"`
	FailureReasonMessage string             `json:"failure_reason_message,omitempty"`
}

type TimelinePointer struct {
	SegmentID       string `json:"segment_id"`
	FrameIndex      int    `json:"frame_index"`
	RecordDigest    string `json:"record_digest"`
	EmitterStreamID string `json:"emitter_stream_id"`
	Sequence        int64  `json:"seq"`
	OccurredAt      string `json:"occurred_at"`
	RunID           string `json:"run_id,omitempty"`
}

type derivedIndex struct {
	SchemaVersion                  int                          `json:"schema_version"`
	BuiltAt                        string                       `json:"built_at"`
	TotalRecords                   int                          `json:"total_records"`
	LastIndexedSegmentID           string                       `json:"last_indexed_segment_id,omitempty"`
	RecordDigestLookup             map[string]RecordLookup      `json:"record_digest_lookup,omitempty"`
	SegmentSealLookup              map[string]SegmentSealLookup `json:"segment_seal_lookup,omitempty"`
	SealChainIndexLookup           map[string]string            `json:"seal_chain_index_lookup,omitempty"`
	LatestVerificationReportDigest string                       `json:"latest_verification_report_digest,omitempty"`
	RunTimeline                    []TimelinePointer            `json:"run_timeline,omitempty"`
}

type RecordLookup struct {
	SegmentID  string `json:"segment_id"`
	FrameIndex int    `json:"frame_index"`
}

type SegmentSealLookup struct {
	SealDigest     string `json:"seal_digest"`
	SealChainIndex int64  `json:"seal_chain_index"`
}

type AuditRecordInclusion struct {
	RecordDigest          string                                  `json:"record_digest"`
	RecordEnvelopeDigest  string                                  `json:"record_envelope_digest"`
	SegmentID             string                                  `json:"segment_id"`
	FrameIndex            int                                     `json:"frame_index"`
	SegmentRecordCount    int                                     `json:"segment_record_count"`
	SegmentSealDigest     string                                  `json:"segment_seal_digest,omitempty"`
	SegmentSealChainIndex *int64                                  `json:"segment_seal_chain_index,omitempty"`
	PreviousSealDigest    string                                  `json:"previous_seal_digest,omitempty"`
	OrderedMerkle         AuditRecordInclusionOrderedMerkleLookup `json:"ordered_merkle"`
}

type AuditRecordInclusionOrderedMerkleLookup struct {
	Profile              string   `json:"profile"`
	LeafIndex            int      `json:"leaf_index"`
	LeafCount            int      `json:"leaf_count"`
	SegmentMerkleRoot    string   `json:"segment_merkle_root,omitempty"`
	SegmentRecordDigests []string `json:"segment_record_digests"`
}

type ledgerState struct {
	SchemaVersion                int    `json:"schema_version"`
	CurrentOpenSegmentID         string `json:"current_open_segment_id"`
	NextSegmentNumber            int64  `json:"next_segment_number"`
	OpenFrameCount               int    `json:"open_frame_count"`
	LastSealEnvelopeDigest       string `json:"last_seal_envelope_digest,omitempty"`
	LastSealedSegmentID          string `json:"last_sealed_segment_id,omitempty"`
	LastVerificationReportDigest string `json:"last_verification_report_digest,omitempty"`
	RecoveryComplete             bool   `json:"recovery_complete"`
	LastIndexedRecordCount       int    `json:"last_indexed_record_count"`
}
