package projectsubstrate

type AdoptionInput struct {
	RepositoryRoot string
	Authority      RepoRootAuthority
}

type AdoptionResult struct {
	SchemaID       string             `json:"schema_id"`
	SchemaVersion  string             `json:"schema_version"`
	RepositoryRoot string             `json:"repository_root"`
	Status         string             `json:"status"`
	ReasonCodes    []string           `json:"reason_codes,omitempty"`
	Snapshot       ValidationSnapshot `json:"snapshot"`
}

type InitPreviewInput struct {
	RepositoryRoot string
	Authority      RepoRootAuthority
}

type InitFileChange struct {
	Path             string `json:"path"`
	Action           string `json:"action"`
	BeforeContentSHA string `json:"before_content_sha256,omitempty"`
	AfterContentSHA  string `json:"after_content_sha256,omitempty"`
}

type InitPreview struct {
	SchemaID         string             `json:"schema_id"`
	SchemaVersion    string             `json:"schema_version"`
	RepositoryRoot   string             `json:"repository_root"`
	Status           string             `json:"status"`
	ReasonCodes      []string           `json:"reason_codes,omitempty"`
	CurrentSnapshot  ValidationSnapshot `json:"current_snapshot"`
	ExpectedSnapshot ValidationSnapshot `json:"expected_snapshot"`
	FileChanges      []InitFileChange   `json:"file_changes,omitempty"`
	ConflictingPaths []string           `json:"conflicting_paths,omitempty"`
	RequiredFollowUp []string           `json:"required_follow_up,omitempty"`
	PreviewToken     string             `json:"preview_token"`
}

type InitApplyInput struct {
	Preview              InitPreview
	ExpectedPreviewToken string
}

type InitApplyResult struct {
	SchemaID          string             `json:"schema_id"`
	SchemaVersion     string             `json:"schema_version"`
	RepositoryRoot    string             `json:"repository_root"`
	Status            string             `json:"status"`
	ReasonCodes       []string           `json:"reason_codes,omitempty"`
	AppliedChanges    []InitFileChange   `json:"applied_changes,omitempty"`
	CurrentSnapshot   ValidationSnapshot `json:"current_snapshot"`
	ResultingSnapshot ValidationSnapshot `json:"resulting_snapshot"`
	PreviewToken      string             `json:"preview_token"`
}
