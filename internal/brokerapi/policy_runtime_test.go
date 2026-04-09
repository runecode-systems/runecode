package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestEvaluateActionCompilesEvaluatesAndPersistsDecision(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-policy-evaluate"
	ctx := putTrustedPolicyContextForRun(t, s, runID, true)
	decision, err := s.EvaluateAction(runID, trustedArtifactReadAction())
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	assertEvaluateActionPersistedDecision(t, s, runID, ctx, decision)
}

func TestHandleArtifactReadUsesTrustedPolicyRuntime(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-artifact-read-policy"
	ctx := putTrustedPolicyContextForRun(t, s, runID, false)
	_ = ctx
	ref := putTrustedRuntimeArtifactForReadTest(t, s, runID)
	handle, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-policy-read",
		Digest:        ref.Digest,
		ProducerRole:  "workspace",
		ConsumerRole:  "model_gateway",
		ManifestOptIn: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactRead error response: %+v", errResp)
	}
	_ = handle.Reader.Close()
	assertSinglePersistedAllowDecision(t, s, runID)
}

func trustedArtifactReadAction() policyengine.ActionRequest {
	return policyengine.ActionRequest{
		SchemaID:              "runecode.protocol.v0.ActionRequest",
		SchemaVersion:         "0.1.0",
		ActionKind:            policyengine.ActionKindArtifactRead,
		CapabilityID:          artifactReadCapabilityID,
		ActionPayloadSchemaID: "runecode.protocol.v0.ActionPayloadArtifactRead",
		ActionPayload: map[string]any{
			"schema_id":      "runecode.protocol.v0.ActionPayloadArtifactRead",
			"schema_version": "0.1.0",
			"artifact_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("8", 64)},
			"read_mode":      "full",
		},
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func assertEvaluateActionPersistedDecision(t *testing.T, s *Service, runID string, ctx trustedPolicyContextDigests, decision policyengine.PolicyDecision) {
	t.Helper()
	if decision.DecisionOutcome != policyengine.DecisionAllow {
		t.Fatalf("decision_outcome = %q, want allow", decision.DecisionOutcome)
	}
	if decision.ManifestHash == "" || !strings.HasPrefix(decision.ManifestHash, "sha256:") {
		t.Fatalf("manifest_hash = %q, want sha256 identity", decision.ManifestHash)
	}
	details := requireSingleDecisionAuditDetails(t, s, runID)
	if manifestHash, _ := details["manifest_hash"].(string); manifestHash != decision.ManifestHash {
		t.Fatalf("persisted manifest_hash = %q, want %q", manifestHash, decision.ManifestHash)
	}
	if runValue, _ := details["run_id"].(string); runValue != runID {
		t.Fatalf("persisted run_id = %q, want %q", runValue, runID)
	}
	hashes := toStringSlice(details["policy_input_hashes"])
	if len(hashes) < 3 {
		t.Fatalf("policy_input_hashes len = %d, want >= 3", len(hashes))
	}
	if !containsString(hashes, ctx.roleDigest) || !containsString(hashes, ctx.runDigest) || !containsString(hashes, ctx.allowlistDigest) {
		t.Fatalf("policy_input_hashes = %#v, want role/run/allowlist digests", hashes)
	}
}

func putTrustedRuntimeArtifactForReadTest(t *testing.T, s *Service, runID string) artifacts.ArtifactReference {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("artifact"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassApprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("d", 64),
		CreatedByRole:         "workspace",
		RunID:                 runID,
		StepID:                "step-1",
	})
	if err != nil {
		t.Fatalf("Put artifact returned error: %v", err)
	}
	return ref
}

func assertSinglePersistedAllowDecision(t *testing.T, s *Service, runID string) {
	t.Helper()
	details := requireSingleDecisionAuditDetails(t, s, runID)
	if outcome, _ := details["decision_outcome"].(string); outcome != string(policyengine.DecisionAllow) {
		t.Fatalf("persisted decision_outcome = %q, want allow", outcome)
	}
}

func requireSingleDecisionAuditDetails(t *testing.T, s *Service, runID string) map[string]interface{} {
	t.Helper()
	refs := s.PolicyDecisionRefsForRun(runID)
	if len(refs) != 1 {
		t.Fatalf("policy decision refs len = %d, want 1", len(refs))
	}
	details, ok := decisionAuditDetailsByDigest(t, s, refs[0])
	if !ok {
		t.Fatalf("missing persisted decision for digest %q", refs[0])
	}
	return details
}
