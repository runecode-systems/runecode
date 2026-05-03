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

type derivedIndexMeta struct {
	SchemaVersion                  int    `json:"schema_version"`
	BuiltAt                        string `json:"built_at"`
	TotalRecords                   int    `json:"total_records"`
	RunTimelineCount               int    `json:"run_timeline_count,omitempty"`
	LastIndexedSegmentID           string `json:"last_indexed_segment_id,omitempty"`
	LatestVerificationReportDigest string `json:"latest_verification_report_digest,omitempty"`
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
	SegmentRecordDigests []string `json:"segment_record_digests,omitempty"`
	CompactPath          []string `json:"compact_path,omitempty"`
}

type AuditEvidenceSnapshot struct {
	SchemaID                    string   `json:"schema_id"`
	SchemaVersion               string   `json:"schema_version"`
	CreatedAt                   string   `json:"created_at"`
	SegmentIDs                  []string `json:"segment_ids,omitempty"`
	SegmentSealDigests          []string `json:"segment_seal_digests,omitempty"`
	AuditReceiptDigests         []string `json:"audit_receipt_digests,omitempty"`
	VerificationReportDigests   []string `json:"verification_report_digests,omitempty"`
	RuntimeEvidenceDigests      []string `json:"runtime_evidence_digests,omitempty"`
	VerifierRecordDigests       []string `json:"verifier_record_digests,omitempty"`
	EventContractCatalogDigests []string `json:"event_contract_catalog_digests,omitempty"`
	SignerEvidenceDigests       []string `json:"signer_evidence_digests,omitempty"`
	StoragePostureDigests       []string `json:"storage_posture_digests,omitempty"`
	TypedRequestDigests         []string `json:"typed_request_digests,omitempty"`
	ActionRequestDigests        []string `json:"action_request_digests,omitempty"`
	ControlPlaneDigests         []string `json:"control_plane_digests,omitempty"`
	AttestationEvidenceDigests  []string `json:"attestation_evidence_digests,omitempty"`
	InstanceIdentityDigests     []string `json:"instance_identity_digests,omitempty"`
	PolicyEvidenceDigests       []string `json:"policy_evidence_digests,omitempty"`
	RequiredApprovalIDs         []string `json:"required_approval_ids,omitempty"`
	ApprovalEvidenceDigests     []string `json:"approval_evidence_digests,omitempty"`
	AnchorEvidenceDigests       []string `json:"anchor_evidence_digests,omitempty"`
	ProviderInvocationDigests   []string `json:"provider_invocation_digests,omitempty"`
	SecretLeaseDigests          []string `json:"secret_lease_digests,omitempty"`
}

type AuditEvidenceSnapshotCompletenessReview struct {
	FullySatisfied        bool                                `json:"fully_satisfied"`
	RequiredIdentityCount int                                 `json:"required_identity_count"`
	Missing               []AuditEvidenceSnapshotCompleteness `json:"missing,omitempty"`
	DeclaredRedactions    []AuditEvidenceSnapshotCompleteness `json:"declared_redactions,omitempty"`
}

type AuditEvidenceSnapshotCompleteness struct {
	Family   string `json:"family"`
	Identity string `json:"identity"`
}
