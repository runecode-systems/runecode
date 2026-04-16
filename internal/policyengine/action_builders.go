package policyengine

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func NewWorkspaceWriteAction(input WorkspaceWriteActionInput) ActionRequest {
	payload := map[string]any{
		"schema_id":      actionPayloadWorkspaceSchemaID,
		"schema_version": "0.1.0",
		"target_path":    input.TargetPath,
		"write_mode":     input.WriteMode,
	}
	if input.SourcePath != "" {
		payload["source_path"] = input.SourcePath
	}
	if input.ContentSHA256 != nil {
		payload["content_sha256"] = *input.ContentSHA256
	}
	if input.Bytes != nil {
		payload["bytes"] = *input.Bytes
	}
	return buildActionRequest(ActionKindWorkspaceWrite, actionPayloadWorkspaceSchemaID, payload, input.ActionEnvelope)
}

func NewExecutorRunAction(input ExecutorRunActionInput) ActionRequest {
	argv := make([]any, 0, len(input.Argv))
	for _, v := range input.Argv {
		argv = append(argv, v)
	}
	payload := map[string]any{
		"schema_id":      actionPayloadExecutorSchemaID,
		"schema_version": "0.1.0",
		"executor_class": input.ExecutorClass,
		"executor_id":    input.ExecutorID,
		"argv":           argv,
	}
	if len(input.Environment) > 0 {
		environment := map[string]any{}
		for key, value := range input.Environment {
			environment[key] = value
		}
		payload["environment"] = environment
	}
	if input.WorkingDirectory != "" {
		payload["working_directory"] = input.WorkingDirectory
	}
	if input.NetworkAccess != "" {
		payload["network_access"] = input.NetworkAccess
	}
	if input.TimeoutSeconds != nil {
		payload["timeout_seconds"] = *input.TimeoutSeconds
	}
	return buildActionRequest(ActionKindExecutorRun, actionPayloadExecutorSchemaID, payload, input.ActionEnvelope)
}

func NewArtifactReadAction(input ArtifactReadActionInput) ActionRequest {
	payload := map[string]any{
		"schema_id":      actionPayloadArtifactSchemaID,
		"schema_version": "0.1.0",
		"artifact_hash":  input.ArtifactHash,
		"read_mode":      input.ReadMode,
	}
	if input.ExpectedDataClass != "" {
		payload["expected_data_class"] = input.ExpectedDataClass
	}
	if input.Purpose != "" {
		payload["purpose"] = input.Purpose
	}
	if input.MaxBytes != nil {
		payload["max_bytes"] = *input.MaxBytes
	}
	return buildActionRequest(ActionKindArtifactRead, actionPayloadArtifactSchemaID, payload, input.ActionEnvelope)
}

func NewPromotionAction(input PromotionActionInput) ActionRequest {
	payload := map[string]any{
		"schema_id":            actionPayloadPromotionSchemaID,
		"schema_version":       "0.1.0",
		"promotion_kind":       input.PromotionKind,
		"source_artifact_hash": input.SourceArtifactHash,
		"target_data_class":    input.TargetDataClass,
	}
	if input.ByteStart != nil {
		payload["byte_start"] = *input.ByteStart
	}
	if input.ByteEnd != nil {
		payload["byte_end"] = *input.ByteEnd
	}
	if input.Justification != "" {
		payload["justification"] = input.Justification
	}
	if input.RepoPath != "" {
		payload["repo_path"] = input.RepoPath
	}
	if input.Commit != "" {
		payload["commit"] = input.Commit
	}
	if input.ExtractorToolVersion != "" {
		payload["extractor_tool_version"] = input.ExtractorToolVersion
	}
	if input.Approver != "" {
		payload["approver"] = input.Approver
	}
	return buildActionRequest(ActionKindPromotion, actionPayloadPromotionSchemaID, payload, input.ActionEnvelope)
}

func NewBackendPostureChangeAction(input BackendPostureChangeActionInput) ActionRequest {
	payload := map[string]any{
		"schema_id":                      actionPayloadBackendSchemaID,
		"schema_version":                 "0.1.0",
		"target_instance_id":             input.TargetInstanceID,
		"target_backend_kind":            input.TargetBackendKind,
		"selection_mode":                 input.SelectionMode,
		"change_kind":                    input.ChangeKind,
		"assurance_change_kind":          input.AssuranceChangeKind,
		"opt_in_kind":                    input.OptInKind,
		"reduced_assurance_acknowledged": input.ReducedAssuranceAcknowledged,
	}
	if input.RunID != "" {
		payload["run_id"] = input.RunID
	}
	if input.Reason != "" {
		payload["reason"] = input.Reason
	}
	return buildActionRequest(ActionKindBackendPosture, actionPayloadBackendSchemaID, payload, input.ActionEnvelope)
}

