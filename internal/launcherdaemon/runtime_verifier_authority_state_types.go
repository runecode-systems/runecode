package launcherdaemon

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	runtimeVerifierAuthorityStateSchemaID      = "runecode.launcher.runtime-verifier-authority-state"
	runtimeVerifierAuthorityStateSchemaVersion = "0.1.0"
	runtimeVerifierAuthorityStateFileName      = "runtime_verifier_authority_state.json"
	runtimeVerifierAuthorityStateDirName       = "authority-state"

	runtimeVerifierAuthorityStatusActive  = "active"
	runtimeVerifierAuthorityStatusRevoked = "revoked"

	runtimeVerifierAuthoritySourceBuiltin  = "builtin"
	runtimeVerifierAuthoritySourceImported = "imported"

	runtimeVerifierAuthorityMergeModeExtend  = "extend"
	runtimeVerifierAuthorityMergeModeReplace = "replace"

	runtimeVerifierAuthorityReceiptSchemaID      = "runecode.launcher.runtime-verifier-authority-state-receipt"
	runtimeVerifierAuthorityReceiptSchemaVersion = "0.1.0"
	runtimeVerifierAuthorityReceiptFileName      = "runtime_verifier_authority_state_receipt.json"
)

type runtimeVerifierAuthorityGeneration struct {
	Revision         uint64 `json:"revision"`
	PreviousRevision uint64 `json:"previous_revision,omitempty"`
	ChangedAt        string `json:"changed_at"`
	Reason           string `json:"reason,omitempty"`
}

type runtimeVerifierAuthorityEntry struct {
	VerifierSetRef string                       `json:"verifier_set_ref"`
	Records        []trustpolicy.VerifierRecord `json:"records"`
	Status         string                       `json:"status"`
	Source         string                       `json:"source"`
	ChangedAt      string                       `json:"changed_at"`
	Reason         string                       `json:"reason,omitempty"`
}

type runtimeVerifierAuthorityState struct {
	SchemaID          string                                     `json:"schema_id"`
	SchemaVersion     string                                     `json:"schema_version"`
	StateDigest       string                                     `json:"state_digest"`
	Generation        runtimeVerifierAuthorityGeneration         `json:"generation"`
	MergeMode         string                                     `json:"merge_mode,omitempty"`
	AuthoritiesByKind map[string][]runtimeVerifierAuthorityEntry `json:"authorities_by_kind"`
}

type RuntimeVerifierAuthorityStateImportReceipt struct {
	SchemaID                string                             `json:"schema_id"`
	SchemaVersion           string                             `json:"schema_version"`
	Action                  string                             `json:"action"`
	ChangedAt               string                             `json:"changed_at"`
	WorkRootScope           string                             `json:"work_root_scope"`
	ImportedStateDigest     string                             `json:"imported_state_digest"`
	EffectiveStateDigest    string                             `json:"effective_state_digest"`
	PreviousEffectiveDigest string                             `json:"previous_effective_digest,omitempty"`
	ImportedGeneration      runtimeVerifierAuthorityGeneration `json:"imported_generation"`
	EffectiveGeneration     runtimeVerifierAuthorityGeneration `json:"effective_generation"`
}

func loadEffectiveRuntimeVerifierAuthorityState(cacheRoot string) (runtimeVerifierAuthorityState, error) {
	builtin := builtInRuntimeVerifierAuthorityState()
	imported, found, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		return runtimeVerifierAuthorityState{}, err
	}
	if !found {
		return builtin, nil
	}
	return mergeRuntimeVerifierAuthorityState(builtin, imported)
}

func ExportBuiltInRuntimeVerifierAuthorityState() ([]byte, error) {
	return marshalRuntimeVerifierAuthorityState(builtInRuntimeVerifierAuthorityState())
}

func builtInRuntimeVerifierAuthorityState() runtimeVerifierAuthorityState {
	state := runtimeVerifierAuthorityState{
		SchemaID:          runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion:     runtimeVerifierAuthorityStateSchemaVersion,
		Generation:        runtimeVerifierAuthorityGeneration{Revision: 1, ChangedAt: "2026-04-29T00:00:00Z", Reason: "launcher built-in baseline authority"},
		MergeMode:         runtimeVerifierAuthorityMergeModeExtend,
		AuthoritiesByKind: builtInRuntimeVerifierAuthorityEntriesByKind(),
	}
	state.StateDigest = mustRuntimeVerifierAuthorityStateDigest(state)
	return state
}

