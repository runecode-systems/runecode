//go:build linux

package launcherdaemon

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestQEMUVerticalSliceHelloWorld(t *testing.T) {
	skipIfVerticalSliceUnavailable(t)

	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	workRoot := t.TempDir()
	qemuBinary := seedVerticalSliceQEMUFixture(t, workRoot)
	brokerSvc, err := brokerapi.NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	svc, err := New(Config{Controller: NewQEMUController(QEMUControllerConfig{WorkRoot: workRoot, QEMUBinary: qemuBinary}), Reporter: brokerSvc})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	runID := "run-vertical-slice"
	spec := validSpecForTests()
	spec.RunID = runID
	spec.StageID = "stage-hello"
	spec.RoleInstanceID = "role-hello"
	spec.ResourceLimits.ActiveTimeoutSeconds = 20
	seedVerticalSliceRuntimeCache(t, workRoot, qemuBinary, &spec)

	if _, err := svc.Launch(context.Background(), spec); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	waitForCompletedTerminalReport(t, brokerSvc, runID, 45*time.Second)
}

func skipIfVerticalSliceUnavailable(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("launcher hardening requires unprivileged execution")
	}
}

func seedVerticalSliceRuntimeCache(t *testing.T, workRoot string, qemuBinary string, spec *launcherbackend.BackendLaunchSpec) {
	t.Helper()
	kernelDigest := writeRuntimeCacheBlob(t, workRoot, []byte("fixture-runtime-kernel-linux-amd64-v1"))
	initrdDigest := writeRuntimeCacheBlob(t, workRoot, []byte("fixture-runtime-initrd-linux-amd64-v1"))
	spec.Image.ComponentDigests = map[string]string{"kernel": kernelDigest, "initrd": initrdDigest}
	digest, err := spec.Image.ExpectedDescriptorDigest()
	if err != nil {
		t.Fatalf("ExpectedDescriptorDigest returned error: %v", err)
	}
	spec.Image.DescriptorDigest = digest
	spec.Image.Signing.PayloadDigest = digest
	seedRuntimeImageVerificationAssets(t, workRoot, qemuBinary, spec)
}

func seedVerticalSliceQEMUFixture(t *testing.T, workRoot string) string {
	t.Helper()
	binaryPath := filepath.Join(workRoot, "fixtures", "qemu-system-x86_64")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(qemu fixture dir) returned error: %v", err)
	}
	const script = `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "QEMU emulator version fixture-vertical-slice-1.0"
  exit 0
fi
printf '%s\n' "` + helloWorldToken + `"
exit 0
`
	if err := os.WriteFile(binaryPath, []byte(script), 0o700); err != nil {
		t.Fatalf("WriteFile(qemu fixture) returned error: %v", err)
	}
	return binaryPath
}

func seedRuntimeImageVerificationAssets(t *testing.T, workRoot string, qemuBinary string, spec *launcherbackend.BackendLaunchSpec) {
	t.Helper()
	seedVerticalSliceImageVerificationAssets(t, workRoot, spec)
	seedVerticalSliceToolchainVerificationAssets(t, workRoot, qemuBinary, spec)
}

func seedVerticalSliceImageVerificationAssets(t *testing.T, workRoot string, spec *launcherbackend.BackendLaunchSpec) {
	t.Helper()
	_, privateKey, keyIDValue := runtimeImageVerifierSignerForTests()
	verifierRecord := runtimeImageVerifierRecordForTests()
	verifierBlob, err := json.Marshal([]trustpolicy.VerifierRecord{verifierRecord})
	if err != nil {
		t.Fatalf("Marshal(verifierRecord) returned error: %v", err)
	}
	verifierDigest := writeRuntimeCacheBlob(t, workRoot, verifierBlob)
	envelope, err := buildVerticalSliceSignedEnvelope(spec.Image, privateKey, keyIDValue)
	if err != nil {
		t.Fatalf("buildVerticalSliceSignedEnvelope returned error: %v", err)
	}
	envelopeBlob, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal(envelope) returned error: %v", err)
	}
	signatureDigest := writeRuntimeCacheBlob(t, workRoot, envelopeBlob)
	spec.Image.Signing.SignatureDigest = signatureDigest
	spec.Image.Signing.VerifierSetRef = verifierDigest
}

