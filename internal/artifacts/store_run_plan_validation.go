package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/runplan"
)

func validateRunPlanAuthorityRecord(rec RunPlanAuthorityRecord) error {
	if rec.RunID == "" || rec.PlanID == "" {
		return fmt.Errorf("run_id and plan_id are required")
	}
	if rec.SupersedesPlanID != "" && rec.SupersedesPlanID == rec.PlanID {
		return fmt.Errorf("supersedes_plan_id must differ from plan_id")
	}
	if err := validateRunPlanAuthorityDigests(rec); err != nil {
		return err
	}
	if rec.ProjectContextIdentityDigest != "" && !isValidDigest(rec.ProjectContextIdentityDigest) {
		return fmt.Errorf("project_context_identity_digest must be sha256 identity when provided")
	}
	for i := range rec.Entries {
		if err := validateRunPlanGateEntryRecord(rec.Entries[i], i); err != nil {
			return err
		}
	}
	return nil
}

func validateRunPlanAuthorityDigests(rec RunPlanAuthorityRecord) error {
	required := map[string]string{
		"run_plan_digest":          rec.RunPlanDigest,
		"workflow_definition_hash": rec.WorkflowDefinitionHash,
		"process_definition_hash":  rec.ProcessDefinitionHash,
		"policy_context_hash":      rec.PolicyContextHash,
	}
	for field, digest := range required {
		if !isValidDigest(digest) {
			return fmt.Errorf("%s must be sha256 identity", field)
		}
	}
	return nil
}

func validateRunPlanGateEntryRecord(entry RunPlanGateEntryRecord, index int) error {
	if err := validateRunPlanGateEntryCore(entry, index); err != nil {
		return err
	}
	if err := validateRunPlanGateEntryDigests(entry, index); err != nil {
		return err
	}
	return validateRunPlanGateEntryHandoffs(entry, index)
}

func validateRunPlanGateEntryCore(entry RunPlanGateEntryRecord, index int) error {
	if strings.TrimSpace(entry.EntryID) == "" {
		return fmt.Errorf("entries[%d].entry_id is required", index)
	}
	if strings.TrimSpace(entry.EntryKind) == "" {
		return fmt.Errorf("entries[%d].entry_kind is required", index)
	}
	if strings.TrimSpace(entry.PlanCheckpointCode) == "" || entry.PlanOrderIndex < 0 {
		return fmt.Errorf("entries[%d] requires plan_checkpoint_code and plan_order_index >= 0", index)
	}
	if strings.TrimSpace(entry.GateID) == "" || strings.TrimSpace(entry.GateKind) == "" || strings.TrimSpace(entry.GateVersion) == "" {
		return fmt.Errorf("entries[%d] requires gate_id, gate_kind, gate_version", index)
	}
	if strings.TrimSpace(entry.StageID) == "" || strings.TrimSpace(entry.StepID) == "" || strings.TrimSpace(entry.RoleInstanceID) == "" {
		return fmt.Errorf("entries[%d] requires stage_id, step_id, role_instance_id", index)
	}
	if entry.MaxAttempts < 1 {
		return fmt.Errorf("entries[%d].max_attempts must be >= 1", index)
	}
	return nil
}

func validateRunPlanGateEntryDigests(entry RunPlanGateEntryRecord, index int) error {
	for j, digest := range entry.ExpectedInputDigests {
		if !isValidDigest(strings.TrimSpace(digest)) {
			return fmt.Errorf("entries[%d].expected_input_digests[%d] must be sha256 identity", index, j)
		}
	}
	return nil
}

func validateRunPlanGateEntryHandoffs(entry RunPlanGateEntryRecord, index int) error {
	for j, handoff := range entry.DependencyCacheHandoffs {
		if !isValidDigest(strings.TrimSpace(handoff.RequestDigest)) {
			return fmt.Errorf("entries[%d].dependency_cache_handoffs[%d].request_digest must be sha256 identity", index, j)
		}
		if strings.TrimSpace(handoff.ConsumerRole) == "" {
			return fmt.Errorf("entries[%d].dependency_cache_handoffs[%d].consumer_role is required", index, j)
		}
		if !handoff.Required {
			return fmt.Errorf("entries[%d].dependency_cache_handoffs[%d].required must be true", index, j)
		}
	}
	return nil
}

func validateRunPlanCompilationRecord(rec RunPlanCompilationRecord) error {
	if err := validateRunPlanCompilationIdentity(rec); err != nil {
		return err
	}
	if err := validateRunPlanCompilationHashes(rec); err != nil {
		return err
	}
	return validateRunPlanCompilationDigests(rec)
}

func validateRunPlanCompilationIdentity(rec RunPlanCompilationRecord) error {
	if rec.RunID == "" || rec.PlanID == "" {
		return fmt.Errorf("run plan compilation run_id and plan_id are required")
	}
	if rec.SupersedesPlanID != "" && rec.SupersedesPlanID == rec.PlanID {
		return fmt.Errorf("run plan compilation supersedes_plan_id must differ from plan_id")
	}
	if rec.CompiledAt.IsZero() {
		return fmt.Errorf("run plan compilation compiled_at is required")
	}
	return nil
}

