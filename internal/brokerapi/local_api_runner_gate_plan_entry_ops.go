package brokerapi

import (
	"fmt"
	"strings"
)

func (p compiledRunGatePlan) hasEntries() bool {
	return len(p.byGate) > 0
}

func (p compiledRunGatePlan) entryFor(gateID, gateKind, gateVersion, checkpointCode string, orderIndex int) (runPlannedGateEntry, bool) {
	entry, ok := p.byGate[runPlanGateKey(gateID, gateKind, gateVersion, checkpointCode, orderIndex)]
	return entry, ok
}

func runPlanGateKey(gateID, gateKind, gateVersion, checkpointCode string, orderIndex int) string {
	return strings.TrimSpace(gateID) + "|" + strings.TrimSpace(gateKind) + "|" + strings.TrimSpace(gateVersion) + "|" + strings.TrimSpace(checkpointCode) + "|" + fmt.Sprintf("%d", orderIndex)
}

func sameRunPlannedGateEntry(a, b runPlannedGateEntry) bool {
	return a.GateID == b.GateID && a.GateKind == b.GateKind &&
		a.GateVersion == b.GateVersion && a.StageID == b.StageID &&
		a.StepID == b.StepID && a.RoleInstanceID == b.RoleInstanceID &&
		a.PlanCheckpointCode == b.PlanCheckpointCode &&
		a.PlanOrderIndex == b.PlanOrderIndex &&
		a.MaxAttempts == b.MaxAttempts &&
		sameStringSlices(a.ExpectedInputDigests, b.ExpectedInputDigests) &&
		sameDependencyHandoffs(a.DependencyCacheHandoffs, b.DependencyCacheHandoffs)
}

func sameStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameDependencyHandoffs(a, b []runPlannedDependencyCacheHandoff) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].RequestDigest != b[i].RequestDigest ||
			a[i].ConsumerRole != b[i].ConsumerRole ||
			a[i].Required != b[i].Required {
			return false
		}
	}
	return true
}

func validatePlannedInputDigestHooks(expected []string, reported []string) error {
	if len(expected) == 0 {
		return nil
	}
	reportedSet := map[string]struct{}{}
	for _, digest := range reported {
		reportedSet[strings.TrimSpace(digest)] = struct{}{}
	}
	for _, digest := range expected {
		if _, ok := reportedSet[digest]; !ok {
			return fmt.Errorf("normalized_input_digests missing planned digest %q", digest)
		}
	}
	return nil
}
