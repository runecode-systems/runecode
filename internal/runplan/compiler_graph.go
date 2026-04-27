package runplan

import (
	"fmt"
	"sort"
	"strings"
)

const stepCompletedDependencyKind = "step_completed"

func compileProcessDependencyEdges(gates []GateDefinition, edges []DependencyEdge) ([]DependencyEdge, error) {
	stepIndex, err := gateDefinitionsByStepID(gates)
	if err != nil {
		return nil, err
	}
	compiled, err := normalizedDependencyEdges(stepIndex, edges)
	if err != nil {
		return nil, err
	}
	if err := validateDependencyEdgesAcyclic(stepIndex, compiled); err != nil {
		return nil, err
	}
	return compiled, nil
}

func gateDefinitionsByStepID(gates []GateDefinition) (map[string]GateDefinition, error) {
	stepIndex := map[string]GateDefinition{}
	for _, gate := range gates {
		stepID := strings.TrimSpace(gate.StepID)
		if existing, seen := stepIndex[stepID]; seen {
			return nil, fmt.Errorf("step_id %q is used by multiple gate definitions (%q and %q)", stepID, existing.Gate.GateID, gate.Gate.GateID)
		}
		stepIndex[stepID] = gate
	}
	return stepIndex, nil
}

func normalizedDependencyEdges(stepIndex map[string]GateDefinition, edges []DependencyEdge) ([]DependencyEdge, error) {
	compiled := make([]DependencyEdge, 0, len(edges))
	seen := map[string]struct{}{}
	for _, edge := range edges {
		normalized, err := validateAndNormalizeDependencyEdge(stepIndex, edge)
		if err != nil {
			return nil, err
		}
		key := normalizedDependencyEdgeKey(normalized)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("dependency edge %q is duplicated", key)
		}
		seen[key] = struct{}{}
		compiled = append(compiled, normalized)
	}
	sort.Slice(compiled, func(i, j int) bool {
		left := normalizedDependencyEdgeKey(compiled[i])
		right := normalizedDependencyEdgeKey(compiled[j])
		return left < right
	})
	return compiled, nil
}

func validateAndNormalizeDependencyEdge(stepIndex map[string]GateDefinition, edge DependencyEdge) (DependencyEdge, error) {
	upstreamStepID := strings.TrimSpace(edge.UpstreamStepID)
	downstreamStepID := strings.TrimSpace(edge.DownstreamStepID)
	dependencyKind := strings.TrimSpace(edge.DependencyKind)
	if upstreamStepID == "" {
		return DependencyEdge{}, fmt.Errorf("dependency edge upstream_step_id is required")
	}
	if downstreamStepID == "" {
		return DependencyEdge{}, fmt.Errorf("dependency edge downstream_step_id is required")
	}
	if dependencyKind != stepCompletedDependencyKind {
		return DependencyEdge{}, fmt.Errorf("dependency edge kind %q is not supported", dependencyKind)
	}
	if upstreamStepID == downstreamStepID {
		return DependencyEdge{}, fmt.Errorf("dependency edge %q -> %q must not self-reference", upstreamStepID, downstreamStepID)
	}
	upstreamGate, ok := stepIndex[upstreamStepID]
	if !ok {
		return DependencyEdge{}, fmt.Errorf("dependency edge references unknown upstream_step_id %q", upstreamStepID)
	}
	downstreamGate, ok := stepIndex[downstreamStepID]
	if !ok {
		return DependencyEdge{}, fmt.Errorf("dependency edge references unknown downstream_step_id %q", downstreamStepID)
	}
	if upstreamGate.OrderIndex >= downstreamGate.OrderIndex {
		return DependencyEdge{}, fmt.Errorf("dependency edge %q -> %q violates deterministic order_index monotonicity", upstreamStepID, downstreamStepID)
	}
	return DependencyEdge{UpstreamStepID: upstreamStepID, DownstreamStepID: downstreamStepID, DependencyKind: dependencyKind}, nil
}

func normalizedDependencyEdgeKey(edge DependencyEdge) string {
	return fmt.Sprintf("%s|%s|%s", edge.UpstreamStepID, edge.DownstreamStepID, edge.DependencyKind)
}

func validateDependencyEdgesAcyclic(stepIndex map[string]GateDefinition, edges []DependencyEdge) error {
	adjacency, indegree := buildGraphMaps(stepIndex, edges)
	roots := findRootNodes(indegree)
	if err := topologicalSort(adjacency, indegree, roots); err != nil {
		return err
	}
	return nil
}

func buildGraphMaps(stepIndex map[string]GateDefinition, edges []DependencyEdge) (map[string][]string, map[string]int) {
	adjacency := map[string][]string{}
	indegree := map[string]int{}
	for stepID := range stepIndex {
		indegree[stepID] = 0
	}
	for _, edge := range edges {
		adjacency[edge.UpstreamStepID] = append(adjacency[edge.UpstreamStepID], edge.DownstreamStepID)
		indegree[edge.DownstreamStepID]++
	}
	return adjacency, indegree
}

func findRootNodes(indegree map[string]int) []string {
	roots := make([]string, 0)
	for stepID, count := range indegree {
		if count == 0 {
			roots = append(roots, stepID)
		}
	}
	sort.Strings(roots)
	return roots
}

func topologicalSort(adjacency map[string][]string, indegree map[string]int, roots []string) error {
	queue := append([]string(nil), roots...)
	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++
		neighbors := adjacency[current]
		sort.Strings(neighbors)
		for _, downstream := range neighbors {
			indegree[downstream]--
			if indegree[downstream] == 0 {
				queue = append(queue, downstream)
				sort.Strings(queue)
			}
		}
	}
	if visited != len(indegree) {
		return fmt.Errorf("process dependency graph must be a DAG (cycle detected)")
	}
	return nil
}
