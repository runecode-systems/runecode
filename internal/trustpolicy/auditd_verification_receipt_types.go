package trustpolicy

import "encoding/json"

type auditReceiptPayloadStrict struct {
	SchemaID             string          `json:"schema_id"`
	SchemaVersion        string          `json:"schema_version"`
	SubjectDigest        Digest          `json:"subject_digest"`
	AuditReceiptKind     string          `json:"audit_receipt_kind"`
	SubjectFamily        string          `json:"subject_family,omitempty"`
	Recorder             json.RawMessage `json:"recorder"`
	RecordedAt           string          `json:"recorded_at"`
	ReceiptPayloadSchema string          `json:"receipt_payload_schema_id,omitempty"`
	ReceiptPayload       json.RawMessage `json:"receipt_payload,omitempty"`
}

type importRestoreReceiptPayload struct {
	ProvenanceAction      string                     `json:"provenance_action"`
	SegmentFileHashScope  string                     `json:"segment_file_hash_scope"`
	ImportedSegments      []importRestoreSegmentLink `json:"imported_segments"`
	SourceManifestDigests []Digest                   `json:"source_manifest_digests"`
	SourceInstanceID      string                     `json:"source_instance_id,omitempty"`
	Operator              *PrincipalIdentity         `json:"operator,omitempty"`
	AuthorityContext      *importRestoreAuthority    `json:"authority_context,omitempty"`
}

type importRestoreAuthority struct {
	AuthorityKind               string  `json:"authority_kind"`
	AuthorityID                 string  `json:"authority_id"`
	AuthorizationManifestDigest *Digest `json:"authorization_manifest_digest,omitempty"`
	Note                        string  `json:"note,omitempty"`
}

type importRestoreSegmentLink struct {
	ImportedSegmentSealDigest Digest `json:"imported_segment_seal_digest"`
	ImportedSegmentRoot       Digest `json:"imported_segment_root"`
	SourceSegmentFileHash     Digest `json:"source_segment_file_hash"`
	LocalSegmentFileHash      Digest `json:"local_segment_file_hash"`
	ByteIdentityVerified      bool   `json:"byte_identity_verified"`
}

type anchorReceiptPayload struct {
	AnchorKind           string               `json:"anchor_kind"`
	KeyProtectionPosture string               `json:"key_protection_posture"`
	PresenceMode         string               `json:"presence_mode"`
	ApprovalAssurance    string               `json:"approval_assurance_level,omitempty"`
	ApprovalDecision     *Digest              `json:"approval_decision_digest,omitempty"`
	AnchorWitness        anchorReceiptWitness `json:"anchor_witness"`
}

type anchorReceiptWitness struct {
	WitnessKind   string `json:"witness_kind"`
	WitnessDigest Digest `json:"witness_digest"`
}
