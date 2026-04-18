package brokerapi

import "github.com/runecode-ai/runecode/internal/secretsd"

type GitRemoteMutationIssueExecuteLeaseRequest struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	RequestID          string `json:"request_id"`
	PreparedMutationID string `json:"prepared_mutation_id"`
	TTLSeconds         int    `json:"ttl_seconds,omitempty"`
}

type GitRemoteMutationIssueExecuteLeaseResponse struct {
	SchemaID            string         `json:"schema_id"`
	SchemaVersion       string         `json:"schema_version"`
	RequestID           string         `json:"request_id"`
	PreparedMutationID  string         `json:"prepared_mutation_id"`
	Lease               secretsd.Lease `json:"lease"`
	ProviderAuthLeaseID string         `json:"provider_auth_lease_id"`
}
