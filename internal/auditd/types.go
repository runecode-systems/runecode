package auditd

import (
	"errors"
	"io"

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

type AuditEvidenceSnapshot struct {
	SchemaID                   string   `json:"schema_id"`
	SchemaVersion              string   `json:"schema_version"`
	CreatedAt                  string   `json:"created_at"`
	SegmentIDs                 []string `json:"segment_ids,omitempty"`
	SegmentSealDigests         []string `json:"segment_seal_digests,omitempty"`
	AuditReceiptDigests        []string `json:"audit_receipt_digests,omitempty"`
	VerificationReportDigests  []string `json:"verification_report_digests,omitempty"`
	RuntimeEvidenceDigests     []string `json:"runtime_evidence_digests,omitempty"`
	AttestationEvidenceDigests []string `json:"attestation_evidence_digests,omitempty"`
	InstanceIdentityDigests    []string `json:"instance_identity_digests,omitempty"`
	PolicyEvidenceDigests      []string `json:"policy_evidence_digests,omitempty"`
	RequiredApprovalIDs        []string `json:"required_approval_ids,omitempty"`
	ApprovalEvidenceDigests    []string `json:"approval_evidence_digests,omitempty"`
	AnchorEvidenceDigests      []string `json:"anchor_evidence_digests,omitempty"`
	ProviderInvocationDigests  []string `json:"provider_invocation_digests,omitempty"`
	SecretLeaseDigests         []string `json:"secret_lease_digests,omitempty"`
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

type AuditEvidenceBundleManifestRequest struct {
	Scope             AuditEvidenceBundleScope
	ExportProfile     string
	CreatedByTool     AuditEvidenceBundleToolIdentity
	DisclosurePosture AuditEvidenceBundleDisclosurePosture
	Redactions        []AuditEvidenceBundleRedaction
}

type AuditEvidenceBundleExportRequest struct {
	ManifestRequest AuditEvidenceBundleManifestRequest
	ArchiveFormat   string
}

type AuditEvidenceBundleExport struct {
	Manifest AuditEvidenceBundleManifest
	Reader   io.ReadCloser
}

type AuditEvidenceBundleManifest struct {
	SchemaID          string                               `json:"schema_id"`
	SchemaVersion     string                               `json:"schema_version"`
	BundleID          string                               `json:"bundle_id"`
	CreatedAt         string                               `json:"created_at"`
	CreatedByTool     AuditEvidenceBundleToolIdentity      `json:"created_by_tool"`
	ExportProfile     string                               `json:"export_profile"`
	Scope             AuditEvidenceBundleScope             `json:"scope"`
	InstanceIdentity  string                               `json:"instance_identity_digest,omitempty"`
	IncludedObjects   []AuditEvidenceBundleIncludedObject  `json:"included_objects,omitempty"`
	RootDigests       []string                             `json:"root_digests,omitempty"`
	SealReferences    []AuditEvidenceBundleSealReference   `json:"seal_references,omitempty"`
	VerifierIdentity  AuditEvidenceBundleVerifierIdentity  `json:"verifier_identity"`
	TrustRootDigests  []string                             `json:"trust_root_digests,omitempty"`
	DisclosurePosture AuditEvidenceBundleDisclosurePosture `json:"disclosure_posture"`
	Redactions        []AuditEvidenceBundleRedaction       `json:"redactions,omitempty"`
}

type AuditEvidenceBundleToolIdentity struct {
	ToolName                   string `json:"tool_name"`
	ToolVersion                string `json:"tool_version"`
	BuildRevision              string `json:"build_revision,omitempty"`
	ProtocolBundleManifestHash string `json:"protocol_bundle_manifest_hash,omitempty"`
}

type AuditEvidenceBundleScope struct {
	ScopeKind       string   `json:"scope_kind"`
	RunID           string   `json:"run_id,omitempty"`
	IncidentID      string   `json:"incident_id,omitempty"`
	ArtifactDigests []string `json:"artifact_digests,omitempty"`
}

type AuditEvidenceBundleIncludedObject struct {
	ObjectFamily string `json:"object_family"`
	Digest       string `json:"digest"`
	Path         string `json:"path"`
	ByteLength   int64  `json:"byte_length"`
}

type AuditEvidenceBundleSealReference struct {
	SegmentID          string `json:"segment_id"`
	SealDigest         string `json:"seal_digest"`
	SealChainIndex     int64  `json:"seal_chain_index"`
	PreviousSealDigest string `json:"previous_seal_digest,omitempty"`
}

type AuditEvidenceBundleVerifierIdentity struct {
	KeyID          string `json:"key_id"`
	KeyIDValue     string `json:"key_id_value"`
	LogicalPurpose string `json:"logical_purpose"`
	LogicalScope   string `json:"logical_scope"`
}

type AuditEvidenceBundleDisclosurePosture struct {
	Posture                    string `json:"posture"`
	SelectiveDisclosureApplied bool   `json:"selective_disclosure_applied"`
}

type AuditEvidenceBundleRedaction struct {
	Path       string `json:"path"`
	ReasonCode string `json:"reason_code"`
}

type AuditEvidenceBundleOfflineVerification struct {
	SchemaID            string                                    `json:"schema_id"`
	SchemaVersion       string                                    `json:"schema_version"`
	VerifiedAt          string                                    `json:"verified_at"`
	ArchiveFormat       string                                    `json:"archive_format"`
	ManifestDigest      string                                    `json:"manifest_digest,omitempty"`
	BundleID            string                                    `json:"bundle_id,omitempty"`
	ExportProfile       string                                    `json:"export_profile,omitempty"`
	Scope               AuditEvidenceBundleScope                  `json:"scope"`
	VerifierIdentity    AuditEvidenceBundleVerifierIdentity       `json:"verifier_identity"`
	TrustRootDigests    []string                                  `json:"trust_root_digests,omitempty"`
	VerificationStatus  string                                    `json:"verification_status"`
	Findings            []AuditEvidenceBundleOfflineFinding       `json:"findings,omitempty"`
	VerificationReports []AuditEvidenceBundleOfflineReportPosture `json:"verification_reports,omitempty"`
}

type AuditEvidenceBundleOfflineFinding struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	ObjectPath string `json:"object_path,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

type AuditEvidenceBundleOfflineReportPosture struct {
	Digest                 string   `json:"digest"`
	IntegrityStatus        string   `json:"integrity_status"`
	AnchoringStatus        string   `json:"anchoring_status"`
	StoragePostureStatus   string   `json:"storage_posture_status"`
	SegmentLifecycleStatus string   `json:"segment_lifecycle_status"`
	CurrentlyDegraded      bool     `json:"currently_degraded"`
	DegradedReasons        []string `json:"degraded_reasons,omitempty"`
	HardFailures           []string `json:"hard_failures,omitempty"`
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
