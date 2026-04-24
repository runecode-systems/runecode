package artifacts

type sessionDurableStateCompare struct {
	SessionID                        string
	WorkspaceID                      string
	CreatedAtUnixNano                int64
	CreatedByRunID                   string
	UpdatedAtUnixNano                int64
	Status                           string
	WorkPosture                      string
	WorkPostureReason                string
	LastActivityUnixNano             int64
	LastActivityKind                 string
	LastActivityPreview              string
	LastInteractionSequence          int64
	TurnCount                        int
	HasIncompleteTurn                bool
	ExecutionTriggers                []sessionExecutionTriggerDurableStateCompare
	TurnExecutions                   []sessionTurnExecutionDurableStateCompare
	TranscriptTurns                  []sessionTranscriptTurnDurableStateCompare
	IdempotencyByKey                 map[string]sessionIdempotencyRecordCompare
	ExecutionTriggerIdempotencyByKey map[string]sessionExecutionTriggerIdempotencyRecordCompare
	LinkedRunIDs                     []string
}

type sessionExecutionTriggerDurableStateCompare struct {
	TriggerID              string
	SessionID              string
	TriggerIndex           int
	TriggerSource          string
	RequestedOperation     string
	UserMessageContentText string
	CreatedAtUnixNano      int64
}

type sessionTurnExecutionDurableStateCompare struct {
	TurnID                               string
	SessionID                            string
	ExecutionIndex                       int
	OrchestrationScopeID                 string
	DependsOnScopeIDs                    []string
	TriggerID                            string
	TriggerSource                        string
	RequestedOperation                   string
	ExecutionState                       string
	WaitKind                             string
	WaitState                            string
	ApprovalProfile                      string
	AutonomyPosture                      string
	PrimaryRunID                         string
	PendingApprovalID                    string
	LinkedRunIDs                         []string
	LinkedApprovalIDs                    []string
	LinkedArtifactDigests                []string
	LinkedAuditRecordDigests             []string
	BoundValidatedProjectSubstrateDigest string
	BlockedReasonCode                    string
	TerminalOutcome                      string
	CreatedAtUnixNano                    int64
	UpdatedAtUnixNano                    int64
}

type sessionTranscriptTurnDurableStateCompare struct {
	TurnID              string
	SessionID           string
	TurnIndex           int
	StartedAtUnixNano   int64
	CompletedAtUnixNano int64
	Status              string
	Messages            []sessionTranscriptMessageDurableStateCompare
}

type sessionTranscriptMessageDurableStateCompare struct {
	MessageID         string
	TurnID            string
	SessionID         string
	MessageIndex      int
	Role              string
	CreatedAtUnixNano int64
	ContentText       string
	RelatedLinks      sessionTranscriptLinksDurableStateCompare
}

type sessionTranscriptLinksDurableStateCompare struct {
	RunIDs             []string
	ApprovalIDs        []string
	ArtifactDigests    []string
	AuditRecordDigests []string
}

type sessionIdempotencyRecordCompare struct {
	RequestHash string
	TurnID      string
	MessageID   string
	Seq         int64
}

type sessionExecutionTriggerIdempotencyRecordCompare struct {
	RequestHash string
	TriggerID   string
	TurnID      string
	Seq         int64
}
