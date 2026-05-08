package launcherbackend

import "testing"

func TestNormalizePostHandshakeRuntimeAttestationInputPopulatesBootDigestsFromNamedIdentity(t *testing.T) {
	input := &PostHandshakeRuntimeAttestationInput{
		RunID:                        "run-1",
		IsolateID:                    "isolate-1",
		SessionID:                    "session-1",
		SessionNonce:                 "nonce-1",
		LaunchContextDigest:          testDigest("1"),
		HandshakeTranscriptHash:      testDigest("2"),
		IsolateSessionKeyIDValue:     testDigest("3")[7:],
		RuntimeImageDescriptorDigest: testDigest("4"),
		RuntimeImageBootProfile:      BootProfileMicroVMLinuxKernelInitrdV1,
		RuntimeImageVerifierRef:      testDigest("7"),
		AuthorityStateDigest:         testDigest("8"),
		BootComponentDigestByName:    map[string]string{"kernel": testDigest("5"), "initrd": testDigest("6")},
		AttestationSourceKind:        AttestationSourceKindTrustedRuntime,
		MeasurementProfile:           MeasurementProfileMicroVMBootV1,
	}

	normalized := NormalizePostHandshakeRuntimeAttestationInput(input)
	if normalized == nil {
		t.Fatal("normalized input should not be nil")
	}
	if len(normalized.BootComponentDigests) != 2 {
		t.Fatalf("boot_component_digests length = %d, want 2", len(normalized.BootComponentDigests))
	}
}
