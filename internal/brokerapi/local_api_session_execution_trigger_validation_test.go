package brokerapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func TestApprovedImplementationInputSetDigestAcceptsDigestObjectIdentity(t *testing.T) {
	want := "sha256:" + strings.Repeat("a", 64)
	decoded := map[string]any{
		"input_set_digest": digestObject(want),
	}
	if got, ok := approvedImplementationInputSetDigest(decoded); !ok || got != want {
		t.Fatalf("approvedImplementationInputSetDigest = (%q, %v), want (%q, true)", got, ok, want)
	}
}

func TestApprovedImplementationInputSetDigestRejectsMalformedDigestObject(t *testing.T) {
	decoded := map[string]any{
		"input_set_digest": map[string]any{"hash_alg": "sha512", "hash": "abc"},
	}
	if _, ok := approvedImplementationInputSetDigest(decoded); ok {
		t.Fatal("approvedImplementationInputSetDigest returned ok for malformed digest object")
	}
}

func TestRecomputeApprovedImplementationInputSetDigestExcludesEmbeddedDigestField(t *testing.T) {
	decoded := map[string]any{
		"schema_id":              "runecode.protocol.v0.RuneContextApprovedImplementationInputSet",
		"schema_version":         "0.1.0",
		"approval_profile":       "moderate",
		"input_set_digest":       digestObject("sha256:" + strings.Repeat("f", 64)),
		"approved_input_digests": []any{digestObject("sha256:" + strings.Repeat("a", 64))},
	}
	got, err := recomputeApprovedImplementationInputSetDigest(decoded)
	if err != nil {
		t.Fatalf("recomputeApprovedImplementationInputSetDigest returned error: %v", err)
	}
	clone := map[string]any{}
	for key, value := range decoded {
		clone[key] = value
	}
	delete(clone, "input_set_digest")
	raw, err := json.Marshal(clone)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(raw)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	want := artifacts.DigestBytes(canonical)
	if got != want {
		t.Fatalf("recomputeApprovedImplementationInputSetDigest = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(decoded["input_set_digest"], digestObject("sha256:"+strings.Repeat("f", 64))) {
		t.Fatal("recomputeApprovedImplementationInputSetDigest mutated input payload")
	}
}

func TestValidateApprovedImplementationCatalogBindingAcceptsDigestObjects(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	catalog := approvedImplementationCatalogEntry()
	decoded := map[string]any{
		"workflow_definition_hash": digestObject(catalog.WorkflowDefinitionHash),
		"process_definition_hash":  digestObject(catalog.ProcessDefinitionHash),
	}
	if errResp := validateApprovedImplementationCatalogBinding(s, "req-approved-catalog-valid", decoded); errResp != nil {
		t.Fatalf("validateApprovedImplementationCatalogBinding returned error: %+v", errResp)
	}
}

func TestValidateApprovedImplementationCatalogBindingRejectsMismatchedDigestObject(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	catalog := approvedImplementationCatalogEntry()
	decoded := map[string]any{
		"workflow_definition_hash": digestObject("sha256:" + strings.Repeat("a", 64)),
		"process_definition_hash":  digestObject(catalog.ProcessDefinitionHash),
	}
	if errResp := validateApprovedImplementationCatalogBinding(s, "req-approved-catalog-drift", decoded); errResp == nil {
		t.Fatal("validateApprovedImplementationCatalogBinding expected mismatch error")
	}
}

func TestDigestIdentityFromApprovedImplementationFieldRejectsMalformedObject(t *testing.T) {
	decoded := map[string]any{
		"validated_project_substrate_digest": map[string]any{"hash_alg": "sha256"},
	}
	if _, ok := digestIdentityFromApprovedImplementationField(decoded, "validated_project_substrate_digest"); ok {
		t.Fatal("digestIdentityFromApprovedImplementationField returned ok for malformed object")
	}
}

func TestValidateSessionWorkflowRoutingSemanticsAllowsEmptyContinueRouting(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	errResp := s.validateSessionWorkflowRoutingSemantics("req-empty-continue", SessionExecutionTriggerRequest{
		RequestedOperation: "continue",
		WorkflowRouting: &SessionWorkflowPackRouting{
			SchemaID:      "runecode.protocol.v0.SessionWorkflowPackRouting",
			SchemaVersion: "0.1.0",
		},
	})
	if errResp != nil {
		t.Fatalf("validateSessionWorkflowRoutingSemantics returned error: %+v", errResp)
	}
}

func TestValidSessionWorkflowRoutingRejectsEmptyContinueRoutingWithArtifacts(t *testing.T) {
	routing := &SessionWorkflowPackRouting{
		SchemaID:      "runecode.protocol.v0.SessionWorkflowPackRouting",
		SchemaVersion: "0.1.0",
		BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{
			ArtifactRef:    "unexpected",
			ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
		}},
	}
	if validSessionWorkflowRouting("continue", routing) {
		t.Fatal("validSessionWorkflowRouting returned true for empty continue routing with bound artifacts")
	}
}

func TestValidSessionWorkflowRoutingRejectsEmptyContinueRoutingWithInvalidHeader(t *testing.T) {
	routing := &SessionWorkflowPackRouting{SchemaID: "invalid", SchemaVersion: "0.1.0"}
	if validSessionWorkflowRouting("continue", routing) {
		t.Fatal("validSessionWorkflowRouting returned true for empty continue routing with invalid header")
	}
}

func TestValidateApprovedImplementationRoutingRejectsDuplicateInputSetBindings(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	errResp := s.validateApprovedImplementationRouting("req-approved-duplicate", &SessionWorkflowPackRouting{
		SchemaID:          "runecode.protocol.v0.SessionWorkflowPackRouting",
		SchemaVersion:     "0.1.0",
		WorkflowFamily:    "runecontext",
		WorkflowOperation: "approved_change_implementation",
		BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{
			{ArtifactRef: "implementation_input_set", ArtifactDigest: "sha256:" + strings.Repeat("1", 64)},
			{ArtifactRef: "implementation_input_set", ArtifactDigest: "sha256:" + strings.Repeat("2", 64)},
		},
	})
	if errResp == nil {
		t.Fatal("validateApprovedImplementationRouting expected duplicate binding validation error")
	}
}

func TestValidateApprovedImplementationRoutingRejectsUnexpectedBindingRef(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	errResp := s.validateApprovedImplementationRouting("req-approved-extra", &SessionWorkflowPackRouting{
		SchemaID:          "runecode.protocol.v0.SessionWorkflowPackRouting",
		SchemaVersion:     "0.1.0",
		WorkflowFamily:    "runecontext",
		WorkflowOperation: "approved_change_implementation",
		BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{
			{ArtifactRef: "implementation_input_set", ArtifactDigest: "sha256:" + strings.Repeat("1", 64)},
			{ArtifactRef: "unexpected", ArtifactDigest: "sha256:" + strings.Repeat("2", 64)},
		},
	})
	if errResp == nil {
		t.Fatal("validateApprovedImplementationRouting expected unexpected-binding validation error")
	}
}
