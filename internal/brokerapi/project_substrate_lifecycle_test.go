package brokerapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func TestHandleProjectSubstrateGetAdoptAndInitLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	assertProjectSubstrateGetMissing(t, service)
	assertProjectSubstrateAdoptBlocked(t, service)
	previewResp := assertProjectSubstrateInitPreviewReady(t, service)
	assertProjectSubstrateInitApplyApplied(t, service, previewResp.Preview.PreviewToken)
	assertProjectSubstrateGetValid(t, service)
}

func TestHandleProjectSubstratePostureGetProjectsBrokerOwnedLifecycleSurface(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.13", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})

	resp, errResp := service.HandleProjectSubstratePostureGet(context.Background(), ProjectSubstratePostureGetRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstratePostureGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-posture-get",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstratePostureGet returned error: %+v", errResp)
	}
	if got := resp.PostureSummary.CompatibilityPosture; got != "supported_with_upgrade_available" {
		t.Fatalf("posture_summary.compatibility_posture = %q, want supported_with_upgrade_available", got)
	}
	if got := resp.Adoption.Status; got != "adopted" {
		t.Fatalf("adoption.status = %q, want adopted", got)
	}
	if got := resp.InitPreview.Status; got != "noop" {
		t.Fatalf("init_preview.status = %q, want noop for canonical repo", got)
	}
	if got := resp.UpgradePreview.Status; got != "ready_for_apply" {
		t.Fatalf("upgrade_preview.status = %q, want ready_for_apply for supported_with_upgrade_available", got)
	}
	if got := resp.UpgradePreview.ExpectedSnapshot.RuneContextVersion; got != "0.1.0-alpha.14" {
		t.Fatalf("upgrade_preview.expected_snapshot.runecontext_version = %q, want 0.1.0-alpha.14", got)
	}
	if len(resp.RemediationGuidance) == 0 {
		t.Fatal("remediation_guidance empty, want broker-projected advisory guidance")
	}
}

func TestHandleProjectSubstrateAdoptBlocksUnsupportedCompatibility(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.99", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})

	resp, errResp := service.HandleProjectSubstrateAdopt(context.Background(), ProjectSubstrateAdoptRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateAdoptRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-adopt-unsupported",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateAdopt returned error: %+v", errResp)
	}
	if got := resp.Adoption.Status; got != "blocked" {
		t.Fatalf("adoption.status = %q, want blocked", got)
	}
	if len(resp.Adoption.ReasonCodes) == 0 {
		t.Fatal("adoption.reason_codes empty, want compatibility blocked reason")
	}
	found := false
	for _, reason := range resp.Adoption.ReasonCodes {
		if reason == "project_substrate_unsupported_too_new" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("adoption.reason_codes = %v, want project_substrate_unsupported_too_new", resp.Adoption.ReasonCodes)
	}
}

func TestHandleProjectSubstrateUpgradePreviewAndApply(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.14", "plain", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})

	previewResp, errResp := service.HandleProjectSubstrateUpgradePreview(context.Background(), ProjectSubstrateUpgradePreviewRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateUpgradePreviewRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-upgrade-preview",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateUpgradePreview returned error: %+v", errResp)
	}
	if got := previewResp.Preview.Status; got != "ready_for_apply" {
		t.Fatalf("upgrade preview status = %q, want ready_for_apply", got)
	}
	if strings.TrimSpace(previewResp.Preview.PreviewDigest) == "" {
		t.Fatal("upgrade preview digest empty, want deterministic digest")
	}

	applyResp, errResp := service.HandleProjectSubstrateUpgradeApply(context.Background(), ProjectSubstrateUpgradeApplyRequest{
		SchemaID:              "runecode.protocol.v0.ProjectSubstrateUpgradeApplyRequest",
		SchemaVersion:         "0.1.0",
		RequestID:             "req-project-substrate-upgrade-apply",
		ExpectedPreviewDigest: previewResp.Preview.PreviewDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateUpgradeApply returned error: %+v", errResp)
	}
	if got := applyResp.ApplyResult.Status; got != "applied" {
		t.Fatalf("upgrade apply status = %q, want applied", got)
	}
	if got := applyResp.ApplyResult.ResultingSnapshot.ValidationState; got != "valid" {
		t.Fatalf("resulting snapshot validation_state = %q, want valid", got)
	}
}

func TestHandleProjectSubstrateApplyReturnsSuccessWhenRefreshFails(t *testing.T) {
	repoRoot := t.TempDir()
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	previewResp := assertProjectSubstrateInitPreviewReady(t, service)
	service.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{}, fmt.Errorf("refresh failed after apply")
	}

	applyResp, errResp := service.HandleProjectSubstrateInitApply(context.Background(), ProjectSubstrateInitApplyRequest{
		SchemaID:             "runecode.protocol.v0.ProjectSubstrateInitApplyRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-project-substrate-init-apply-refresh-failure",
		ExpectedPreviewToken: previewResp.Preview.PreviewToken,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateInitApply returned error: %+v", errResp)
	}
	if got := applyResp.ApplyResult.Status; got != "applied" {
		t.Fatalf("apply_result.status = %q, want applied", got)
	}
	content, err := os.ReadFile(filepath.Join(repoRoot, "runecontext.yaml"))
	if err != nil {
		t.Fatalf("ReadFile runecontext.yaml returned error: %v", err)
	}
	if !strings.Contains(string(content), "runecontext_version: \"0.1.0-alpha.14\"") {
		t.Fatalf("runecontext.yaml missing canonical version after apply:\n%s", string(content))
	}
	service.discoverProjectSubstrateFn = nil
	assertProjectSubstrateGetValid(t, service)
}

func TestHandleProjectSubstrateInitPreviewRefusesConflicts(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".runecontext"), 0o755); err != nil {
		t.Fatalf("MkdirAll .runecontext returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "runecontext-legacy"), 0o755); err != nil {
		t.Fatalf("MkdirAll runecontext-legacy returned error: %v", err)
	}

	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	previewResp, errResp := service.HandleProjectSubstrateInitPreview(context.Background(), ProjectSubstrateInitPreviewRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateInitPreviewRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-init-preview-conflict",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateInitPreview returned error: %+v", errResp)
	}
	if previewResp.Preview.Status != "blocked_conflict" {
		t.Fatalf("preview.status = %q, want blocked_conflict", previewResp.Preview.Status)
	}
	if len(previewResp.Preview.ConflictingPaths) == 0 {
		t.Fatal("conflicting_paths empty, want surfaced conflicts")
	}
}
