//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestTUIRoutesUseRealLocalRPCBrokerContracts(t *testing.T) {
	listener, errCh := startTUILocalRPCServer(t)
	defer func() {
		_ = listener.Close()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("local RPC server returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for local RPC server shutdown")
		}
	}()
	probeTUILocalRPCClient(t)
	assertTUIBrokerBackedRoutes(t)
}

func startTUILocalRPCServer(t *testing.T) (*brokerapi.LocalIPCListener, <-chan error) {
	t.Helper()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	service := newTUILocalRPCService(t)
	listener, err := brokerapi.ListenLocalIPC(brokerapi.LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	configureTUILocalRPCClient(t, runtimeDir)
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveTUILocalRPCConn(t, listener, service)
	}()
	return listener, errCh
}

func configureTUILocalRPCClient(t *testing.T, runtimeDir string) {
	t.Helper()
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"}, nil
	}
	localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
		return brokerapi.DialLocalRPC(ctx, cfg)
	}
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})
}

func probeTUILocalRPCClient(t *testing.T) {
	t.Helper()
	client := &rpcBrokerClient{}
	resp, err := client.RunList(context.Background(), 5)
	if err != nil {
		t.Fatalf("RunList returned error: %v", err)
	}
	if len(resp.Runs) == 0 {
		t.Fatal("RunList returned no runs")
	}
	if resp.Runs[0].RunID != "run-tui" {
		t.Fatalf("RunList first run_id = %q, want run-tui", resp.Runs[0].RunID)
	}
	if _, err := client.ReadinessGet(context.Background()); err != nil {
		t.Fatalf("ReadinessGet returned error: %v", err)
	}
	if _, err := client.VersionInfoGet(context.Background()); err != nil {
		t.Fatalf("VersionInfoGet returned error: %v", err)
	}
	if _, err := client.ApprovalList(context.Background(), 5); err != nil {
		t.Fatalf("ApprovalList returned error: %v", err)
	}
	if _, err := client.AuditVerificationGet(context.Background(), 20); err != nil {
		t.Fatalf("AuditVerificationGet returned error: %v", err)
	}
	if _, err := client.RunWatch(context.Background(), brokerapi.RunWatchRequest{StreamID: "probe-run-watch", IncludeSnapshot: true}); err != nil {
		t.Fatalf("RunWatch returned error: %v", err)
	}
	if _, err := client.ApprovalWatch(context.Background(), brokerapi.ApprovalWatchRequest{StreamID: "probe-approval-watch", IncludeSnapshot: true}); err != nil {
		t.Fatalf("ApprovalWatch returned error: %v", err)
	}
	if _, err := client.SessionWatch(context.Background(), brokerapi.SessionWatchRequest{StreamID: "probe-session-watch", IncludeSnapshot: true}); err != nil {
		t.Fatalf("SessionWatch returned error: %v", err)
	}
}

func assertTUIBrokerBackedRoutes(t *testing.T) {
	t.Helper()
	recording := newRecordingBrokerClient(&rpcBrokerClient{})

	dashboard := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, recording)
	assertRouteOutputContainsAll(t, dashboard, routeDashboard,
		"Now",
		"Safety strip",
		"backend_kind=unknown",
		"Live Activity",
		"Live activity (typed watch families; logs are supplemental inspection only):",
		"totals events=2 snapshot=1 upsert=0 terminal=1 errors=0",
		"last_event=run_watch_terminal subject=run-tui status=completed",
	)

	chat := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, recording)
	assertRouteOutputContainsAll(t, chat, routeChat,
		"Sessions: 1 active=session-tui",
		"Composer: idle",
		"Inspector",
		"Linked runs: run-tui",
	)

	runs := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, recording)
	assertRouteOutputContainsAll(t, runs, routeRuns,
		"backend_kind=unknown",
		"Authoritative broker state (control-plane truth):",
		"Coordination summary:",
	)

	approvals := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, recording)
	assertRouteOutputContainsAll(t, approvals, routeApprovals,
		"Approval safety strip",
		"Approval trigger code:",
		"Canonical bound identity:",
	)

	artifacts := newArtifactsRouteModel(routeDefinition{ID: routeArtifacts, Label: "Artifacts"}, recording)
	assertArtifactsRouteRedactsDiffContent(t, artifacts)

	audit := newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, recording)
	assertAuditRouteSupportsDrillDown(t, audit)

	status := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, recording)
	assertRouteOutputContainsAll(t, status, routeStatus,
		"Runtime/audit readiness strip",
		"Broker ready=true local_only=true",
		"Protocol posture:",
	)

	if !containsCall(recording.Calls(), "RunWatch") || !containsCall(recording.Calls(), "ArtifactRead") || !containsCall(recording.Calls(), "AuditVerificationGet") || !containsCall(recording.Calls(), "AuditRecordGet") {
		t.Fatalf("expected broker-backed route calls, got %v", recording.Calls())
	}
}

