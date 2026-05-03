package auditd

import "io"

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
	SchemaID          string                                `json:"schema_id"`
	SchemaVersion     string                                `json:"schema_version"`
	BundleID          string                                `json:"bundle_id"`
	CreatedAt         string                                `json:"created_at"`
	CreatedByTool     AuditEvidenceBundleToolIdentity       `json:"created_by_tool"`
	ExportProfile     string                                `json:"export_profile"`
	Scope             AuditEvidenceBundleScope              `json:"scope"`
	ControlPlane      *AuditEvidenceBundleControlProvenance `json:"control_plane_provenance,omitempty"`
	InstanceIdentity  string                                `json:"instance_identity_digest,omitempty"`
	IncludedObjects   []AuditEvidenceBundleIncludedObject   `json:"included_objects,omitempty"`
	RootDigests       []string                              `json:"root_digests,omitempty"`
	SealReferences    []AuditEvidenceBundleSealReference    `json:"seal_references,omitempty"`
	VerifierIdentity  AuditEvidenceBundleVerifierIdentity   `json:"verifier_identity"`
	TrustRootDigests  []string                              `json:"trust_root_digests,omitempty"`
	DisclosurePosture AuditEvidenceBundleDisclosurePosture  `json:"disclosure_posture"`
	Redactions        []AuditEvidenceBundleRedaction        `json:"redactions,omitempty"`
}

type AuditEvidenceBundleToolIdentity struct {
	ToolName                   string `json:"tool_name"`
	ToolVersion                string `json:"tool_version"`
	BuildRevision              string `json:"build_revision,omitempty"`
	ProtocolBundleManifestHash string `json:"protocol_bundle_manifest_hash,omitempty"`
}

type AuditEvidenceBundleControlProvenance struct {
	WorkflowDefinitionHash string `json:"workflow_definition_hash,omitempty"`
	ToolManifestDigest     string `json:"tool_manifest_digest,omitempty"`
	PromptTemplateDigest   string `json:"prompt_template_digest,omitempty"`
	ProtocolBundleHash     string `json:"protocol_bundle_manifest_hash,omitempty"`
	VerifierImplDigest     string `json:"verifier_implementation_digest,omitempty"`
	TrustPolicyDigest      string `json:"trust_policy_digest,omitempty"`
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
