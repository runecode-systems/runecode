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

func TestHandleReadinessGetProjectsProjectSubstrateSummary(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	summary := readinessProjectSubstrateSummary(t, service)
	assertProjectSubstrateSummaryCore(t, summary)
	assertProjectSubstrateSummaryVersionRange(t, summary)
	assertProjectSubstrateSummaryIdentity(t, summary)
}

func TestHandleVersionInfoGetProjectsProjectContextIdentityDigest(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	resp, errResp := service.HandleVersionInfoGet(context.Background(), VersionInfoGetRequest{
		SchemaID:      "runecode.protocol.v0.VersionInfoGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-version",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleVersionInfoGet returned error: %+v", errResp)
	}
	if resp.VersionInfo.ProjectSubstrateContractID != "runecode.runecontext.project_substrate.v0" {
		t.Fatalf("project_substrate_contract_id = %q, want runecode.runecontext.project_substrate.v0", resp.VersionInfo.ProjectSubstrateContractID)
	}
	if resp.VersionInfo.ProjectSubstrateContractVersion != "v0" {
		t.Fatalf("project_substrate_contract_version = %q, want v0", resp.VersionInfo.ProjectSubstrateContractVersion)
	}
	if resp.VersionInfo.ProjectSubstrateValidationState != "valid" {
		t.Fatalf("project_substrate_validation_state = %q, want valid", resp.VersionInfo.ProjectSubstrateValidationState)
	}
	if resp.VersionInfo.ProjectSubstratePosture != "supported_current" {
		t.Fatalf("project_substrate_posture = %q, want supported_current", resp.VersionInfo.ProjectSubstratePosture)
	}
	if resp.VersionInfo.ProjectSubstrateVersion == "" {
		t.Fatal("project_substrate_version empty, want declared version")
	}
	if resp.VersionInfo.ProjectSubstrateSupportedMin == "" || resp.VersionInfo.ProjectSubstrateSupportedMax == "" || resp.VersionInfo.ProjectSubstrateRecommended == "" {
		t.Fatalf("supported/recommended range missing: min=%q max=%q recommended=%q", resp.VersionInfo.ProjectSubstrateSupportedMin, resp.VersionInfo.ProjectSubstrateSupportedMax, resp.VersionInfo.ProjectSubstrateRecommended)
	}
	if resp.VersionInfo.ProjectContextIdentityDigest == "" {
		t.Fatal("project_context_identity_digest empty, want digest")
	}
	if resp.VersionInfo.ProjectSubstratePostureSummary == nil {
		t.Fatal("project_substrate_posture_summary = nil, want summary projection on version surface")
	}
}

func TestProjectSubstrateCompatibilitySupportedWithUpgradeAvailable(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.13", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})
	resp, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-upgrade-available",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	summary := resp.Readiness.ProjectSubstrateSummary
	if summary == nil {
		t.Fatal("project_substrate_posture_summary = nil, want value")
	}
	if got := summary.CompatibilityPosture; got != "supported_with_upgrade_available" {
		t.Fatalf("compatibility_posture = %q, want supported_with_upgrade_available", got)
	}
	if !summary.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed = false, want true")
	}
	if len(summary.ReasonCodes) == 0 {
		t.Fatal("reason_codes empty, want upgrade advisory reason")
	}
	if len(summary.BlockedReasonCodes) != 0 {
		t.Fatalf("blocked_reason_codes = %v, want empty", summary.BlockedReasonCodes)
	}
}

func TestProjectSubstrateCompatibilityUnsupportedBlocksNormalOperationsButKeepsDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.99", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})
	assertProjectSubstrateRunListAllowed(t, service, "req-project-substrate-runlist-allowed")
	assertBlockedProjectSubstrateReadiness(t, service, "unsupported_too_new")
	assertProjectSubstrateVersionDiagnostics(t, service, "req-project-substrate-version-diagnostics")
}

func TestNoAutoUpgradeDuringOrdinaryBrokerOperations(t *testing.T) {
	repoRoot := t.TempDir()
	writeProjectSubstrateDirs(t, repoRoot)
	configPath := filepath.Join(repoRoot, "runecontext.yaml")
	before := "schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext\n"
	if err := os.WriteFile(configPath, []byte(before), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}

	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	assertNonVerifiedReadiness(t, service)
	assertProjectSubstrateRunListAllowed(t, service, "req-no-auto-upgrade-runlist")
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(after) returned error: %v", err)
	}
	if string(after) != before {
		t.Fatalf("runecontext.yaml mutated during ordinary flows:\nbefore:\n%s\nafter:\n%s", before, string(after))
	}
}