func seedVerticalSliceToolchainVerificationAssets(t *testing.T, workRoot string, qemuBinary string, spec *launcherbackend.BackendLaunchSpec) {
	t.Helper()
	if spec.Image.Signing.Toolchain == nil {
		return
	}
	_, toolchainPriv, toolchainKeyID := runtimeToolchainVerifierSignerForTests()
	toolchainVerifierBlob, err := json.Marshal([]trustpolicy.VerifierRecord{runtimeToolchainVerifierRecordForTests()})
	if err != nil {
		t.Fatalf("Marshal(toolchain verifier) returned error: %v", err)
	}
	spec.Image.Signing.Toolchain.VerifierSetRef = writeRuntimeCacheBlob(t, workRoot, toolchainVerifierBlob)
	toolchainEnvelope, descriptorDigest, err := buildVerticalSliceToolchainSignedEnvelope(qemuBinary, spec.Image.Signing.Toolchain, toolchainPriv, toolchainKeyID)
	if err != nil {
		t.Fatalf("buildVerticalSliceToolchainSignedEnvelope returned error: %v", err)
	}
	spec.Image.Signing.Toolchain.DescriptorDigest = descriptorDigest
	toolchainEnvelopeBlob, err := json.Marshal(toolchainEnvelope)
	if err != nil {
		t.Fatalf("Marshal(toolchain envelope) returned error: %v", err)
	}
	spec.Image.Signing.Toolchain.SignatureDigest = writeRuntimeCacheBlob(t, workRoot, toolchainEnvelopeBlob)
}

func buildVerticalSliceSignedEnvelope(image launcherbackend.RuntimeImageDescriptor, privateKey ed25519.PrivateKey, keyIDValue string) (trustpolicy.SignedObjectEnvelope, error) {
	payloadBytes, err := image.SignedPayloadCanonicalBytes()
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	signature := ed25519.Sign(privateKey, payloadBytes)
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      launcherbackend.RuntimeImageSignedPayloadSchemaID,
		PayloadSchemaVersion: launcherbackend.RuntimeImageSignedPayloadSchemaVersion,
		Payload:              json.RawMessage(payloadBytes),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}, nil
}

func buildVerticalSliceToolchainSignedEnvelope(qemuBinary string, toolchain *launcherbackend.RuntimeToolchainSigningHooks, privateKey ed25519.PrivateKey, keyIDValue string) (trustpolicy.SignedObjectEnvelope, string, error) {
	payload, err := buildVerticalSliceToolchainPayload(qemuBinary, toolchain)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", err
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", err
	}
	descriptorDigest := sha256Digest(canonicalPayload)
	payload["descriptor_digest"] = descriptorDigest
	payloadBytes, err = json.Marshal(payload)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", err
	}
	payloadBytes, err = jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", err
	}
	signature := ed25519.Sign(privateKey, payloadBytes)
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      launcherbackend.RuntimeToolchainDescriptorSchemaID,
		PayloadSchemaVersion: launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		Payload:              json.RawMessage(payloadBytes),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}, descriptorDigest, nil
}

func buildVerticalSliceToolchainPayload(qemuBinary string, toolchain *launcherbackend.RuntimeToolchainSigningHooks) (map[string]any, error) {
	qemuDigest, err := digestFileSHA256(qemuBinary)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"schema_id":         launcherbackend.RuntimeToolchainDescriptorSchemaID,
		"schema_version":    launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		"toolchain_family":  "qemu",
		"toolchain_version": "vertical-slice",
		"artifact_digests":  map[string]string{"qemu-system-x86_64": qemuDigest},
	}
	if toolchain != nil && strings.TrimSpace(toolchain.BundleDigest) != "" {
		payload["publication_bundle_digest"] = toolchain.BundleDigest
	}
	return payload, nil
}

func writeRuntimeCacheBlob(t *testing.T, workRoot string, data []byte) string {
	t.Helper()
	path := filepath.Join(workRoot, "blob-seed")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile(blob-seed) returned error: %v", err)
	}
	digest, err := digestFileSHA256(path)
	if err != nil {
		t.Fatalf("digestFileSHA256(blob-seed) returned error: %v", err)
	}
	parts := strings.SplitN(digest, ":", 2)
	cachePath := filepath.Join(verifiedRuntimeCacheRoot(workRoot), parts[0], parts[1])
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		t.Fatalf("MkdirAll(cache dir) returned error: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile(cache blob) returned error: %v", err)
	}
	return digest
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func waitForCompletedTerminalReport(t *testing.T, brokerSvc *brokerapi.Service, runID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		facts := brokerSvc.RuntimeFacts(runID)
		if facts.TerminalReport != nil {
			if facts.TerminalReport.TerminationKind != launcherbackend.BackendTerminationKindCompleted {
				t.Fatalf("terminal kind = %q, failure=%q", facts.TerminalReport.TerminationKind, facts.TerminalReport.FailureReasonCode)
			}
			if facts.LaunchReceipt.BackendKind != launcherbackend.BackendKindMicroVM {
				t.Fatalf("backend_kind = %q, want %q", facts.LaunchReceipt.BackendKind, launcherbackend.BackendKindMicroVM)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("timed out waiting for terminal report")
}