func validateRunPlanCompilationHashes(rec RunPlanCompilationRecord) error {
	checks := map[string]string{
		"run plan compilation run_plan_digest":          rec.RunPlanDigest,
		"run plan compilation workflow_definition_ref":  rec.WorkflowDefinitionRef,
		"run plan compilation process_definition_ref":   rec.ProcessDefinitionRef,
		"run plan compilation workflow_definition_hash": rec.WorkflowDefinitionHash,
		"run plan compilation process_definition_hash":  rec.ProcessDefinitionHash,
		"run plan compilation policy_context_hash":      rec.PolicyContextHash,
	}
	for field, value := range checks {
		if !isValidDigest(value) {
			return fmt.Errorf("%s must be sha256 identity", field)
		}
	}
	if rec.ProjectContextIdentityDigest != "" && !isValidDigest(rec.ProjectContextIdentityDigest) {
		return fmt.Errorf("run plan compilation project_context_identity_digest must be sha256 identity when provided")
	}
	return nil
}

func validateRunPlanCompilationDigests(rec RunPlanCompilationRecord) error {
	bindingDigest, recordDigest, err := computeRunPlanCompilationDigests(rec)
	if err != nil {
		return err
	}
	if rec.BindingDigest == "" || rec.RecordDigest == "" {
		return fmt.Errorf("run plan compilation binding_digest and record_digest are required")
	}
	if rec.BindingDigest != bindingDigest {
		return fmt.Errorf("run plan compilation binding_digest mismatch")
	}
	if rec.RecordDigest != recordDigest {
		return fmt.Errorf("run plan compilation record_digest mismatch")
	}
	return nil
}

func validateRunPlanAuthorityCompilationBinding(authority RunPlanAuthorityRecord, compilation RunPlanCompilationRecord) error {
	if authority.RunID != compilation.RunID || authority.PlanID != compilation.PlanID {
		return fmt.Errorf("run plan authority and compilation must share run_id and plan_id")
	}
	pairs := []struct {
		name      string
		authority string
		compiled  string
	}{
		{name: "run_plan_digest", authority: authority.RunPlanDigest, compiled: compilation.RunPlanDigest},
		{name: "workflow_definition_hash", authority: authority.WorkflowDefinitionHash, compiled: compilation.WorkflowDefinitionHash},
		{name: "process_definition_hash", authority: authority.ProcessDefinitionHash, compiled: compilation.ProcessDefinitionHash},
		{name: "policy_context_hash", authority: authority.PolicyContextHash, compiled: compilation.PolicyContextHash},
		{name: "project_context_identity_digest", authority: authority.ProjectContextIdentityDigest, compiled: compilation.ProjectContextIdentityDigest},
		{name: "supersedes_plan_id", authority: authority.SupersedesPlanID, compiled: compilation.SupersedesPlanID},
	}
	for _, pair := range pairs {
		if pair.authority != pair.compiled {
			return fmt.Errorf("run plan authority and compilation %s mismatch", pair.name)
		}
	}
	return nil
}

func validateRunPlanAuthorityArtifactConsistency(authority RunPlanAuthorityRecord, compilation RunPlanCompilationRecord, artifactRecord ArtifactRecord, ioStore *storeIO) error {
	compiled, err := decodeRunPlanArtifact(authority, artifactRecord, ioStore)
	if err != nil {
		return err
	}
	if err := validateCompiledRunPlanAuthorityFields(authority, compiled); err != nil {
		return err
	}
	artifactEntries := runPlanGateEntriesFromCompiledEntries(compiled.Entries)
	if !sameRunPlanGateEntryRecordSlices(authority.Entries, artifactEntries) {
		return fmt.Errorf("run plan artifact %q entries mismatch persisted authority", authority.RunPlanDigest)
	}
	if strings.TrimSpace(compilation.RunPlanDigest) != authority.RunPlanDigest {
		return fmt.Errorf("run plan compilation run_plan_digest mismatch")
	}
	return nil
}

func decodeRunPlanArtifact(authority RunPlanAuthorityRecord, artifactRecord ArtifactRecord, ioStore *storeIO) (runplan.RunPlan, error) {
	payload, err := ioStore.readBlob(artifactRecord.BlobPath)
	if err != nil {
		return runplan.RunPlan{}, fmt.Errorf("read run plan artifact %q: %w", authority.RunPlanDigest, err)
	}
	compiled := runplan.RunPlan{}
	if err := json.Unmarshal(payload, &compiled); err != nil {
		return runplan.RunPlan{}, fmt.Errorf("decode run plan artifact %q: %w", authority.RunPlanDigest, err)
	}
	return compiled, nil
}

func validateCompiledRunPlanAuthorityFields(authority RunPlanAuthorityRecord, compiled runplan.RunPlan) error {
	pairs := []struct {
		name string
		want string
		got  string
	}{
		{name: "run_id", want: authority.RunID, got: strings.TrimSpace(compiled.RunID)},
		{name: "plan_id", want: authority.PlanID, got: strings.TrimSpace(compiled.PlanID)},
		{name: "supersedes_plan_id", want: authority.SupersedesPlanID, got: strings.TrimSpace(compiled.SupersedesPlanID)},
		{name: "workflow_definition_hash", want: authority.WorkflowDefinitionHash, got: strings.TrimSpace(compiled.WorkflowDefinitionHash)},
		{name: "process_definition_hash", want: authority.ProcessDefinitionHash, got: strings.TrimSpace(compiled.ProcessDefinitionHash)},
		{name: "policy_context_hash", want: authority.PolicyContextHash, got: strings.TrimSpace(compiled.PolicyContextHash)},
		{name: "project_context_identity_digest", want: authority.ProjectContextIdentityDigest, got: strings.TrimSpace(compiled.ProjectContextIdentityDigest)},
	}
	for _, pair := range pairs {
		if pair.got != pair.want {
			return fmt.Errorf("run plan artifact %q %s mismatch", authority.RunPlanDigest, pair.name)
		}
	}
	return nil
}