func readinessProjectSubstrateSummary(t *testing.T, service *Service) *ProjectSubstratePostureSummary {
	t.Helper()
	resp, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-readiness"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if resp.Readiness.ProjectSubstrateSummary == nil {
		t.Fatal("project_substrate_posture_summary = nil, want value")
	}
	return resp.Readiness.ProjectSubstrateSummary
}

func assertProjectSubstrateSummaryCore(t *testing.T, summary *ProjectSubstratePostureSummary) {
	t.Helper()
	if summary.ContractID != "runecode.runecontext.project_substrate.v0" {
		t.Fatalf("contract_id = %q, want runecode.runecontext.project_substrate.v0", summary.ContractID)
	}
	if summary.ContractVersion != "v0" {
		t.Fatalf("contract_version = %q, want v0", summary.ContractVersion)
	}
	if summary.ValidationState != "valid" {
		t.Fatalf("validation_state = %q, want valid", summary.ValidationState)
	}
	if summary.CompatibilityPosture != "supported_current" {
		t.Fatalf("compatibility_posture = %q, want supported_current", summary.CompatibilityPosture)
	}
	if !summary.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed = false, want true")
	}
	if summary.ActiveRuneContextVersion == "" {
		t.Fatal("active_runecontext_version empty, want declared version")
	}
}

func assertProjectSubstrateSummaryVersionRange(t *testing.T, summary *ProjectSubstratePostureSummary) {
	t.Helper()
	if summary.SupportedRuneContextMin == "" || summary.SupportedRuneContextMax == "" || summary.RecommendedRuneContextTarget == "" {
		t.Fatalf("supported/recommended range missing: min=%q max=%q recommended=%q", summary.SupportedRuneContextMin, summary.SupportedRuneContextMax, summary.RecommendedRuneContextTarget)
	}
}

func assertProjectSubstrateSummaryIdentity(t *testing.T, summary *ProjectSubstratePostureSummary) {
	t.Helper()
	if summary.ValidatedSnapshotDigest == "" {
		t.Fatal("validated_snapshot_digest empty, want digest")
	}
	if summary.ProjectContextIdentityDigest != summary.ValidatedSnapshotDigest {
		t.Fatalf("project_context_identity_digest = %q, want %q", summary.ProjectContextIdentityDigest, summary.ValidatedSnapshotDigest)
	}
}

func assertProjectSubstrateRunListAllowed(t *testing.T, service *Service, requestID string) {
	t.Helper()
	if _, runErr := service.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: requestID, Limit: 10}, RequestContext{}); runErr != nil {
		t.Fatalf("HandleRunList returned error in diagnostics-only posture: %+v", runErr)
	}
}

func assertBlockedProjectSubstrateReadiness(t *testing.T, service *Service, posture string) {
	t.Helper()
	resp, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-readiness-diagnostics"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if resp.Readiness.Ready {
		t.Fatal("readiness.ready = true, want false when project substrate is blocked")
	}
	if resp.Readiness.ProjectSubstrateSummary == nil {
		t.Fatal("project_substrate_posture_summary = nil, want value")
	}
	if got := resp.Readiness.ProjectSubstrateSummary.CompatibilityPosture; got != posture {
		t.Fatalf("compatibility_posture = %q, want %q", got, posture)
	}
	if len(resp.Readiness.ProjectSubstrateSummary.BlockedReasonCodes) == 0 {
		t.Fatal("blocked_reason_codes empty, want blocked reasons")
	}
}

func assertProjectSubstrateVersionDiagnostics(t *testing.T, service *Service, requestID string) {
	t.Helper()
	_, errResp := service.HandleVersionInfoGet(context.Background(), VersionInfoGetRequest{SchemaID: "runecode.protocol.v0.VersionInfoGetRequest", SchemaVersion: "0.1.0", RequestID: requestID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleVersionInfoGet returned error: %+v", errResp)
	}
}

func writeProjectSubstrateDirs(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "runecontext", "assurance"), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "runecontext"), 0o755); err != nil {
		t.Fatalf("MkdirAll runecontext returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "runecontext", "assurance", "baseline.yaml"), []byte("canonicalization: runecontext-canonical-json-v1\ncreated_at: 0\nkind: baseline\nschema_version: 1\nsubject_id: project-root\nvalue:\n  adoption_commit: 0000000000000000000000000000000000000000\n  source_posture: embedded\n"), 0o644); err != nil {
		t.Fatalf("WriteFile baseline.yaml returned error: %v", err)
	}
}

