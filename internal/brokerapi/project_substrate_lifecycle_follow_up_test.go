package brokerapi

import (
	"context"
	"testing"
)

func TestHandleProjectSubstrateInitApplyRevalidatesFollowUpPostureAndStatus(t *testing.T) {
	repoRoot := t.TempDir()
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	previewResp := assertProjectSubstrateInitPreviewReady(t, service)
	assertProjectSubstrateInitApplyApplied(t, service, previewResp.Preview.PreviewToken)

	postureResp := mustProjectSubstratePostureGet(t, service, "req-project-substrate-posture-after-init-apply")
	if got := postureResp.PostureSummary.ValidationState; got != "valid" {
		t.Fatalf("posture_summary.validation_state = %q, want valid", got)
	}
	if got := postureResp.PostureSummary.CompatibilityPosture; got != "supported_current" {
		t.Fatalf("posture_summary.compatibility_posture = %q, want supported_current", got)
	}
	if !postureResp.PostureSummary.NormalOperationAllowed {
		t.Fatal("posture_summary.normal_operation_allowed = false, want true")
	}

	readinessResp := mustReadinessGetForProjectSubstrateSmoke(t, service, "req-project-substrate-readiness-after-init-apply")
	if readinessResp.Readiness.ProjectSubstrateSummary == nil {
		t.Fatal("readiness.project_substrate_posture_summary = nil, want projection after init apply")
	}
	if got := readinessResp.Readiness.ProjectSubstrateSummary.CompatibilityPosture; got != "supported_current" {
		t.Fatalf("readiness.project_substrate_posture_summary.compatibility_posture = %q, want supported_current", got)
	}
	if !readinessResp.Readiness.ProjectSubstrateSummary.NormalOperationAllowed {
		t.Fatal("readiness.project_substrate_posture_summary.normal_operation_allowed = false, want true")
	}
}

func TestHandleProjectSubstrateUpgradeApplyRevalidatesFollowUpPostureAndStatus(t *testing.T) {
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.13", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	previewResp := mustProjectSubstrateUpgradePreview(t, service, "req-project-substrate-upgrade-preview-follow-up")
	applyProjectSubstrateUpgradePreview(t, service, "req-project-substrate-upgrade-apply-follow-up", previewResp.Preview.PreviewDigest)

	postureResp := mustProjectSubstratePostureGet(t, service, "req-project-substrate-posture-after-upgrade-apply")
	if got := postureResp.PostureSummary.ValidationState; got != "valid" {
		t.Fatalf("posture_summary.validation_state = %q, want valid", got)
	}
	if got := postureResp.PostureSummary.CompatibilityPosture; got != "supported_current" {
		t.Fatalf("posture_summary.compatibility_posture = %q, want supported_current", got)
	}
	if got := postureResp.UpgradePreview.Status; got != "noop" {
		t.Fatalf("upgrade_preview.status = %q, want noop after upgrade apply", got)
	}

	readinessResp := mustReadinessGetForProjectSubstrateSmoke(t, service, "req-project-substrate-readiness-after-upgrade-apply")
	if readinessResp.Readiness.ProjectSubstrateSummary == nil {
		t.Fatal("readiness.project_substrate_posture_summary = nil, want projection after upgrade apply")
	}
	if got := readinessResp.Readiness.ProjectSubstrateSummary.CompatibilityPosture; got != "supported_current" {
		t.Fatalf("readiness.project_substrate_posture_summary.compatibility_posture = %q, want supported_current", got)
	}
	if !readinessResp.Readiness.ProjectSubstrateSummary.NormalOperationAllowed {
		t.Fatal("readiness.project_substrate_posture_summary.normal_operation_allowed = false, want true")
	}
}

func mustProjectSubstrateUpgradePreview(t *testing.T, service *Service, requestID string) ProjectSubstrateUpgradePreviewResponse {
	t.Helper()
	resp, errResp := service.HandleProjectSubstrateUpgradePreview(context.Background(), ProjectSubstrateUpgradePreviewRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradePreviewRequest", SchemaVersion: "0.1.0", RequestID: requestID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateUpgradePreview returned error: %+v", errResp)
	}
	return resp
}

func applyProjectSubstrateUpgradePreview(t *testing.T, service *Service, requestID, previewDigest string) {
	t.Helper()
	_, errResp := service.HandleProjectSubstrateUpgradeApply(context.Background(), ProjectSubstrateUpgradeApplyRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradeApplyRequest", SchemaVersion: "0.1.0", RequestID: requestID, ExpectedPreviewDigest: previewDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateUpgradeApply returned error: %+v", errResp)
	}
}

