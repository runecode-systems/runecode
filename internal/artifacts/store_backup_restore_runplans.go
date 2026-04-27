package artifacts

import "fmt"

func loadRestoredRunPlans(next *StoreState, authorities []RunPlanAuthorityRecord, compilations []RunPlanCompilationRecord, ioStore *storeIO) error {
	if err := restoreRunPlanAuthorities(next, authorities); err != nil {
		return err
	}
	if err := restoreRunPlanCompilations(next, compilations); err != nil {
		return err
	}
	if err := validateRestoredRunPlanBindings(next, ioStore); err != nil {
		return err
	}
	rebuildRunPlanRefsByRunLocked(next)
	return nil
}

func restoreRunPlanAuthorities(next *StoreState, authorities []RunPlanAuthorityRecord) error {
	for _, rec := range authorities {
		normalized := normalizeRunPlanAuthorityRecord(rec)
		if err := validateRunPlanAuthorityRecord(normalized); err != nil {
			return err
		}
		if _, ok := next.Artifacts[normalized.RunPlanDigest]; !ok {
			return fmt.Errorf("run plan digest %q for run %q plan %q not found in restored artifacts", normalized.RunPlanDigest, normalized.RunID, normalized.PlanID)
		}
		next.RunPlanAuthorities[runPlanStateKey(normalized.RunID, normalized.PlanID)] = normalized
	}
	return nil
}

func restoreRunPlanCompilations(next *StoreState, compilations []RunPlanCompilationRecord) error {
	for _, rec := range compilations {
		normalized := normalizeRunPlanCompilationRecord(rec)
		if err := validateRunPlanCompilationRecord(normalized); err != nil {
			return err
		}
		next.RunPlanCompilations[runPlanStateKey(normalized.RunID, normalized.PlanID)] = normalized
	}
	return nil
}

func validateRestoredRunPlanBindings(next *StoreState, ioStore *storeIO) error {
	for key, authority := range next.RunPlanAuthorities {
		compilation, ok := next.RunPlanCompilations[key]
		if !ok {
			return fmt.Errorf("run plan compilation missing for run %q plan %q", authority.RunID, authority.PlanID)
		}
		if err := validateRunPlanAuthorityCompilationBinding(authority, compilation); err != nil {
			return err
		}
		record, ok := next.Artifacts[authority.RunPlanDigest]
		if !ok {
			return fmt.Errorf("run plan digest %q for run %q plan %q not found in restored artifacts", authority.RunPlanDigest, authority.RunID, authority.PlanID)
		}
		if err := validateRunPlanAuthorityArtifactConsistency(authority, compilation, record, ioStore); err != nil {
			return err
		}
	}
	return nil
}
