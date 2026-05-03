package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type AuditEvidenceSnapshotGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type AuditEvidenceRetentionReviewRequest struct {
	SchemaID      string                   `json:"schema_id"`
	SchemaVersion string                   `json:"schema_version"`
	RequestID     string                   `json:"request_id"`
	Scope         AuditEvidenceBundleScope `json:"scope"`
}

type AuditEvidenceRetentionReviewResponse struct {
	SchemaID      string                            `json:"schema_id"`
	SchemaVersion string                            `json:"schema_version"`
	RequestID     string                            `json:"request_id"`
	Snapshot      AuditEvidenceSnapshot             `json:"snapshot"`
	Manifest      AuditEvidenceBundleManifest       `json:"manifest"`
	Completeness  AuditEvidenceSnapshotCompleteness `json:"completeness_review"`
}

type AuditEvidenceSnapshotCompleteness struct {
	FullySatisfied        bool                            `json:"fully_satisfied"`
	RequiredIdentityCount int                             `json:"required_identity_count"`
	Missing               []AuditEvidenceSnapshotIdentity `json:"missing,omitempty"`
	DeclaredRedactions    []AuditEvidenceSnapshotIdentity `json:"declared_redactions,omitempty"`
}

type AuditEvidenceSnapshotIdentity struct {
	Family   string              `json:"family"`
	Identity *trustpolicy.Digest `json:"identity,omitempty"`
}

type AuditEvidenceSnapshotGetResponse struct {
	SchemaID      string                `json:"schema_id"`
	SchemaVersion string                `json:"schema_version"`
	RequestID     string                `json:"request_id"`
	Snapshot      AuditEvidenceSnapshot `json:"snapshot"`
}

type AuditEvidenceSnapshot struct {
	SchemaID                   string               `json:"schema_id"`
	SchemaVersion              string               `json:"schema_version"`
	CreatedAt                  string               `json:"created_at"`
	SegmentIDs                 []string             `json:"segment_ids,omitempty"`
	SegmentSealDigests         []trustpolicy.Digest `json:"segment_seal_digests,omitempty"`
	AuditReceiptDigests        []trustpolicy.Digest `json:"audit_receipt_digests,omitempty"`
	VerificationReportDigests  []trustpolicy.Digest `json:"verification_report_digests,omitempty"`
	RuntimeEvidenceDigests     []trustpolicy.Digest `json:"runtime_evidence_digests,omitempty"`
	AttestationEvidenceDigests []trustpolicy.Digest `json:"attestation_evidence_digests,omitempty"`
	InstanceIdentityDigests    []trustpolicy.Digest `json:"instance_identity_digests,omitempty"`
	PolicyEvidenceDigests      []trustpolicy.Digest `json:"policy_evidence_digests,omitempty"`
	RequiredApprovalIDs        []string             `json:"required_approval_ids,omitempty"`
	ApprovalEvidenceDigests    []trustpolicy.Digest `json:"approval_evidence_digests,omitempty"`
	AnchorEvidenceDigests      []trustpolicy.Digest `json:"anchor_evidence_digests,omitempty"`
	ProviderInvocationDigests  []trustpolicy.Digest `json:"provider_invocation_digests,omitempty"`
	SecretLeaseDigests         []trustpolicy.Digest `json:"secret_lease_digests,omitempty"`
}

type AuditEvidenceBundleManifestGetRequest struct {
	SchemaID                string                               `json:"schema_id"`
	SchemaVersion           string                               `json:"schema_version"`
	RequestID               string                               `json:"request_id"`
	Scope                   AuditEvidenceBundleScope             `json:"scope"`
	ExportProfile           string                               `json:"export_profile"`
	CreatedByTool           AuditEvidenceBundleToolIdentity      `json:"created_by_tool"`
	DisclosurePosture       AuditEvidenceBundleDisclosurePosture `json:"disclosure_posture"`
	Redactions              []AuditEvidenceBundleRedaction       `json:"redactions,omitempty"`
	ExternalSharingIntended bool                                 `json:"external_sharing_intended,omitempty"`
}

