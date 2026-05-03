package brokerapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestNewServiceDerivesProductInstanceIDViaSharedHelper(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) returned error: %v", err)
	}
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	configuredRoot := "  " + repoRoot + "  "

	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: configuredRoot})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}

	want := localbootstrap.DeriveProductInstanceID(repoRoot)
	if service.productInstanceID != want {
		t.Fatalf("productInstanceID = %q, want %q", service.productInstanceID, want)
	}
}

func TestDefaultVersionInfoUsesConcreteMetadata(t *testing.T) {
	service, err := NewService(t.TempDir(), filepath.Join(t.TempDir(), "ledger"))
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	info := service.versionInfo
	assertVersionInfoFieldConcrete(t, "product_version", info.ProductVersion)
	assertVersionInfoFieldConcrete(t, "build_revision", info.BuildRevision)
	assertVersionInfoFieldConcrete(t, "build_time", info.BuildTime)
	if info.ProtocolBundleVersion != "0.9.0" {
		t.Fatalf("protocol_bundle_version = %q, want 0.9.0", info.ProtocolBundleVersion)
	}
	if !strings.HasPrefix(info.ProtocolBundleManifestHash, "sha256:") || len(info.ProtocolBundleManifestHash) != 71 {
		t.Fatalf("protocol_bundle_manifest_hash = %q, want sha256 identity", info.ProtocolBundleManifestHash)
	}
	if info.ProtocolBundleManifestHash == "sha256:"+strings.Repeat("0", 64) {
		t.Fatal("protocol_bundle_manifest_hash must not be all-zero placeholder")
	}
}

func assertVersionInfoFieldConcrete(t *testing.T, name, value string) {
	t.Helper()
	if value == "" || value == "unknown" {
		t.Fatalf("%s = %q, want concrete value", name, value)
	}
}

func TestSeedDevManualScenarioReturnsApprovalForSeededDecision(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	if err := seedConflictingApprovalDecision(t, service, devManualSeedRunID, digestWithByte("0"), digestWithByte("9")); err != nil {
		t.Fatalf("seedConflictingApprovalDecision returned error: %v", err)
	}
	result, err := service.SeedDevManualScenario()
	if err != nil {
		t.Fatalf("SeedDevManualScenario returned error: %v", err)
	}
	approval, ok := service.ApprovalGet(result.ApprovalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing seeded approval", result.ApprovalID)
	}
	decision, ok := service.PolicyDecisionGet(approval.PolicyDecisionHash)
	if !ok {
		t.Fatalf("PolicyDecisionGet(%q) missing seeded decision", approval.PolicyDecisionHash)
	}
	if !isDevManualApprovalDecision(decision) {
		t.Fatalf("approval %q linked to non-seed decision: %+v", result.ApprovalID, decision)
	}
}

func TestSeedDevManualScenarioAddsManualSeedLinkWhenDifferentEventSharesDigest(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	if err := service.AppendTrustedAuditEvent("other_event", "brokerapi", map[string]interface{}{
		"run_id":        devManualSeedRunID,
		"session_id":    devManualSeedSessionID,
		"record_digest": "sha256:" + strings.Repeat("1", 64),
		"seed_profile":  devManualSeedDefaultProfile,
	}); err != nil {
		t.Fatalf("AppendTrustedAuditEvent returned error: %v", err)
	}
	result, err := service.SeedDevManualScenario()
	if err != nil {
		t.Fatalf("SeedDevManualScenario returned error: %v", err)
	}
	events, err := service.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	count := 0
	for _, event := range events {
		if devManualSessionAuditLinkMatches(event, result.AuditRecordDigest, result.Profile) {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("manual_seed_link count = %d, want 1", count)
	}
}

func TestSeedDevManualScenarioRejectsDefaultLedgerRoot(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service, err := NewService(t.TempDir(), auditd.DefaultLedgerRoot())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	t.Setenv(devManualSeedEnvVar, "1")
	_, err = service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected default-ledger-root rejection")
	}
	if err.Error() != "dev manual seeding refuses default audit ledger root" {
		t.Fatalf("SeedDevManualScenario error = %q, want sanitized default-ledger refusal", err.Error())
	}
}

