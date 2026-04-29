package launcherdaemon

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func ensureHelloWorldAuthorityState(workRoot string) (helloWorldSignerMaterial, helloWorldSignerMaterial, error) {
	imageSigner, toolchainSigner, err := loadHelloWorldSignerPair(workRoot)
	if err != nil {
		return helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	if err := seedHelloWorldVerifierSetAsset(cacheRoot, imageSigner); err != nil {
		return helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	if err := seedHelloWorldVerifierSetAsset(cacheRoot, toolchainSigner); err != nil {
		return helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	if err := importHelloWorldAuthorityState(workRoot, cacheRoot, imageSigner, toolchainSigner); err != nil {
		return helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	return imageSigner, toolchainSigner, nil
}

func loadHelloWorldSignerPair(workRoot string) (helloWorldSignerMaterial, helloWorldSignerMaterial, error) {
	imageSigner, err := ensureHelloWorldSignerMaterial(workRoot, runtimeVerifierKindImage, "runtime_image_signing", "runecode-launcher-hello-world-image")
	if err != nil {
		return helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	toolchainSigner, err := ensureHelloWorldSignerMaterial(workRoot, runtimeVerifierKindToolchain, "runtime_toolchain_signing", "runecode-launcher-hello-world-toolchain")
	if err != nil {
		return helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	return imageSigner, toolchainSigner, nil
}

func importHelloWorldAuthorityState(workRoot string, cacheRoot string, imageSigner helloWorldSignerMaterial, toolchainSigner helloWorldSignerMaterial) error {
	state, foundImported, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		return err
	}
	if foundImported && runtimeVerifierAuthorityStateContainsHelloWorldSigners(state, imageSigner, toolchainSigner) {
		return nil
	}
	state = nextHelloWorldAuthorityImportState(state, foundImported, imageSigner, toolchainSigner)
	path, err := writeHelloWorldAuthorityImportFile(workRoot, state)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(path) }()
	_, err = ImportRuntimeVerifierAuthorityStateForWorkRootWithReceipt(workRoot, path)
	return err
}

func writeHelloWorldAuthorityImportFile(workRoot string, state runtimeVerifierAuthorityState) (string, error) {
	path := filepath.Join(workRoot, "hello-world-runtime-authority-state.json")
	raw, err := marshalRuntimeVerifierAuthorityState(state)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func runtimeVerifierAuthorityStateContainsHelloWorldSigners(state runtimeVerifierAuthorityState, imageSigner helloWorldSignerMaterial, toolchainSigner helloWorldSignerMaterial) bool {
	return helloWorldAuthorityEntryPresent(state.AuthoritiesByKind[runtimeVerifierKindImage], imageSigner) &&
		helloWorldAuthorityEntryPresent(state.AuthoritiesByKind[runtimeVerifierKindToolchain], toolchainSigner)
}

func helloWorldAuthorityEntryPresent(entries []runtimeVerifierAuthorityEntry, signer helloWorldSignerMaterial) bool {
	for _, entry := range entries {
		if entry.VerifierSetRef != signer.verifierSetRef {
			continue
		}
		if runtimeVerifierRecordsEqual(entry.Records, []trustpolicy.VerifierRecord{signer.record}) {
			return true
		}
	}
	return false
}

func nextHelloWorldAuthorityImportState(previous runtimeVerifierAuthorityState, foundImported bool, imageSigner helloWorldSignerMaterial, toolchainSigner helloWorldSignerMaterial) runtimeVerifierAuthorityState {
	state := baseHelloWorldAuthorityImportState(previous, foundImported)
	appendHelloWorldAuthorityEntry(&state, runtimeVerifierKindImage, imageSigner)
	appendHelloWorldAuthorityEntry(&state, runtimeVerifierKindToolchain, toolchainSigner)
	return normalizeRuntimeVerifierAuthorityState(state)
}

func baseHelloWorldAuthorityImportState(previous runtimeVerifierAuthorityState, foundImported bool) runtimeVerifierAuthorityState {
	if !foundImported {
		return runtimeVerifierAuthorityState{
			SchemaID:          runtimeVerifierAuthorityStateSchemaID,
			SchemaVersion:     runtimeVerifierAuthorityStateSchemaVersion,
			Generation:        runtimeVerifierAuthorityGeneration{Revision: 2, PreviousRevision: 1, ChangedAt: helloWorldAuthorityChangedAt, Reason: helloWorldAuthorityReason},
			MergeMode:         runtimeVerifierAuthorityMergeModeExtend,
			AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{},
		}
	}
	state := previous
	if state.AuthoritiesByKind == nil {
		state.AuthoritiesByKind = map[string][]runtimeVerifierAuthorityEntry{}
	}
	if strings.TrimSpace(state.SchemaID) == "" {
		state.SchemaID = runtimeVerifierAuthorityStateSchemaID
	}
	if strings.TrimSpace(state.SchemaVersion) == "" {
		state.SchemaVersion = runtimeVerifierAuthorityStateSchemaVersion
	}
	if strings.TrimSpace(state.MergeMode) == "" {
		state.MergeMode = runtimeVerifierAuthorityMergeModeExtend
	}
	state.Generation = runtimeVerifierAuthorityGeneration{
		Revision:         state.Generation.Revision + 1,
		PreviousRevision: state.Generation.Revision,
		ChangedAt:        helloWorldAuthorityChangedAt,
		Reason:           helloWorldAuthorityReason,
	}
	return state
}

func appendHelloWorldAuthorityEntry(state *runtimeVerifierAuthorityState, kind string, signer helloWorldSignerMaterial) {
	state.AuthoritiesByKind[kind] = mergeRuntimeVerifierAuthorityEntries(state.AuthoritiesByKind[kind], []runtimeVerifierAuthorityEntry{authorityEntryForHelloWorldSigner(signer)})
}

func ensureHelloWorldSignerMaterial(workRoot string, kind string, purpose string, principalID string) (helloWorldSignerMaterial, error) {
	path := filepath.Join(workRoot, "hello-world-signers", kind+".json")
	if existing, err := loadHelloWorldSignerMaterial(path); err == nil {
		return existing, nil
	} else if !os.IsNotExist(err) {
		return helloWorldSignerMaterial{}, err
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return helloWorldSignerMaterial{}, err
	}
	record := verifierRecordForHelloWorldSigner(publicKey, purpose, principalID)
	material := helloWorldSignerMaterial{record: record, privateKey: privateKey, verifierSetRef: mustRuntimeVerifierSetDigest([]trustpolicy.VerifierRecord{record})}
	if err := persistHelloWorldSignerMaterial(path, material); err != nil {
		return helloWorldSignerMaterial{}, err
	}
	return material, nil
}

func seedHelloWorldVerifierSetAsset(cacheRoot string, material helloWorldSignerMaterial) error {
	blob, err := json.Marshal([]trustpolicy.VerifierRecord{material.record})
	if err != nil {
		return err
	}
	digest, err := seedHelloWorldRuntimeAsset(cacheRoot, blob)
	if err != nil {
		return err
	}
	if digest != material.verifierSetRef {
		return fmt.Errorf("hello-world verifier set digest mismatch")
	}
	return nil
}

type persistedHelloWorldSignerMaterial struct {
	Record         trustpolicy.VerifierRecord `json:"record"`
	PrivateKeySeed string                     `json:"private_key_seed"`
	VerifierSetRef string                     `json:"verifier_set_ref"`
}

func loadHelloWorldSignerMaterial(path string) (helloWorldSignerMaterial, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return helloWorldSignerMaterial{}, err
	}
	persisted := persistedHelloWorldSignerMaterial{}
	if err := json.Unmarshal(raw, &persisted); err != nil {
		return helloWorldSignerMaterial{}, err
	}
	seed, err := hex.DecodeString(strings.TrimSpace(persisted.PrivateKeySeed))
	if err != nil {
		return helloWorldSignerMaterial{}, err
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if sha256HexStringHelloWorld(publicKey) != persisted.Record.KeyIDValue {
		return helloWorldSignerMaterial{}, fmt.Errorf("hello-world signer key does not match persisted record")
	}
	return helloWorldSignerMaterial{record: persisted.Record, privateKey: privateKey, verifierSetRef: persisted.VerifierSetRef}, nil
}

func persistHelloWorldSignerMaterial(path string, material helloWorldSignerMaterial) error {
	persisted := persistedHelloWorldSignerMaterial{
		Record:         material.record,
		PrivateKeySeed: hex.EncodeToString(material.privateKey.Seed()),
		VerifierSetRef: material.verifierSetRef,
	}
	raw, err := json.Marshal(persisted)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeRuntimeStateFile(path, filepath.Base(path)+".*.tmp", raw)
}

func sha256HexStringHelloWorld(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
