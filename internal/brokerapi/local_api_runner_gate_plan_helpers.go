package brokerapi

import (
	"fmt"
	"sort"
	"strings"
)

func sortRunPlanEntries(entries []runPlannedGateEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].PlanOrderIndex != entries[j].PlanOrderIndex {
			return entries[i].PlanOrderIndex < entries[j].PlanOrderIndex
		}
		if entries[i].PlanCheckpointCode != entries[j].PlanCheckpointCode {
			return entries[i].PlanCheckpointCode < entries[j].PlanCheckpointCode
		}
		if entries[i].GateID != entries[j].GateID {
			return entries[i].GateID < entries[j].GateID
		}
		if entries[i].GateKind != entries[j].GateKind {
			return entries[i].GateKind < entries[j].GateKind
		}
		return entries[i].GateVersion < entries[j].GateVersion
	})
}

func buildRunPlanEntryIndex(entries []runPlannedGateEntry) (map[string]runPlannedGateEntry, error) {
	byGate := make(map[string]runPlannedGateEntry, len(entries))
	for _, entry := range entries {
		key := runPlanGateKey(entry.GateID, entry.GateKind, entry.GateVersion, entry.PlanCheckpointCode, entry.PlanOrderIndex)
		if existing, ok := byGate[key]; ok {
			if !sameRunPlannedGateEntry(existing, entry) {
				return nil, fmt.Errorf("ambiguous gate plan entry for gate %q at %s[%d]", entry.GateID, entry.PlanCheckpointCode, entry.PlanOrderIndex)
			}
			continue
		}
		byGate[key] = entry
	}
	return byGate, nil
}

func isTrustedRunPlanPutCandidate(trustedSource bool, createdByRole, contentType, runID, stepID string) bool {
	if !trustedSource {
		return false
	}
	if strings.TrimSpace(runID) == "" {
		return false
	}
	switch strings.TrimSpace(createdByRole) {
	case "broker", "brokerapi":
	default:
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(contentType), "application/json") {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(stepID), runPlanAuthorityStepPrefix)
}
