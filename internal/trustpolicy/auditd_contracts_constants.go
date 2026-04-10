package trustpolicy

import (
	"encoding/json"
	"regexp"
)

const (
	AuditEventSchemaID      = "runecode.protocol.v0.AuditEvent"
	AuditEventSchemaVersion = "0.5.0"

	IsolateSessionStartedPayloadSchemaID      = "runecode.protocol.v0.IsolateSessionStartedPayload"
	IsolateSessionStartedPayloadSchemaVersion = "0.1.0"
	IsolateSessionBoundPayloadSchemaID        = "runecode.protocol.v0.IsolateSessionBoundPayload"
	IsolateSessionBoundPayloadSchemaVersion   = "0.1.0"

	AuditSegmentSealSchemaID      = "runecode.protocol.v0.AuditSegmentSeal"
	AuditSegmentSealSchemaVersion = "0.2.0"

	AuditEventContractCatalogSchemaID      = "runecode.protocol.v0.AuditEventContractCatalog"
	AuditEventContractCatalogSchemaVersion = "0.1.0"

	AuditSegmentStateOpen        = "open"
	AuditSegmentStateSealed      = "sealed"
	AuditSegmentStateAnchored    = "anchored"
	AuditSegmentStateImported    = "imported"
	AuditSegmentStateQuarantined = "quarantined"

	AuditSegmentOwnershipScopeInstanceGlobal = "instance_global"

	AuditSegmentCutTriggerSizeWindow = "size_window"
	AuditSegmentCutTriggerTimeWindow = "time_window"

	AuditSegmentMerkleProfileOrderedDSEv1 = "sha256_ordered_dse_v1"
	AuditSegmentFileHashScopeRawFramedV1  = "raw_framed_segment_bytes_v1"
	AuditSegmentAnchoringSubjectSeal      = "audit_segment_seal"
)

var sealReasonPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type AuditTypedReference struct {
	ObjectFamily string `json:"object_family"`
	Digest       Digest `json:"digest"`
	RefRole      string `json:"ref_role"`
}

type AuditGatewayContext struct {
	EgressCategory string `json:"egress_category"`
}

type IsolateSessionStartedPayload struct {
	SchemaID                      string `json:"schema_id"`
	SchemaVersion                 string `json:"schema_version"`
	RunID                         string `json:"run_id"`
	IsolateID                     string `json:"isolate_id"`
	SessionID                     string `json:"session_id"`
	BackendKind                   string `json:"backend_kind"`
	IsolationAssuranceLevel       string `json:"isolation_assurance_level"`
	ProvisioningPosture           string `json:"provisioning_posture"`
	LaunchContextDigest           string `json:"launch_context_digest"`
	HandshakeTranscriptHash       string `json:"handshake_transcript_hash"`
	LaunchReceiptDigest           string `json:"launch_receipt_digest"`
	RuntimeImageDescriptorDigest  string `json:"runtime_image_descriptor_digest"`
	AppliedHardeningPostureDigest string `json:"applied_hardening_posture_digest"`
	AttestationEvidenceDigest     string `json:"attestation_evidence_digest,omitempty"`
}

type IsolateSessionBoundPayload struct {
	SchemaID                      string `json:"schema_id"`
	SchemaVersion                 string `json:"schema_version"`
	RunID                         string `json:"run_id"`
	IsolateID                     string `json:"isolate_id"`
	SessionID                     string `json:"session_id"`
	BackendKind                   string `json:"backend_kind"`
	IsolationAssuranceLevel       string `json:"isolation_assurance_level"`
	ProvisioningPosture           string `json:"provisioning_posture"`
	LaunchContextDigest           string `json:"launch_context_digest"`
	HandshakeTranscriptHash       string `json:"handshake_transcript_hash"`
	SessionBindingDigest          string `json:"session_binding_digest"`
	RuntimeImageDescriptorDigest  string `json:"runtime_image_descriptor_digest"`
	AppliedHardeningPostureDigest string `json:"applied_hardening_posture_digest"`
	AttestationEvidenceDigest     string `json:"attestation_evidence_digest,omitempty"`
}

type AuditEventPayload struct {
	SchemaID                     string                `json:"schema_id"`
	SchemaVersion                string                `json:"schema_version"`
	AuditEventType               string                `json:"audit_event_type"`
	EmitterStreamID              string                `json:"emitter_stream_id"`
	Seq                          int64                 `json:"seq"`
	OccurredAt                   string                `json:"occurred_at"`
	Principal                    PrincipalIdentity     `json:"principal"`
	PreviousEventHash            *Digest               `json:"previous_event_hash,omitempty"`
	ActiveRoleManifestHash       *Digest               `json:"active_role_manifest_hash,omitempty"`
	ActiveCapabilityManifestHash *Digest               `json:"active_capability_manifest_hash,omitempty"`
	EventPayloadSchemaID         string                `json:"event_payload_schema_id"`
	EventPayload                 json.RawMessage       `json:"event_payload"`
	EventPayloadHash             Digest                `json:"event_payload_hash"`
	ProtocolBundleManifestHash   Digest                `json:"protocol_bundle_manifest_hash"`
	SubjectRef                   *AuditTypedReference  `json:"subject_ref,omitempty"`
	CauseRefs                    []AuditTypedReference `json:"cause_refs,omitempty"`
	RelatedRefs                  []AuditTypedReference `json:"related_refs,omitempty"`
	SignerEvidenceRefs           []AuditTypedReference `json:"signer_evidence_refs,omitempty"`
	Scope                        map[string]string     `json:"scope,omitempty"`
	Correlation                  map[string]string     `json:"correlation,omitempty"`
	GatewayContext               *AuditGatewayContext  `json:"gateway_context,omitempty"`
}

type AuditEventContractCatalog struct {
	SchemaID      string                           `json:"schema_id"`
	SchemaVersion string                           `json:"schema_version"`
	CatalogID     string                           `json:"catalog_id"`
	Entries       []AuditEventContractCatalogEntry `json:"entries"`
}

type AuditEventContractCatalogEntry struct {
	AuditEventType                 string   `json:"audit_event_type"`
	AllowedPayloadSchemaIDs        []string `json:"allowed_payload_schema_ids"`
	AllowedSignerPurposes          []string `json:"allowed_signer_purposes"`
	AllowedSignerScopes            []string `json:"allowed_signer_scopes"`
	RequiredScopeFields            []string `json:"required_scope_fields"`
	RequiredCorrelationFields      []string `json:"required_correlation_fields"`
	RequireSubjectRef              bool     `json:"require_subject_ref"`
	AllowedSubjectRefRoles         []string `json:"allowed_subject_ref_roles"`
	AllowedCauseRefRoles           []string `json:"allowed_cause_ref_roles"`
	AllowedRelatedRefRoles         []string `json:"allowed_related_ref_roles"`
	RequireGatewayContext          bool     `json:"require_gateway_context"`
	AllowedGatewayEgressCategories []string `json:"allowed_gateway_egress_categories"`
	RequireSignerEvidenceRefs      bool     `json:"require_signer_evidence_refs"`
	AllowedSignerEvidenceRefRoles  []string `json:"allowed_signer_evidence_ref_roles"`
}

type AuditSignerEvidenceReference struct {
	Digest   Digest              `json:"digest"`
	Evidence AuditSignerEvidence `json:"evidence"`
}
