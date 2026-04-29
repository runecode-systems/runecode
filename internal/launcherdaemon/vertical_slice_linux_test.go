//go:build linux

package launcherdaemon

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestQEMUVerticalSliceHelloWorld(t *testing.T) {
	skipIfVerticalSliceUnavailable(t)

	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	workRoot := t.TempDir()
	brokerSvc, err := brokerapi.NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	svc, err := New(Config{Controller: NewQEMUController(QEMUControllerConfig{WorkRoot: workRoot}), Reporter: brokerSvc})
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
	seedVerticalSliceRuntimeCache(t, workRoot, &spec)

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
	if _, err := os.Stat("/usr/bin/qemu-system-x86_64"); err != nil {
		t.Skip("qemu-system-x86_64 unavailable")
	}
	if _, err := os.Stat("/dev/kvm"); err != nil {
		t.Skip("/dev/kvm unavailable")
	}
	if kernels, _ := filepath.Glob("/boot/vmlinuz-*"); len(kernels) == 0 {
		t.Skip("no readable /boot/vmlinuz-* kernel")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable for initramfs build")
	}
}

func seedVerticalSliceRuntimeCache(t *testing.T, workRoot string, spec *launcherbackend.BackendLaunchSpec) {
	t.Helper()
	kernels, err := filepath.Glob("/boot/vmlinuz-*")
	if err != nil || len(kernels) == 0 {
		t.Skip("no kernel available for vertical slice cache seeding")
	}
	sort.Strings(kernels)
	kernelPath := kernels[len(kernels)-1]
	seedDir := filepath.Join(workRoot, "seed")
	if err := os.MkdirAll(seedDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(seedDir) returned error: %v", err)
	}
	initrdPath, err := buildHelloInitramfs(context.Background(), seedDir)
	if err != nil {
		t.Fatalf("buildHelloInitramfs returned error: %v", err)
	}
	kernelDigest, err := digestFileSHA256(kernelPath)
	if err != nil {
		t.Fatalf("digestFileSHA256(kernel) returned error: %v", err)
	}
	initrdDigest, err := digestFileSHA256(initrdPath)
	if err != nil {
		t.Fatalf("digestFileSHA256(initrd) returned error: %v", err)
	}
	writeRuntimeCacheAsset(t, workRoot, kernelDigest, kernelPath)
	writeRuntimeCacheAsset(t, workRoot, initrdDigest, initrdPath)
	spec.Image.ComponentDigests = map[string]string{"kernel": kernelDigest, "initrd": initrdDigest}
	digest, err := spec.Image.ExpectedDescriptorDigest()
	if err != nil {
		t.Fatalf("ExpectedDescriptorDigest returned error: %v", err)
	}
	spec.Image.DescriptorDigest = digest
	spec.Image.Signing.PayloadDigest = digest
	seedRuntimeImageVerificationAssets(t, workRoot, spec)
}

func writeRuntimeCacheAsset(t *testing.T, workRoot string, digest string, sourcePath string) {
	t.Helper()
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid digest %q", digest)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", sourcePath, err)
	}
	cachePath := filepath.Join(verifiedRuntimeCacheRoot(workRoot), parts[0], parts[1])
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		t.Fatalf("MkdirAll(cache dir) returned error: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile(cachePath) returned error: %v", err)
	}
}

func seedRuntimeImageVerificationAssets(t *testing.T, workRoot string, spec *launcherbackend.BackendLaunchSpec) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyIDValue := sha256Hex(publicKey)
	verifierRecord := buildVerticalSliceVerifierRecord(publicKey, keyIDValue)
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

func buildVerticalSliceVerifierRecord(publicKey ed25519.PublicKey, keyIDValue string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "runtime_image_signing",
		LogicalScope:           "vertical-slice",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.1.0", ActorKind: "service", PrincipalID: "vertical-slice-signer", InstanceID: "test"},
		KeyProtectionPosture:   "ephemeral_memory",
		IdentityBindingPosture: "tofu",
		PresenceMode:           "none",
		CreatedAt:              time.Now().UTC().Format(time.RFC3339),
		Status:                 "active",
	}
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