func builtInRuntimeVerifierAuthorityEntriesByKind() map[string][]runtimeVerifierAuthorityEntry {
	authorities := map[string][]runtimeVerifierAuthorityEntry{}
	for kind, records := range builtInRuntimeVerifierPoliciesByKind() {
		authorities[kind] = []runtimeVerifierAuthorityEntry{newBuiltInRuntimeVerifierAuthorityEntry(records)}
	}
	return authorities
}

func newBuiltInRuntimeVerifierAuthorityEntry(records []trustpolicy.VerifierRecord) runtimeVerifierAuthorityEntry {
	return runtimeVerifierAuthorityEntry{
		VerifierSetRef: mustRuntimeVerifierSetDigest(records),
		Records:        records,
		Status:         runtimeVerifierAuthorityStatusActive,
		Source:         runtimeVerifierAuthoritySourceBuiltin,
		ChangedAt:      "2026-04-29T00:00:00Z",
		Reason:         "launcher built-in baseline authority",
	}
}

func normalizeRuntimeVerifierAuthorityState(state runtimeVerifierAuthorityState) runtimeVerifierAuthorityState {
	if state.AuthoritiesByKind == nil {
		state.AuthoritiesByKind = map[string][]runtimeVerifierAuthorityEntry{}
	}
	for kind, entries := range state.AuthoritiesByKind {
		sort.SliceStable(entries, func(i int, j int) bool {
			if entries[i].VerifierSetRef == entries[j].VerifierSetRef {
				return entries[i].Status < entries[j].Status
			}
			return entries[i].VerifierSetRef < entries[j].VerifierSetRef
		})
		state.AuthoritiesByKind[kind] = entries
	}
	state.StateDigest = mustRuntimeVerifierAuthorityStateDigest(state)
	return state
}

func loadCurrentEffectiveRuntimeVerifierAuthorityState(cacheRoot string) (runtimeVerifierAuthorityState, bool, error) {
	if _, err := os.Stat(runtimeVerifierAuthorityStatePath(cacheRoot)); err != nil {
		if os.IsNotExist(err) {
			state, loadErr := loadEffectiveRuntimeVerifierAuthorityState(cacheRoot)
			if loadErr != nil {
				return runtimeVerifierAuthorityState{}, false, loadErr
			}
			return state, false, nil
		}
		return runtimeVerifierAuthorityState{}, false, err
	}
	state, err := loadEffectiveRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		return runtimeVerifierAuthorityState{}, false, err
	}
	return state, true, nil
}

func validateRuntimeVerifierAuthorityState(state runtimeVerifierAuthorityState) error {
	if err := validateRuntimeVerifierAuthorityStateEnvelope(state); err != nil {
		return err
	}
	if err := validateRuntimeVerifierAuthorityGeneration(state.Generation); err != nil {
		return err
	}
	if err := validateRuntimeVerifierAuthorityEntries(state.AuthoritiesByKind); err != nil {
		return err
	}
	return nil
}

func validateRuntimeVerifierAuthorityStateEnvelope(state runtimeVerifierAuthorityState) error {
	if strings.TrimSpace(state.SchemaID) != runtimeVerifierAuthorityStateSchemaID {
		return fmt.Errorf("runtime verifier authority state schema_id is invalid")
	}
	if strings.TrimSpace(state.SchemaVersion) != runtimeVerifierAuthorityStateSchemaVersion {
		return fmt.Errorf("runtime verifier authority state schema_version is invalid")
	}
	if mode := strings.TrimSpace(state.MergeMode); mode != "" && mode != runtimeVerifierAuthorityMergeModeExtend && mode != runtimeVerifierAuthorityMergeModeReplace {
		return fmt.Errorf("runtime verifier authority state merge_mode is invalid")
	}
	return nil
}

func validateRuntimeVerifierAuthorityGeneration(generation runtimeVerifierAuthorityGeneration) error {
	if strings.TrimSpace(generation.ChangedAt) == "" {
		return fmt.Errorf("runtime verifier authority state generation.changed_at is required")
	}
	if generation.Revision < 1 {
		return fmt.Errorf("runtime verifier authority state generation.revision must be >= 1")
	}
	if _, err := time.Parse(time.RFC3339, generation.ChangedAt); err != nil {
		return fmt.Errorf("runtime verifier authority state generation.changed_at is invalid")
	}
	return nil
}
