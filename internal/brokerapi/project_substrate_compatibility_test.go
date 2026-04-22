package brokerapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectSubstrateCompatibilityBlocksNormalOpsButAllowsDiagnostics(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "runecontext", "assurance"), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "runecontext", "assurance", "baseline.yaml"), []byte("canonicalization: runecontext-canonical-json-v1\ncreated_at: 0\nkind: baseline\nschema_version: 1\nsubject_id: project-root\nvalue:\n  adoption_commit: 0000000000000000000000000000000000000000\n  source_posture: embedded\n"), 0o644); err != nil {
		t.Fatalf("WriteFile baseline.yaml returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "runecontext.yaml"), []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext\n"), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}

	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	if _, errResp := service.HandleRunList(context.Background(), RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-list-blocked",
		Limit:         10,
	}, RequestContext{}); errResp == nil {
		t.Fatal("HandleRunList error = nil, want blocked by project substrate gate")
	} else if errResp.Error.Code != "project_substrate_operation_blocked" {
		t.Fatalf("error.code = %q, want project_substrate_operation_blocked", errResp.Error.Code)
	}

	if _, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-diagnostics",
	}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if _, errResp := service.HandleProjectSubstrateGet(context.Background(), ProjectSubstrateGetRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-substrate-get-diagnostics",
	}, RequestContext{}); errResp != nil {
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
