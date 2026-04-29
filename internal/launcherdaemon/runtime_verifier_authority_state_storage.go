package launcherdaemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func ImportRuntimeVerifierAuthorityState(cacheRoot string, sourcePath string) error {
	_, err := ImportRuntimeVerifierAuthorityStateWithReceipt(cacheRoot, sourcePath)
	return err
}

func ImportRuntimeVerifierAuthorityStateWithReceipt(cacheRoot string, sourcePath string) (RuntimeVerifierAuthorityStateImportReceipt, error) {
	context, err := loadRuntimeVerifierAuthorityImportContext(cacheRoot)
	if err != nil {
		return RuntimeVerifierAuthorityStateImportReceipt{}, err
	}
	state, err := loadRuntimeVerifierAuthorityStateFromPath(sourcePath)
	if err != nil {
		return RuntimeVerifierAuthorityStateImportReceipt{}, err
	}
	if err := validateRuntimeVerifierAuthorityStateProgression(state, context.previousEffective, context.previousImported, context.foundImported); err != nil {
		return RuntimeVerifierAuthorityStateImportReceipt{}, err
	}
	effective, err := runtimeVerifierAuthorityEffectiveStateForImport(state)
	if err != nil {
		return RuntimeVerifierAuthorityStateImportReceipt{}, err
	}
	receipt := newRuntimeVerifierAuthorityImportReceipt(state, effective, context.previousEffective)
	if err := persistImportedRuntimeVerifierAuthorityState(cacheRoot, state); err != nil {
		return RuntimeVerifierAuthorityStateImportReceipt{}, err
	}
	if err := persistRuntimeVerifierAuthorityImportReceipt(cacheRoot, receipt); err != nil {
		return receipt, fmt.Errorf("persist runtime verifier authority import receipt: %w", err)
	}
	return receipt, nil
}

type runtimeVerifierAuthorityImportContext struct {
	previousEffective runtimeVerifierAuthorityState
	previousImported  runtimeVerifierAuthorityState
	foundImported     bool
}

func loadRuntimeVerifierAuthorityImportContext(cacheRoot string) (runtimeVerifierAuthorityImportContext, error) {
	previousEffective, _, err := loadCurrentEffectiveRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		return runtimeVerifierAuthorityImportContext{}, err
	}
	previousImported, foundImported, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		return runtimeVerifierAuthorityImportContext{}, err
	}
	return runtimeVerifierAuthorityImportContext{
		previousEffective: previousEffective,
		previousImported:  previousImported,
		foundImported:     foundImported,
	}, nil
}

func loadRuntimeVerifierAuthorityStateFromPath(path string) (runtimeVerifierAuthorityState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runtimeVerifierAuthorityState{}, err
	}
	return decodeAndValidateRuntimeVerifierAuthorityState(data)
}

func decodeAndValidateRuntimeVerifierAuthorityState(data []byte) (runtimeVerifierAuthorityState, error) {
	state, err := decodeRuntimeVerifierAuthorityState(data)
	if err != nil {
		return runtimeVerifierAuthorityState{}, err
	}
	importedDigest := strings.TrimSpace(state.StateDigest)
	state = normalizeRuntimeVerifierAuthorityState(state)
	digestValidationState := state
	digestValidationState.StateDigest = importedDigest
	if err := validateRuntimeVerifierAuthorityStateDigest(digestValidationState); err != nil {
		return runtimeVerifierAuthorityState{}, err
	}
	if err := validateRuntimeVerifierAuthorityState(state); err != nil {
		return runtimeVerifierAuthorityState{}, err
	}
	return state, nil
}

func runtimeVerifierAuthorityEffectiveStateForImport(imported runtimeVerifierAuthorityState) (runtimeVerifierAuthorityState, error) {
	return mergeRuntimeVerifierAuthorityState(builtInRuntimeVerifierAuthorityState(), imported)
}

func newRuntimeVerifierAuthorityImportReceipt(imported runtimeVerifierAuthorityState, effective runtimeVerifierAuthorityState, previousEffective runtimeVerifierAuthorityState) RuntimeVerifierAuthorityStateImportReceipt {
	receipt := RuntimeVerifierAuthorityStateImportReceipt{
		SchemaID:             runtimeVerifierAuthorityReceiptSchemaID,
		SchemaVersion:        runtimeVerifierAuthorityReceiptSchemaVersion,
		Action:               "import",
		ChangedAt:            time.Now().UTC().Format(time.RFC3339),
		WorkRootScope:        "runtime-cache",
		ImportedStateDigest:  imported.StateDigest,
		EffectiveStateDigest: effective.StateDigest,
		ImportedGeneration:   imported.Generation,
		EffectiveGeneration:  effective.Generation,
	}
	if previousEffective.StateDigest != "" {
		receipt.PreviousEffectiveDigest = previousEffective.StateDigest
	}
	return receipt
}

