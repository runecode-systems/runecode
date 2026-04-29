package launcherdaemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestPrepareHelloWorldRuntimeImageForLaunchSeedsAdmittableImage(t *testing.T) {
	workRoot := t.TempDir()
	image, err := PrepareHelloWorldRuntimeImageForLaunch(workRoot)
	if err != nil {
		t.Fatalf("PrepareHelloWorldRuntimeImageForLaunch returned error: %v", err)
	}
	if err := image.Validate(); err != nil {
		t.Fatalf("prepared hello-world image validation failed: %v", err)
	}
	if _, err := admitRuntimeImage(workRoot, image); err != nil {
		t.Fatalf("admitRuntimeImage returned error for prepared hello-world image: %v", err)
	}
	effective, err := loadEffectiveRuntimeVerifierAuthorityState(verifiedRuntimeCacheRoot(workRoot))
	if err != nil {
		t.Fatalf("loadEffectiveRuntimeVerifierAuthorityState returned error: %v", err)
	}
	if len(effective.AuthoritiesByKind[runtimeVerifierKindImage]) == 0 || len(effective.AuthoritiesByKind[runtimeVerifierKindToolchain]) == 0 {
		t.Fatal("expected hello-world authority import to seed both runtime-image and runtime-toolchain authorities")
	}
}

func TestPrepareHelloWorldRuntimeImageForLaunchUsesQEMUDigestWhenPresent(t *testing.T) {
	workRoot := t.TempDir()
	qemuPath := filepath.Join(workRoot, "qemu-system-x86_64")
	if err := os.WriteFile(qemuPath, []byte("deterministic-qemu-fixture"), 0o700); err != nil {
		t.Fatalf("WriteFile(qemu fixture) returned error: %v", err)
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	toolchain := &launcherbackend.RuntimeToolchainSigningHooks{
		DescriptorSchemaID:      launcherbackend.RuntimeToolchainDescriptorSchemaID,
		DescriptorSchemaVersion: launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		SignerRef:               "launcher-cli-hello-world-runtime-toolchain",
	}
	signer, err := ensureHelloWorldSignerMaterial(workRoot, runtimeVerifierKindToolchain, "runtime_toolchain_signing", "runecode-launcher-hello-world-toolchain")
	if err != nil {
		t.Fatalf("ensureHelloWorldSignerMaterial returned error: %v", err)
	}
	if err := seedHelloWorldToolchainVerificationAssets(cacheRoot, toolchain, qemuPath, signer); err != nil {
		t.Fatalf("seedHelloWorldToolchainVerificationAssets returned error: %v", err)
	}
	actualDigest, err := digestFileSHA256(qemuPath)
	if err != nil {
		t.Fatalf("digestFileSHA256(qemu fixture) returned error: %v", err)
	}
	envelopePath, err := resolveVerifiedRuntimeAsset(cacheRoot, toolchain.SignatureDigest)
	if err != nil {
		t.Fatalf("resolveVerifiedRuntimeAsset returned error: %v", err)
	}
	envelope, err := readSignedEnvelope(envelopePath)
	if err != nil {
		t.Fatalf("readSignedEnvelope returned error: %v", err)
	}
	descriptor, err := decodeVerifiedToolchainDescriptor(envelope)
	if err != nil {
		t.Fatalf("decodeVerifiedToolchainDescriptor returned error: %v", err)
	}
	if got := descriptor.ArtifactDigests["qemu-system-x86_64"]; got != actualDigest {
		t.Fatalf("toolchain qemu digest = %q, want %q", got, actualDigest)
	}
}

func TestPrepareHelloWorldRuntimeImageForLaunchDeterministicWarmCache(t *testing.T) {
	workRoot := t.TempDir()
	first, err := PrepareHelloWorldRuntimeImageForLaunch(workRoot)
	if err != nil {
		t.Fatalf("first PrepareHelloWorldRuntimeImageForLaunch returned error: %v", err)
	}
	second, err := PrepareHelloWorldRuntimeImageForLaunch(workRoot)
	if err != nil {
		t.Fatalf("second PrepareHelloWorldRuntimeImageForLaunch returned error: %v", err)
	}
	if first.DescriptorDigest != second.DescriptorDigest {
		t.Fatalf("descriptor digest mismatch: first=%q second=%q", first.DescriptorDigest, second.DescriptorDigest)
	}
	if first.Signing == nil || second.Signing == nil {
		t.Fatal("expected signing material for prepared hello-world image")
	}
	if first.Signing.VerifierSetRef != second.Signing.VerifierSetRef {
		t.Fatalf("verifier set digest mismatch: first=%q second=%q", first.Signing.VerifierSetRef, second.Signing.VerifierSetRef)
	}
	if first.Signing.SignatureDigest != second.Signing.SignatureDigest {
		t.Fatalf("signature digest mismatch: first=%q second=%q", first.Signing.SignatureDigest, second.Signing.SignatureDigest)
	}
	if first.ComponentDigests["kernel"] != second.ComponentDigests["kernel"] || first.ComponentDigests["initrd"] != second.ComponentDigests["initrd"] {
		t.Fatalf("component digest mismatch: first=%v second=%v", first.ComponentDigests, second.ComponentDigests)
	}
}

func TestPrepareHelloWorldRuntimeImageForLaunchPreservesExistingImportedAuthorities(t *testing.T) {
	workRoot := t.TempDir()
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	input := writeHelloWorldExistingAuthorityFixture(t, workRoot, importedExtendRuntimeVerifierAuthorityStateForTests(t))
	if _, err := ImportRuntimeVerifierAuthorityStateForWorkRootWithReceipt(workRoot, input); err != nil {
		t.Fatalf("ImportRuntimeVerifierAuthorityStateForWorkRootWithReceipt returned error: %v", err)
	}
	before, found, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		t.Fatalf("loadImportedRuntimeVerifierAuthorityState(before) returned error: %v", err)
	}
	if !found {
		t.Fatal("expected imported runtime verifier authority state before hello-world preparation")
	}
	if _, err := PrepareHelloWorldRuntimeImageForLaunch(workRoot); err != nil {
		t.Fatalf("PrepareHelloWorldRuntimeImageForLaunch returned error: %v", err)
	}
	after, found, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		t.Fatalf("loadImportedRuntimeVerifierAuthorityState(after) returned error: %v", err)
	}
	if !found {
		t.Fatal("expected imported runtime verifier authority state after hello-world preparation")
	}
	assertHelloWorldImportedAuthorityExtended(t, workRoot, before, after)
	assertHelloWorldSignerPrincipal(t, workRoot, runtimeVerifierKindImage, "runecode-launcher-hello-world-image")
}

