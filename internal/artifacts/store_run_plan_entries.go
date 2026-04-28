package artifacts

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/runplan"
)

func runPlanGateEntriesFromCompiledEntries(entries []runplan.Entry) []RunPlanGateEntryRecord {
	if len(entries) == 0 {
		return nil
	}
	out := make([]RunPlanGateEntryRecord, 0, len(entries))
	for _, entry := range entries {
		out = append(out, runPlanGateEntryFromCompiledEntry(entry))
	}
	return out
}

func runPlanGateEntryFromCompiledEntry(entry runplan.Entry) RunPlanGateEntryRecord {
	return RunPlanGateEntryRecord{
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
		MaxAttempts:             maxAttemptsFromRetrySemantics(entry.Gate.RetrySemantics),
		ExpectedInputDigests:    normalizedInputDigestsFromGateInputs(entry.Gate.NormalizedInputs),
		DependencyCacheHandoffs: runPlanDependencyCacheHandoffsFromCompiledEntry(entry.DependencyCacheHandoffs),
	}
}

func runPlanDependencyCacheHandoffsFromCompiledEntry(handoffs []runplan.DependencyCacheHandoff) []RunPlanDependencyCacheHandoffRecord {
	if len(handoffs) == 0 {
		return nil
	}
	out := make([]RunPlanDependencyCacheHandoffRecord, 0, len(handoffs))
	for _, handoff := range handoffs {
		requestDigest, err := handoff.RequestDigest.Identity()
		if err != nil {
			continue
		}
		out = append(out, RunPlanDependencyCacheHandoffRecord{
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

func normalizedInputDigestsFromGateInputs(inputs []map[string]any) []string {
	if len(inputs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(inputs))
	for _, input := range inputs {
		digest, _ := input["input_digest"].(string)
		digest = strings.TrimSpace(digest)
		if digest == "" {
			continue
		}
		if _, ok := seen[digest]; ok {
			continue
		}
		seen[digest] = struct{}{}
		out = append(out, digest)
	}
	sort.Strings(out)
	return out
}

func maxAttemptsFromRetrySemantics(retrySemantics map[string]any) int {
	raw, ok := retrySemantics["max_attempts"].(float64)
	if !ok || raw < 1 {
		return 1
	}
	return int(raw)
}

func cloneRunPlanGateEntries(entries []RunPlanGateEntryRecord) []RunPlanGateEntryRecord {
	if len(entries) == 0 {
		return nil
	}
	out := make([]RunPlanGateEntryRecord, len(entries))
	for i := range entries {
		out[i] = entries[i]
		out[i].ExpectedInputDigests = append([]string{}, entries[i].ExpectedInputDigests...)
		if len(entries[i].DependencyCacheHandoffs) > 0 {
			out[i].DependencyCacheHandoffs = append([]RunPlanDependencyCacheHandoffRecord{}, entries[i].DependencyCacheHandoffs...)
		}
	}
	return out
}

func sameRunPlanGateEntryRecordSlices(left, right []RunPlanGateEntryRecord) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !sameRunPlanGateEntryRecord(left[i], right[i]) {
			return false
		}
	}
	return true
}

func sameRunPlanGateEntryRecord(left, right RunPlanGateEntryRecord) bool {
	if left.EntryID != right.EntryID || left.EntryKind != right.EntryKind || left.PlanCheckpointCode != right.PlanCheckpointCode || left.PlanOrderIndex != right.PlanOrderIndex || left.GateID != right.GateID || left.GateKind != right.GateKind || left.GateVersion != right.GateVersion || left.StageID != right.StageID || left.StepID != right.StepID || left.RoleInstanceID != right.RoleInstanceID || left.MaxAttempts != right.MaxAttempts {
		return false
	}
	if !sameStringSlices(left.ExpectedInputDigests, right.ExpectedInputDigests) {
		return false
	}
	if len(left.DependencyCacheHandoffs) != len(right.DependencyCacheHandoffs) {
		return false
	}
	for i := range left.DependencyCacheHandoffs {
		if left.DependencyCacheHandoffs[i] != right.DependencyCacheHandoffs[i] {
			return false
		}
	}
	return true
}

func sameStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