func assertNonVerifiedReadiness(t *testing.T, service *Service) {
	t.Helper()
	resp, errResp := service.HandleReadinessGet(context.Background(), ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: "req-no-auto-upgrade-readiness"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if resp.Readiness.ProjectSubstrateSummary == nil {
		t.Fatal("project_substrate_posture_summary = nil, want value")
	}
	if got := resp.Readiness.ProjectSubstrateSummary.ValidationState; got != "invalid" {
		t.Fatalf("validation_state = %q, want invalid for non-verified posture", got)
	}
}

func assertProjectSubstrateGetMissing(t *testing.T, service *Service) {
	t.Helper()

	getResp, errResp := service.HandleProjectSubstrateGet(context.Background(), ProjectSubstrateGetRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-get",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateGet returned error: %+v", errResp)
	}
	if getResp.Snapshot.ValidationState != "missing" {
		t.Fatalf("snapshot.validation_state = %q, want missing", getResp.Snapshot.ValidationState)
	}
}

func assertProjectSubstrateAdoptBlocked(t *testing.T, service *Service) {
	t.Helper()
	adoptResp, errResp := service.HandleProjectSubstrateAdopt(context.Background(), ProjectSubstrateAdoptRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateAdoptRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-adopt",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateAdopt returned error: %+v", errResp)
	}
	if adoptResp.Adoption.Status != "blocked" {
		t.Fatalf("adoption.status = %q, want blocked", adoptResp.Adoption.Status)
	}
}

func assertProjectSubstrateInitPreviewReady(t *testing.T, service *Service) ProjectSubstrateInitPreviewResponse {
	t.Helper()
	previewResp, errResp := service.HandleProjectSubstrateInitPreview(context.Background(), ProjectSubstrateInitPreviewRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateInitPreviewRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-init-preview",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateInitPreview returned error: %+v", errResp)
	}
	if previewResp.Preview.Status != "ready_for_apply" {
		t.Fatalf("preview.status = %q, want ready_for_apply", previewResp.Preview.Status)
	}
	if previewResp.Preview.PreviewToken == "" {
		t.Fatal("preview.preview_token empty, want deterministic token")
	}
	return previewResp
}

func assertProjectSubstrateInitApplyApplied(t *testing.T, service *Service, previewToken string) {
	t.Helper()
	applyResp, errResp := service.HandleProjectSubstrateInitApply(context.Background(), ProjectSubstrateInitApplyRequest{
		SchemaID:             "runecode.protocol.v0.ProjectSubstrateInitApplyRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-project-substrate-init-apply",
		ExpectedPreviewToken: previewToken,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateInitApply returned error: %+v", errResp)
	}
	if applyResp.ApplyResult.Status != "applied" {
		t.Fatalf("apply_result.status = %q, want applied", applyResp.ApplyResult.Status)
	}
}

func assertProjectSubstrateGetValid(t *testing.T, service *Service) {
	t.Helper()
	postGetResp, errResp := service.HandleProjectSubstrateGet(context.Background(), ProjectSubstrateGetRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstrateGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-project-substrate-get-post",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProjectSubstrateGet(post) returned error: %+v", errResp)
	}
	if postGetResp.Snapshot.ValidationState != "valid" {
		t.Fatalf("post snapshot.validation_state = %q, want valid", postGetResp.Snapshot.ValidationState)
	}
}

func TestHandleProductLifecyclePostureGetSupportedWithUpgradeProjectsDegradedAttachableSurface(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.13", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})

	resp, errResp := service.HandleProductLifecyclePostureGet(context.Background(), ProductLifecyclePostureGetRequest{
		SchemaID:      "runecode.protocol.v0.ProductLifecyclePostureGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-product-lifecycle-posture-upgrade-available",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProductLifecyclePostureGet returned error: %+v", errResp)
	}
	if got := resp.ProductLifecycle.AttachMode; got != "full" {
		t.Fatalf("attach_mode = %q, want full", got)
	}
	if got := resp.ProductLifecycle.LifecyclePosture; got != "degraded" {
		t.Fatalf("lifecycle_posture = %q, want degraded", got)
	}
	if !resp.ProductLifecycle.Attachable {
		t.Fatal("attachable = false, want true")
	}
	if !resp.ProductLifecycle.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed = false, want true")
	}
	if len(resp.ProductLifecycle.BlockedReasonCodes) != 0 {
		t.Fatalf("blocked_reason_codes = %v, want empty", resp.ProductLifecycle.BlockedReasonCodes)
	}
	if len(resp.ProductLifecycle.DegradedReasonCodes) != 1 || resp.ProductLifecycle.DegradedReasonCodes[0] != "project_substrate_upgrade_available" {
		t.Fatalf("degraded_reason_codes = %v, want [project_substrate_upgrade_available]", resp.ProductLifecycle.DegradedReasonCodes)
	}
	if strings.TrimSpace(resp.ProductLifecycle.ProductInstanceID) == "" {
		t.Fatal("product_instance_id empty, want stable repo-scoped identity")
	}
	if strings.TrimSpace(resp.ProductLifecycle.LifecycleGeneration) == "" {
		t.Fatal("lifecycle_generation empty, want broker generation identity")
	}
}

