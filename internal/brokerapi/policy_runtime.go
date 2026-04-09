package brokerapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var errPolicyContextUnavailable = errors.New("trusted policy context unavailable")

const artifactReadCapabilityID = "cap_artifact_read"

type policyRuntime struct {
	service *Service
}

type trustedPolicyCatalog struct {
	byKind    map[string][]artifacts.ArtifactRecord
	verifiers []trustpolicy.VerifierRecord
}

func (s *Service) EvaluateAction(runID string, action policyengine.ActionRequest) (policyengine.PolicyDecision, error) {
	runtime := policyRuntime{service: s}
	return runtime.EvaluateAction(runID, action)
}

func (r policyRuntime) EvaluateAction(runID string, action policyengine.ActionRequest) (policyengine.PolicyDecision, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return policyengine.PolicyDecision{}, fmt.Errorf("run_id is required")
	}
	compileInput, err := r.loadCompileInput(runID)
	if err != nil {
		return policyengine.PolicyDecision{}, err
	}
	compiled, err := policyengine.Compile(compileInput)
	if err != nil {
		return policyengine.PolicyDecision{}, err
	}
	decision, err := policyengine.Evaluate(compiled, action)
	if err != nil {
		return policyengine.PolicyDecision{}, err
	}
	if err := r.service.RecordPolicyDecision(runID, "", decision); err != nil {
		return policyengine.PolicyDecision{}, err
	}
	return decision, nil
}

func (r policyRuntime) loadCompileInput(runID string) (policyengine.CompileInput, error) {
	catalog, err := r.trustedPolicyCatalog()
	if err != nil {
		return policyengine.CompileInput{}, err
	}
	selected, err := r.loadRuntimeContextRecords(catalog, runID)
	if err != nil {
		return policyengine.CompileInput{}, err
	}
	roleInput, roleManifest, runInput, runManifest, err := r.loadRequiredRuntimeManifests(selected, runID)
	if err != nil {
		return policyengine.CompileInput{}, err
	}
	compileInput := policyengine.CompileInput{
		FixedInvariants:            fixedBrokerPolicyInvariants(),
		RoleManifest:               roleInput,
		RunManifest:                runInput,
		Allowlists:                 []policyengine.ManifestInput{},
		VerifierRecords:            catalog.verifiers,
		RequireSignedContextVerify: true,
	}
	allowlistRefs, err := requiredAllowlistRefs(roleManifest.AllowlistRefs, runManifest.AllowlistRefs)
	if err != nil {
		return policyengine.CompileInput{}, err
	}
	if err := r.applyOptionalStageManifest(&compileInput, selected.stageRecord, runID, &allowlistRefs); err != nil {
		return policyengine.CompileInput{}, err
	}
	if err := r.attachAllowlists(&compileInput, catalog.byKind[artifacts.TrustedContractImportKindPolicyAllowlist], allowlistRefs); err != nil {
		return policyengine.CompileInput{}, err
	}
	if err := r.attachOptionalRuleSet(&compileInput, catalog.byKind[artifacts.TrustedContractImportKindPolicyRuleSet], runID); err != nil {
		return policyengine.CompileInput{}, err
	}
	return compileInput, nil
}

type runtimeContextRecords struct {
	roleRecord  artifacts.ArtifactRecord
	runRecord   artifacts.ArtifactRecord
	stageRecord *artifacts.ArtifactRecord
}

func (r policyRuntime) loadRuntimeContextRecords(catalog trustedPolicyCatalog, runID string) (runtimeContextRecords, error) {
	roleRecord, err := pickRequiredExactRunRecord(catalog.byKind[artifacts.TrustedContractImportKindRoleManifest], runID, artifacts.TrustedContractImportKindRoleManifest)
	if err != nil {
		return runtimeContextRecords{}, err
	}
	runRecord, err := pickRequiredExactRunRecord(catalog.byKind[artifacts.TrustedContractImportKindRunCapability], runID, artifacts.TrustedContractImportKindRunCapability)
	if err != nil {
		return runtimeContextRecords{}, err
	}
	return runtimeContextRecords{
		roleRecord:  roleRecord,
		runRecord:   runRecord,
		stageRecord: pickOptionalExactRunRecord(catalog.byKind[artifacts.TrustedContractImportKindStageCapability], runID),
	}, nil
}

