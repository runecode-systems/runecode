package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type runPlannedGateEntry struct {
	GateID               string
	GateKind             string
	GateVersion          string
	PlanCheckpointCode   string
	PlanOrderIndex       int
	MaxAttempts          int
	ExpectedInputDigests []string
}

type compiledRunGatePlan struct {
	entries []runPlannedGateEntry
	byGate  map[string]runPlannedGateEntry
}

func (s *Service) compileRunGatePlan(runID string) (compiledRunGatePlan, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return compiledRunGatePlan{}, nil
	}
	records := s.List()
	entries := make([]runPlannedGateEntry, 0, 8)
	foundRun := false
	for _, record := range records {
		if strings.TrimSpace(record.RunID) != runID {
			continue
		}
		foundRun = true
		if !isTrustedGatePlanSourceRole(record.CreatedByRole) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(record.Reference.ContentType), "application/json") {
			continue
		}
		obj, ok, err := s.readWorkflowOrProcessDefinition(record.Reference.Digest)
		if err != nil {
			return compiledRunGatePlan{}, err
		}
		if !ok {
			continue
		}
		definitions, err := extractGateDefinitionsForRunPlan(obj)
		if err != nil {
			return compiledRunGatePlan{}, err
		}
		entries = append(entries, definitions...)
	}
	if !foundRun {
		status, ok := s.RunStatuses()[runID]
		if !ok || strings.TrimSpace(status) == "" {
			return compiledRunGatePlan{}, fmt.Errorf("run %q not found", runID)
		}
	}
	if len(entries) == 0 {
		return compiledRunGatePlan{}, nil
	}
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
	byGate := make(map[string]runPlannedGateEntry, len(entries))
	for _, entry := range entries {
		key := runPlanGateKey(entry.GateID, entry.GateKind, entry.GateVersion, entry.PlanCheckpointCode, entry.PlanOrderIndex)
		if existing, ok := byGate[key]; ok {
			if !sameRunPlannedGateEntry(existing, entry) {
				return compiledRunGatePlan{}, fmt.Errorf("ambiguous gate plan entry for gate %q at %s[%d]", entry.GateID, entry.PlanCheckpointCode, entry.PlanOrderIndex)
			}
			continue
		}
		byGate[key] = entry
	}
	return compiledRunGatePlan{entries: entries, byGate: byGate}, nil
}

func isTrustedGatePlanSourceRole(role string) bool {
	switch strings.TrimSpace(role) {
	case "broker", "brokerapi":
		return true
	default:
		return false
	}
}

func (s *Service) readWorkflowOrProcessDefinition(digest string) (map[string]any, bool, error) {
	r, err := s.Get(digest)
	if err != nil {
		return nil, false, fmt.Errorf("read trusted plan artifact %q: %w", digest, err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, false, fmt.Errorf("read trusted plan artifact body %q: %w", digest, err)
	}
	obj := map[string]any{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, false, nil
	}
	schemaID, _ := obj["schema_id"].(string)
	switch strings.TrimSpace(schemaID) {
	case "runecode.protocol.v0.WorkflowDefinition", "runecode.protocol.v0.ProcessDefinition":
		return obj, true, nil
	default:
		return nil, false, nil
	}
}

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
	checkpoint, _ := def["checkpoint_code"].(string)
	if strings.TrimSpace(checkpoint) == "" {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].checkpoint_code is required", index)
	}
	orderFloat, ok := def["order_index"].(float64)
	if !ok || orderFloat < 0 {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].order_index must be >= 0", index)
	}
	orderIndex := int(orderFloat)
	gate, ok := def["gate"].(map[string]any)
	if !ok {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].gate is required", index)
	}
	gateID, _ := gate["gate_id"].(string)
	gateKind, _ := gate["gate_kind"].(string)
	gateVersion, _ := gate["gate_version"].(string)
	if strings.TrimSpace(gateID) == "" || strings.TrimSpace(gateKind) == "" || strings.TrimSpace(gateVersion) == "" {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].gate requires gate_id, gate_kind, gate_version", index)
	}
	retry, ok := gate["retry_semantics"].(map[string]any)
	if !ok {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].gate.retry_semantics is required", index)
	}
	maxAttemptsFloat, ok := retry["max_attempts"].(float64)
	if !ok || maxAttemptsFloat < 1 {
		return runPlannedGateEntry{}, fmt.Errorf("gate_definitions[%d].gate.retry_semantics.max_attempts must be >= 1", index)
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
		MaxAttempts:          int(maxAttemptsFloat),
		ExpectedInputDigests: inputDigests,
	}, nil
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
	if a.GateID != b.GateID || a.GateKind != b.GateKind || a.GateVersion != b.GateVersion || a.PlanCheckpointCode != b.PlanCheckpointCode || a.PlanOrderIndex != b.PlanOrderIndex || a.MaxAttempts != b.MaxAttempts {
		return false
	}
	if len(a.ExpectedInputDigests) != len(b.ExpectedInputDigests) {
		return false
	}
	for i := range a.ExpectedInputDigests {
		if a.ExpectedInputDigests[i] != b.ExpectedInputDigests[i] {
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
