package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type ZKProofGenerateRequest struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	RecordDigest  trustpolicy.Digest `json:"record_digest"`
}

type ZKProofGenerateResponse struct {
	SchemaID                  string              `json:"schema_id"`
	SchemaVersion             string              `json:"schema_version"`
	RequestID                 string              `json:"request_id"`
	StatementFamily           string              `json:"statement_family"`
	StatementVersion          string              `json:"statement_version"`
	NormalizationProfileID    string              `json:"normalization_profile_id"`
	SchemeAdapterID           string              `json:"scheme_adapter_id"`
	RecordDigest              trustpolicy.Digest  `json:"record_digest"`
	AuditProofBindingDigest   trustpolicy.Digest  `json:"audit_proof_binding_digest"`
	ZKProofArtifactDigest     trustpolicy.Digest  `json:"zk_proof_artifact_digest"`
	ZKProofVerificationDigest *trustpolicy.Digest `json:"zk_proof_verification_record_digest,omitempty"`
	EvaluationGate            string              `json:"evaluation_gate"`
	UserCheckInRequired       bool                `json:"user_check_in_required"`
	CheckInNote               string              `json:"check_in_note"`
}

type ZKProofVerifyRequest struct {
	SchemaID              string             `json:"schema_id"`
	SchemaVersion         string             `json:"schema_version"`
	RequestID             string             `json:"request_id"`
	ZKProofArtifactDigest trustpolicy.Digest `json:"zk_proof_artifact_digest"`
}

type ZKProofVerifyResponse struct {
	SchemaID                        string             `json:"schema_id"`
	SchemaVersion                   string             `json:"schema_version"`
	RequestID                       string             `json:"request_id"`
	ZKProofArtifactDigest           trustpolicy.Digest `json:"zk_proof_artifact_digest"`
	ZKProofVerificationRecordDigest trustpolicy.Digest `json:"zk_proof_verification_record_digest"`
	VerificationOutcome             string             `json:"verification_outcome"`
	ReasonCodes                     []string           `json:"reason_codes"`
	CacheProvenance                 string             `json:"cache_provenance"`
	EvaluationGate                  string             `json:"evaluation_gate"`
	UserCheckInRequired             bool               `json:"user_check_in_required"`
	CheckInNote                     string             `json:"check_in_note"`
}
