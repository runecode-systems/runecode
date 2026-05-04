package secretsd

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	stateFileName                   = "state.json"
	secretsDirName                  = "secrets"
	stateVersion                    = 1
	defaultTTLSeconds               = 900
	hardCapTTLSeconds               = 3600
	leaseStatusActive               = "active"
	leaseStatusRevoked              = "revoked"
	deliveryKindGitGateway          = "git_gateway"
	deliveryKindEnvironmentVariable = "environment_variable"
	deliveryKindCLIArgument         = "cli_argument"
)

var (
	ErrAccessDenied        = errors.New("access denied")
	ErrNotFound            = errors.New("not found")
	ErrStateRecoveryFailed = errors.New("state recovery failed")
)

type Service struct {
	mu   sync.Mutex
	root string
	now  func() time.Time
	rand io.Reader

	auditAnchorSignerProfile auditAnchorSignerProfile

	leaseAuditHook atomic.Pointer[leaseAuditHookHolder]

	state state
}

type LeaseAuditEvent struct {
	Action string `json:"action"`
	Lease  Lease  `json:"lease"`
}

type leaseAuditHookHolder struct {
	hook func(LeaseAuditEvent)
}

type auditAnchorSignerProfile string

const (
	auditAnchorSignerProfileDefault   auditAnchorSignerProfile = "default"
	auditAnchorSignerProfileMetaAudit auditAnchorSignerProfile = "meta_audit"
)

type SecretMetadata struct {
	SecretRef      string    `json:"secret_ref"`
	SecretID       string    `json:"secret_id"`
	MaterialDigest string    `json:"material_digest"`
	ImportedAt     time.Time `json:"imported_at"`
}

type Lease struct {
	LeaseID      string           `json:"lease_id"`
	SecretRef    string           `json:"secret_ref"`
	ConsumerID   string           `json:"consumer_id"`
	RoleKind     string           `json:"role_kind"`
	Scope        string           `json:"scope"`
	DeliveryKind string           `json:"delivery_kind,omitempty"`
	GitBinding   *GitLeaseBinding `json:"git_binding,omitempty"`
	IssuedAt     time.Time        `json:"issued_at"`
	ExpiresAt    time.Time        `json:"expires_at"`
	Status       string           `json:"status"`
	RevokedAt    *time.Time       `json:"revoked_at,omitempty"`
	Reason       string           `json:"reason,omitempty"`
}

type GitLeaseBinding struct {
	RepositoryIdentity string   `json:"repository_identity"`
	AllowedOperations  []string `json:"allowed_operations"`
	ActionRequestHash  string   `json:"action_request_hash"`
	PolicyContextHash  string   `json:"policy_context_hash"`
}

type GitLeaseUseContext struct {
	RepositoryIdentity string `json:"repository_identity"`
	Operation          string `json:"operation"`
	ActionRequestHash  string `json:"action_request_hash"`
	PolicyContextHash  string `json:"policy_context_hash"`
}

type RuntimeSnapshot struct {
	LeaseIssueCount   int
	LeaseRenewCount   int
	LeaseRevokeCount  int
	LeaseDenyCount    int
	ActiveLeaseCount  int
	ExpiredLeaseCount int
	RevokedLeaseCount int
	SecretRecordCount int
	LastRecoveredAt   time.Time
	LastUpdatedAt     time.Time
}

type IssueLeaseRequest struct {
	SecretRef    string
	ConsumerID   string
	RoleKind     string
	Scope        string
	DeliveryKind string
	GitBinding   *GitLeaseBinding
	TTLSeconds   int
}

type RenewLeaseRequest struct {
	LeaseID    string
	ConsumerID string
	RoleKind   string
	Scope      string
	TTLSeconds int
}

type RevokeLeaseRequest struct {
	LeaseID    string
	ConsumerID string
	RoleKind   string
	Scope      string
	Reason     string
}

type RetrieveRequest struct {
	LeaseID       string
	ConsumerID    string
	RoleKind      string
	Scope         string
	DeliveryKind  string
	GitUseContext *GitLeaseUseContext
}

type RevokeGitLeasesRequest struct {
	RepositoryIdentity string
	ActionRequestHash  string
	PolicyContextHash  string
	Reason             string
}

type state struct {
	Version int                     `json:"version"`
	Secrets map[string]secretRecord `json:"secrets"`
	Leases  map[string]leaseRecord  `json:"leases"`
	Metrics metrics                 `json:"metrics"`
}

type secretRecord struct {
	SecretID       string    `json:"secret_id"`
	MaterialFile   string    `json:"material_file"`
	MaterialDigest string    `json:"material_digest"`
	ImportedAt     time.Time `json:"imported_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type leaseRecord struct {
	LeaseID      string           `json:"lease_id"`
	SecretRef    string           `json:"secret_ref"`
	ConsumerID   string           `json:"consumer_id"`
	RoleKind     string           `json:"role_kind"`
	Scope        string           `json:"scope"`
	DeliveryKind string           `json:"delivery_kind,omitempty"`
	GitBinding   *GitLeaseBinding `json:"git_binding,omitempty"`
	IssuedAt     time.Time        `json:"issued_at"`
	ExpiresAt    time.Time        `json:"expires_at"`
	Status       string           `json:"status"`
	RevokedAt    *time.Time       `json:"revoked_at,omitempty"`
	Reason       string           `json:"reason,omitempty"`
}

type metrics struct {
	LeaseIssueCount  int `json:"lease_issue_count"`
	LeaseRenewCount  int `json:"lease_renew_count"`
	LeaseRevokeCount int `json:"lease_revoke_count"`
	LeaseDenyCount   int `json:"lease_deny_count"`
}

func (r leaseRecord) bindingMatches(consumerID, roleKind, scope string) bool {
	return r.ConsumerID == consumerID && r.RoleKind == roleKind && r.Scope == scope
}

func (r leaseRecord) public() Lease {
	return Lease{
		LeaseID:      r.LeaseID,
		SecretRef:    r.SecretRef,
		ConsumerID:   r.ConsumerID,
		RoleKind:     r.RoleKind,
		Scope:        r.Scope,
		DeliveryKind: r.DeliveryKind,
		GitBinding:   cloneGitLeaseBinding(r.GitBinding),
		IssuedAt:     r.IssuedAt,
		ExpiresAt:    r.ExpiresAt,
		Status:       r.Status,
		RevokedAt:    r.RevokedAt,
		Reason:       r.Reason,
	}
}