func assertRouteOutputContainsAll(t *testing.T, model routeModel, id routeID, want ...string) {
	t.Helper()
	updated, cmd := model.Update(routeActivatedMsg{RouteID: id})
	if cmd == nil {
		t.Fatalf("route %s activation returned nil command", id)
	}
	loaded := cmd()
	updated, _ = updated.Update(loaded)
	view := updated.View(120, 40, focusContent)
	for _, needle := range want {
		if !strings.Contains(view, needle) {
			t.Fatalf("route %s view missing %q: %s", id, needle, view)
		}
	}
}

func assertArtifactsRouteRedactsDiffContent(t *testing.T, model routeModel) {
	t.Helper()
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeArtifacts})
	if cmd == nil {
		t.Fatal("artifacts activation returned nil command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(teaKey("j"))
	updated, cmd = updated.Update(teaKey("enter"))
	if cmd == nil {
		t.Fatal("artifacts enter returned nil command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	for _, needle := range []string{"Typed detail mode:", "diff content unavailable: broker_limit_policy_rejected"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("artifacts view missing %q: %s", needle, view)
		}
	}
}

func assertAuditRouteSupportsDrillDown(t *testing.T, model routeModel) {
	t.Helper()
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAudit})
	if cmd == nil {
		t.Fatal("audit activation returned nil command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	for _, needle := range []string{"Audit safety strip", "Timeline paging: page=1 entries=1 has_next=no", "Verification posture: integrity=ok", "Verification findings: none"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("audit view missing %q: %s", needle, view)
		}
	}
	updated, cmd = updated.Update(teaKey("enter"))
	if cmd == nil {
		t.Fatal("audit enter returned nil command")
	}
	updated, _ = updated.Update(cmd())
	view = updated.View(120, 40, focusContent)
	for _, needle := range []string{"Record family:", "Verification posture:", "Linked references:"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("audit drill-down view missing %q: %s", needle, view)
		}
	}
}

