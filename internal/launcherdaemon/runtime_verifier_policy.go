package launcherdaemon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	runtimeVerifierKindImage     = "runtime_image"
	runtimeVerifierKindToolchain = "runtime_toolchain"

	runtimeImageVerifierPublicKeyBase64     = "0EqyMnQrtKs6E2i9RhXk5tAiSrcaAWuvhSCjMsl3hzc="
	runtimeImageVerifierKeyIDValue          = "10ba682c8ad13513971e8b56881aab8bd702bb807796eca81932c735a94d6e6d"
	runtimeToolchainVerifierPublicKeyBase64 = "oJql9HpnWYAv+VX43C0qFKXJnSO+l/hkEn/5ODRVpPA="
	runtimeToolchainVerifierKeyIDValue      = "1325b850c2871916eae203f0efc3c8987f64e5e3cdb27679e6d1fa97808357e6"
)

func loadAuthorizedRuntimeVerifierRegistry(cacheRoot string, digest string, kind string) (*trustpolicy.VerifierRegistry, error) {
	verifierSetPath, err := resolveVerifiedRuntimeAsset(cacheRoot, digest)
	if err != nil {
		return nil, fmt.Errorf("verifier set unavailable")
	}
	data, err := os.ReadFile(verifierSetPath)
	if err != nil {
		return nil, err
	}
	var records []trustpolicy.VerifierRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode verifier set: %w", err)
	}
	if err := authorizeRuntimeVerifierSet(cacheRoot, kind, strings.TrimSpace(digest), records); err != nil {
		return nil, err
	}
	registry, err := trustpolicy.NewVerifierRegistry(records)
	if err != nil {
		return nil, fmt.Errorf("load verifier set: %w", err)
	}
	return registry, nil
}

func authorizeRuntimeVerifierSet(cacheRoot string, kind string, digest string, records []trustpolicy.VerifierRecord) error {
	state, err := loadEffectiveRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		return err
	}
	entries, ok := state.AuthoritiesByKind[kind]
	if !ok || len(entries) == 0 {
		return unsupportedRuntimeVerifierPolicyKindError(kind)
	}
	matchedEntry, found := findRuntimeVerifierAuthorityEntry(entries, digest)
	if found {
		return authorizeMatchedRuntimeVerifierEntry(matchedEntry, records)
	}
	if isSupportedRuntimeVerifierKind(kind) {
		return fmt.Errorf("runtime verifier set is not authorized")
	}
	return unsupportedRuntimeVerifierPolicyKindError(kind)
}

func findRuntimeVerifierAuthorityEntry(entries []runtimeVerifierAuthorityEntry, digest string) (runtimeVerifierAuthorityEntry, bool) {
	for _, entry := range entries {
		if entry.VerifierSetRef == digest {
			return entry, true
		}
	}
	return runtimeVerifierAuthorityEntry{}, false
}

func authorizeMatchedRuntimeVerifierEntry(entry runtimeVerifierAuthorityEntry, records []trustpolicy.VerifierRecord) error {
	if entry.Status != runtimeVerifierAuthorityStatusActive {
		return fmt.Errorf("runtime verifier set is revoked")
	}
	if runtimeVerifierRecordsEqual(records, entry.Records) {
		return nil
	}
	return fmt.Errorf("runtime verifier set contents are not authorized")
}

func isSupportedRuntimeVerifierKind(kind string) bool {
	return kind == runtimeVerifierKindImage || kind == runtimeVerifierKindToolchain
}

func unsupportedRuntimeVerifierPolicyKindError(kind string) error {
	return fmt.Errorf("runtime verifier policy kind %q is unsupported", kind)
}

func builtInRuntimeVerifierPoliciesByKind() map[string][]trustpolicy.VerifierRecord {
	imageRecords := []trustpolicy.VerifierRecord{builtInRuntimeImageVerifierRecord()}
	toolchainRecords := []trustpolicy.VerifierRecord{builtInRuntimeToolchainVerifierRecord()}
	return map[string][]trustpolicy.VerifierRecord{
		runtimeVerifierKindImage:     imageRecords,
		runtimeVerifierKindToolchain: toolchainRecords,
	}
}

func builtInRuntimeImageVerifierRecord() trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:      trustpolicy.VerifierSchemaID,
		SchemaVersion: trustpolicy.VerifierSchemaVersion,
		KeyID:         trustpolicy.KeyIDProfile,
		KeyIDValue:    runtimeImageVerifierKeyIDValue,
		Alg:           "ed25519",
		PublicKey: trustpolicy.PublicKey{
			Encoding: "base64",
			Value:    runtimeImageVerifierPublicKeyBase64,
		},
		LogicalPurpose:         "runtime_image_signing",
		LogicalScope:           "publisher",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.1.0", ActorKind: "service", PrincipalID: "runecode-runtime-image-publisher", InstanceID: "builtin"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-04-29T00:00:00Z",
		Status:                 "active",
	}
}

func builtInRuntimeToolchainVerifierRecord() trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:      trustpolicy.VerifierSchemaID,
		SchemaVersion: trustpolicy.VerifierSchemaVersion,
		KeyID:         trustpolicy.KeyIDProfile,
		KeyIDValue:    runtimeToolchainVerifierKeyIDValue,
		Alg:           "ed25519",
		PublicKey: trustpolicy.PublicKey{
			Encoding: "base64",
			Value:    runtimeToolchainVerifierPublicKeyBase64,
		},
		LogicalPurpose:         "runtime_toolchain_signing",
		LogicalScope:           "publisher",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.1.0", ActorKind: "service", PrincipalID: "runecode-runtime-toolchain-publisher", InstanceID: "builtin"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-04-29T00:00:00Z",
		Status:                 "active",
	}
}

func mustRuntimeVerifierSetDigest(records []trustpolicy.VerifierRecord) string {
	data, err := json.Marshal(records)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func runtimeVerifierRecordEqual(left trustpolicy.VerifierRecord, right trustpolicy.VerifierRecord) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}

func runtimeVerifierRecordsEqual(left []trustpolicy.VerifierRecord, right []trustpolicy.VerifierRecord) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if !runtimeVerifierRecordEqual(left[idx], right[idx]) {
			return false
		}
	}
	return true
}
