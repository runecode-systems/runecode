package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (r policyRuntime) selectInstanceControlManifestRecords(catalog trustedPolicyCatalog, action policyengine.ActionRequest) (artifacts.ArtifactRecord, artifacts.ArtifactRecord, error) {
	runID := instanceControlRunIDFromAction(action)
	if runID == "" {
		inferred, err := inferSingleInstanceControlRunID(catalog.byKind[artifacts.TrustedContractImportKindRoleManifest], catalog.byKind[artifacts.TrustedContractImportKindRunCapability])
		if err != nil {
			return artifacts.ArtifactRecord{}, artifacts.ArtifactRecord{}, err
		}
		runID = inferred
	}
	roleRecord, err := pickRequiredExactRunRecord(catalog.byKind[artifacts.TrustedContractImportKindRoleManifest], runID, artifacts.TrustedContractImportKindRoleManifest)
	if err != nil {
		return artifacts.ArtifactRecord{}, artifacts.ArtifactRecord{}, err
	}
	runRecord, err := pickRequiredExactRunRecord(catalog.byKind[artifacts.TrustedContractImportKindRunCapability], runID, artifacts.TrustedContractImportKindRunCapability)
	if err != nil {
		return artifacts.ArtifactRecord{}, artifacts.ArtifactRecord{}, err
	}
	return roleRecord, runRecord, nil
}

func instanceControlRunIDFromAction(action policyengine.ActionRequest) string {
	if action.ActionPayload == nil {
		return ""
	}
	raw, _ := action.ActionPayload["run_id"].(string)
	runID := strings.TrimSpace(raw)
	if strings.HasPrefix(runID, "instance-control:") {
		return runID
	}
	return ""
}

func inferSingleInstanceControlRunID(roleRecords, runRecords []artifacts.ArtifactRecord) (string, error) {
	runsWithRole := map[string]struct{}{}
	for _, rec := range roleRecords {
		runID := strings.TrimSpace(rec.RunID)
		if runID != "" {
			runsWithRole[runID] = struct{}{}
		}
	}
	common := make([]string, 0)
	for _, rec := range runRecords {
		runID := strings.TrimSpace(rec.RunID)
		if runID == "" {
			continue
		}
		if _, ok := runsWithRole[runID]; ok {
			common = append(common, runID)
		}
	}
	common = sortedUniquePolicyRefs(common)
	if len(common) == 1 {
		return common[0], nil
	}
	if len(common) == 0 {
		return "", fmt.Errorf("%w: no shared trusted instance-control run context", errPolicyContextUnavailable)
	}
	return "", fmt.Errorf("%w: ambiguous trusted instance-control run contexts; require explicit run_id selector (candidates=%s)", errPolicyContextUnavailable, strings.Join(common, ","))
}
