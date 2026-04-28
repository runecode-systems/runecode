package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
)

func runPlanEntryRecords(entries []runplan.Entry) []artifacts.RunPlanGateEntryRecord {
	if len(entries) == 0 {
		return nil
	}
	out := make([]artifacts.RunPlanGateEntryRecord, 0, len(entries))
	for _, entry := range entries {
		out = append(out, runPlanEntryRecord(entry))
	}
	return out
}

func runPlanEntryRecord(entry runplan.Entry) artifacts.RunPlanGateEntryRecord {
	return artifacts.RunPlanGateEntryRecord{
		EntryID:                 strings.TrimSpace(entry.EntryID),
		EntryKind:               strings.TrimSpace(entry.EntryKind),
		PlanCheckpointCode:      strings.TrimSpace(entry.CheckpointCode),
		PlanOrderIndex:          entry.OrderIndex,
		GateID:                  strings.TrimSpace(entry.Gate.GateID),
		GateKind:                strings.TrimSpace(entry.Gate.GateKind),
		GateVersion:             strings.TrimSpace(entry.Gate.GateVersion),
		StageID:                 strings.TrimSpace(entry.StageID),
		StepID:                  strings.TrimSpace(entry.StepID),
		RoleInstanceID:          strings.TrimSpace(entry.RoleInstanceID),
		MaxAttempts:             maxAttemptsFromEntry(entry),
		ExpectedInputDigests:    normalizedInputDigestsFromEntry(entry),
		DependencyCacheHandoffs: runPlanDependencyCacheHandoffs(entry.DependencyCacheHandoffs),
	}
}

func runPlanDependencyCacheHandoffs(handoffs []runplan.DependencyCacheHandoff) []artifacts.RunPlanDependencyCacheHandoffRecord {
	if len(handoffs) == 0 {
		return nil
	}
	out := make([]artifacts.RunPlanDependencyCacheHandoffRecord, 0, len(handoffs))
	for _, handoff := range handoffs {
		requestDigest, err := handoff.RequestDigest.Identity()
		if err != nil {
			continue
		}
		out = append(out, artifacts.RunPlanDependencyCacheHandoffRecord{
			RequestDigest: strings.TrimSpace(requestDigest),
			ConsumerRole:  strings.TrimSpace(handoff.ConsumerRole),
			Required:      handoff.Required,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizedInputDigestsFromEntry(entry runplan.Entry) []string {
	if len(entry.Gate.NormalizedInputs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(entry.Gate.NormalizedInputs))
	for _, input := range entry.Gate.NormalizedInputs {
		digest, _ := input["input_digest"].(string)
		digest = strings.TrimSpace(digest)
		if digest == "" {
			continue
		}
		if _, exists := seen[digest]; exists {
			continue
		}
		seen[digest] = struct{}{}
		out = append(out, digest)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}
