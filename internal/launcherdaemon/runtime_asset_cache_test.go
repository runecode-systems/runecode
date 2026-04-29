package launcherdaemon

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestAdmitRuntimeImageVerifiesToolchainAndShapesAdmissionRecord(t *testing.T) {
	workRoot := t.TempDir()
	image := validRuntimeImageForTests()
	materializeComponentDigests(t, workRoot, &image)
	seedRuntimeImageSignatureAssets(t, workRoot, &image)
	seedRuntimeToolchainSignatureAssets(t, workRoot, &image, false)

	admitted, err := admitRuntimeImage(workRoot, image)
	if err != nil {
		t.Fatalf("admitRuntimeImage returned error: %v", err)
	}
	if err := admitted.admissionRecord.Validate(); err != nil {
		t.Fatalf("admission record validation failed: %v", err)
	}
	if admitted.admissionRecord.RuntimeToolchainDescriptorDigest == "" {
		t.Fatal("expected toolchain identity in admitted runtime record")
	}
	if len(admitted.componentPaths) != 2 {
		t.Fatalf("expected resolved component paths, got %d", len(admitted.componentPaths))
	}
}

func TestAdmitRuntimeImageRejectsUnauthorizedToolchainVerifierSet(t *testing.T) {
	workRoot := t.TempDir()
	image := validRuntimeImageForTests()
	materializeComponentDigests(t, workRoot, &image)
	seedRuntimeImageSignatureAssets(t, workRoot, &image)
	seedRuntimeToolchainSignatureAssets(t, workRoot, &image, true)

	_, err := admitRuntimeImage(workRoot, image)
	if err == nil || !strings.Contains(err.Error(), "runtime verifier set is not authorized") {
		t.Fatalf("expected unauthorized toolchain verifier rejection, got %v", err)
	}
}

func TestAdmitRuntimeImageRejectsUnauthorizedImageVerifierSet(t *testing.T) {
	workRoot := t.TempDir()
	image := validRuntimeImageForTests()
	materializeComponentDigests(t, workRoot, &image)
	seedRuntimeImageSignatureAssets(t, workRoot, &image)
	seedRuntimeToolchainSignatureAssets(t, workRoot, &image, false)
	image.Signing.VerifierSetRef = seedUnauthorizedRuntimeImageVerifierSet(t, workRoot)

	_, err := admitRuntimeImage(workRoot, image)
	if err == nil || !strings.Contains(err.Error(), "runtime verifier set is not authorized") {
		t.Fatalf("expected unauthorized image verifier rejection, got %v", err)
	}
}

func TestAdmitRuntimeImageRejectsHostPathToolchainSignerRef(t *testing.T) {
	workRoot := t.TempDir()
	image := validRuntimeImageForTests()
	image.Signing.Toolchain.SignerRef = "/var/lib/private/toolchain-signer"
	materializeComponentDigests(t, workRoot, &image)
	seedRuntimeImageSignatureAssets(t, workRoot, &image)
	seedRuntimeToolchainSignatureAssets(t, workRoot, &image, false)

	_, err := admitRuntimeImage(workRoot, image)
	if err == nil || !strings.Contains(err.Error(), "signing.toolchain.signer_ref must not include host-local path material") {
		t.Fatalf("expected host-path signer rejection, got %v", err)
	}
}

