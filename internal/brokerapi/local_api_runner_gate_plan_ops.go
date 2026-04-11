package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
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
	records, foundRun := runScopedPlanRecords(s.List(), runID)
	if err := ensureRunExistsForGatePlan(s.RunStatuses(), runID, foundRun); err != nil {
		return compiledRunGatePlan{}, err
	}
	entries, err := s.collectRunPlanEntries(records)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	if len(entries) == 0 {
		return compiledRunGatePlan{}, nil
	}
	sortRunPlanEntries(entries)
	byGate, err := buildRunPlanEntryIndex(entries)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	return compiledRunGatePlan{entries: entries, byGate: byGate}, nil
}

func runScopedPlanRecords(records []artifacts.ArtifactRecord, runID string) ([]artifacts.ArtifactRecord, bool) {
	out := make([]artifacts.ArtifactRecord, 0, len(records))
	foundRun := false
	for _, record := range records {
		if strings.TrimSpace(record.RunID) != runID {
			continue
		}
		foundRun = true
		if !isTrustedGatePlanSourceRecord(record) {
			continue
		}
		out = append(out, record)
	}
	return out, foundRun
}

func isTrustedGatePlanSourceRecord(record artifacts.ArtifactRecord) bool {
	if !isTrustedGatePlanSourceRole(record.CreatedByRole) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(record.Reference.ContentType), "application/json")
}

func ensureRunExistsForGatePlan(statuses map[string]string, runID string, foundRun bool) error {
	if foundRun {
		return nil
	}
	status, ok := statuses[runID]
	if !ok || strings.TrimSpace(status) == "" {
		return fmt.Errorf("run %q not found", runID)
	}
	return nil
}

func (s *Service) collectRunPlanEntries(records []artifacts.ArtifactRecord) ([]runPlannedGateEntry, error) {
	entries := make([]runPlannedGateEntry, 0, len(records))
	for _, record := range records {
		definitions, err := s.runPlanEntriesFromRecord(record)
		if err != nil {
			return nil, err
		}
		entries = append(entries, definitions...)
	}
	return entries, nil
}

func (s *Service) runPlanEntriesFromRecord(record artifacts.ArtifactRecord) ([]runPlannedGateEntry, error) {
	obj, ok, err := s.readWorkflowOrProcessDefinition(record.Reference.Digest)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return extractGateDefinitionsForRunPlan(obj)
}

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
