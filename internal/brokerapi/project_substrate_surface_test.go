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
	assertProjectSubstrateRunListBlocked(t, service, "req-project-substrate-runlist-blocked")
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
	assertProjectSubstrateRunListBlocked(t, service, "req-no-auto-upgrade-runlist")
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

func assertProjectSubstrateRunListBlocked(t *testing.T, service *Service, requestID string) {
	t.Helper()
	_, runErr := service.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: requestID, Limit: 10}, RequestContext{})
	if runErr == nil {
		t.Fatal("HandleRunList error = nil, want project substrate policy block")
	}
	if runErr.Error.Code != "project_substrate_operation_blocked" {
		t.Fatalf("error code = %q, want project_substrate_operation_blocked", runErr.Error.Code)
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

func TestHandleProjectSubstrateGetAdoptAndInitLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	service := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	assertProjectSubstrateGetMissing(t, service)
	assertProjectSubstrateAdoptBlocked(t, service)
	previewResp := assertProjectSubstrateInitPreviewReady(t, service)
	assertProjectSubstrateInitApplyApplied(t, service, previewResp.Preview.PreviewToken)
	assertProjectSubstrateGetValid(t, service)
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

func repositoryRootForProjectSubstrateTests(t *testing.T) string {
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
