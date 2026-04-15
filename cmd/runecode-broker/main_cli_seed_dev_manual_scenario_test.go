package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestSeedDevManualScenarioCommandSeedsDeterministicBrokerSurfaceState(t *testing.T) {
	if !brokerapi.DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	setBrokerServiceForTest(t)
	t.Setenv("RUNECODE_DEV_MODE", "1")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	result := runSeedDevManualScenario(t, stdout, stderr)
	if result.Profile != "tui-rich-v1" {
		t.Fatalf("profile = %q, want tui-rich-v1", result.Profile)
	}
	if result.RunID == "" || result.SessionID == "" || result.ApprovalID == "" || result.AuditRecordDigest == "" {
		t.Fatalf("seed result missing required identifiers: %+v", result)
	}
	if len(result.ArtifactDigests) < 3 {
		t.Fatalf("artifact digests len = %d, want >=3", len(result.ArtifactDigests))
	}
	runs := decodeCLIJSON[[]brokerapi.RunSummary](t, stdout, stderr, []string{"run-list", "--limit", "20"}, "run-list")
	if !containsRunID(runs, result.RunID) {
		t.Fatalf("run-list missing seeded run %q", result.RunID)
	}
	sessions := decodeCLIJSON[[]brokerapi.SessionSummary](t, stdout, stderr, []string{"session-list", "--limit", "20"}, "session-list")
	if !containsSessionID(sessions, result.SessionID) {
		t.Fatalf("session-list missing seeded session %q", result.SessionID)
	}
	approvals := decodeCLIJSON[[]brokerapi.ApprovalSummary](t, stdout, stderr, []string{"approval-list", "--limit", "20"}, "approval-list")
	if !containsApprovalID(approvals, result.ApprovalID) {
		t.Fatalf("approval-list missing seeded approval %q", result.ApprovalID)
	}
	auditSurface := decodeCLIJSON[brokerapi.AuditVerificationSurface](t, stdout, stderr, []string{"audit-verification", "--limit", "20"}, "audit-verification")
	if len(auditSurface.Views) == 0 {
		t.Fatal("audit-verification views empty, want seeded record")
	}
	runEvents := decodeCLIJSON[[]brokerapi.RunWatchEvent](t, stdout, stderr, []string{"run-watch", "--include-snapshot", "--follow"}, "run-watch")
	if len(runEvents) < 2 {
		t.Fatalf("run-watch events len = %d, want >=2", len(runEvents))
	}
	sessionEvents := decodeCLIJSON[[]brokerapi.SessionWatchEvent](t, stdout, stderr, []string{"session-watch", "--include-snapshot", "--follow"}, "session-watch")
	if len(sessionEvents) < 2 {
		t.Fatalf("session-watch events len = %d, want >=2", len(sessionEvents))
	}
	approvalEvents := decodeCLIJSON[[]brokerapi.ApprovalWatchEvent](t, stdout, stderr, []string{"approval-watch", "--include-snapshot", "--follow"}, "approval-watch")
	if len(approvalEvents) < 2 {
		t.Fatalf("approval-watch events len = %d, want >=2", len(approvalEvents))
	}
	logEvents := decodeCLIJSON[[]brokerapi.LogStreamEvent](t, stdout, stderr, []string{"stream-logs", "--run-id", result.RunID, "--include-backlog", "--follow"}, "stream-logs")
	if len(logEvents) < 3 {
		t.Fatalf("stream-logs events len = %d, want >=3", len(logEvents))
	}
}

func TestSeedDevManualScenarioRequiresDevOnlyAck(t *testing.T) {
	if !brokerapi.DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	setBrokerServiceForTest(t)
	t.Setenv("RUNECODE_DEV_MODE", "1")
	err := run([]string{"seed-dev-manual-scenario"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("seed-dev-manual-scenario expected usage error when --dev-only is missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("seed-dev-manual-scenario error type = %T, want *usageError", err)
	}
}

func TestSeedDevManualScenarioRequiresExplicitDevModeEnv(t *testing.T) {
	if !brokerapi.DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	setBrokerServiceForTest(t)
	err := run([]string{"seed-dev-manual-scenario", "--dev-only"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("seed-dev-manual-scenario expected dev mode guard error")
	}
	if got := err.Error(); got != "seed-dev-manual-scenario failed: dev manual seeding requires RUNECODE_DEV_MODE=1" {
		t.Fatalf("seed-dev-manual-scenario error = %q", got)
	}
}

func TestSeedDevManualScenarioIsIdempotentForPolicyAndPrimaryIdentifiers(t *testing.T) {
	if !brokerapi.DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	setBrokerServiceForTest(t)
	t.Setenv("RUNECODE_DEV_MODE", "1")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	first := runSeedDevManualScenario(t, stdout, stderr)
	second := runSeedDevManualScenario(t, stdout, stderr)
	if first.RunID != second.RunID || first.SessionID != second.SessionID || first.ApprovalID != second.ApprovalID || first.AuditRecordDigest != second.AuditRecordDigest {
		t.Fatalf("seed identifiers changed across repeated runs: first=%+v second=%+v", first, second)
	}
	service, err := brokerServiceFactory(defaultBrokerServiceRoots())
	if err != nil {
		t.Fatalf("brokerServiceFactory returned error: %v", err)
	}
	policy := service.Policy()
	count := 0
	for _, rule := range policy.FlowMatrix {
		if rule.ProducerRole == "workspace" && rule.ConsumerRole == "model_gateway" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("workspace->model_gateway flow rule count = %d, want 1 after repeated seeding", count)
	}
}

func TestSeedDevManualScenarioUnavailableWhenBuildTagDisabled(t *testing.T) {
	if brokerapi.DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is enabled in this build")
	}
	setBrokerServiceForTest(t)
	t.Setenv("RUNECODE_DEV_MODE", "1")
	err := run([]string{"seed-dev-manual-scenario", "--dev-only"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("seed-dev-manual-scenario expected build-disabled usage error")
	}
	uerr, ok := err.(*usageError)
	if !ok {
		t.Fatalf("seed-dev-manual-scenario error type = %T, want *usageError", err)
	}
	if uerr.Error() != "seed-dev-manual-scenario unavailable in this build" {
		t.Fatalf("seed-dev-manual-scenario error = %q", uerr.Error())
	}
}

func runSeedDevManualScenario(t *testing.T, stdout, stderr *bytes.Buffer) brokerapi.DevManualSeedResult {
	t.Helper()
	return decodeCLIJSON[brokerapi.DevManualSeedResult](t, stdout, stderr, []string{"seed-dev-manual-scenario", "--dev-only"}, "seed-dev-manual-scenario")
}

func decodeCLIJSON[T any](t *testing.T, stdout, stderr *bytes.Buffer, args []string, label string) T {
	t.Helper()
	stdout.Reset()
	if err := run(args, stdout, stderr); err != nil {
		t.Fatalf("%s returned error: %v", label, err)
	}
	var out T
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("%s parse error: %v", label, err)
	}
	return out
}

func containsRunID(items []brokerapi.RunSummary, runID string) bool {
	for _, item := range items {
		if item.RunID == runID {
			return true
		}
	}
	return false
}

func containsSessionID(items []brokerapi.SessionSummary, sessionID string) bool {
	for _, item := range items {
		if item.Identity.SessionID == sessionID {
			return true
		}
	}
	return false
}

func containsApprovalID(items []brokerapi.ApprovalSummary, approvalID string) bool {
	for _, item := range items {
		if item.ApprovalID == approvalID {
			return true
		}
	}
	return false
}