func TestProjectSubstrateLifecycleSmokeCoversInitAndUpgradeProductSurfaces(t *testing.T) {
	t.Run("init lifecycle", func(t *testing.T) {
		assertProjectSubstrateInitLifecycleSmoke(t)
	})

	t.Run("adopt and upgrade lifecycle", func(t *testing.T) {
		assertProjectSubstrateAdoptAndUpgradeLifecycleSmoke(t)
	})
}

func assertProjectSubstrateInitLifecycleSmoke(t *testing.T) {
	t.Helper()
	repoRoot := t.TempDir()
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	assertProjectSubstrateGetMissing(t, service)
	postureResp := mustProjectSubstratePostureGet(t, service, "req-project-substrate-smoke-init-posture-before")
	if postureResp.PostureSummary.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed before init = true, want false")
	}
	assertProjectSubstrateAdoptBlocked(t, service)
	previewResp := assertProjectSubstrateInitPreviewReady(t, service)
	assertProjectSubstrateInitApplyApplied(t, service, previewResp.Preview.PreviewToken)
	assertProjectSubstrateGetValid(t, service)
	postureResp = mustProjectSubstratePostureGet(t, service, "req-project-substrate-smoke-init-posture-after")
	if got := postureResp.PostureSummary.CompatibilityPosture; got != "supported_current" {
		t.Fatalf("compatibility_posture after init = %q, want supported_current", got)
	}
	if !postureResp.PostureSummary.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed after init = false, want true")
	}
	readinessResp := mustReadinessGetForProjectSubstrateSmoke(t, service, "req-project-substrate-smoke-init-readiness")
	if readinessResp.Readiness.ProjectSubstrateSummary == nil {
		t.Fatal("readiness.project_substrate_summary = nil, want broker-owned summary after init")
	}
	if got := readinessResp.Readiness.ProjectSubstrateSummary.ValidationState; got != "valid" {
		t.Fatalf("readiness project substrate validation_state = %q, want valid", got)
	}
}

func assertProjectSubstrateAdoptAndUpgradeLifecycleSmoke(t *testing.T) {
	t.Helper()
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.13", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	adoptResp, errResp := service.HandleProjectSubstrateAdopt(context.Background(), ProjectSubstrateAdoptRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateAdoptRequest", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-smoke-adopt"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateAdopt returned error: %+v", errResp)
	}
	if got := adoptResp.Adoption.Status; got != "adopted" {
		t.Fatalf("adoption.status = %q, want adopted", got)
	}
	postureResp := mustProjectSubstratePostureGet(t, service, "req-project-substrate-smoke-upgrade-before")
	if got := postureResp.UpgradePreview.Status; got != "ready_for_apply" {
		t.Fatalf("upgrade_preview.status before upgrade = %q, want ready_for_apply", got)
	}
	previewResp, errResp := service.HandleProjectSubstrateUpgradePreview(context.Background(), ProjectSubstrateUpgradePreviewRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradePreviewRequest", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-smoke-upgrade-preview"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateUpgradePreview returned error: %+v", errResp)
	}
	if got := previewResp.Preview.Status; got != "ready_for_apply" {
		t.Fatalf("upgrade preview status = %q, want ready_for_apply", got)
	}
	applyResp, errResp := service.HandleProjectSubstrateUpgradeApply(context.Background(), ProjectSubstrateUpgradeApplyRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradeApplyRequest", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-smoke-upgrade-apply", ExpectedPreviewDigest: previewResp.Preview.PreviewDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateUpgradeApply returned error: %+v", errResp)
	}
	if got := applyResp.ApplyResult.Status; got != "applied" {
		t.Fatalf("upgrade apply status = %q, want applied", got)
	}
	postureResp = mustProjectSubstratePostureGet(t, service, "req-project-substrate-smoke-upgrade-after")
	if got := postureResp.PostureSummary.CompatibilityPosture; got != "supported_current" {
		t.Fatalf("compatibility_posture after upgrade = %q, want supported_current", got)
	}
	if got := postureResp.UpgradePreview.Status; got != "noop" {
		t.Fatalf("upgrade_preview.status after upgrade = %q, want noop", got)
	}
}

func mustProjectSubstratePostureGet(t *testing.T, service *Service, requestID string) ProjectSubstratePostureGetResponse {
	t.Helper()
	resp, errResp := service.HandleProjectSubstratePostureGet(context.Background(), ProjectSubstratePostureGetRequest{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureGetRequest", SchemaVersion: "0.1.0", RequestID: requestID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstratePostureGet returned error: %+v", errResp)
	}
	return resp
}

func mustReadinessGetForProjectSubstrateSmoke(t *testing.T, service *Service, requestID string) ReadinessGetResponse {
	t.Helper()
	resp, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: requestID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	return resp
}
