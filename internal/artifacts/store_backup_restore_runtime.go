package artifacts

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func deriveRestorableRuntimeEvidence(factsByRun map[string]launcherbackend.RuntimeFactsSnapshot) (map[string]launcherbackend.RuntimeEvidenceSnapshot, error) {
	derived := make(map[string]launcherbackend.RuntimeEvidenceSnapshot, len(factsByRun))
	for runID, facts := range factsByRun {
		evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(cloneRuntimeFactsSnapshot(facts))
		if err != nil {
			return nil, fmt.Errorf("runtime facts restore run %q: %w", runID, err)
		}
		derived[runID] = evidence
	}
	return derived, nil
}

func loadRestoredRuntimeEvidence(next *StoreState, evidenceByRun map[string]launcherbackend.RuntimeEvidenceSnapshot, restorableEvidence map[string]launcherbackend.RuntimeEvidenceSnapshot) error {
	for runID, derivedEvidence := range restorableEvidence {
		next.RuntimeEvidenceByRun[runID] = derivedEvidence
	}
	for runID := range evidenceByRun {
		trimmedRunID, err := validateRestoredRuntimeRunID(runID, "runtime evidence")
		if err != nil {
			return err
		}
		if _, ok := next.RuntimeFactsByRun[trimmedRunID]; ok {
			continue
		}
		return fmt.Errorf("runtime evidence restore run %q requires runtime facts", trimmedRunID)
	}
	return nil
}