func writeHelloWorldExistingAuthorityFixture(t *testing.T, workRoot string, state runtimeVerifierAuthorityState) string {
	t.Helper()
	raw, err := marshalRuntimeVerifierAuthorityState(state)
	if err != nil {
		t.Fatalf("marshalRuntimeVerifierAuthorityState returned error: %v", err)
	}
	input := filepath.Join(workRoot, "existing-authority-state.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile(existing authority state) returned error: %v", err)
	}
	return input
}

func assertHelloWorldImportedAuthorityExtended(t *testing.T, workRoot string, before runtimeVerifierAuthorityState, after runtimeVerifierAuthorityState) {
	t.Helper()
	if len(after.AuthoritiesByKind[runtimeVerifierKindImage]) <= len(before.AuthoritiesByKind[runtimeVerifierKindImage]) {
		t.Fatalf("expected hello-world image authority to extend imported state: before=%d after=%d", len(before.AuthoritiesByKind[runtimeVerifierKindImage]), len(after.AuthoritiesByKind[runtimeVerifierKindImage]))
	}
	if after.Generation.Revision <= before.Generation.Revision {
		t.Fatalf("expected hello-world authority import to advance revision: before=%d after=%d", before.Generation.Revision, after.Generation.Revision)
	}
	if _, err := os.Stat(filepath.Join(workRoot, "hello-world-runtime-authority-state.json")); !os.IsNotExist(err) {
		t.Fatalf("temporary hello-world authority import file should be removed, stat err=%v", err)
	}
}

func assertHelloWorldSignerPrincipal(t *testing.T, workRoot string, kind string, principalID string) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(workRoot, "hello-world-signers", kind+".json"))
	if err != nil {
		t.Fatalf("ReadFile(%s signer) returned error: %v", kind, err)
	}
	persisted := persistedHelloWorldSignerMaterial{}
	if err := json.Unmarshal(blob, &persisted); err != nil {
		t.Fatalf("Unmarshal(%s signer) returned error: %v", kind, err)
	}
	if persisted.Record.OwnerPrincipal.PrincipalID != principalID {
		t.Fatalf("persisted %s signer principal_id = %q, want %q", kind, persisted.Record.OwnerPrincipal.PrincipalID, principalID)
	}
}