func teaKey(key string) tea.KeyMsg {
	if len(key) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func serveTUILocalRPCConn(t *testing.T, listener *brokerapi.LocalIPCListener, service *brokerapi.Service) error {
	t.Helper()
	for {
		conn, err := listener.Listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		if err := serveTUIRPCConnLoop(conn, service); err != nil {
			return err
		}
	}
}

func serveTUIRPCConnLoop(conn net.Conn, service *brokerapi.Service) error {
	defer conn.Close()
	// Production local IPC derives peer identity from the transport; this test
	// server uses fixed metadata only to exercise the TUI's typed broker calls.
	meta := brokerapi.RequestContext{ClientID: "test-client", LaneID: "local_ipc"}
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	for {
		var wire brokerapi.LocalRPCRequest
		if err := decoder.Decode(&wire); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := validateTUIRawMessageLimits(service, wire.Request); err != nil {
			if encodeErr := encoder.Encode(brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		response := dispatchTUILocalRPC(service, wire, meta)
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
}

func dispatchTUILocalRPC(service *brokerapi.Service, wire brokerapi.LocalRPCRequest, meta brokerapi.RequestContext) brokerapi.LocalRPCResponse {
	if response, ok := dispatchTUILocalRPCJSONOps(service, wire, meta); ok {
		return response
	}
	switch wire.Operation {
	case "artifact_read":
		return dispatchTUIArtifactRead(service, wire.Request, meta)
	case "run_watch":
		return dispatchTUIRunWatch(service, wire.Request, meta)
	case "approval_watch":
		return dispatchTUIApprovalWatch(service, wire.Request, meta)
	case "session_watch":
		return dispatchTUISessionWatch(service, wire.Request, meta)
	default:
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(fmt.Errorf("unsupported operation %q", wire.Operation))}
	}
}

func dispatchTUILocalRPCJSONOps(service *brokerapi.Service, wire brokerapi.LocalRPCRequest, meta brokerapi.RequestContext) (brokerapi.LocalRPCResponse, bool) {
	switch wire.Operation {
	case "run_list":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleRunList), true
	case "run_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleRunGet), true
	case "session_list":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleSessionList), true
	case "session_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleSessionGet), true
	case "session_send_message":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleSessionSendMessage), true
	case "approval_list":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleApprovalList), true
	case "approval_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleApprovalGet), true
	case "approval_resolve":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleApprovalResolve), true
	case "artifact_list":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleArtifactListV0), true
	case "artifact_head":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleArtifactHeadV0), true
	case "audit_timeline":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleAuditTimeline), true
	case "audit_verification_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleAuditVerificationGet), true
	case "audit_record_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleAuditRecordGet), true
	case "readiness_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleReadinessGet), true
	case "version_info_get":
		return dispatchTUILocalRPCJSON(service, wire.Request, meta, (*brokerapi.Service).HandleVersionInfoGet), true
	default:
		return brokerapi.LocalRPCResponse{}, false
	}
}

func dispatchTUILocalRPCJSON[Req any, Resp any](
	service *brokerapi.Service,
	raw json.RawMessage,
	meta brokerapi.RequestContext,
	handler func(*brokerapi.Service, context.Context, Req, brokerapi.RequestContext) (Resp, *brokerapi.ErrorResponse),
) brokerapi.LocalRPCResponse {
	var req Req
	if err := json.Unmarshal(raw, &req); err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	resp, errResp := handler(service, context.Background(), req, meta)
	if errResp != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: errResp}
	}
	return encodeTUILocalRPCResponse(resp)
}

func dispatchTUIArtifactRead(service *brokerapi.Service, raw json.RawMessage, meta brokerapi.RequestContext) brokerapi.LocalRPCResponse {
	req := brokerapi.ArtifactReadRequest{}
	if err := json.Unmarshal(raw, &req); err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	handle, errResp := service.HandleArtifactRead(context.Background(), req, meta)
	if errResp != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamArtifactReadEvents(handle)
	if err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	return encodeTUILocalRPCResponse(events)
}

func dispatchTUIRunWatch(service *brokerapi.Service, raw json.RawMessage, meta brokerapi.RequestContext) brokerapi.LocalRPCResponse {
	req := brokerapi.RunWatchRequest{}
	if err := json.Unmarshal(raw, &req); err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	ack, errResp := service.HandleRunWatchRequest(context.Background(), req, meta)
	if errResp != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamRunWatchEvents(ack)
	if err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	return encodeTUILocalRPCResponse(events)
}

func dispatchTUIApprovalWatch(service *brokerapi.Service, raw json.RawMessage, meta brokerapi.RequestContext) brokerapi.LocalRPCResponse {
	req := brokerapi.ApprovalWatchRequest{}
	if err := json.Unmarshal(raw, &req); err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	ack, errResp := service.HandleApprovalWatchRequest(context.Background(), req, meta)
	if errResp != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamApprovalWatchEvents(ack)
	if err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	return encodeTUILocalRPCResponse(events)
}

func dispatchTUISessionWatch(service *brokerapi.Service, raw json.RawMessage, meta brokerapi.RequestContext) brokerapi.LocalRPCResponse {
	req := brokerapi.SessionWatchRequest{}
	if err := json.Unmarshal(raw, &req); err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	ack, errResp := service.HandleSessionWatchRequest(context.Background(), req, meta)
	if errResp != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamSessionWatchEvents(ack)
	if err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	return encodeTUILocalRPCResponse(events)
}