func TestSeedDevManualScenarioRejectsLedgerWithMultipleBootstrapSegments(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	if err := os.MkdirAll(filepath.Join(service.auditRoot, "segments"), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := writeBootstrapOpenSegmentForDevManualSeedTest(filepath.Join(service.auditRoot, "segments", "segment-000001.json"), "segment-000001"); err != nil {
		t.Fatalf("writeBootstrapOpenSegmentForDevManualSeedTest(1) returned error: %v", err)
	}
	if err := writeBootstrapOpenSegmentForDevManualSeedTest(filepath.Join(service.auditRoot, "segments", "segment-000002.json"), "segment-000002"); err != nil {
		t.Fatalf("writeBootstrapOpenSegmentForDevManualSeedTest(2) returned error: %v", err)
	}
	_, err := service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected populated-ledger rejection for multiple bootstrap segments")
	}
	if err.Error() != "dev manual seeding refuses populated audit ledger root" {
		t.Fatalf("SeedDevManualScenario error = %q, want populated-ledger refusal", err.Error())
	}
}

func TestSeedDevManualScenarioRejectsLedgerWithExistingReceiptSidecar(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	receiptsDir := filepath.Join(service.auditRoot, "sidecar", "receipts")
	if err := os.MkdirAll(receiptsDir, 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(receiptsDir, strings.Repeat("a", 64)+".json"), []byte(`{"schema_id":"x"}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err := service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected populated-ledger rejection for receipt sidecar")
	}
	if err.Error() != "dev manual seeding refuses populated audit ledger root" {
		t.Fatalf("SeedDevManualScenario error = %q, want populated-ledger refusal", err.Error())
	}
}

func TestSeedDevManualScenarioRejectsLedgerWithExistingExternalAnchorEvidence(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	evidenceDir := filepath.Join(service.auditRoot, "sidecar", "external-anchor-evidence")
	if err := os.MkdirAll(evidenceDir, 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, strings.Repeat("b", 64)+".json"), []byte(`{"schema_id":"x"}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err := service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected populated-ledger rejection for external anchor evidence")
	}
	if err.Error() != "dev manual seeding refuses populated audit ledger root" {
		t.Fatalf("SeedDevManualScenario error = %q, want populated-ledger refusal", err.Error())
	}
}

func TestSeedDevManualScenarioRejectsOversizedSeedMarker(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	markerPath := devManualLedgerSeedMarkerPath(service.auditRoot)
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(markerPath, []byte(strings.Repeat("x", devManualSeedMarkerMaxBytes+1)), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err := service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected oversized marker rejection")
	}
	if err.Error() != "dev manual seeding refuses tampered seed marker" {
		t.Fatalf("SeedDevManualScenario error = %q, want tampered-marker refusal", err.Error())
	}
}

func TestSeedDevManualScenarioRejectsInvalidSeedMarkerContent(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	markerPath := devManualLedgerSeedMarkerPath(service.auditRoot)
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(markerPath, []byte("tampered\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err := service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected invalid marker rejection")
	}
	if err.Error() != "dev manual seeding refuses tampered seed marker" {
		t.Fatalf("SeedDevManualScenario error = %q, want tampered-marker refusal", err.Error())
	}
}

func TestSeedDevManualScenarioUnavailableWhenBuildTagDisabled(t *testing.T) {
	if DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is enabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	_, err := service.SeedDevManualScenario()
	if err == nil {
		t.Fatal("SeedDevManualScenario expected build-disabled error")
	}
	if err.Error() != "dev manual seeding unavailable in this build" {
		t.Fatalf("SeedDevManualScenario error = %q, want build-disabled message", err.Error())
	}
}

func TestSeedDevManualScenarioSupportsBackendPostureApprovalFlow(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	if _, err := service.SeedDevManualScenario(); err != nil {
		t.Fatalf("SeedDevManualScenario returned error: %v", err)
	}
	resp, errResp := service.HandleBackendPostureChange(context.Background(), BackendPostureChangeRequest{
		SchemaID:                     "runecode.protocol.v0.BackendPostureChangeRequest",
		SchemaVersion:                "0.1.0",
		RequestID:                    "req-dev-seed-backend-posture",
		TargetInstanceID:             devManualSeedInstanceID,
		TargetBackendKind:            "container",
		SelectionMode:                "explicit_selection",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
		Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleBackendPostureChange error response: %+v", errResp)
	}
	if resp.Outcome.Outcome != "approval_required" {
		t.Fatalf("outcome = %q, want approval_required", resp.Outcome.Outcome)
	}
	if resp.Outcome.ApprovalID == "" {
		t.Fatal("approval_id = empty, want backend posture approval")
	}
}

func TestSeedDevManualScenarioSupportsDegradedProfile(t *testing.T) {
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	result, err := service.SeedDevManualScenarioWithProfile(devManualSeedDegradedProfile)
	if err != nil {
		t.Fatalf("SeedDevManualScenarioWithProfile returned error: %v", err)
	}
	if result.Profile != devManualSeedDegradedProfile {
		t.Fatalf("profile = %q, want %q", result.Profile, devManualSeedDegradedProfile)
	}
}

func TestNormalizeDevManualSeedProfileRejectsUnsupportedValue(t *testing.T) {
	if _, err := NormalizeDevManualSeedProfile("not-a-profile"); err == nil {
		t.Fatal("NormalizeDevManualSeedProfile expected error for unsupported profile")
	}
}

func newDevManualSeedService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	service, err := NewService(filepath.Join(root, "store"), filepath.Join(root, "ledger"))
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	return service
}

func seedConflictingApprovalDecision(t *testing.T, service *Service, runID, manifestHash, actionHash string) error {
	t.Helper()
	return service.RecordPolicyDecision(runID, "", policyengine.PolicyDecision{
		SchemaID:                 "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:            "0.3.0",
		DecisionOutcome:          policyengine.DecisionRequireHumanApproval,
		PolicyReasonCode:         "approval_required",
		ManifestHash:             manifestHash,
		ActionRequestHash:        actionHash,
		PolicyInputHashes:        []string{digestWithByte("7")},
		RelevantArtifactHashes:   []string{digestWithByte("8")},
		DetailsSchemaID:          "runecode.protocol.details.policy.evaluation.v0",
		Details:                  map[string]any{"precedence": "test_conflict"},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "excerpt_promotion",
			"approval_assurance_level": "moderate",
			"presence_mode":            "os_confirmation",
			"approval_ttl_seconds":     1800,
			"changes_if_approved":      "Conflict decision",
			"scope": map[string]any{
				"workspace_id":     devManualSeedWorkspaceID,
				"run_id":           runID,
				"stage_id":         devManualSeedStageID,
				"role_instance_id": devManualSeedRoleInstanceID,
				"action_kind":      "promotion",
			},
		},
	})
}

func writeBootstrapOpenSegmentForDevManualSeedTest(path string, segmentID string) error {
	segment := trustpolicy.AuditSegmentFilePayload{
		SchemaID:      "runecode.protocol.v0.AuditSegmentFile",
		SchemaVersion: "0.1.0",
		Header: trustpolicy.AuditSegmentHeader{
			Format:       "audit_segment_framed_v1",
			SegmentID:    segmentID,
			SegmentState: trustpolicy.AuditSegmentStateOpen,
			CreatedAt:    "2026-03-13T12:21:00Z",
			Writer:       "auditd",
		},
		Frames:          []trustpolicy.AuditSegmentRecordFrame{},
		LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateOpen, MarkedAt: "2026-03-13T12:21:00Z"},
	}
	b, err := json.Marshal(segment)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, canonical, 0o600)
}