func TestAdmitRuntimeImageReusesPersistedAdmissionRecordOnWarmPath(t *testing.T) {
	workRoot := t.TempDir()
	image := validRuntimeImageForTests()
	materializeComponentDigests(t, workRoot, &image)
	seedRuntimeImageSignatureAssets(t, workRoot, &image)
	seedRuntimeToolchainSignatureAssets(t, workRoot, &image, false)

	first, err := admitRuntimeImage(workRoot, image)
	if err != nil {
		t.Fatalf("first admitRuntimeImage returned error: %v", err)
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	if _, found, err := loadRuntimeAdmissionRecord(cacheRoot, image.DescriptorDigest); err != nil || !found {
		t.Fatalf("persisted runtime admission record missing after first admit: found=%v err=%v", found, err)
	}

	second, err := admitRuntimeImage(workRoot, image)
	if err != nil {
		t.Fatalf("warm admitRuntimeImage returned error: %v", err)
	}
	if second.admissionRecord.DescriptorDigest != first.admissionRecord.DescriptorDigest {
		t.Fatalf("warm admission descriptor digest = %q, want %q", second.admissionRecord.DescriptorDigest, first.admissionRecord.DescriptorDigest)
	}
}

func TestAdmitRuntimeImageRejectsTamperedPersistedAdmissionRecord(t *testing.T) {
	workRoot := t.TempDir()
	image := validRuntimeImageForTests()
	materializeComponentDigests(t, workRoot, &image)
	seedRuntimeImageSignatureAssets(t, workRoot, &image)
	seedRuntimeToolchainSignatureAssets(t, workRoot, &image, false)

	admitted, err := admitRuntimeImage(workRoot, image)
	if err != nil {
		t.Fatalf("first admitRuntimeImage returned error: %v", err)
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	path, err := runtimeAdmissionRecordPath(cacheRoot, image.DescriptorDigest)
	if err != nil {
		t.Fatalf("runtimeAdmissionRecordPath returned error: %v", err)
	}
	tampered := admitted.admissionRecord
	tampered.RuntimeImageVerifierSetRef = "sha256:" + repeatHex('f')
	raw, err := json.Marshal(tampered)
	if err != nil {
		t.Fatalf("Marshal(tampered) returned error: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("WriteFile(path) returned error: %v", err)
	}
	_, err = admitRuntimeImage(workRoot, image)
	if err == nil || !strings.Contains(err.Error(), "persisted runtime admission record does not match image identity") {
		t.Fatalf("expected persisted admission mismatch, got %v", err)
	}
}

func materializeComponentDigests(t *testing.T, workRoot string, image *launcherbackend.RuntimeImageDescriptor) {
	t.Helper()
	componentDigests := map[string]string{}
	for name := range image.ComponentDigests {
		payload := []byte("component:" + name)
		digest := sha256Digest(payload)
		writeDigestAddressedAsset(t, workRoot, digest, payload)
		componentDigests[name] = digest
	}
	image.ComponentDigests = componentDigests
	descriptorDigest, err := image.ExpectedDescriptorDigest()
	if err != nil {
		t.Fatalf("ExpectedDescriptorDigest returned error: %v", err)
	}
	image.DescriptorDigest = descriptorDigest
	image.Signing.PayloadDigest = descriptorDigest
}

func seedRuntimeImageSignatureAssets(t *testing.T, workRoot string, image *launcherbackend.RuntimeImageDescriptor) {
	t.Helper()
	_, privateKey, keyIDValue := runtimeImageVerifierSignerForTests()
	payloadBytes, err := image.SignedPayloadCanonicalBytes()
	if err != nil {
		t.Fatalf("SignedPayloadCanonicalBytes returned error: %v", err)
	}
	envelope := trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      launcherbackend.RuntimeImageSignedPayloadSchemaID,
		PayloadSchemaVersion: launcherbackend.RuntimeImageSignedPayloadSchemaVersion,
		Payload:              json.RawMessage(payloadBytes),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payloadBytes))},
	}
	verifier := []trustpolicy.VerifierRecord{runtimeImageVerifierRecordForTests()}
	verifierBlob, err := json.Marshal(verifier)
	if err != nil {
		t.Fatalf("Marshal(verifier) returned error: %v", err)
	}
	verifierDigest := writeCacheBlob(t, workRoot, verifierBlob)
	envelopeBlob, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal(envelope) returned error: %v", err)
	}
	signatureDigest := writeCacheBlob(t, workRoot, envelopeBlob)
	image.Signing.VerifierSetRef = verifierDigest
	image.Signing.SignatureDigest = signatureDigest
}

func seedRuntimeToolchainSignatureAssets(t *testing.T, workRoot string, image *launcherbackend.RuntimeImageDescriptor, useWrongVerifier bool) {
	t.Helper()
	canonicalPayload, digest := buildToolchainCanonicalPayloadForTests(t, image.Signing.Toolchain)

	publicKey, privateKey, keyIDValue := runtimeToolchainVerifierSignerForTests()
	envelope := trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      launcherbackend.RuntimeToolchainDescriptorSchemaID,
		PayloadSchemaVersion: launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		Payload:              json.RawMessage(canonicalPayload),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, canonicalPayload))},
	}
	envelopeBlob, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal(toolchain envelope) returned error: %v", err)
	}
	image.Signing.Toolchain.SignatureDigest = writeCacheBlob(t, workRoot, envelopeBlob)
	image.Signing.Toolchain.DescriptorDigest = digest
	image.Signing.Toolchain.VerifierSetRef = writeToolchainVerifierSet(t, workRoot, publicKey, keyIDValue, useWrongVerifier)
}

