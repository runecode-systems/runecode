package policyengine

import "slices"

// ExecutorRegistryProjection is a read-only, deterministic projection of the
// trusted executor registry suitable for transport in plan/runtime envelopes.
// It is advisory and must not be treated as policy authority.
type ExecutorRegistryProjection struct {
	Version   string                     `json:"version"`
	Executors []ExecutorProjectionRecord `json:"executors"`
}

// ExecutorProjectionRecord exposes executor metadata needed by untrusted
// components without exposing evaluators or mutability.
type ExecutorProjectionRecord struct {
	ExecutorID    string   `json:"executor_id"`
	ExecutorClass string   `json:"executor_class"`
	AllowedRoles  []string `json:"allowed_role_kinds"`
}

var trustedWorkspaceExecutorContractsByID = map[string]typedExecutorContract{
	"workspace-runner": {
		ID:               "workspace-runner",
		AllowedRoles:     roleSet("workspace-edit", "workspace-test"),
		AllowedClass:     "workspace_ordinary",
		AllowedNetwork:   roleSet("none"),
		AllowedEnvKeys:   roleSet("CI", "HOME", "LANG", "LC_ALL", "PATH", "PWD", "TMP", "TMPDIR", "TEMP"),
		AllowEmptyEnv:    true,
		RequireWorkspace: true,
		AllowEnvWrapper:  true,
		AllowedArgvHeads: [][]string{{"go", "test"}, {"go", "build"}, {"go", "vet"}, {"go", "fmt"}, {"go", "list"}, {"python"}, {"node", "--test"}, {"npm", "test"}, {"just", "test"}, {"just", "lint"}, {"just", "fmt"}, {"just", "ci"}},
		MaxArgvItems:     64,
		MaxTimeoutSecs:   3600,
	},
	"python": {
		ID:               "python",
		AllowedRoles:     roleSet("workspace-edit", "workspace-test"),
		AllowedClass:     "workspace_ordinary",
		AllowedNetwork:   roleSet("none"),
		AllowedEnvKeys:   roleSet("PYTHONPATH", "PYTHONWARNINGS", "CI", "HOME", "LANG", "LC_ALL", "PATH", "PWD", "TMP", "TMPDIR", "TEMP"),
		AllowEmptyEnv:    true,
		RequireWorkspace: true,
		AllowEnvWrapper:  false,
		AllowedArgvHeads: [][]string{{"python"}},
		MaxArgvItems:     64,
		MaxTimeoutSecs:   3600,
	},
}

func workspaceExecutorContractByID(executorID string) (typedExecutorContract, bool) {
	contract, ok := trustedWorkspaceExecutorContractsByID[executorID]
	return contract, ok
}

func BuildExecutorRegistryProjection() ExecutorRegistryProjection {
	executorIDs := make([]string, 0, len(trustedWorkspaceExecutorContractsByID))
	for executorID := range trustedWorkspaceExecutorContractsByID {
		executorIDs = append(executorIDs, executorID)
	}
	slices.Sort(executorIDs)

	records := make([]ExecutorProjectionRecord, 0, len(executorIDs))
	for _, executorID := range executorIDs {
		contract := trustedWorkspaceExecutorContractsByID[executorID]
		records = append(records, ExecutorProjectionRecord{
			ExecutorID:    contract.ID,
			ExecutorClass: contract.AllowedClass,
			AllowedRoles:  sortedKeys(contract.AllowedRoles),
		})
	}

	return ExecutorRegistryProjection{
		Version:   "trusted-v1",
		Executors: records,
	}
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func roleSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
