package secretsd

import (
	"errors"
	"io"
	"sync"
	"time"
)

const (
	stateFileName      = "state.json"
	secretsDirName     = "secrets"
	stateVersion       = 1
	defaultTTLSeconds  = 900
	hardCapTTLSeconds  = 3600
	leaseStatusActive  = "active"
	leaseStatusRevoked = "revoked"
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

	state state
}

type SecretMetadata struct {
	SecretRef      string    `json:"secret_ref"`
	SecretID       string    `json:"secret_id"`
	MaterialDigest string    `json:"material_digest"`
	ImportedAt     time.Time `json:"imported_at"`
}

type Lease struct {
	LeaseID    string     `json:"lease_id"`
	SecretRef  string     `json:"secret_ref"`
	ConsumerID string     `json:"consumer_id"`
	RoleKind   string     `json:"role_kind"`
	Scope      string     `json:"scope"`
	IssuedAt   time.Time  `json:"issued_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Status     string     `json:"status"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	Reason     string     `json:"reason,omitempty"`
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
	SecretRef  string
	ConsumerID string
	RoleKind   string
	Scope      string
	TTLSeconds int
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
	LeaseID    string
	ConsumerID string
	RoleKind   string
	Scope      string
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
	LeaseID    string     `json:"lease_id"`
	SecretRef  string     `json:"secret_ref"`
	ConsumerID string     `json:"consumer_id"`
	RoleKind   string     `json:"role_kind"`
	Scope      string     `json:"scope"`
	IssuedAt   time.Time  `json:"issued_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Status     string     `json:"status"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	Reason     string     `json:"reason,omitempty"`
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
		LeaseID:    r.LeaseID,
		SecretRef:  r.SecretRef,
		ConsumerID: r.ConsumerID,
		RoleKind:   r.RoleKind,
		Scope:      r.Scope,
		IssuedAt:   r.IssuedAt,
		ExpiresAt:  r.ExpiresAt,
		Status:     r.Status,
		RevokedAt:  r.RevokedAt,
		Reason:     r.Reason,
	}
}
