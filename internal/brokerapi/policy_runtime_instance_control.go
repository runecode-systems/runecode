package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) EvaluateInstanceControlAction(action policyengine.ActionRequest) (policyengine.PolicyDecision, error) {
	runtime := policyRuntime{service: s}
	return runtime.EvaluateInstanceControlAction(action)
}

func (r policyRuntime) EvaluateInstanceControlAction(action policyengine.ActionRequest) (policyengine.PolicyDecision, error) {
	compileInput, err := r.loadInstanceControlCompileInput(action)
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
	if err := r.service.RecordPolicyDecision("", "", decision); err != nil {
		return policyengine.PolicyDecision{}, err
	}
	return decision, nil
}

func (r policyRuntime) loadInstanceControlCompileInput(action policyengine.ActionRequest) (policyengine.CompileInput, error) {
	catalog, err := r.trustedPolicyCatalog()
	if err != nil {
		return policyengine.CompileInput{}, err
	}
	roleInput, roleManifest, runInput, runManifest, err := r.readInstanceControlManifests(catalog)
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
	if err := r.attachAllowlists(&compileInput, catalog.byKind[artifacts.TrustedContractImportKindPolicyAllowlist], allowlistRefs); err != nil {
		return policyengine.CompileInput{}, err
	}
	if err := r.attachOptionalRuleSet(&compileInput, catalog.byKind[artifacts.TrustedContractImportKindPolicyRuleSet], strings.TrimSpace(runManifest.RunID)); err != nil {
		return policyengine.CompileInput{}, err
	}
	if strings.TrimSpace(action.RoleFamily) != "" && strings.TrimSpace(action.RoleFamily) != compileInputFixedRoleFamily(compileInput) {
		return policyengine.CompileInput{}, fmt.Errorf("instance-control action role_family mismatch")
	}
	return compileInput, nil
}

func (r policyRuntime) readInstanceControlManifests(catalog trustedPolicyCatalog) (policyengine.ManifestInput, policyengine.RoleManifest, policyengine.ManifestInput, policyengine.CapabilityManifest, error) {
	roleRecord, err := pickLatestRecord(catalog.byKind[artifacts.TrustedContractImportKindRoleManifest], artifacts.TrustedContractImportKindRoleManifest)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	runRecord, err := pickLatestRecord(catalog.byKind[artifacts.TrustedContractImportKindRunCapability], artifacts.TrustedContractImportKindRunCapability)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	roleInput, roleManifest, err := r.readRoleManifest(roleRecord)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	runInput, runManifest, err := r.readCapabilityManifest(runRecord)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	return roleInput, roleManifest, runInput, runManifest, nil
}

func compileInputFixedRoleFamily(input policyengine.CompileInput) string {
	manifest := policyengine.RoleManifest{}
	if err := json.Unmarshal(input.RoleManifest.Payload, &manifest); err != nil {
		return ""
	}
	return strings.TrimSpace(manifest.RoleFamily)
}
