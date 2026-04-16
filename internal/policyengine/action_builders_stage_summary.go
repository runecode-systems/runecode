package policyengine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func canonicalStageSummaryPayloadAndHash(input StageSummarySignOffActionInput, planID string) (map[string]any, string, error) {
	stageSummary, err := canonicalStageSummaryPayload(input, planID)
	if err != nil {
		return nil, "", err
	}
	summaryHashIdentity, err := canonicalHashValue(stageSummary)
	if err != nil {
		return nil, "", fmt.Errorf("stage summary sign-off action canonical stage_summary hash: %w", err)
	}
	if err := validateProvidedStageSummaryHash(input.StageSummaryHash, summaryHashIdentity); err != nil {
		return nil, "", err
	}
	return stageSummary, summaryHashIdentity, nil
}

func canonicalStageSummaryPayload(input StageSummarySignOffActionInput, planID string) (map[string]any, error) {
	stageSummary, err := cloneJSONObject(input.StageSummary)
	if err != nil {
		return nil, fmt.Errorf("stage summary sign-off action invalid stage_summary payload: %w", err)
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
	return stageSummary, nil
}

func validateProvidedStageSummaryHash(provided trustpolicy.Digest, canonical string) error {
	if strings.TrimSpace(provided.HashAlg) == "" && strings.TrimSpace(provided.Hash) == "" {
		return nil
	}
	providedIdentity, err := provided.Identity()
	if err != nil {
		return fmt.Errorf("stage summary sign-off action invalid stage_summary_hash: %w", err)
	}
	if providedIdentity != canonical {
		return fmt.Errorf("stage summary sign-off action stage_summary_hash must match canonical stage_summary digest")
	}
	return nil
}

func cloneJSONObject(input map[string]any) (map[string]any, error) {
	if len(input) == 0 {
		return map[string]any{}, nil
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	cloned := map[string]any{}
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
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
