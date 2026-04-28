package artifacts

import (
	"fmt"
	"sort"
	"strings"
)

func sameRunPlanRefsByRun(left, right map[string][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for runID, leftRefs := range left {
		rightRefs, ok := right[runID]
		if !ok || !sameStringSlices(leftRefs, rightRefs) {
			return false
		}
	}
	return true
}

func selectActiveRunPlanAuthorityRecord(authorities []RunPlanAuthorityRecord) (RunPlanAuthorityRecord, bool, error) {
	if len(authorities) == 0 {
		return RunPlanAuthorityRecord{}, false, nil
	}
	byPlanID, err := collectRunPlanAuthoritiesByPlanID(authorities)
	if err != nil {
		return RunPlanAuthorityRecord{}, false, err
	}
	active := filterActiveRunPlanAuthorities(byPlanID, collectSupersededRunPlanIDs(byPlanID))
	if len(active) == 0 {
		return RunPlanAuthorityRecord{}, false, fmt.Errorf("run has no active trusted run plan authority")
	}
	if len(active) == 1 {
		return active[0], true, nil
	}
	return RunPlanAuthorityRecord{}, false, fmt.Errorf("run has ambiguous active trusted run plan authority: %s", joinAuthorityPlanIDs(active))
}

func joinAuthorityPlanIDs(active []RunPlanAuthorityRecord) string {
	planIDs := make([]string, 0, len(active))
	for _, authority := range active {
		planIDs = append(planIDs, authority.PlanID)
	}
	sort.Strings(planIDs)
	return strings.Join(planIDs, ",")
}

func collectRunPlanAuthoritiesByPlanID(authorities []RunPlanAuthorityRecord) (map[string]RunPlanAuthorityRecord, error) {
	byPlanID := map[string]RunPlanAuthorityRecord{}
	for _, authority := range authorities {
		existing, ok := byPlanID[authority.PlanID]
		if !ok {
			byPlanID[authority.PlanID] = authority
			continue
		}
		if existing.RunPlanDigest != authority.RunPlanDigest {
			return nil, fmt.Errorf("run has conflicting trusted run plans for plan_id %q", authority.PlanID)
		}
		if authority.RecordedAt.After(existing.RecordedAt) {
			byPlanID[authority.PlanID] = authority
		}
	}
	return byPlanID, nil
}

func collectSupersededRunPlanIDs(byPlanID map[string]RunPlanAuthorityRecord) map[string]struct{} {
	superseded := map[string]struct{}{}
	for _, authority := range byPlanID {
		if authority.SupersedesPlanID != "" {
			superseded[authority.SupersedesPlanID] = struct{}{}
		}
	}
	return superseded
}

func filterActiveRunPlanAuthorities(byPlanID map[string]RunPlanAuthorityRecord, superseded map[string]struct{}) []RunPlanAuthorityRecord {
	active := make([]RunPlanAuthorityRecord, 0, len(byPlanID))
	for planID, authority := range byPlanID {
		if _, ok := superseded[planID]; ok {
			continue
		}
		active = append(active, authority)
	}
	return active
}