func encodeTUILocalRPCResponse(resp any) brokerapi.LocalRPCResponse {
	raw, err := json.Marshal(resp)
	if err != nil {
		return brokerapi.LocalRPCResponse{OK: false, Error: tuiRPCError(err)}
	}
	return brokerapi.LocalRPCResponse{OK: true, Response: json.RawMessage(raw)}
}

func validateTUIRawMessageLimits(service *brokerapi.Service, raw json.RawMessage) error {
	return brokerapi.ValidateRawMessageLimits(raw, service.APILimits())
}

func newTUILocalRPCService(t *testing.T) *brokerapi.Service {
	t.Helper()
	storeRoot := t.TempDir()
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForTUILocalRPCTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForTUILocalRPCTest returned error: %v", err)
	}
	service, err := brokerapi.NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	policy := artifacts.DefaultPolicy()
	policy.FlowMatrix = append(policy.FlowMatrix,
		artifacts.FlowRule{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []artifacts.DataClass{artifacts.DataClassDiffs, artifacts.DataClassBuildLogs, artifacts.DataClassGateEvidence, artifacts.DataClassAuditVerificationReport}},
	)
	if err := service.SetPolicy(policy); err != nil {
		t.Fatalf("SetPolicy returned error: %v", err)
	}

	seedTUIRunArtifacts(t, service)
	if err := service.RecordRuntimeFacts("run-tui", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-tui", SessionID: "session-tui"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	seedTUIApproval(t, service)
	seedTUIAudit(t, service)
	return service
}

