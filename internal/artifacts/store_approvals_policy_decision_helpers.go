package artifacts

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const approvalMaxTTLSeconds = int64(24 * 60 * 60)

type approvalDecisionDerivedSummary struct {
	trigger        string
	changes        string
	assurance      string
	presence       string
	workspaceID    string
	instanceID     string
	runID          string
	stageID        string
	stepID         string
	roleInstanceID string
	actionKind     string
	relevant       []string
	sourceDigest   string
}

func approvalTiming(record PolicyDecisionRecord, required map[string]any, nowFn func() time.Time) (time.Time, time.Time) {
	requestedAt := record.RecordedAt.UTC()
	if requestedAt.IsZero() {
		requestedAt = nowFn().UTC()
	}
	expiresAt := requestedAt.Add(30 * time.Minute)
	if ttlSeconds, ok := approvalTTLSeconds(required); ok && ttlSeconds > 0 {
		if ttlSeconds > approvalMaxTTLSeconds {
			ttlSeconds = approvalMaxTTLSeconds
		}
		expiresAt = requestedAt.Add(time.Duration(ttlSeconds) * time.Second)
	}
	return requestedAt, expiresAt
}

func approvalDecisionSummary(record PolicyDecisionRecord, required map[string]any) approvalDecisionDerivedSummary {
	scope := mapField(required, "scope")
	relevant := uniqueSortedStrings(record.RelevantArtifactHashes)
	return approvalDecisionDerivedSummary{
		trigger:        stringField(required, "approval_trigger_code", "approval_required"),
		changes:        stringField(required, "changes_if_approved", approvalChangesIfApprovedDefault),
		assurance:      stringField(required, "approval_assurance_level", approvalDefaultAssuranceLevel),
		presence:       stringField(required, "presence_mode", approvalDefaultPresenceMode),
		workspaceID:    stringField(scope, "workspace_id", ""),
		instanceID:     stringField(scope, "instance_id", ""),
		runID:          stringField(scope, "run_id", strings.TrimSpace(record.RunID)),
		stageID:        stringField(scope, "stage_id", ""),
		stepID:         stringField(scope, "step_id", ""),
		roleInstanceID: stringField(scope, "role_instance_id", ""),
		actionKind:     stringField(scope, "action_kind", "unknown"),
		relevant:       relevant,
		sourceDigest:   approvalSourceDigest(relevant),
	}
}

func approvalSourceDigest(relevant []string) string {
	if len(relevant) == 1 {
		return relevant[0]
	}
	return ""
}

func approvalTTLSeconds(required map[string]any) (int64, bool) {
	raw, ok := required["approval_ttl_seconds"]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func stringField(object map[string]any, key string, fallback string) string {
	if object == nil {
		return fallback
	}
	value, ok := object[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func mapField(object map[string]any, key string) map[string]any {
	if object == nil {
		return map[string]any{}
	}
	v, ok := object[key].(map[string]any)
	if !ok || v == nil {
		return map[string]any{}
	}
	return v
}

func approvalRequestPayloadFromDecision(record PolicyDecisionRecord, requestedAt, expiresAt time.Time, trigger, changes, assurance, presence, runID, stepID string) (map[string]any, error) {
	manifestHash, err := digestObjectForIdentity(record.ManifestHash)
	if err != nil {
		return nil, fmt.Errorf("manifest_hash: %w", err)
	}
	actionRequestHash, err := digestObjectForIdentity(record.ActionRequestHash)
	if err != nil {
		return nil, fmt.Errorf("action_request_hash: %w", err)
	}
	relevantArtifactHashes, err := digestObjectSliceForIdentities(record.RelevantArtifactHashes)
	if err != nil {
		return nil, fmt.Errorf("relevant_artifact_hashes: %w", err)
	}
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                approvalRequesterIdentity(runID),
		"approval_trigger_code":    trigger,
		"manifest_hash":            manifestHash,
		"action_request_hash":      actionRequestHash,
		"relevant_artifact_hashes": relevantArtifactHashes,
		"details_schema_id":        record.RequiredApprovalSchemaID,
		"details":                  approvalDetailsFromRequired(record.RequiredApproval, runID, stepID),
		"approval_assurance_level": assurance,
		"presence_mode":            presence,
		"requested_at":             requestedAt.UTC().Format(time.RFC3339),
		"expires_at":               expiresAt.UTC().Format(time.RFC3339),
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      changes,
		"signatures":               pendingApprovalSignatures(),
	}, nil
}

func approvalRequesterIdentity(runID string) map[string]any {
	identity := map[string]any{
		"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
		"schema_version": "0.2.0",
		"actor_kind":     "daemon",
		"principal_id":   "broker",
		"instance_id":    "broker-local",
	}
	if strings.TrimSpace(runID) != "" {
		identity["run_id"] = runID
	}
	return identity
}

func approvalDetailsFromRequired(required map[string]any, runID, stepID string) map[string]any {
	if details, ok := required["details"].(map[string]any); ok && len(details) > 0 {
		return addApprovalContext(copyAnyMap(details), runID, stepID)
	}
	if len(required) == 0 {
		return map[string]any{"run_id": runID, "step_id": stepID}
	}
	return addApprovalContext(copyAnyMap(required), runID, stepID)
}

func copyAnyMap(src map[string]any) map[string]any {
	copyMap := map[string]any{}
	for k, v := range src {
		copyMap[k] = v
	}
	return copyMap
}

func addApprovalContext(details map[string]any, runID, stepID string) map[string]any {
	if runID != "" {
		if _, ok := details["run_id"]; !ok {
			details["run_id"] = runID
		}
	}
	if stepID != "" {
		if _, ok := details["step_id"]; !ok {
			details["step_id"] = stepID
		}
	}
	return details
}

func pendingApprovalSignatures() []map[string]any {
	return []map[string]any{{
		"alg":          "ed25519",
		"key_id":       trustpolicy.KeyIDProfile,
		"key_id_value": strings.Repeat("0", 64),
		"signature":    "cGVuZGluZw==",
	}}
}

func digestObjectForIdentity(identity string) (map[string]any, error) {
	if len(identity) == len("sha256:")+64 && strings.HasPrefix(identity, "sha256:") {
		return map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(identity, "sha256:")}, nil
	}
	return nil, fmt.Errorf("invalid digest identity %q", identity)
}

func digestObjectSliceForIdentities(identities []string) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(identities))
	for _, identity := range uniqueSortedStrings(identities) {
		d, err := digestObjectForIdentity(identity)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if out == nil {
		return []map[string]any{}, nil
	}
	return out, nil
}
