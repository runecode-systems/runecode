package launcherbackend

import (
	"strings"
	"testing"
)

func TestLaunchContextValidateRequiresCanonicalDigest(t *testing.T) {
	ctx := validLaunchContextForContractTests()
	digest, err := ctx.CanonicalDigest()
	if err != nil {
		t.Fatalf("CanonicalDigest returned error: %v", err)
	}
	ctx.LaunchContextDigest = digest
	if err := ctx.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	ctx.PolicyDecisionRefs = append(ctx.PolicyDecisionRefs, testDigest("9"))
	if err := ctx.Validate(); err == nil {
		t.Fatal("Validate expected mismatch when launch_context_digest no longer matches canonical content")
	}
}

func TestValidateSessionHandshakeAcceptsTOFUHandshake(t *testing.T) {
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
	record, err := ValidateSessionHandshake(ctx, host, isolate, ready, nil)
	if err != nil {
		t.Fatalf("ValidateSessionHandshake returned error: %v", err)
	}
	if record.IsolateKeyIDValue != isolate.IsolateSessionKey.KeyIDValue {
		t.Fatalf("isolate key pin = %q, want %q", record.IsolateKeyIDValue, isolate.IsolateSessionKey.KeyIDValue)
	}
}

func TestValidateSessionHandshakeFailsClosedOnReplayAndIdentityChange(t *testing.T) {
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
	previous := &SessionBindingRecord{
		RunID: host.RunID, IsolateID: host.IsolateID, SessionID: host.SessionID, SessionNonce: host.SessionNonce,
		IsolateKeyIDValue: isolate.IsolateSessionKey.KeyIDValue, HandshakeTranscriptHash: transcript,
	}
	if _, err := ValidateSessionHandshake(ctx, host, isolate, ready, previous); err == nil {
		t.Fatal("ValidateSessionHandshake expected replay rejection")
	}
	previous.SessionNonce = "nonce-previous"
	previous.HandshakeTranscriptHash = testDigest("1")
	previous.IsolateKeyIDValue = strings.Repeat("f", 64)
	if _, err := ValidateSessionHandshake(ctx, host, isolate, ready, previous); err == nil {
		t.Fatal("ValidateSessionHandshake expected mid-session identity change rejection")
	}
}

func TestValidateSessionHandshakeRejectsTranscriptMismatch(t *testing.T) {
	ctx := validLaunchContextForContractTests()
	digest, err := ctx.CanonicalDigest()
	if err != nil {
		t.Fatalf("CanonicalDigest returned error: %v", err)
	}
	ctx.LaunchContextDigest = digest
	host := validHostHelloForContractTests(ctx)
	isolate := validIsolateHelloForContractTests(host)
	isolate.HandshakeTranscriptHash = testDigest("a")
	ready := validSessionReadyForContractTests(host, isolate)
	ready.HandshakeTranscriptHash = testDigest("b")
	if _, err := ValidateSessionHandshake(ctx, host, isolate, ready, nil); err == nil {
		t.Fatal("ValidateSessionHandshake expected transcript mismatch rejection")
	}
}