func seedTUIRunArtifacts(t *testing.T, service *brokerapi.Service) {
	t.Helper()
	puts := []artifacts.PutRequest{
		{Payload: []byte("spec body"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-tui", StepID: "plan"},
		{Payload: []byte("diff --git a/file b/file\n+new line\ntoken=fixture-redaction-value-not-a-secret\n"), ContentType: "text/plain", DataClass: artifacts.DataClassDiffs, ProvenanceReceiptHash: "sha256:" + strings.Repeat("2", 64), CreatedByRole: "workspace", RunID: "run-tui", StepID: "apply"},
	}
	for _, req := range puts {
		if _, err := service.Put(req); err != nil {
			t.Fatalf("Put returned error: %v", err)
		}
	}
}

func seedTUIApproval(t *testing.T, service *brokerapi.Service) {
	t.Helper()
	if err := service.RecordPolicyDecision("run-tui", "", policyengine.PolicyDecision{
		SchemaID:                 "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:            "0.3.0",
		DecisionOutcome:          policyengine.DecisionRequireHumanApproval,
		PolicyReasonCode:         "approval_required",
		ManifestHash:             "sha256:" + strings.Repeat("1", 64),
		ActionRequestHash:        "sha256:" + strings.Repeat("7", 64),
		PolicyInputHashes:        []string{"sha256:" + strings.Repeat("2", 64)},
		RelevantArtifactHashes:   []string{"sha256:" + strings.Repeat("3", 64)},
		DetailsSchemaID:          "runecode.protocol.details.policy.evaluation.v0",
		Details:                  map[string]any{"precedence": "approval_profile_moderate"},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "excerpt_promotion",
			"approval_assurance_level": "moderate",
			"presence_mode":            "os_confirmation",
			"scope": map[string]any{
				"schema_id":      "runecode.protocol.v0.ApprovalBoundScope",
				"schema_version": "0.1.0",
				"workspace_id":   "ws-tui",
				"run_id":         "run-tui",
				"stage_id":       "stage-1",
				"action_kind":    "promotion",
			},
			"changes_if_approved":  "Promotion continues",
			"approval_ttl_seconds": 1800,
		},
	}); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	if len(service.ApprovalList()) == 0 {
		t.Fatal("expected broker-created pending approval from policy decision")
	}
}

func seedTUIAudit(t *testing.T, service *brokerapi.Service) {
	t.Helper()
	if err := service.AppendTrustedAuditEvent("run_state", "brokerapi", map[string]any{"run_id": "run-tui", "session_id": "session-tui", "event_summary": "Run state changed"}); err != nil {
		t.Fatalf("AppendTrustedAuditEvent returned error: %v", err)
	}
}

func seedLedgerForTUILocalRPCTest(root string) error {
	if err := prepareTUILedgerDirs(root); err != nil {
		return err
	}
	evidence, err := buildTUISeedEventEvidence("session-tui")
	if err != nil {
		return err
	}
	if err := writeTUISeedSegment(root, "segment-000001", evidence.recordDigest, evidence.canonicalEnvelope); err != nil {
		return err
	}
	if err := writeTUISeedSeal(root, "segment-000001", evidence.recordDigest, 0); err != nil {
		return err
	}
	ledger, err := auditd.Open(root)
	if err != nil {
		return err
	}
	if err := configureTUISeedContractsAndIndex(ledger); err != nil {
		return err
	}
	return persistTUISeedReport(ledger)
}

type tuiSeedEvidence struct {
	recordDigest      trustpolicy.Digest
	canonicalEnvelope []byte
}

func prepareTUILedgerDirs(root string) error {
	for _, path := range []string{filepath.Join(root, "segments"), filepath.Join(root, "sidecar", "segment-seals")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func buildTUISeedEventEvidence(sessionID string) (tuiSeedEvidence, error) {
	eventPayload := map[string]any{"session_id": sessionID}
	eventPayloadHash := sha256.Sum256(mustTUICanonicalJSON(eventPayload))
	event := tuiSeedAuditEventEnvelopePayload(sessionID, eventPayloadHash)
	envelope := tuiSeedSignedEventEnvelope(event)
	canonicalEnvelope := mustTUICanonicalJSON(envelope)
	sum := sha256.Sum256(canonicalEnvelope)
	return tuiSeedEvidence{recordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, canonicalEnvelope: canonicalEnvelope}, nil
}

func tuiSeedAuditEventEnvelopePayload(sessionID string, eventPayloadHash [32]byte) map[string]any {
	return map[string]any{
		"schema_id":               trustpolicy.AuditEventSchemaID,
		"schema_version":          trustpolicy.AuditEventSchemaVersion,
		"audit_event_type":        "isolate_session_bound",
		"emitter_stream_id":       "auditd-stream-1",
		"seq":                     1,
		"occurred_at":             "2026-03-13T12:15:00Z",
		"principal":               map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"event_payload_schema_id": trustpolicy.IsolateSessionBoundPayloadSchemaID,
		"event_payload": map[string]any{
			"schema_id":                        trustpolicy.IsolateSessionBoundPayloadSchemaID,
			"schema_version":                   trustpolicy.IsolateSessionBoundPayloadSchemaVersion,
			"run_id":                           "run-tui",
			"isolate_id":                       "isolate-1",
			"session_id":                       sessionID,
			"backend_kind":                     "microvm",
			"isolation_assurance_level":        "isolated",
			"provisioning_posture":             "tofu",
			"launch_context_digest":            "sha256:" + strings.Repeat("1", 64),
			"handshake_transcript_hash":        "sha256:" + strings.Repeat("2", 64),
			"session_binding_digest":           "sha256:" + strings.Repeat("3", 64),
			"runtime_image_descriptor_digest":  "sha256:" + strings.Repeat("4", 64),
			"applied_hardening_posture_digest": "sha256:" + strings.Repeat("5", 64),
		},
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": hex.EncodeToString(eventPayloadHash[:])},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": "ws-tui", "run_id": "run-tui", "stage_id": "stage-1"},
		"correlation":                   map[string]any{"session_id": sessionID, "operation_id": "op-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
}

func tuiSeedSignedEventEnvelope(event map[string]any) trustpolicy.SignedObjectEnvelope {
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditEventSchemaID, PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion, Payload: mustTUIJSON(event), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString([]byte("sig"))}}
}

func writeTUISeedSegment(root, segmentID string, recordDigest trustpolicy.Digest, canonicalEnvelope []byte) error {
	segment := trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: segmentID, SegmentState: trustpolicy.AuditSegmentStateSealed, CreatedAt: "2026-03-13T12:00:00Z", Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{{RecordDigest: recordDigest, ByteLength: int64(len(canonicalEnvelope)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelope)}}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateSealed, MarkedAt: "2026-03-13T12:20:00Z"}}
	return writeTUICanonicalJSON(filepath.Join(root, "segments", segmentID+".json"), segment)
}

