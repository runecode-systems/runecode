package brokerapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectSubstrateCompatibilityAllowsInspectSurfacesInDiagnosticsMode(t *testing.T) {
	repoRoot := t.TempDir()
	writeDiagnosticsModeProjectSubstrateAnchors(t, repoRoot)

	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	missingDigest := "sha256:" + strings.Repeat("a", 64)
	assertDiagnosticsModeRunInspectSurfaces(t, service)
	assertDiagnosticsModeApprovalInspectSurfaces(t, service, missingDigest)
	assertDiagnosticsModeArtifactInspectSurfaces(t, service, missingDigest)
	seedSessionRuntimeFactsForOpsTest(t, service, "run-blocked", "sess-blocked")
	assertDiagnosticsModeExecutionStillBlocked(t, service)
	assertDiagnosticsModeReadinessAndSubstrateReads(t, service)
}

func writeDiagnosticsModeProjectSubstrateAnchors(t *testing.T, repoRoot string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repoRoot, "runecontext", "assurance"), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "runecontext", "assurance", "baseline.yaml"), []byte("canonicalization: runecontext-canonical-json-v1\ncreated_at: 0\nkind: baseline\nschema_version: 1\nsubject_id: project-root\nvalue:\n  adoption_commit: 0000000000000000000000000000000000000000\n  source_posture: embedded\n"), 0o644); err != nil {
		t.Fatalf("WriteFile baseline.yaml returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "runecontext.yaml"), []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext\n"), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
}

func assertDiagnosticsModeRunInspectSurfaces(t *testing.T, service *Service) {
	t.Helper()
	if _, errResp := service.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-run-list-blocked", Limit: 10}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleRunList returned error in diagnostics-only posture: %+v", errResp)
	}
	if _, errResp := service.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get-not-found", RunID: "run-missing"}, RequestContext{}); errResp == nil {
		t.Fatal("HandleRunGet error = nil, want run-specific not-found in diagnostics-only posture")
	} else if errResp.Error.Code != "broker_not_found_run" {
		t.Fatalf("error.code = %q, want broker_not_found_run", errResp.Error.Code)
	}
}

func assertDiagnosticsModeApprovalInspectSurfaces(t *testing.T, service *Service, missingDigest string) {
	t.Helper()
	if _, errResp := service.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list"}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleApprovalList returned error in diagnostics-only posture: %+v", errResp)
	}
	if _, errResp := service.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-not-found", ApprovalID: missingDigest}, RequestContext{}); errResp == nil {
		t.Fatal("HandleApprovalGet error = nil, want approval-specific not-found in diagnostics-only posture")
	} else if errResp.Error.Code != "broker_not_found_approval" {
		t.Fatalf("error.code = %q, want broker_not_found_approval", errResp.Error.Code)
	}
}

func assertDiagnosticsModeArtifactInspectSurfaces(t *testing.T, service *Service, missingDigest string) {
	t.Helper()
	if _, errResp := service.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: "0.1.0", RequestID: "req-artifact-list"}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleArtifactListV0 returned error in diagnostics-only posture: %+v", errResp)
	}
	if _, errResp := service.HandleArtifactHeadV0(context.Background(), LocalArtifactHeadRequest{SchemaID: "runecode.protocol.v0.ArtifactHeadRequest", SchemaVersion: "0.1.0", RequestID: "req-artifact-head-not-found", Digest: missingDigest}, RequestContext{}); errResp == nil {
		t.Fatal("HandleArtifactHeadV0 error = nil, want storage not-found in diagnostics-only posture")
	} else if errResp.Error.Code != "broker_not_found_artifact" {
		t.Fatalf("error.code = %q, want broker_not_found_artifact", errResp.Error.Code)
	}
}

func assertDiagnosticsModeExecutionStillBlocked(t *testing.T, service *Service) {
	t.Helper()
	if _, errResp := service.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-blocked", SessionID: "sess-blocked", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultWorkflowRoutingForTriggerTests(), UserMessageContentText: "run"}, RequestContext{}); errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger error = nil, want blocked execution-sensitive operation")
	} else if errResp.Error.Code != "project_substrate_operation_blocked" {
		t.Fatalf("error.code = %q, want project_substrate_operation_blocked", errResp.Error.Code)
	}
}

func assertDiagnosticsModeReadinessAndSubstrateReads(t *testing.T, service *Service) {
	t.Helper()
	if _, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: "req-readiness-diagnostics"}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if _, errResp := service.HandleProjectSubstrateGet(context.Background(), ProjectSubstrateGetRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateGetRequest", SchemaVersion: "0.1.0", RequestID: "req-substrate-get-diagnostics"}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleProjectSubstrateGet returned error: %+v", errResp)
	}
}

func TestProjectSubstrateGateRefreshesStateBeforeBlocking(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "runecontext", "assurance"), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "runecontext", "assurance", "baseline.yaml"), []byte("canonicalization: runecontext-canonical-json-v1\ncreated_at: 0\nkind: baseline\nschema_version: 1\nsubject_id: project-root\nvalue:\n  adoption_commit: 0000000000000000000000000000000000000000\n  source_posture: embedded\n"), 0o644); err != nil {
		t.Fatalf("WriteFile baseline.yaml returned error: %v", err)
	}
	configPath := filepath.Join(repoRoot, "runecontext.yaml")
	if err := os.WriteFile(configPath, []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext\n"), 0o644); err != nil {
		t.Fatalf("WriteFile invalid runecontext.yaml returned error: %v", err)
	}

	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	if err := os.WriteFile(configPath, []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: verified\nsource:\n  type: embedded\n  path: runecontext\n"), 0o644); err != nil {
		t.Fatalf("WriteFile valid runecontext.yaml returned error: %v", err)
	}

	if _, errResp := service.HandleRunList(context.Background(), RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-list-refreshed",
		Limit:         10,
	}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleRunList returned error after substrate fix: %+v", errResp)
	}
}