type AuditEvidenceBundleManifestGetResponse struct {
	SchemaID       string                            `json:"schema_id"`
	SchemaVersion  string                            `json:"schema_version"`
	RequestID      string                            `json:"request_id"`
	Manifest       AuditEvidenceBundleManifest       `json:"manifest"`
	SignedManifest *trustpolicy.SignedObjectEnvelope `json:"signed_manifest,omitempty"`
}

type AuditEvidenceBundleExportRequest struct {
	SchemaID          string                               `json:"schema_id"`
	SchemaVersion     string                               `json:"schema_version"`
	RequestID         string                               `json:"request_id"`
	Scope             AuditEvidenceBundleScope             `json:"scope"`
	ExportProfile     string                               `json:"export_profile"`
	CreatedByTool     AuditEvidenceBundleToolIdentity      `json:"created_by_tool"`
	DisclosurePosture AuditEvidenceBundleDisclosurePosture `json:"disclosure_posture"`
	Redactions        []AuditEvidenceBundleRedaction       `json:"redactions,omitempty"`
	ArchiveFormat     string                               `json:"archive_format,omitempty"`
}

type AuditEvidenceBundleExportEvent struct {
	SchemaID       string                       `json:"schema_id"`
	SchemaVersion  string                       `json:"schema_version"`
	StreamID       string                       `json:"stream_id"`
	RequestID      string                       `json:"request_id"`
	Seq            int64                        `json:"seq"`
	EventType      string                       `json:"event_type"`
	Manifest       *AuditEvidenceBundleManifest `json:"manifest,omitempty"`
	ArchiveFormat  string                       `json:"archive_format,omitempty"`
	ChunkBase64    string                       `json:"chunk_base64,omitempty"`
	ChunkBytes     int                          `json:"chunk_bytes,omitempty"`
	ManifestDigest *trustpolicy.Digest          `json:"manifest_digest,omitempty"`
	Terminal       bool                         `json:"terminal,omitempty"`
	TerminalStatus string                       `json:"terminal_status,omitempty"`
	EOF            bool                         `json:"eof,omitempty"`
	Error          *ProtocolError               `json:"error,omitempty"`
}

type AuditEvidenceBundleOfflineVerifyRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	BundlePath    string `json:"bundle_path"`
	ArchiveFormat string `json:"archive_format,omitempty"`
}

type AuditEvidenceBundleOfflineVerifyResponse struct {
	SchemaID      string                                 `json:"schema_id"`
	SchemaVersion string                                 `json:"schema_version"`
	RequestID     string                                 `json:"request_id"`
	Verification  AuditEvidenceBundleOfflineVerification `json:"verification"`
}

type AuditEvidenceBundleOfflineVerification struct {
	SchemaID            string                                    `json:"schema_id"`
	SchemaVersion       string                                    `json:"schema_version"`
	VerifiedAt          string                                    `json:"verified_at"`
	ArchiveFormat       string                                    `json:"archive_format"`
	ManifestDigest      *trustpolicy.Digest                       `json:"manifest_digest,omitempty"`
	BundleID            string                                    `json:"bundle_id,omitempty"`
	ExportProfile       string                                    `json:"export_profile,omitempty"`
	Scope               AuditEvidenceBundleScope                  `json:"scope"`
	VerifierIdentity    AuditEvidenceBundleVerifierIdentity       `json:"verifier_identity"`
	TrustRootDigests    []trustpolicy.Digest                      `json:"trust_root_digests,omitempty"`
	VerificationStatus  string                                    `json:"verification_status"`
	Findings            []AuditEvidenceBundleOfflineFinding       `json:"findings,omitempty"`
	VerificationReports []AuditEvidenceBundleOfflineReportPosture `json:"verification_reports,omitempty"`
}

type AuditEvidenceBundleOfflineFinding struct {
	Code       string              `json:"code"`
	Severity   string              `json:"severity"`
	Message    string              `json:"message"`
	ObjectPath string              `json:"object_path,omitempty"`
	Digest     *trustpolicy.Digest `json:"digest,omitempty"`
}

