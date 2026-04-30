package launcherdaemon

import (
	"reflect"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestDeriveRuntimeSessionBindingDeterministicForStableInputs(t *testing.T) {
	spec := validSpecForTests()

	first, err := deriveRuntimeSessionBinding(spec, spec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	if err != nil {
		t.Fatalf("deriveRuntimeSessionBinding(first) returned error: %v", err)
	}
	second, err := deriveRuntimeSessionBinding(spec, spec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	if err != nil {
		t.Fatalf("deriveRuntimeSessionBinding(second) returned error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("deriveRuntimeSessionBinding should be deterministic for stable inputs; first=%+v second=%+v", first, second)
	}
}

func TestDeriveRuntimeSessionBindingDomainSeparatedOutputs(t *testing.T) {
	spec := validSpecForTests()
	base := mustDeriveRuntimeSessionBinding(t, spec, spec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	if base.LaunchContextDigest == base.HandshakeTranscriptHash {
		t.Fatalf("domain separation failure: launch context digest equals handshake transcript hash (%q)", base.LaunchContextDigest)
	}
	assertSessionBindingFieldRelationships(t, base, mustDeriveRuntimeSessionBinding(t, spec, "sha256:"+repeatHex('f'), "isolate-1", "session-1", strings.Repeat("a", 32)), false, true, true)
	assertSessionBindingFieldRelationships(t, base, mustDeriveRuntimeSessionBinding(t, spec, spec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("b", 32)), true, true, true)
	assertSessionBindingFieldRelationships(t, base, mustDeriveRuntimeSessionBinding(t, spec, spec.Image.DescriptorDigest, "isolate-2", "session-1", strings.Repeat("a", 32)), true, true, true)
	assertSessionBindingFieldRelationships(t, base, mustDeriveRuntimeSessionBinding(t, spec, spec.Image.DescriptorDigest, "isolate-1", "session-2", strings.Repeat("a", 32)), true, true, true)
}

func TestDeriveRuntimeSessionBindingUsesUnambiguousHashInputEncoding(t *testing.T) {
	firstSpec := validSpecForTests()
	firstSpec.RunID = "run"
	firstSpec.StageID = "stage|alpha"
	firstSpec.RoleInstanceID = "role"

	secondSpec := validSpecForTests()
	secondSpec.RunID = "run|stage"
	secondSpec.StageID = "alpha"
	secondSpec.RoleInstanceID = "role"

	first := mustDeriveRuntimeSessionBinding(t, firstSpec, firstSpec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	second := mustDeriveRuntimeSessionBinding(t, secondSpec, secondSpec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	if first.LaunchContextDigest == second.LaunchContextDigest {
		t.Fatal("launch context digest should remain distinct for delimiter-ambiguous inputs")
	}
	if first.HandshakeTranscriptHash == second.HandshakeTranscriptHash {
		t.Fatal("handshake transcript hash should remain distinct for delimiter-ambiguous inputs")
	}
	if first.IsolateSessionKeyIDValue == second.IsolateSessionKeyIDValue {
		t.Fatal("session key id should remain distinct for delimiter-ambiguous inputs")
	}
}

func TestDeriveRuntimeSessionBindingFailsClosedOnMissingRequiredInputs(t *testing.T) {
	base := validSpecForTests()
	for _, tc := range missingRuntimeSessionBindingInputTests(base) {
		t.Run(tc.name, func(t *testing.T) {
			_, err := deriveRuntimeSessionBinding(tc.spec, tc.descriptor, tc.isolateID, tc.sessionID, tc.nonce)
			if err == nil {
				t.Fatal("deriveRuntimeSessionBinding expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("deriveRuntimeSessionBinding error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func mustDeriveRuntimeSessionBinding(t *testing.T, spec launcherbackend.BackendLaunchSpec, descriptorDigest, isolateID, sessionID, nonce string) runtimeSessionBinding {
	t.Helper()
	binding, err := deriveRuntimeSessionBinding(spec, descriptorDigest, isolateID, sessionID, nonce)
	if err != nil {
		t.Fatalf("deriveRuntimeSessionBinding returned error: %v", err)
	}
	return binding
}

func assertSessionBindingFieldRelationships(t *testing.T, base, changed runtimeSessionBinding, wantLaunchContextChange, wantTranscriptChange, wantKeyIDChange bool) {
	t.Helper()
	assertBindingFieldChange(t, "launch context digest", base.LaunchContextDigest, changed.LaunchContextDigest, wantLaunchContextChange)
	assertBindingFieldChange(t, "handshake transcript hash", base.HandshakeTranscriptHash, changed.HandshakeTranscriptHash, wantTranscriptChange)
	assertBindingFieldChange(t, "session key id", base.IsolateSessionKeyIDValue, changed.IsolateSessionKeyIDValue, wantKeyIDChange)
}

func assertBindingFieldChange(t *testing.T, fieldName, base, changed string, wantChange bool) {
	t.Helper()
	if wantChange && changed == base {
		t.Fatalf("%s should change", fieldName)
	}
	if !wantChange && changed != base {
		t.Fatalf("%s should not change: got %q want %q", fieldName, changed, base)
	}
}

func missingRuntimeSessionBindingInputTests(base launcherbackend.BackendLaunchSpec) []struct {
	name       string
	spec       launcherbackend.BackendLaunchSpec
	descriptor string
	isolateID  string
	sessionID  string
	nonce      string
	wantErr    string
} {
	return []struct {
		name       string
		spec       launcherbackend.BackendLaunchSpec
		descriptor string
		isolateID  string
		sessionID  string
		nonce      string
		wantErr    string
	}{
		{name: "missing run id", spec: withRunID(base, ""), descriptor: base.Image.DescriptorDigest, isolateID: "isolate-1", sessionID: "session-1", nonce: strings.Repeat("a", 32), wantErr: "requires run id"},
		{name: "missing stage id", spec: withStageID(base, ""), descriptor: base.Image.DescriptorDigest, isolateID: "isolate-1", sessionID: "session-1", nonce: strings.Repeat("a", 32), wantErr: "requires stage id"},
		{name: "missing role instance id", spec: withRoleInstanceID(base, ""), descriptor: base.Image.DescriptorDigest, isolateID: "isolate-1", sessionID: "session-1", nonce: strings.Repeat("a", 32), wantErr: "requires role instance id"},
		{name: "missing descriptor digest", spec: base, descriptor: "", isolateID: "isolate-1", sessionID: "session-1", nonce: strings.Repeat("a", 32), wantErr: "requires runtime image descriptor digest"},
		{name: "missing isolate id", spec: base, descriptor: base.Image.DescriptorDigest, isolateID: "", sessionID: "session-1", nonce: strings.Repeat("a", 32), wantErr: "requires isolate id"},
		{name: "missing session id", spec: base, descriptor: base.Image.DescriptorDigest, isolateID: "isolate-1", sessionID: "", nonce: strings.Repeat("a", 32), wantErr: "requires session id"},
		{name: "missing nonce", spec: base, descriptor: base.Image.DescriptorDigest, isolateID: "isolate-1", sessionID: "session-1", nonce: "", wantErr: "requires session nonce"},
	}
}

func withRunID(spec launcherbackend.BackendLaunchSpec, value string) launcherbackend.BackendLaunchSpec {
	spec.RunID = value
	return spec
}

func withStageID(spec launcherbackend.BackendLaunchSpec, value string) launcherbackend.BackendLaunchSpec {
	spec.StageID = value
	return spec
}

func withRoleInstanceID(spec launcherbackend.BackendLaunchSpec, value string) launcherbackend.BackendLaunchSpec {
	spec.RoleInstanceID = value
	return spec
}
