//go:build linux

package launcherdaemon

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestMergeQEMURuntimePostHandshakeMaterialPreservesRuntimeProducedAttestationFields(t *testing.T) {
	merged := mergeQEMURuntimePostHandshakeMaterial(seedRuntimeAttestationMaterial(), runtimeProducedAttestationMaterial())
	assertRuntimeAttestationFields(t, merged)
}

func TestMergeQEMURuntimePostHandshakeMaterialRetainsSeedBindingWhenRuntimeOmitsIt(t *testing.T) {
	seed := seedRuntimeAttestationMaterial()
	merged := mergeQEMURuntimePostHandshakeMaterial(seed, runtimeProducedAttestationMaterial())
	assertSeedBindingRetained(t, merged, seed)
}

func TestMergeQEMURuntimePostHandshakeMaterialAccumulatesAcrossIncrementalRuntimeLines(t *testing.T) {
	secureSession := &launcherbackend.RuntimeSecureSessionMaterial{
		LaunchContext: launcherbackend.LaunchContext{RunID: "run-1", SessionID: "session-1", SessionNonce: "nonce-1", LaunchContextDigest: "sha256:launch"},
		SessionReady:  launcherbackend.SessionReady{RunID: "run-1", IsolateID: "iso-1", SessionID: "session-1", SessionNonce: "nonce-1", HandshakeTranscriptHash: "sha256:transcript", IsolateKeyIDValue: "key-id"},
	}
	first := &launcherbackend.RuntimePostHandshakeMaterial{SecureSession: secureSession}
	second := &launcherbackend.RuntimePostHandshakeMaterial{Attestation: &launcherbackend.PostHandshakeRuntimeAttestationInput{RuntimeEvidenceCollected: true, AttestationSourceKind: launcherbackend.AttestationSourceKindTrustedRuntime, MeasurementProfile: launcherbackend.MeasurementProfileMicroVMBootV1, EvidenceClaimsDigest: "sha256:evidence"}}

	merged := mergeQEMURuntimePostHandshakeMaterial(first, second)
	if merged == nil || merged.SecureSession == nil {
		t.Fatal("merged secure session missing after incremental runtime lines")
	}
	if merged.Attestation == nil {
		t.Fatal("merged attestation missing after incremental runtime lines")
	}
	if got, want := merged.Attestation.EvidenceClaimsDigest, "sha256:evidence"; got != want {
		t.Fatalf("evidence claims digest = %q, want %q", got, want)
	}
}

func seedRuntimeAttestationMaterial() *launcherbackend.RuntimePostHandshakeMaterial {
	return &launcherbackend.RuntimePostHandshakeMaterial{
		Attestation: &launcherbackend.PostHandshakeRuntimeAttestationInput{
			RunID:                        "run-1",
			IsolateID:                    "isolate-1",
			SessionID:                    "session-1",
			SessionNonce:                 "nonce-1",
			LaunchContextDigest:          "sha256:launch",
			HandshakeTranscriptHash:      "sha256:transcript",
			IsolateSessionKeyIDValue:     "key-id",
			RuntimeImageDescriptorDigest: "sha256:image",
			RuntimeImageBootProfile:      launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1,
			RuntimeEvidenceCollected:     false,
			AttestationSourceKind:        launcherbackend.AttestationSourceKindUnknown,
			MeasurementProfile:           launcherbackend.MeasurementProfileUnknown,
			EvidenceClaimsDigest:         "sha256:seed-evidence",
		},
	}
}

func runtimeProducedAttestationMaterial() *launcherbackend.RuntimePostHandshakeMaterial {
	return &launcherbackend.RuntimePostHandshakeMaterial{
		Attestation: &launcherbackend.PostHandshakeRuntimeAttestationInput{
			RuntimeEvidenceCollected: true,
			AttestationSourceKind:    launcherbackend.AttestationSourceKindTrustedRuntime,
			MeasurementProfile:       launcherbackend.MeasurementProfileMicroVMBootV1,
			FreshnessMaterial:        []string{"session_nonce"},
			FreshnessBindingClaims:   []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
			EvidenceClaimsDigest:     "sha256:runtime-evidence",
		},
	}
}

func assertRuntimeAttestationFields(t *testing.T, merged *launcherbackend.RuntimePostHandshakeMaterial) {
	t.Helper()
	if merged == nil || merged.Attestation == nil {
		t.Fatal("merged runtime post-handshake material attestation is required")
	}
	if got, want := merged.Attestation.EvidenceClaimsDigest, "sha256:runtime-evidence"; got != want {
		t.Fatalf("evidence claims digest = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.AttestationSourceKind, launcherbackend.AttestationSourceKindTrustedRuntime; got != want {
		t.Fatalf("attestation source kind = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.MeasurementProfile, launcherbackend.MeasurementProfileMicroVMBootV1; got != want {
		t.Fatalf("measurement profile = %q, want %q", got, want)
	}
	if !merged.Attestation.RuntimeEvidenceCollected {
		t.Fatal("runtime evidence collected should remain true")
	}
}

func assertSeedBindingRetained(t *testing.T, merged *launcherbackend.RuntimePostHandshakeMaterial, seed *launcherbackend.RuntimePostHandshakeMaterial) {
	t.Helper()
	if merged == nil || merged.Attestation == nil || seed == nil || seed.Attestation == nil {
		t.Fatal("merged runtime post-handshake material attestation is required")
	}
	if got, want := merged.Attestation.RunID, seed.Attestation.RunID; got != want {
		t.Fatalf("run_id = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.IsolateID, seed.Attestation.IsolateID; got != want {
		t.Fatalf("isolate_id = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.SessionID, seed.Attestation.SessionID; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.LaunchContextDigest, seed.Attestation.LaunchContextDigest; got != want {
		t.Fatalf("launch_context_digest = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.HandshakeTranscriptHash, seed.Attestation.HandshakeTranscriptHash; got != want {
		t.Fatalf("handshake_transcript_hash = %q, want %q", got, want)
	}
	if got, want := merged.Attestation.EvidenceClaimsDigest, "sha256:runtime-evidence"; got != want {
		t.Fatalf("evidence claims digest = %q, want %q", got, want)
	}
}
