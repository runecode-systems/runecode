package launcherbackend

import "testing"

func TestBuildSecureSessionSummaryIncludesIdentityAndChannelSeparation(t *testing.T) {
	ctx := validLaunchContextForContractTests()
	digest, err := ctx.CanonicalDigest()
	if err != nil {
		t.Fatalf("CanonicalDigest returned error: %v", err)
	}
	ctx.LaunchContextDigest = digest
	host := validHostHelloForContractTests(ctx)
	isolate := validIsolateHelloForContractTests(host)
	transcript, err := HandshakeTranscriptHash(host, isolate)
	if err != nil {
		t.Fatalf("HandshakeTranscriptHash returned error: %v", err)
	}
	isolate.HandshakeTranscriptHash = transcript
	ready := validSessionReadyForContractTests(host, isolate)
	ready.HandshakeTranscriptHash = transcript
	binding, err := ValidateSessionHandshake(ctx, host, isolate, ready, nil)
	if err != nil {
		t.Fatalf("ValidateSessionHandshake returned error: %v", err)
	}
	summary, err := BuildSecureSessionSummary(host, isolate, ready, binding)
	if err != nil {
		t.Fatalf("BuildSecureSessionSummary returned error: %v", err)
	}
	if summary.Identity.KeyIDValue == "" || summary.Identity.PublicKeyDigest == "" {
		t.Fatalf("identity = %#v, want populated key identity and digest", summary.Identity)
	}
	if summary.Channel.ChannelKeyMode != SessionChannelKeyModeDistinct {
		t.Fatalf("channel_key_mode = %q, want %q", summary.Channel.ChannelKeyMode, SessionChannelKeyModeDistinct)
	}
	if summary.TranscriptBinding != transcript {
		t.Fatalf("transcript_binding = %q, want %q", summary.TranscriptBinding, transcript)
	}
}