func ImportRuntimeVerifierAuthorityStateForWorkRoot(workRoot string, sourcePath string) error {
	return ImportRuntimeVerifierAuthorityState(verifiedRuntimeCacheRoot(workRoot), sourcePath)
}

func ImportRuntimeVerifierAuthorityStateForWorkRootWithReceipt(workRoot string, sourcePath string) (RuntimeVerifierAuthorityStateImportReceipt, error) {
	return ImportRuntimeVerifierAuthorityStateWithReceipt(verifiedRuntimeCacheRoot(workRoot), sourcePath)
}

func ExportEffectiveRuntimeVerifierAuthorityStateForWorkRoot(workRoot string) ([]byte, error) {
	state, err := loadEffectiveRuntimeVerifierAuthorityState(verifiedRuntimeCacheRoot(workRoot))
	if err != nil {
		return nil, err
	}
	return marshalRuntimeVerifierAuthorityState(state)
}

func loadImportedRuntimeVerifierAuthorityState(cacheRoot string) (runtimeVerifierAuthorityState, bool, error) {
	path := runtimeVerifierAuthorityStatePath(cacheRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return runtimeVerifierAuthorityState{}, false, nil
		}
		return runtimeVerifierAuthorityState{}, false, err
	}
	state, err := decodeAndValidateRuntimeVerifierAuthorityState(data)
	if err != nil {
		return runtimeVerifierAuthorityState{}, false, err
	}
	return state, true, nil
}

func decodeRuntimeVerifierAuthorityState(data []byte) (runtimeVerifierAuthorityState, error) {
	state := runtimeVerifierAuthorityState{}
	if err := json.Unmarshal(data, &state); err != nil {
		return runtimeVerifierAuthorityState{}, fmt.Errorf("decode runtime verifier authority state: %w", err)
	}
	return state, nil
}

func persistImportedRuntimeVerifierAuthorityState(cacheRoot string, state runtimeVerifierAuthorityState) error {
	state = normalizeRuntimeVerifierAuthorityState(state)
	canonical, err := marshalRuntimeVerifierAuthorityState(state)
	if err != nil {
		return err
	}
	path := runtimeVerifierAuthorityStatePath(cacheRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeRuntimeStateFile(path, runtimeVerifierAuthorityStateFileName+".*.tmp", canonical)
}

func persistRuntimeVerifierAuthorityImportReceipt(cacheRoot string, receipt RuntimeVerifierAuthorityStateImportReceipt) error {
	b, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	path := filepath.Join(cacheRoot, runtimeVerifierAuthorityStateDirName, runtimeVerifierAuthorityReceiptFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeRuntimeStateFile(path, runtimeVerifierAuthorityReceiptFileName+".*.tmp", canonical)
}

func marshalRuntimeVerifierAuthorityState(state runtimeVerifierAuthorityState) ([]byte, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

func runtimeVerifierAuthorityStatePath(cacheRoot string) string {
	return filepath.Join(cacheRoot, runtimeVerifierAuthorityStateDirName, runtimeVerifierAuthorityStateFileName)
}

func mergeRuntimeVerifierAuthorityState(builtin runtimeVerifierAuthorityState, imported runtimeVerifierAuthorityState) (runtimeVerifierAuthorityState, error) {
	if strings.TrimSpace(imported.MergeMode) == runtimeVerifierAuthorityMergeModeReplace {
		return imported, nil
	}
	result := imported
	result.AuthoritiesByKind = mergeRuntimeVerifierAuthorityEntriesByKind(builtin.AuthoritiesByKind, imported.AuthoritiesByKind)
	result = normalizeRuntimeVerifierAuthorityState(result)
	if err := validateRuntimeVerifierAuthorityState(result); err != nil {
		return runtimeVerifierAuthorityState{}, err
	}
	return result, nil
}

func mergeRuntimeVerifierAuthorityEntriesByKind(builtin map[string][]runtimeVerifierAuthorityEntry, imported map[string][]runtimeVerifierAuthorityEntry) map[string][]runtimeVerifierAuthorityEntry {
	merged := map[string][]runtimeVerifierAuthorityEntry{}
	for kind, entries := range builtin {
		merged[kind] = append([]runtimeVerifierAuthorityEntry{}, entries...)
	}
	for kind, entries := range imported {
		merged[kind] = mergeRuntimeVerifierAuthorityEntries(merged[kind], entries)
	}
	return merged
}

func mergeRuntimeVerifierAuthorityEntries(base []runtimeVerifierAuthorityEntry, incoming []runtimeVerifierAuthorityEntry) []runtimeVerifierAuthorityEntry {
	byRef := map[string]runtimeVerifierAuthorityEntry{}
	for _, entry := range base {
		byRef[entry.VerifierSetRef] = entry
	}
	for _, entry := range incoming {
		byRef[entry.VerifierSetRef] = entry
	}
	mergedEntries := make([]runtimeVerifierAuthorityEntry, 0, len(byRef))
	for _, entry := range byRef {
		mergedEntries = append(mergedEntries, entry)
	}
	return mergedEntries
}
