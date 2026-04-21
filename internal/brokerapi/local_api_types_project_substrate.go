package brokerapi

import "github.com/runecode-ai/runecode/internal/projectsubstrate"

type ProjectSubstrateGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProjectSubstrateGetResponse struct {
	SchemaID       string                              `json:"schema_id"`
	SchemaVersion  string                              `json:"schema_version"`
	RequestID      string                              `json:"request_id"`
	RepositoryRoot string                              `json:"repository_root"`
	Contract       projectsubstrate.ContractState      `json:"contract"`
	Snapshot       projectsubstrate.ValidationSnapshot `json:"snapshot"`
}

type ProjectSubstratePostureGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProjectSubstratePostureGetResponse struct {
	SchemaID            string                              `json:"schema_id"`
	SchemaVersion       string                              `json:"schema_version"`
	RequestID           string                              `json:"request_id"`
	RepositoryRoot      string                              `json:"repository_root"`
	Contract            projectsubstrate.ContractState      `json:"contract"`
	Snapshot            projectsubstrate.ValidationSnapshot `json:"snapshot"`
	PostureSummary      ProjectSubstratePostureSummary      `json:"posture_summary"`
	Adoption            projectsubstrate.AdoptionResult     `json:"adoption"`
	InitPreview         projectsubstrate.InitPreview        `json:"init_preview"`
	UpgradePreview      projectsubstrate.UpgradePreview     `json:"upgrade_preview"`
	BlockedExplanation  string                              `json:"blocked_explanation,omitempty"`
	RemediationGuidance []string                            `json:"remediation_guidance,omitempty"`
}

type ProjectSubstrateAdoptRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProjectSubstrateAdoptResponse struct {
	SchemaID      string                          `json:"schema_id"`
	SchemaVersion string                          `json:"schema_version"`
	RequestID     string                          `json:"request_id"`
	Adoption      projectsubstrate.AdoptionResult `json:"adoption"`
}

type ProjectSubstrateInitPreviewRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProjectSubstrateInitPreviewResponse struct {
	SchemaID      string                       `json:"schema_id"`
	SchemaVersion string                       `json:"schema_version"`
	RequestID     string                       `json:"request_id"`
	Preview       projectsubstrate.InitPreview `json:"preview"`
}

type ProjectSubstrateInitApplyRequest struct {
	SchemaID             string `json:"schema_id"`
	SchemaVersion        string `json:"schema_version"`
	RequestID            string `json:"request_id"`
	ExpectedPreviewToken string `json:"expected_preview_token,omitempty"`
}

type ProjectSubstrateInitApplyResponse struct {
	SchemaID      string                           `json:"schema_id"`
	SchemaVersion string                           `json:"schema_version"`
	RequestID     string                           `json:"request_id"`
	ApplyResult   projectsubstrate.InitApplyResult `json:"apply_result"`
}

type ProjectSubstrateUpgradePreviewRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProjectSubstrateUpgradePreviewResponse struct {
	SchemaID      string                          `json:"schema_id"`
	SchemaVersion string                          `json:"schema_version"`
	RequestID     string                          `json:"request_id"`
	Preview       projectsubstrate.UpgradePreview `json:"preview"`
}

type ProjectSubstrateUpgradeApplyRequest struct {
	SchemaID              string `json:"schema_id"`
	SchemaVersion         string `json:"schema_version"`
	RequestID             string `json:"request_id"`
	ExpectedPreviewDigest string `json:"expected_preview_digest,omitempty"`
}

type ProjectSubstrateUpgradeApplyResponse struct {
	SchemaID      string                              `json:"schema_id"`
	SchemaVersion string                              `json:"schema_version"`
	RequestID     string                              `json:"request_id"`
	ApplyResult   projectsubstrate.UpgradeApplyResult `json:"apply_result"`
}
