package launcherdaemon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
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

var trustedRuntimeVerifierPolicies = runtimeVerifierPoliciesByKind()

type trustedRuntimeVerifierPolicy struct {
	allowedDigest string
	records       []trustpolicy.VerifierRecord
}

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
	if err := authorizeRuntimeVerifierSet(kind, strings.TrimSpace(digest), records); err != nil {
		return nil, err
	}
	registry, err := trustpolicy.NewVerifierRegistry(records)
	if err != nil {
		return nil, fmt.Errorf("load verifier set: %w", err)
	}
	return registry, nil
}

func authorizeRuntimeVerifierSet(kind string, digest string, records []trustpolicy.VerifierRecord) error {
	policy, ok := trustedRuntimeVerifierPolicies[kind]
	if !ok {
		return fmt.Errorf("runtime verifier policy kind %q is unsupported", kind)
	}
	if digest != policy.allowedDigest {
		return fmt.Errorf("runtime verifier set is not authorized")
	}
	if !slices.EqualFunc(records, policy.records, runtimeVerifierRecordEqual) {
		return fmt.Errorf("runtime verifier set contents are not authorized")
	}
	return nil
}

func runtimeVerifierPoliciesByKind() map[string]trustedRuntimeVerifierPolicy {
	imageRecords := []trustpolicy.VerifierRecord{builtInRuntimeImageVerifierRecord()}
	toolchainRecords := []trustpolicy.VerifierRecord{builtInRuntimeToolchainVerifierRecord()}
	return map[string]trustedRuntimeVerifierPolicy{
		runtimeVerifierKindImage: {
			allowedDigest: mustRuntimeVerifierSetDigest(imageRecords),
			records:       imageRecords,
		},
		runtimeVerifierKindToolchain: {
			allowedDigest: mustRuntimeVerifierSetDigest(toolchainRecords),
			records:       toolchainRecords,
		},
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
