package projectsubstrate

type AuditEventAppender interface {
	AppendTrustedAuditEvent(eventType, emitter string, details map[string]interface{}) error
}

type UpgradePreviewInput struct {
	RepositoryRoot string
	Authority      RepoRootAuthority
}

type UpgradeFileChange struct {
	Path             string `json:"path"`
	Action           string `json:"action"`
	BeforeContentSHA string `json:"before_content_sha256,omitempty"`
	AfterContentSHA  string `json:"after_content_sha256,omitempty"`
}

type UpgradePrecondition struct {
	Code      string `json:"code"`
	Satisfied bool   `json:"satisfied"`
}

type UpgradePreview struct {
	SchemaID         string                `json:"schema_id"`
	SchemaVersion    string                `json:"schema_version"`
	RepositoryRoot   string                `json:"repository_root"`
	Status           string                `json:"status"`
	ReasonCodes      []string              `json:"reason_codes,omitempty"`
	CurrentSnapshot  ValidationSnapshot    `json:"current_snapshot"`
	ExpectedSnapshot ValidationSnapshot    `json:"expected_snapshot"`
	FileChanges      []UpgradeFileChange   `json:"file_changes,omitempty"`
	Preconditions    []UpgradePrecondition `json:"preconditions,omitempty"`
	RequiredFollowUp []string              `json:"required_follow_up,omitempty"`
	PreviewDigest    string                `json:"preview_digest"`
}

type UpgradeApplyInput struct {
	Preview             UpgradePreview
	ExpectedPreviewHash string
	AuditAppender       AuditEventAppender
}

type UpgradeApplyResult struct {
	SchemaID          string              `json:"schema_id"`
	SchemaVersion     string              `json:"schema_version"`
	RepositoryRoot    string              `json:"repository_root"`
	Status            string              `json:"status"`
	ReasonCodes       []string            `json:"reason_codes,omitempty"`
	AppliedChanges    []UpgradeFileChange `json:"applied_changes,omitempty"`
	CurrentSnapshot   ValidationSnapshot  `json:"current_snapshot"`
	ResultingSnapshot ValidationSnapshot  `json:"resulting_snapshot"`
	PreviewDigest     string              `json:"preview_digest"`
}