func buildToolchainCanonicalPayloadForTests(t *testing.T, toolchain *launcherbackend.RuntimeToolchainSigningHooks) ([]byte, string) {
	t.Helper()
	toolchainPayload := map[string]any{
		"schema_id":         launcherbackend.RuntimeToolchainDescriptorSchemaID,
		"schema_version":    launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		"toolchain_family":  "qemu",
		"toolchain_version": "8.2.2",
		"artifact_digests":  map[string]string{"qemu-system-x86_64": "sha256:" + repeatHex('1')},
	}
	if toolchain != nil && toolchain.BundleDigest != "" {
		toolchainPayload["publication_bundle_digest"] = toolchain.BundleDigest
	}
	payloadJSON, err := json.Marshal(toolchainPayload)
	if err != nil {
		t.Fatalf("Marshal(toolchain payload) returned error: %v", err)
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadJSON)
	if err != nil {
		t.Fatalf("canonicalize(toolchain payload) returned error: %v", err)
	}
	digest := sha256Digest(canonicalPayload)
	toolchainPayload["descriptor_digest"] = digest
	payloadJSON, err = json.Marshal(toolchainPayload)
	if err != nil {
		t.Fatalf("Marshal(toolchain payload with descriptor) returned error: %v", err)
	}
	canonicalPayload, err = jsoncanonicalizer.Transform(payloadJSON)
	if err != nil {
		t.Fatalf("canonicalize(toolchain payload with descriptor) returned error: %v", err)
	}
	return canonicalPayload, digest
}

func writeToolchainVerifierSet(t *testing.T, workRoot string, expectedKey ed25519.PublicKey, expectedKeyID string, wrong bool) string {
	t.Helper()
	pub := expectedKey
	keyID := expectedKeyID
	record := runtimeToolchainVerifierRecordForTests()
	if wrong {
		var err error
		pub, _, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("GenerateKey returned error: %v", err)
		}
		keyID = sha256HexString(pub)
		record = buildVerifierRecord(pub, keyID)
	}
	blob, err := json.Marshal([]trustpolicy.VerifierRecord{record})
	if err != nil {
		t.Fatalf("Marshal(toolchain verifier set) returned error: %v", err)
	}
	return writeCacheBlob(t, workRoot, blob)
}

func buildVerifierRecord(publicKey ed25519.PublicKey, keyIDValue string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
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

func seedUnauthorizedRuntimeImageVerifierSet(t *testing.T, workRoot string) string {
	t.Helper()
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyIDValue := sha256HexString(publicKey)
	record := trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "runtime_image_signing",
		LogicalScope:           "publisher",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.1.0", ActorKind: "service", PrincipalID: "runecode-runtime-image-publisher", InstanceID: "builtin"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-04-29T00:00:00Z",
		Status:                 "active",
	}
	blob, err := json.Marshal([]trustpolicy.VerifierRecord{record})
	if err != nil {
		t.Fatalf("Marshal(unauthorized image verifier) returned error: %v", err)
	}
	return writeCacheBlob(t, workRoot, blob)
}

func writeCacheBlob(t *testing.T, workRoot string, data []byte) string {
	t.Helper()
	digest := sha256Digest(data)
	writeDigestAddressedAsset(t, workRoot, digest, data)
	return digest
}

func writeDigestAddressedAsset(t *testing.T, workRoot string, digest string, data []byte) {
	t.Helper()
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid digest %q", digest)
	}
	path := filepath.Join(verifiedRuntimeCacheRoot(workRoot), parts[0], parts[1])
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func removeDigestAddressedAsset(t *testing.T, cacheRoot string, digest string) {
	t.Helper()
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid digest %q", digest)
	}
	path := filepath.Join(cacheRoot, parts[0], parts[1])
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove(%s) returned error: %v", path, err)
	}
}

func sha256HexString(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
