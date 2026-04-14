package policyengine

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type SecretLeaseRenewalContext struct {
	ConsumerPrincipalRef string
	TargetRef            string
	PolicyContextHash    trustpolicy.Digest
}

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
	TimeoutSeconds  *int
	PayloadHash     *trustpolicy.Digest
	AuditContext    *GatewayAuditContextInput
	QuotaContext    *GatewayQuotaContextInput
}

type GatewayAuditContextInput struct {
	OutboundBytes      int64
	StartedAt          string
	CompletedAt        string
	Outcome            string
	RequestHash        *trustpolicy.Digest
	ResponseHash       *trustpolicy.Digest
	LeaseID            string
	PolicyDecisionHash *trustpolicy.Digest
}

type GatewayQuotaContextInput struct {
	QuotaProfileKind    string
	Phase               string
	EnforceDuringStream bool
	StreamLimitBytes    *int64
	Meters              GatewayQuotaMetersInput
}

type GatewayQuotaMetersInput struct {
	RequestUnits     *int64
	InputTokens      *int64
	OutputTokens     *int64
	StreamedBytes    *int64
	ConcurrencyUnits *int64
	SpendMicros      *int64
	EntitlementUnits *int64
}

type BackendPostureChangeActionInput struct {
	ActionEnvelope
	TargetInstanceID             string
	TargetBackendKind            string
	SelectionMode                string
	ChangeKind                   string
	AssuranceChangeKind          string
	OptInKind                    string
	ReducedAssuranceAcknowledged bool
	Reason                       string
}

type GateOverrideActionInput struct {
	ActionEnvelope
	GateID                    string
	GateKind                  string
	GateVersion               string
	GateAttemptID             string
	OverriddenFailedResultRef string
	PolicyContextHash         string
	OverrideMode              string
	Justification             string
	ExpiresAt                 string
	TicketRef                 string
}

type SecretAccessActionInput struct {
	ActionEnvelope
	SecretRef       string
	LeaseID         string
	AccessMode      string
	LeaseTTLSeconds *int
	RenewalContext  *SecretLeaseRenewalContext
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
