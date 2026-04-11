package brokerapi

import (
	"fmt"
	"sort"
	"strings"
)

func extractGateDefinitionsForRunPlan(definition map[string]any) ([]runPlannedGateEntry, error) {
	rawDefs, ok := definition["gate_definitions"].([]any)
	if !ok || len(rawDefs) == 0 {
		return nil, nil
	}
	entries := make([]runPlannedGateEntry, 0, len(rawDefs))
	for index, rawDef := range rawDefs {
		def, ok := rawDef.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("gate_definitions[%d] must be object", index)
		}
		entry, err := extractRunPlannedGateEntry(def, index)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func extractRunPlannedGateEntry(def map[string]any, index int) (runPlannedGateEntry, error) {
	checkpoint, orderIndex, gate, err := parseGateDefinitionCoreFields(def, index)
	if err != nil {
		return runPlannedGateEntry{}, err
	}
	gateID, gateKind, gateVersion, maxAttempts, err := parseGateDefinitionGateFields(gate, index)
	if err != nil {
		return runPlannedGateEntry{}, err
	}
	inputDigests, err := extractExpectedInputDigests(gate["normalized_inputs"])
	if err != nil {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].gate.normalized_inputs: %w", index, err)
	}
	return runPlannedGateEntry{
		GateID:               strings.TrimSpace(gateID),
		GateKind:             strings.TrimSpace(gateKind),
		GateVersion:          strings.TrimSpace(gateVersion),
		PlanCheckpointCode:   strings.TrimSpace(checkpoint),
		PlanOrderIndex:       orderIndex,
		MaxAttempts:          maxAttempts,
		ExpectedInputDigests: inputDigests,
	}, nil
}

func parseGateDefinitionCoreFields(def map[string]any, index int) (string, int, map[string]any, error) {
	checkpoint, _ := def["checkpoint_code"].(string)
	if strings.TrimSpace(checkpoint) == "" {
		return "", 0, nil, fmt.Errorf("gate_definitions[%d].checkpoint_code is required", index)
	}
	orderFloat, ok := def["order_index"].(float64)
	if !ok || orderFloat < 0 {
		return "", 0, nil, fmt.Errorf("gate_definitions[%d].order_index must be >= 0", index)
	}
	gate, ok := def["gate"].(map[string]any)
	if !ok {
		return "", 0, nil, fmt.Errorf("gate_definitions[%d].gate is required", index)
	}
	return checkpoint, int(orderFloat), gate, nil
}

func parseGateDefinitionGateFields(gate map[string]any, index int) (string, string, string, int, error) {
	gateID, _ := gate["gate_id"].(string)
	gateKind, _ := gate["gate_kind"].(string)
	gateVersion, _ := gate["gate_version"].(string)
	if strings.TrimSpace(gateID) == "" || strings.TrimSpace(gateKind) == "" || strings.TrimSpace(gateVersion) == "" {
		return "", "", "", 0, fmt.Errorf("gate_definitions[%d].gate requires gate_id, gate_kind, gate_version", index)
	}
	retry, ok := gate["retry_semantics"].(map[string]any)
	if !ok {
		return "", "", "", 0, fmt.Errorf("gate_definitions[%d].gate.retry_semantics is required", index)
	}
	maxAttemptsFloat, ok := retry["max_attempts"].(float64)
	if !ok || maxAttemptsFloat < 1 {
		return "", "", "", 0, fmt.Errorf("gate_definitions[%d].gate.retry_semantics.max_attempts must be >= 1", index)
	}
	return gateID, gateKind, gateVersion, int(maxAttemptsFloat), nil
}

func extractExpectedInputDigests(raw any) ([]string, error) {
	inputs, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("must be array")
	}
	if len(inputs) == 0 {
		return nil, nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(inputs))
	for index, rawInput := range inputs {
		input, ok := rawInput.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("entry %d must be object", index)
		}
		digest, _ := input["input_digest"].(string)
		digest = strings.TrimSpace(digest)
		if !isValidDigestIdentity(digest) {
			return nil, fmt.Errorf("entry %d has invalid input_digest", index)
		}
		if _, dup := seen[digest]; dup {
			continue
		}
		seen[digest] = struct{}{}
		out = append(out, digest)
	}
	sort.Strings(out)
	return out, nil
}
