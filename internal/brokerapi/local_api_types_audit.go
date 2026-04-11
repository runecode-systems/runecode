package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type AuditTimelineRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
}

type AuditTimelineResponse struct {
	SchemaID      string                             `json:"schema_id"`
	SchemaVersion string                             `json:"schema_version"`
	RequestID     string                             `json:"request_id"`
	Order         string                             `json:"order"`
	Views         []trustpolicy.AuditOperationalView `json:"views"`
	NextCursor    string                             `json:"next_cursor,omitempty"`
}

type AuditVerificationGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	ViewLimit     int    `json:"view_limit,omitempty"`
}

type AuditVerificationGetResponse struct {
	SchemaID      string                                         `json:"schema_id"`
	SchemaVersion string                                         `json:"schema_version"`
	RequestID     string                                         `json:"request_id"`
	Summary       trustpolicy.DerivedRunAuditVerificationSummary `json:"summary"`
	Report        trustpolicy.AuditVerificationReportPayload     `json:"report"`
	Views         []trustpolicy.AuditOperationalView             `json:"views"`
}
