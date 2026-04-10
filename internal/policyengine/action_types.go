package policyengine

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type ActionActor struct {
	ActorKind  string
	RoleFamily string
	RoleKind   string
}

type ActionEnvelope struct {
	CapabilityID           string
	AllowlistRefs          []string
	RelevantArtifactHashes []trustpolicy.Digest
	Actor                  ActionActor
}

type WorkspaceWriteActionInput struct {
	ActionEnvelope
	TargetPath    string
	WriteMode     string
	SourcePath    string
	ContentSHA256 *trustpolicy.Digest
	Bytes         *int64
}

type ExecutorRunActionInput struct {
	ActionEnvelope
	ExecutorClass    string
	ExecutorID       string
	Argv             []string
	Environment      map[string]string
	WorkingDirectory string
	NetworkAccess    string
	TimeoutSeconds   *int
}

type ArtifactReadActionInput struct {
	ActionEnvelope
	ArtifactHash      trustpolicy.Digest
	ReadMode          string
	ExpectedDataClass string
	Purpose           string
	MaxBytes          *int64
}

type PromotionActionInput struct {
	ActionEnvelope
	PromotionKind        string
	SourceArtifactHash   trustpolicy.Digest
	TargetDataClass      string
	ByteStart            *int64
	ByteEnd              *int64
	Justification        string
	RepoPath             string
	Commit               string
	ExtractorToolVersion string
	Approver             string
}

type GatewayEgressActionInput struct {
	ActionEnvelope
	GatewayRoleKind string
	DestinationKind string
	DestinationRef  string
	EgressDataClass string
	Operation       string
	PayloadHash     *trustpolicy.Digest
}

type BackendPostureChangeActionInput struct {
	ActionEnvelope
	BackendClass     string
	ChangeKind       string
	RequestedPosture string
	RequiresOptIn    *bool
	Reason           string
}

type GateOverrideActionInput struct {
	ActionEnvelope
	GateName      string
	OverrideMode  string
	Justification string
	ExpiresAt     string
	TicketRef     string
}

type SecretAccessActionInput struct {
	ActionEnvelope
	SecretRef       string
	AccessMode      string
	LeaseTTLSeconds *int
	Justification   string
	RequiresEgress  *bool
	TargetSystem    string
}

type StageSummarySignOffActionInput struct {
	ActionEnvelope
	RunID            string
	StageID          string
	StageSummaryHash trustpolicy.Digest
	ApprovalProfile  string
	SummaryRevision  *int64
}