func NewGateOverrideAction(input GateOverrideActionInput) ActionRequest {
	payload := map[string]any{
		"schema_id":                    actionPayloadGateSchemaID,
		"schema_version":               "0.1.0",
		"gate_id":                      input.GateID,
		"gate_kind":                    input.GateKind,
		"gate_version":                 input.GateVersion,
		"gate_attempt_id":              input.GateAttemptID,
		"overridden_failed_result_ref": input.OverriddenFailedResultRef,
		"policy_context_hash":          input.PolicyContextHash,
		"override_mode":                input.OverrideMode,
		"justification":                input.Justification,
	}
	if input.ExpiresAt != "" {
		payload["expires_at"] = input.ExpiresAt
	}
	if input.TicketRef != "" {
		payload["ticket_ref"] = input.TicketRef
	}
	return buildActionRequest(ActionKindGateOverride, actionPayloadGateSchemaID, payload, input.ActionEnvelope)
}

func NewSecretAccessAction(input SecretAccessActionInput) ActionRequest {
	payload := map[string]any{
		"schema_id":      actionPayloadSecretAccessID,
		"schema_version": "0.1.0",
		"access_mode":    input.AccessMode,
	}
	if input.SecretRef != "" {
		payload["secret_ref"] = input.SecretRef
	}
	if input.LeaseID != "" {
		payload["lease_id"] = input.LeaseID
	}
	if input.LeaseTTLSeconds != nil {
		payload["lease_ttl_seconds"] = *input.LeaseTTLSeconds
	}
	if input.RenewalContext != nil {
		renewalContext := map[string]any{
			"consumer_principal_ref": input.RenewalContext.ConsumerPrincipalRef,
			"target_ref":             input.RenewalContext.TargetRef,
			"policy_context_hash":    input.RenewalContext.PolicyContextHash,
		}
		payload["renewal_context"] = renewalContext
	}
	if input.Justification != "" {
		payload["justification"] = input.Justification
	}
	if input.RequiresEgress != nil {
		payload["requires_egress"] = *input.RequiresEgress
	}
	if input.TargetSystem != "" {
		payload["target_system"] = input.TargetSystem
	}
	return buildActionRequest(ActionKindSecretAccess, actionPayloadSecretAccessID, payload, input.ActionEnvelope)
}

func NewStageSummarySignOffAction(input StageSummarySignOffActionInput) (ActionRequest, error) {
	planID := strings.TrimSpace(input.PlanID)
	if planID == "" {
		return ActionRequest{}, fmt.Errorf("stage summary sign-off action requires plan_id")
	}
	payload := map[string]any{
		"schema_id":          actionPayloadStageSchemaID,
		"schema_version":     "0.1.0",
		"run_id":             input.RunID,
		"stage_id":           input.StageID,
		"stage_summary":      canonicalStageSummaryPayload(input, planID),
		"stage_summary_hash": input.StageSummaryHash,
	}
	if input.ApprovalProfile != "" {
		payload["approval_profile"] = input.ApprovalProfile
	}
	if input.SummaryRevision != nil {
		payload["summary_revision"] = *input.SummaryRevision
	}
	return buildActionRequest(ActionKindStageSummarySign, actionPayloadStageSchemaID, payload, input.ActionEnvelope), nil
}

func canonicalStageSummaryPayload(input StageSummarySignOffActionInput, planID string) map[string]any {
	stageSummary := map[string]any{}
	for key, value := range input.StageSummary {
		stageSummary[key] = value
	}
	stageSummary["schema_id"] = "runecode.protocol.v0.StageSummary"
	stageSummary["schema_version"] = "0.1.0"
	stageSummary["run_id"] = input.RunID
	stageSummary["plan_id"] = planID
	stageSummary["stage_id"] = input.StageID
	stageSummary["summary_revision"] = int64(1)
	if input.SummaryRevision != nil {
		stageSummary["summary_revision"] = *input.SummaryRevision
	}
	stageSummary["manifest_hash"] = input.ManifestHash
	ensureStageSummaryPayloadDefaults(stageSummary)
	return stageSummary
}

func ensureStageSummaryPayloadDefaults(stageSummary map[string]any) {
	if _, ok := stageSummary["stage_capability_context"]; !ok {
		stageSummary["stage_capability_context"] = map[string]any{}
	}
	if _, ok := stageSummary["requested_high_risk_capability_categories"]; !ok {
		stageSummary["requested_high_risk_capability_categories"] = []any{}
	}
	if _, ok := stageSummary["requested_scope_change_types"]; !ok {
		stageSummary["requested_scope_change_types"] = []any{}
	}
	if _, ok := stageSummary["relevant_artifact_hashes"]; !ok {
		stageSummary["relevant_artifact_hashes"] = []any{}
	}
}

func CanonicalActionRequestHash(action ActionRequest) (string, error) {
	return canonicalHashValue(action)
}

func buildActionRequest(kind string, payloadSchemaID string, payload map[string]any, envelope ActionEnvelope) ActionRequest {
	return ActionRequest{
		SchemaID:               actionRequestSchemaID,
		SchemaVersion:          actionRequestSchemaVersion,
		ActionKind:             kind,
		CapabilityID:           envelope.CapabilityID,
		AllowlistRefs:          append([]string{}, envelope.AllowlistRefs...),
		RelevantArtifactHashes: append([]trustpolicy.Digest{}, envelope.RelevantArtifactHashes...),
		ActionPayloadSchemaID:  payloadSchemaID,
		ActionPayload:          payload,
		ActorKind:              envelope.Actor.ActorKind,
		RoleFamily:             envelope.Actor.RoleFamily,
		RoleKind:               envelope.Actor.RoleKind,
	}
}