func (r policyRuntime) loadRequiredRuntimeManifests(selected runtimeContextRecords, runID string) (policyengine.ManifestInput, policyengine.RoleManifest, policyengine.ManifestInput, policyengine.CapabilityManifest, error) {
	roleInput, roleManifest, err := r.readRoleManifest(selected.roleRecord)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	runInput, runManifest, err := r.readCapabilityManifest(selected.runRecord)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	if strings.TrimSpace(runManifest.RunID) != runID {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, fmt.Errorf("%w: run capability manifest run_id %q does not match run %q", errPolicyContextUnavailable, runManifest.RunID, runID)
	}
	return roleInput, roleManifest, runInput, runManifest, nil
}

func requiredAllowlistRefs(roleRefs, runRefs []trustpolicy.Digest) ([]string, error) {
	roleAllowlistRefs, err := allowlistIdentities(roleRefs)
	if err != nil {
		return nil, err
	}
	runAllowlistRefs, err := allowlistIdentities(runRefs)
	if err != nil {
		return nil, err
	}
	return append(roleAllowlistRefs, runAllowlistRefs...), nil
}

func (r policyRuntime) applyOptionalStageManifest(input *policyengine.CompileInput, stageRecord *artifacts.ArtifactRecord, runID string, allowlistRefs *[]string) error {
	if stageRecord == nil {
		return nil
	}
	stageInput, stageManifest, err := r.readCapabilityManifest(*stageRecord)
	if err != nil {
		return err
	}
	if strings.TrimSpace(stageManifest.RunID) != runID {
		return fmt.Errorf("%w: stage capability manifest run_id %q does not match run %q", errPolicyContextUnavailable, stageManifest.RunID, runID)
	}
	input.StageManifest = &stageInput
	stageAllowlistRefs, err := allowlistIdentities(stageManifest.AllowlistRefs)
	if err != nil {
		return err
	}
	*allowlistRefs = append(*allowlistRefs, stageAllowlistRefs...)
	return nil
}

func (r policyRuntime) attachAllowlists(input *policyengine.CompileInput, records []artifacts.ArtifactRecord, allowlistRefs []string) error {
	allowlistRefs = sortedUniquePolicyRefs(allowlistRefs)
	allowlistByDigest := map[string]artifacts.ArtifactRecord{}
	for _, rec := range records {
		allowlistByDigest[rec.Reference.Digest] = rec
	}
	for _, ref := range allowlistRefs {
		rec, ok := allowlistByDigest[ref]
		if !ok {
			return fmt.Errorf("%w: missing trusted allowlist %q", errPolicyContextUnavailable, ref)
		}
		manifestInput, err := r.readManifestInput(rec)
		if err != nil {
			return err
		}
		input.Allowlists = append(input.Allowlists, manifestInput)
	}
	return nil
}

func (r policyRuntime) attachOptionalRuleSet(input *policyengine.CompileInput, records []artifacts.ArtifactRecord, runID string) error {
	ruleSet := pickOptionalExactRunRecord(records, runID)
	if ruleSet == nil {
		return nil
	}
	manifestInput, err := r.readManifestInput(*ruleSet)
	if err != nil {
		return err
	}
	input.RuleSet = &manifestInput
	return nil
}

func fixedBrokerPolicyInvariants() policyengine.FixedInvariants {
	return policyengine.FixedInvariants{
		DeniedCapabilities: []string{},
		DeniedActionKinds:  []string{},
	}
}

func policyActionForArtifactRead(req ArtifactReadRequest, record artifacts.ArtifactRecord) (policyengine.ActionRequest, error) {
	artifactHash, err := digestFromIdentity(req.Digest)
	if err != nil {
		return policyengine.ActionRequest{}, err
	}
	roleFamily, roleKind := normalizeRoleForSummary(req.ProducerRole)
	if roleKind == "workspace-edit" && strings.HasPrefix(req.ProducerRole, "workspace") {
		roleKind = "workspace-edit"
	}
	return policyengine.NewArtifactReadAction(policyengine.ArtifactReadActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID:           artifactReadCapabilityID,
			RelevantArtifactHashes: []trustpolicy.Digest{artifactHash},
			Actor: policyengine.ActionActor{
				ActorKind:  "role_instance",
				RoleFamily: roleFamily,
				RoleKind:   roleKind,
			},
		},
		ArtifactHash:      artifactHash,
		ReadMode:          "full",
		ExpectedDataClass: string(record.Reference.DataClass),
		Purpose:           "artifact_read",
	}), nil
}
