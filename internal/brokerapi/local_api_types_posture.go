package brokerapi

type BackendPostureGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type BackendPostureAvailability struct {
	SchemaID            string `json:"schema_id"`
	SchemaVersion       string `json:"schema_version"`
	BackendKind         string `json:"backend_kind"`
	Available           bool   `json:"available"`
	AvailabilitySummary string `json:"availability_summary,omitempty"`
}

type BackendPostureState struct {
	SchemaID                 string                       `json:"schema_id"`
	SchemaVersion            string                       `json:"schema_version"`
	InstanceID               string                       `json:"instance_id"`
	BackendKind              string                       `json:"backend_kind"`
	PreferredBackendKind     string                       `json:"preferred_backend_kind"`
	ReducedAssuranceActive   bool                         `json:"reduced_assurance_active"`
	PendingApproval          bool                         `json:"pending_approval"`
	PendingApprovalID        string                       `json:"pending_approval_id,omitempty"`
	LatestPolicyDecisionHash string                       `json:"latest_policy_decision_hash,omitempty"`
	LatestAppliedEvidenceRef string                       `json:"latest_applied_evidence_ref,omitempty"`
	Availability             []BackendPostureAvailability `json:"availability"`
}

type BackendPostureGetResponse struct {
	SchemaID      string              `json:"schema_id"`
	SchemaVersion string              `json:"schema_version"`
	RequestID     string              `json:"request_id"`
	Posture       BackendPostureState `json:"posture"`
}

type BackendPostureChangeRequest struct {
	SchemaID                     string `json:"schema_id"`
	SchemaVersion                string `json:"schema_version"`
	RequestID                    string `json:"request_id"`
	TargetInstanceID             string `json:"target_instance_id"`
	TargetBackendKind            string `json:"target_backend_kind"`
	SelectionMode                string `json:"selection_mode"`
	ChangeKind                   string `json:"change_kind"`
	AssuranceChangeKind          string `json:"assurance_change_kind"`
	OptInKind                    string `json:"opt_in_kind"`
	ReducedAssuranceAcknowledged bool   `json:"reduced_assurance_acknowledged"`
	Reason                       string `json:"reason,omitempty"`
}

type BackendPostureChangeOutcome struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	Outcome            string `json:"outcome"`
	OutcomeReasonCode  string `json:"outcome_reason_code"`
	ApprovalID         string `json:"approval_id,omitempty"`
	PolicyDecisionHash string `json:"policy_decision_hash,omitempty"`
	ActionRequestHash  string `json:"action_request_hash,omitempty"`
}

type BackendPostureChangeResponse struct {
	SchemaID      string                      `json:"schema_id"`
	SchemaVersion string                      `json:"schema_version"`
	RequestID     string                      `json:"request_id"`
	Outcome       BackendPostureChangeOutcome `json:"outcome"`
	Posture       BackendPostureState         `json:"posture"`
}
