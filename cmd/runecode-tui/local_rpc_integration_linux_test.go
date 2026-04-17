//go:build linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestTUIRoutesUseRealLocalRPCBrokerContracts(t *testing.T) {
	listener, service, ledgerRoot, errCh := startTUILocalRPCServer(t)
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
	probeTUILocalRPCClient(t, service, ledgerRoot)
	assertTUIBrokerBackedRoutes(t)
}

func startTUILocalRPCServer(t *testing.T) (*brokerapi.LocalIPCListener, *brokerapi.Service, string, <-chan error) {
	t.Helper()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	service, ledgerRoot := newTUILocalRPCService(t)
	listener, err := brokerapi.ListenLocalIPC(brokerapi.LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	configureTUILocalRPCClient(t, runtimeDir)
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveTUILocalRPCConn(t, listener, service)
	}()
	return listener, service, ledgerRoot, errCh
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

func probeTUILocalRPCClient(t *testing.T, service *brokerapi.Service, ledgerRoot string) {
	t.Helper()
	client := &rpcBrokerClient{}
	assertTUILocalRPCRunList(t, client)
	assertTUILocalRPCReadSurfaces(t, client)
	assertTUILocalRPCAnchorFlow(t, client, service, ledgerRoot)
}

func assertTUILocalRPCRunList(t *testing.T, client *rpcBrokerClient) {
	t.Helper()
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
}

func assertTUILocalRPCReadSurfaces(t *testing.T, client *rpcBrokerClient) {
	t.Helper()
	if _, err := client.ReadinessGet(context.Background()); err != nil {
		t.Fatalf("ReadinessGet returned error: %v", err)
	}
	if _, err := client.VersionInfoGet(context.Background()); err != nil {
		t.Fatalf("VersionInfoGet returned error: %v", err)
	}
	if _, err := client.ApprovalList(context.Background(), 5); err != nil {
		t.Fatalf("ApprovalList returned error: %v", err)
	}
	if _, err := client.AuditTimeline(context.Background(), 5, ""); err != nil {
		t.Fatalf("AuditTimeline returned error: %v", err)
	}
	if _, err := client.AuditVerificationGet(context.Background(), 20); err != nil {
		t.Fatalf("AuditVerificationGet returned error: %v", err)
	}
	if _, err := client.AuditFinalizeVerify(context.Background()); err != nil {
		t.Fatalf("AuditFinalizeVerify returned error: %v", err)
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

func assertTUILocalRPCAnchorFlow(t *testing.T, client *rpcBrokerClient, service *brokerapi.Service, ledgerRoot string) {
	t.Helper()
	sealDigest := mustLatestSealDigestForTUILocalRPCProbe(t, service, ledgerRoot)
	failedResp, err := client.AuditAnchorSegment(context.Background(), brokerapi.AuditAnchorSegmentRequest{SealDigest: sealDigest})
	if err != nil {
		t.Fatalf("AuditAnchorSegment (missing presence) returned error: %v", err)
	}
	if strings.TrimSpace(failedResp.AnchoringStatus) != "failed" {
		t.Fatalf("AuditAnchorSegment (missing presence) status = %q, want failed", failedResp.AnchoringStatus)
	}
	if strings.TrimSpace(failedResp.FailureCode) != "anchor_request_invalid" {
		t.Fatalf("AuditAnchorSegment (missing presence) failure_code = %q, want anchor_request_invalid", failedResp.FailureCode)
	}
	presenceResp, err := client.AuditAnchorPresenceGet(context.Background(), brokerapi.AuditAnchorPresenceGetRequest{SealDigest: sealDigest})
	if err != nil {
		t.Fatalf("AuditAnchorPresenceGet returned error: %v", err)
	}
	preflightResp, err := client.AuditAnchorPreflightGet(context.Background(), brokerapi.AuditAnchorPreflightGetRequest{})
	if err != nil {
		t.Fatalf("AuditAnchorPreflightGet returned error: %v", err)
	}
	if preflightResp.LatestAnchorableSeal == nil {
		t.Fatal("AuditAnchorPreflightGet latest_anchorable_seal missing")
	}
	preflightSeal, err := preflightResp.LatestAnchorableSeal.SealDigest.Identity()
	if err != nil {
		t.Fatalf("AuditAnchorPreflightGet latest_anchorable_seal.seal_digest invalid: %v", err)
	}
	wantSeal, _ := sealDigest.Identity()
	if preflightSeal != wantSeal {
		t.Fatalf("AuditAnchorPreflightGet latest seal = %q, want %q", preflightSeal, wantSeal)
	}
	if !preflightResp.SignerReadiness.Ready {
		t.Fatalf("AuditAnchorPreflightGet signer readiness = not ready: %+v", preflightResp.SignerReadiness)
	}
	if strings.TrimSpace(presenceResp.PresenceMode) != "os_confirmation" {
		t.Fatalf("AuditAnchorPresenceGet presence_mode = %q, want os_confirmation", presenceResp.PresenceMode)
	}
	if presenceResp.PresenceAttestation == nil {
		t.Fatal("AuditAnchorPresenceGet expected presence_attestation for os_confirmation")
	}
	successResp, err := client.AuditAnchorSegment(context.Background(), brokerapi.AuditAnchorSegmentRequest{SealDigest: sealDigest, PresenceAttestation: presenceResp.PresenceAttestation})
	if err != nil {
		t.Fatalf("AuditAnchorSegment (with broker-owned presence) returned error: %v", err)
	}
	if strings.TrimSpace(successResp.AnchoringStatus) != "ok" {
		t.Fatalf("AuditAnchorSegment (with broker-owned presence) status = %q, want ok (failure_code=%q failure_message=%q)", successResp.AnchoringStatus, successResp.FailureCode, successResp.FailureMessage)
	}
}

func mustLatestSealDigestForTUILocalRPCProbe(t *testing.T, service *brokerapi.Service, ledgerRoot string) trustpolicy.Digest {
	t.Helper()
	if service == nil {
		t.Fatal("service is required")
	}
	entries, err := os.ReadDir(filepath.Join(ledgerRoot, "sidecar", "segment-seals"))
	if err != nil {
		t.Fatalf("ReadDir(segment-seals) returned error: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		hash := strings.TrimSuffix(name, ".json")
		digest := trustpolicy.Digest{HashAlg: "sha256", Hash: hash}
		if _, err := digest.Identity(); err == nil {
			return digest
		}
	}
	t.Fatal("no valid segment seal digest found in sidecar")
	return trustpolicy.Digest{}
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
		"feed: waiting for shell watch manager",
	)

	chat := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, recording)
	assertRouteOutputContainsAll(t, chat, routeChat,
		"Sessions: 1 active=session-tui",
		"Composer: idle",
	)
	assertRouteInspectorContainsAll(t, chat, routeChat,
		"Inspector",
		"Linked runs: run-tui",
	)

	runs := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, recording)
	assertRouteOutputContainsAll(t, runs, routeRuns,
		"backend_kind=unknown",
	)
	assertRouteInspectorContainsAll(t, runs, routeRuns,
		"Authoritative broker state (control-plane truth):",
		"Coordination summary:",
	)

	approvals := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, recording)
	assertRouteOutputContainsAll(t, approvals, routeApprovals,
		"Approval safety strip",
	)
	assertRouteInspectorContainsAll(t, approvals, routeApprovals,
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

	if !containsCall(recording.Calls(), "ArtifactRead") || !containsCall(recording.Calls(), "AuditVerificationGet") || !containsCall(recording.Calls(), "AuditRecordGet") {
		t.Fatalf("expected broker-backed route calls, got %v", recording.Calls())
	}
}

func assertRouteInspectorContainsAll(t *testing.T, model routeModel, id routeID, want ...string) {
	t.Helper()
	updated, cmd := model.Update(routeActivatedMsg{RouteID: id})
	if cmd == nil {
		t.Fatalf("route %s activation returned nil command", id)
	}
	loaded := cmd()
	updated, _ = updated.Update(loaded)
	inspector := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body
	for _, needle := range want {
		if !strings.Contains(inspector, needle) {
			t.Fatalf("route %s inspector missing %q: %s", id, needle, inspector)
		}
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
	view := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body
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
	for _, needle := range []string{"Audit safety strip", "Timeline paging: page=1 entries=1 has_next=no", "Verification posture:"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("audit view missing %q: %s", needle, view)
		}
	}
	if !strings.Contains(view, "Verification findings:") && !strings.Contains(view, "Verification findings (machine-readable):") {
		t.Fatalf("audit view missing verification findings section: %s", view)
	}
	updated, cmd = updated.Update(teaKey("enter"))
	if cmd == nil {
		t.Fatal("audit enter returned nil command")
	}
	updated, _ = updated.Update(cmd())
	view = updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body
	for _, needle := range []string{"Record family:", "Verification posture:", "Linked references:"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("audit drill-down view missing %q: %s", needle, view)
		}
	}
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