func writeTUISeedSeal(root, segmentID string, recordDigest trustpolicy.Digest, chainIndex int64) error {
	sealPayload := trustpolicy.AuditSegmentSealPayload{SchemaID: trustpolicy.AuditSegmentSealSchemaID, SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, SegmentID: segmentID, SealedAfterState: trustpolicy.AuditSegmentStateOpen, SegmentState: trustpolicy.AuditSegmentStateSealed, SegmentCut: trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 2048, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow}, EventCount: 1, FirstRecordDigest: recordDigest, LastRecordDigest: recordDigest, MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1, MerkleRoot: recordDigest, SegmentFileHashScope: trustpolicy.AuditSegmentFileHashScopeRawFramedV1, SegmentFileHash: recordDigest, SealChainIndex: chainIndex, AnchoringSubject: trustpolicy.AuditSegmentAnchoringSubjectSeal, SealedAt: "2026-03-13T12:20:00Z", ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SealReason: "size_threshold"}
	sealEnvelope := trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditSegmentSealSchemaID, PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, Payload: mustTUIJSON(sealPayload), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString([]byte("sig"))}}
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(sealEnvelope)
	if err != nil {
		return err
	}
	identity, _ := sealDigest.Identity()
	return writeTUICanonicalJSON(filepath.Join(root, "sidecar", "segment-seals", strings.TrimPrefix(identity, "sha256:")+".json"), sealEnvelope)
}

func configureTUISeedContractsAndIndex(ledger *auditd.Ledger) error {
	if err := ledger.ConfigureVerificationInputs(auditd.VerificationConfiguration{VerifierRecords: []trustpolicy.VerifierRecord{tuiSeedVerifierRecord()}, EventContractCatalog: tuiSeedEventContractCatalog()}); err != nil {
		return err
	}
	_, err := ledger.BuildIndex()
	return err
}

func persistTUISeedReport(ledger *auditd.Ledger) error {
	report := trustpolicy.AuditVerificationReportPayload{SchemaID: trustpolicy.AuditVerificationReportSchemaID, SchemaVersion: trustpolicy.AuditVerificationReportSchemaVersion, VerifiedAt: time.Now().UTC().Format(time.RFC3339), VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: "segment-000001"}, CryptographicallyValid: true, HistoricallyAdmissible: true, CurrentlyDegraded: false, IntegrityStatus: trustpolicy.AuditVerificationStatusOK, AnchoringStatus: trustpolicy.AuditVerificationStatusOK, StoragePostureStatus: trustpolicy.AuditVerificationStatusOK, SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK, DegradedReasons: []string{}, HardFailures: []string{}, Findings: []trustpolicy.AuditVerificationFinding{}, Summary: "ok"}
	_, err := ledger.PersistVerificationReport(report)
	return err
}

func tuiSeedVerifierRecord() trustpolicy.VerifierRecord {
	publicKey := []byte(strings.Repeat("k", 32))
	keyID := sha256.Sum256(publicKey)
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: hex.EncodeToString(keyID[:]), Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
}

func tuiSeedEventContractCatalog() trustpolicy.AuditEventContractCatalog {
	return trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{trustpolicy.IsolateSessionBoundPayloadSchemaID}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
}

func mustTUIJSON(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return b
}

func mustTUICanonicalJSON(value any) []byte {
	b, err := jsoncanonicalizer.Transform(mustTUIJSON(value))
	if err != nil {
		panic(err)
	}
	return b
}

func writeTUICanonicalJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, canonical, 0o600)
}

func tuiRPCError(err error) *brokerapi.ErrorResponse {
	resp := brokerapi.ErrorResponse{
		SchemaID:      "runecode.protocol.v0.BrokerErrorResponse",
		SchemaVersion: localAPISchemaVersion,
		RequestID:     "tui-local-rpc-error",
		Error: brokerapi.ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: "0.3.0",
			Code:          "broker_validation_schema_invalid",
			Category:      "validation",
			Retryable:     false,
			Message:       "request validation failed",
		},
	}
	return &resp
}
