package runplan

import (
	"fmt"
	"sort"
)

const (
	runPlanEntryKindGate     = "gate"
	waitingOperatorInputKind = "waiting_operator_input"
	waitingApprovalKind      = "waiting_approval"
)

func compileRunPlanEntries(gates []GateDefinition, dependencyEdges []DependencyEdge) ([]Entry, error) {
	dependsOnByStep, blocksByStep := dependencyAdjacencyByStep(dependencyEdges)
	entries := make([]Entry, 0, len(gates))
	entryIDs := map[string]struct{}{}
	for _, gate := range gates {
		entry := newRunPlanEntry(gate, dependsOnByStep[gate.StepID], blocksByStep[gate.StepID])
		if _, seen := entryIDs[entry.EntryID]; seen {
			return nil, fmt.Errorf("run plan entry_id %q is duplicated", entry.EntryID)
		}
		entryIDs[entry.EntryID] = struct{}{}
		entries = append(entries, entry)
	}
	return entries, nil
}

func dependencyAdjacencyByStep(edges []DependencyEdge) (map[string][]string, map[string][]string) {
	dependsOnByStep := map[string][]string{}
	blocksByStep := map[string][]string{}
	for _, edge := range edges {
		dependsOnByStep[edge.DownstreamStepID] = append(dependsOnByStep[edge.DownstreamStepID], edge.UpstreamStepID)
		blocksByStep[edge.UpstreamStepID] = append(blocksByStep[edge.UpstreamStepID], edge.DownstreamStepID)
	}
	for stepID := range dependsOnByStep {
		sort.Strings(dependsOnByStep[stepID])
	}
	for stepID := range blocksByStep {
		sort.Strings(blocksByStep[stepID])
	}
	return dependsOnByStep, blocksByStep
}

func newRunPlanEntry(gate GateDefinition, dependsOnStepIDs []string, blocksStepIDs []string) Entry {
	return Entry{
		EntryID:                 gate.StepID,
		EntryKind:               runPlanEntryKindGate,
		OrderIndex:              gate.OrderIndex,
		StageID:                 gate.StageID,
		StepID:                  gate.StepID,
		RoleInstanceID:          gate.RoleInstanceID,
		ExecutorBindingID:       gate.ExecutorBindingID,
		CheckpointCode:          gate.CheckpointCode,
		Gate:                    gate.Gate,
		DependencyCacheHandoffs: gate.DependencyCacheHandoffs,
		DependsOnEntryIDs:       append([]string{}, dependsOnStepIDs...),
		BlocksEntryIDs:          append([]string{}, blocksStepIDs...),
		SupportedWaitKinds:      []string{waitingOperatorInputKind, waitingApprovalKind},
	}
}
