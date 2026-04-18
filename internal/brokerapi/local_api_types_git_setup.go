package brokerapi

type GitProviderAccountState struct {
	SchemaID        string `json:"schema_id"`
	SchemaVersion   string `json:"schema_version"`
	Provider        string `json:"provider"`
	AccountID       string `json:"account_id"`
	AccountUsername string `json:"account_username"`
	Linked          bool   `json:"linked"`
	Source          string `json:"source"`
}

type GitCommitIdentityProfile struct {
	SchemaID       string `json:"schema_id"`
	SchemaVersion  string `json:"schema_version"`
	ProfileID      string `json:"profile_id"`
	DisplayName    string `json:"display_name"`
	AuthorName     string `json:"author_name"`
	AuthorEmail    string `json:"author_email"`
	CommitterName  string `json:"committer_name"`
	CommitterEmail string `json:"committer_email"`
	SignoffName    string `json:"signoff_name"`
	SignoffEmail   string `json:"signoff_email"`
	DefaultProfile bool   `json:"default_profile"`
}

type GitAuthPostureState struct {
	SchemaID                        string `json:"schema_id"`
	SchemaVersion                   string `json:"schema_version"`
	Provider                        string `json:"provider"`
	AuthStatus                      string `json:"auth_status"`
	BootstrapMode                   string `json:"bootstrap_mode"`
	HeadlessBootstrapSupported      bool   `json:"headless_bootstrap_supported"`
	InteractiveTokenFallbackSupport bool   `json:"interactive_token_fallback_supported"`
}

type GitControlPlaneState struct {
	SchemaID                 string   `json:"schema_id"`
	SchemaVersion            string   `json:"schema_version"`
	Provider                 string   `json:"provider"`
	DefaultIdentityProfileID string   `json:"default_identity_profile_id"`
	LastSetupView            string   `json:"last_setup_view"`
	RecentRepositories       []string `json:"recent_repositories"`
}

type GitPolicySurfaceState struct {
	ArtifactManagedOnly   bool `json:"artifact_managed_only"`
	InspectionSupported   bool `json:"inspection_supported"`
	PrepareChangesSupport bool `json:"prepare_changes_supported"`
	DirectMutationSupport bool `json:"direct_mutation_supported"`
}

type GitSetupGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Provider      string `json:"provider"`
}

type GitSetupGetResponse struct {
	SchemaID          string                     `json:"schema_id"`
	SchemaVersion     string                     `json:"schema_version"`
	RequestID         string                     `json:"request_id"`
	ProviderAccount   GitProviderAccountState    `json:"provider_account"`
	IdentityProfiles  []GitCommitIdentityProfile `json:"identity_profiles"`
	AuthPosture       GitAuthPostureState        `json:"auth_posture"`
	ControlPlaneState GitControlPlaneState       `json:"control_plane_state"`
	PolicySurface     GitPolicySurfaceState      `json:"policy_surface"`
}

type GitSetupAuthBootstrapRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Provider      string `json:"provider"`
	Mode          string `json:"mode"`
}

type GitSetupAuthBootstrapResponse struct {
	SchemaID              string                  `json:"schema_id"`
	SchemaVersion         string                  `json:"schema_version"`
	RequestID             string                  `json:"request_id"`
	Provider              string                  `json:"provider"`
	Mode                  string                  `json:"mode"`
	Status                string                  `json:"status"`
	DeviceVerificationURI string                  `json:"device_verification_uri,omitempty"`
	DeviceUserCode        string                  `json:"device_user_code,omitempty"`
	NextPollAfterSeconds  int                     `json:"next_poll_after_seconds,omitempty"`
	AccountState          GitProviderAccountState `json:"account_state"`
	AuthPosture           GitAuthPostureState     `json:"auth_posture"`
}

type GitSetupIdentityUpsertRequest struct {
	SchemaID      string                   `json:"schema_id"`
	SchemaVersion string                   `json:"schema_version"`
	RequestID     string                   `json:"request_id"`
	Provider      string                   `json:"provider"`
	Profile       GitCommitIdentityProfile `json:"profile"`
}

type GitSetupIdentityUpsertResponse struct {
	SchemaID          string                   `json:"schema_id"`
	SchemaVersion     string                   `json:"schema_version"`
	RequestID         string                   `json:"request_id"`
	Provider          string                   `json:"provider"`
	Profile           GitCommitIdentityProfile `json:"profile"`
	ControlPlaneState GitControlPlaneState     `json:"control_plane_state"`
}