type AuditEvidenceBundleOfflineReportPosture struct {
	Digest                 trustpolicy.Digest `json:"digest"`
	IntegrityStatus        string             `json:"integrity_status"`
	AnchoringStatus        string             `json:"anchoring_status"`
	StoragePostureStatus   string             `json:"storage_posture_status"`
	SegmentLifecycleStatus string             `json:"segment_lifecycle_status"`
	CurrentlyDegraded      bool               `json:"currently_degraded"`
	DegradedReasons        []string           `json:"degraded_reasons,omitempty"`
	HardFailures           []string           `json:"hard_failures,omitempty"`
}

type AuditEvidenceBundleManifest struct {
	SchemaID          string                                `json:"schema_id"`
	SchemaVersion     string                                `json:"schema_version"`
	BundleID          string                                `json:"bundle_id"`
	CreatedAt         string                                `json:"created_at"`
	CreatedByTool     AuditEvidenceBundleToolIdentity       `json:"created_by_tool"`
	ExportProfile     string                                `json:"export_profile"`
	Scope             AuditEvidenceBundleScope              `json:"scope"`
	ControlPlane      *AuditEvidenceBundleControlProvenance `json:"control_plane_provenance,omitempty"`
	InstanceIdentity  *trustpolicy.Digest                   `json:"instance_identity_digest,omitempty"`
	IncludedObjects   []AuditEvidenceBundleIncludedObject   `json:"included_objects,omitempty"`
	RootDigests       []trustpolicy.Digest                  `json:"root_digests,omitempty"`
	SealReferences    []AuditEvidenceBundleSealReference    `json:"seal_references,omitempty"`
	VerifierIdentity  AuditEvidenceBundleVerifierIdentity   `json:"verifier_identity"`
	TrustRootDigests  []trustpolicy.Digest                  `json:"trust_root_digests,omitempty"`
	DisclosurePosture AuditEvidenceBundleDisclosurePosture  `json:"disclosure_posture"`
	Redactions        []AuditEvidenceBundleRedaction        `json:"redactions,omitempty"`
}

type AuditEvidenceBundleToolIdentity struct {
	ToolName                   string              `json:"tool_name"`
	ToolVersion                string              `json:"tool_version"`
	BuildRevision              string              `json:"build_revision,omitempty"`
	ProtocolBundleManifestHash *trustpolicy.Digest `json:"protocol_bundle_manifest_hash,omitempty"`
}

type AuditEvidenceBundleControlProvenance struct {
	WorkflowDefinitionHash *trustpolicy.Digest `json:"workflow_definition_hash,omitempty"`
	ToolManifestDigest     *trustpolicy.Digest `json:"tool_manifest_digest,omitempty"`
	PromptTemplateDigest   *trustpolicy.Digest `json:"prompt_template_digest,omitempty"`
	ProtocolBundleHash     *trustpolicy.Digest `json:"protocol_bundle_manifest_hash,omitempty"`
	VerifierImplDigest     *trustpolicy.Digest `json:"verifier_implementation_digest,omitempty"`
	TrustPolicyDigest      *trustpolicy.Digest `json:"trust_policy_digest,omitempty"`
}

type AuditEvidenceBundleScope struct {
	ScopeKind       string               `json:"scope_kind"`
	RunID           string               `json:"run_id,omitempty"`
	IncidentID      string               `json:"incident_id,omitempty"`
	ArtifactDigests []trustpolicy.Digest `json:"artifact_digests,omitempty"`
}

type AuditEvidenceBundleIncludedObject struct {
	ObjectFamily string             `json:"object_family"`
	Digest       trustpolicy.Digest `json:"digest"`
	Path         string             `json:"path"`
	ByteLength   int64              `json:"byte_length"`
}

type AuditEvidenceBundleSealReference struct {
	SegmentID          string              `json:"segment_id"`
	SealDigest         trustpolicy.Digest  `json:"seal_digest"`
	SealChainIndex     int64               `json:"seal_chain_index"`
	PreviousSealDigest *trustpolicy.Digest `json:"previous_seal_digest,omitempty"`
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