func TestHandleProductLifecyclePostureGetUnsupportedProjectsDiagnosticsOnlyAttachMode(t *testing.T) {
	root := t.TempDir()
	writeProjectSubstrateAnchors(t, root, "0.1.0-alpha.99", "verified", "runecontext")
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: root})

	resp, errResp := service.HandleProductLifecyclePostureGet(context.Background(), ProductLifecyclePostureGetRequest{
		SchemaID:      "runecode.protocol.v0.ProductLifecyclePostureGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-product-lifecycle-posture-unsupported",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProductLifecyclePostureGet returned error: %+v", errResp)
	}
	if got := resp.ProductLifecycle.AttachMode; got != "diagnostics_only" {
		t.Fatalf("attach_mode = %q, want diagnostics_only", got)
	}
	if got := resp.ProductLifecycle.LifecyclePosture; got != "blocked" {
		t.Fatalf("lifecycle_posture = %q, want blocked", got)
	}
	if !resp.ProductLifecycle.Attachable {
		t.Fatal("attachable = false, want true in diagnostics mode")
	}
	if resp.ProductLifecycle.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed = true, want false")
	}
	if len(resp.ProductLifecycle.BlockedReasonCodes) == 0 {
		t.Fatal("blocked_reason_codes empty, want stable blocked reason codes")
	}
	foundUnsupportedTooNew := false
	for _, reason := range resp.ProductLifecycle.BlockedReasonCodes {
		if reason == "project_substrate_unsupported_too_new" {
			foundUnsupportedTooNew = true
			break
		}
	}
	if !foundUnsupportedTooNew {
		t.Fatalf("blocked_reason_codes = %v, want project_substrate_unsupported_too_new", resp.ProductLifecycle.BlockedReasonCodes)
	}
}

func TestHandleProductLifecyclePostureGetDiscoveryFailureStillProjectsDiagnosticsAttach(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: t.TempDir()})
	service.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{}, fmt.Errorf("broken substrate state")
	}

	resp, errResp := service.HandleProductLifecyclePostureGet(context.Background(), ProductLifecyclePostureGetRequest{
		SchemaID:      "runecode.protocol.v0.ProductLifecyclePostureGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-product-lifecycle-posture-discovery-failure",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProductLifecyclePostureGet returned error: %+v", errResp)
	}
	if got := resp.ProductLifecycle.AttachMode; got != "diagnostics_only" {
		t.Fatalf("attach_mode = %q, want diagnostics_only", got)
	}
	if got := resp.ProductLifecycle.LifecyclePosture; got != "blocked" {
		t.Fatalf("lifecycle_posture = %q, want blocked", got)
	}
	if !resp.ProductLifecycle.Attachable {
		t.Fatal("attachable = false, want true")
	}
	if len(resp.ProductLifecycle.BlockedReasonCodes) != 1 || resp.ProductLifecycle.BlockedReasonCodes[0] != "project_substrate_discovery_failed" {
		t.Fatalf("blocked_reason_codes = %v, want [project_substrate_discovery_failed]", resp.ProductLifecycle.BlockedReasonCodes)
	}
}

func repositoryRootForProjectSubstrateTests(t testing.TB) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func writeProjectSubstrateAnchors(t *testing.T, root, version, assuranceTier, sourcePath string) {
	t.Helper()
	writeProjectSubstrateDirs(t, root)
	content := "schema_version: 1\nrunecontext_version: \"" + version + "\"\nassurance_tier: " + assuranceTier + "\nsource:\n  type: embedded\n  path: " + sourcePath + "\n"
	if err := os.WriteFile(filepath.Join(root, "runecontext.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
}
