package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestHelpAndUnknownCommand(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"--help"}, stdout, stderr); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage: runecode-broker") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--state-root path") {
		t.Fatalf("help output missing --state-root global option: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--audit-ledger-root path") {
		t.Fatalf("help output missing --audit-ledger-root global option: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "audit-anchor-segment") {
		t.Fatalf("help output missing audit-anchor-segment command: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "audit-finalize-verify") {
		t.Fatalf("help output missing audit-finalize-verify command: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "backend-posture-get") {
		t.Fatalf("help output missing backend-posture-get command: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "git-setup-get") {
		t.Fatalf("help output missing git-setup-get command: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "git-remote-mutation-prepare") {
		t.Fatalf("help output missing git-remote-mutation-prepare command: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "approval-resolve") {
		t.Fatalf("help output missing approval-resolve command: %q", stdout.String())
	}
	err := run([]string{"not-a-command"}, stdout, stderr)
	if err == nil {
		t.Fatal("expected usage error for unknown command")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("unknown command error type = %T, want *usageError", err)
	}
}

func TestGlobalStateAndLedgerRootsAreAppliedBeforeCommand(t *testing.T) {
	setBrokerServiceForTest(t)
	var captured brokerServiceRoots
	originalFactory := brokerServiceFactory
	brokerServiceFactory = func(roots brokerServiceRoots) (*brokerapi.Service, error) {
		captured = roots
		return newBrokerService(roots)
	}
	t.Cleanup(func() {
		brokerServiceFactory = originalFactory
	})

	stateRoot := filepath.Join(t.TempDir(), "state-root")
	ledgerRoot := filepath.Join(t.TempDir(), "audit-ledger-root")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"--state-root", stateRoot, "--audit-ledger-root", ledgerRoot, "run-list", "--limit", "1"}, stdout, stderr); err != nil {
		t.Fatalf("run with global roots returned error: %v", err)
	}
	if captured.stateRoot != stateRoot {
		t.Fatalf("captured state root = %q, want %q", captured.stateRoot, stateRoot)
	}
	if captured.auditLedgerRoot != ledgerRoot {
		t.Fatalf("captured audit ledger root = %q, want %q", captured.auditLedgerRoot, ledgerRoot)
	}
}

func TestDefaultCLICommandsDoNotStartLocalListener(t *testing.T) {
	setBrokerServiceForTest(t)
	originalListen := localIPCListen
	localIPCListen = func(_ brokerapi.LocalIPCConfig) (*brokerapi.LocalIPCListener, error) {
		t.Fatal("localIPCListen should not be called for non-serve-local commands")
		return nil, nil
	}
	t.Cleanup(func() { localIPCListen = originalListen })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"list-artifacts"}, stdout, stderr); err != nil {
		t.Fatalf("list-artifacts returned error: %v", err)
	}
}

func TestServeLocalUsesLocalIPCListener(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("serve-local IPC peer-credential path is linux-only")
	}
	setBrokerServiceForTest(t)
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	clientErr, clientDone := startServeLocalClientProbe(t, runtimeDir)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"serve-local", "--once", "--runtime-dir", runtimeDir}, stdout, stderr); err != nil {
		t.Fatalf("serve-local --once returned error: %v", err)
	}
	awaitServeLocalClientProbe(t, clientErr, clientDone)
	if !strings.Contains(stdout.String(), "local broker api listening") {
		t.Fatalf("serve-local output = %q, want listening banner", stdout.String())
	}
}

func TestServeLocalArtifactReadRejectsRangeWithTypedCode(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("serve-local IPC peer-credential path is linux-only")
	}
	setBrokerServiceForTest(t)
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	clientErr := make(chan error, 1)
	go func() {
		for i := 0; i < 200; i++ {
			socketPath := filepath.Join(runtimeDir, "broker.sock")
			conn, err := net.Dial("unix", socketPath)
			if err == nil {
				clientErr <- requestArtifactReadRangeViaLocalRPC(t, conn)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		clientErr <- fmt.Errorf("failed to connect to serve-local socket")
	}()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"serve-local", "--once", "--runtime-dir", runtimeDir}, stdout, stderr); err != nil {
		t.Fatalf("serve-local --once returned error: %v", err)
	}
	if err := <-clientErr; err != nil {
		t.Fatalf("serve-local artifact-read range probe failed: %v", err)
	}
}

func TestServeLocalRejectsOversizedRequest(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("serve-local IPC peer-credential path is linux-only")
	}
	setBrokerServiceForTest(t)
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	clientErr := make(chan error, 1)
	go func() {
		for i := 0; i < 200; i++ {
			socketPath := filepath.Join(runtimeDir, "broker.sock")
			conn, err := net.Dial("unix", socketPath)
			if err == nil {
				clientErr <- requestOversizedPayloadViaLocalRPC(t, conn)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		clientErr <- fmt.Errorf("failed to connect to serve-local socket")
	}()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"serve-local", "--once", "--runtime-dir", runtimeDir}, stdout, stderr); err != nil {
		t.Fatalf("serve-local --once returned error: %v", err)
	}
	if err := <-clientErr; err != nil {
		t.Fatalf("serve-local oversized payload probe failed: %v", err)
	}
}

func startServeLocalClientProbe(t *testing.T, runtimeDir string) (<-chan error, <-chan struct{}) {
	t.Helper()
	clientErr := make(chan error, 1)
	clientDone := make(chan struct{}, 1)
	go func() {
		probeServeLocalClient(t, runtimeDir, clientErr, clientDone)
	}()
	return clientErr, clientDone
}

func awaitServeLocalClientProbe(t *testing.T, clientErr <-chan error, clientDone <-chan struct{}) {
	t.Helper()
	select {
	case err := <-clientErr:
		t.Fatalf("serve-local client request failed: %v", err)
	case <-clientDone:
	case <-time.After(3 * time.Second):
		t.Fatal("serve-local client did not complete in time")
	}
}

func probeServeLocalClient(t *testing.T, runtimeDir string, clientErr chan<- error, clientDone chan<- struct{}) {
	t.Helper()
	for i := 0; i < 200; i++ {
		socketPath := filepath.Join(runtimeDir, "broker.sock")
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			if err := requestRunListViaLocalRPC(t, conn); err != nil {
				clientErr <- err
				return
			}
			clientDone <- struct{}{}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	clientErr <- fmt.Errorf("failed to connect to serve-local socket")
}

func requestRunListViaLocalRPC(t *testing.T, conn net.Conn) error {
	t.Helper()
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	wire := localRPCRequest{Operation: "run_list", Request: mustJSONRawMessage(t, brokerapi.RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-serve-local-test",
		Limit:         10,
	})}
	if err := encoder.Encode(wire); err != nil {
		return err
	}
	resp := localRPCResponse{}
	if err := decoder.Decode(&resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("rpc failed: %+v", resp.Error)
	}
	if len(resp.Response) == 0 {
		return fmt.Errorf("rpc succeeded with empty response payload")
	}
	runList := brokerapi.RunListResponse{}
	if err := json.Unmarshal(resp.Response, &runList); err != nil {
		return err
	}
	if runList.RequestID != "req-serve-local-test" {
		return fmt.Errorf("run_list request_id = %q, want req-serve-local-test", runList.RequestID)
	}
	return nil
}

func requestSessionSendMessageViaLocalRPC(t *testing.T, conn net.Conn) error {
	t.Helper()
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	sessionID, err := sessionIDForLocalRPCMessageTest(t, encoder, decoder)
	if err != nil {
		return err
	}
	ack, err := sendSessionMessageViaLocalRPC(t, encoder, decoder, sessionID)
	if err != nil {
		return err
	}
	return assertSessionSendAck(ack)
}

func sessionIDForLocalRPCMessageTest(t *testing.T, encoder *json.Encoder, decoder *json.Decoder) (string, error) {
	t.Helper()
	seedReq := localRPCRequest{Operation: "session_list", Request: mustJSONRawMessage(t, brokerapi.SessionListRequest{
		SchemaID:      "runecode.protocol.v0.SessionListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-session-seed",
		Limit:         1,
	})}
	if err := encoder.Encode(seedReq); err != nil {
		return "", err
	}
	seedResp := localRPCResponse{}
	if err := decoder.Decode(&seedResp); err != nil {
		return "", err
	}
	if !seedResp.OK {
		return "", fmt.Errorf("session_list failed: %+v", seedResp.Error)
	}
	list := brokerapi.SessionListResponse{}
	if err := json.Unmarshal(seedResp.Response, &list); err != nil {
		return "", err
	}
	if len(list.Sessions) == 0 {
		return "", fmt.Errorf("session_list returned no sessions")
	}
	return list.Sessions[0].Identity.SessionID, nil
}

func sendSessionMessageViaLocalRPC(t *testing.T, encoder *json.Encoder, decoder *json.Decoder, sessionID string) (brokerapi.SessionSendMessageResponse, error) {
	t.Helper()
	wire := localRPCRequest{Operation: "session_send_message", Request: mustJSONRawMessage(t, brokerapi.SessionSendMessageRequest{
		SchemaID:       "runecode.protocol.v0.SessionSendMessageRequest",
		SchemaVersion:  "0.1.0",
		RequestID:      "req-session-send",
		SessionID:      sessionID,
		Role:           "user",
		ContentText:    "hello",
		IdempotencyKey: "idem-local-rpc",
	})}
	if err := encoder.Encode(wire); err != nil {
		return brokerapi.SessionSendMessageResponse{}, err
	}
	resp := localRPCResponse{}
	if err := decoder.Decode(&resp); err != nil {
		return brokerapi.SessionSendMessageResponse{}, err
	}
	if !resp.OK {
		return brokerapi.SessionSendMessageResponse{}, fmt.Errorf("session_send_message failed: %+v", resp.Error)
	}
	ack := brokerapi.SessionSendMessageResponse{}
	if err := json.Unmarshal(resp.Response, &ack); err != nil {
		return brokerapi.SessionSendMessageResponse{}, err
	}
	return ack, nil
}

func assertSessionSendAck(ack brokerapi.SessionSendMessageResponse) error {
	if ack.EventType != "session_message_ack" {
		return fmt.Errorf("event_type = %q, want session_message_ack", ack.EventType)
	}
	if ack.StreamID == "" || ack.Seq < 1 {
		return fmt.Errorf("invalid ack stream metadata: stream_id=%q seq=%d", ack.StreamID, ack.Seq)
	}
	if ack.Message.ContentText != "hello" {
		return fmt.Errorf("message content_text = %q, want hello", ack.Message.ContentText)
	}
	return nil
}

func requestArtifactReadRangeViaLocalRPC(t *testing.T, conn net.Conn) error {
	t.Helper()
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	wire := localRPCRequest{Operation: "artifact_read", Request: mustJSONRawMessage(t, map[string]any{
		"schema_id":      "runecode.protocol.v0.ArtifactReadRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-art-range",
		"digest":         "sha256:" + strings.Repeat("a", 64),
		"producer_role":  "workspace",
		"consumer_role":  "model_gateway",
		"range_start":    0,
	})}
	if err := encoder.Encode(wire); err != nil {
		return err
	}
	resp := localRPCResponse{}
	if err := decoder.Decode(&resp); err != nil {
		return err
	}
	if resp.OK {
		return fmt.Errorf("expected artifact_read range rejection")
	}
	if resp.Error == nil {
		return fmt.Errorf("expected typed error envelope")
	}
	if resp.Error.Error.Code != "broker_validation_range_not_supported" {
		return fmt.Errorf("error code = %q, want broker_validation_range_not_supported", resp.Error.Error.Code)
	}
	return nil
}

func TestServeLocalSessionSendMessageReturnsTypedAck(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("serve-local IPC peer-credential path is linux-only")
	}
	setBrokerServiceForTest(t)
	seedService, err := brokerServiceFactory(defaultBrokerServiceRoots())
	if err != nil {
		t.Fatalf("brokerServiceFactory seed returned error: %v", err)
	}
	if _, err := seedService.Put(artifacts.PutRequest{Payload: []byte("seed"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace", RunID: "run-session-send", StepID: "step-1"}); err != nil {
		t.Fatalf("seed Put returned error: %v", err)
	}
	if err := seedService.RecordRuntimeFacts("run-session-send", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-send", SessionID: "sess-send-cli"}}); err != nil {
		t.Fatalf("seed RecordRuntimeFacts returned error: %v", err)
	}
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	clientErr := make(chan error, 1)
	go func() {
		for i := 0; i < 200; i++ {
			socketPath := filepath.Join(runtimeDir, "broker.sock")
			conn, err := net.Dial("unix", socketPath)
			if err == nil {
				clientErr <- requestSessionSendMessageViaLocalRPC(t, conn)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		clientErr <- fmt.Errorf("failed to connect to serve-local socket")
	}()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"serve-local", "--once", "--runtime-dir", runtimeDir}, stdout, stderr); err != nil {
		t.Fatalf("serve-local --once returned error: %v", err)
	}
	if err := <-clientErr; err != nil {
		t.Fatalf("serve-local session-send-message probe failed: %v", err)
	}
}

func requestOversizedPayloadViaLocalRPC(t *testing.T, conn net.Conn) error {
	t.Helper()
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	oversized := strings.Repeat("a", (1<<20)+1024)
	wire := localRPCRequest{Operation: "run_list", Request: mustJSONRawMessage(t, map[string]any{
		"schema_id":      "runecode.protocol.v0.RunListRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-oversized",
		"cursor":         oversized,
		"limit":          1,
	})}
	if err := encoder.Encode(wire); err != nil {
		return err
	}
	resp := localRPCResponse{}
	if err := decoder.Decode(&resp); err != nil {
		return err
	}
	if resp.OK {
		return fmt.Errorf("expected oversized request rejection")
	}
	if resp.Error == nil {
		return fmt.Errorf("expected typed error envelope")
	}
	if resp.Error.Error.Code != "broker_limit_message_size_exceeded" {
		return fmt.Errorf("error code = %q, want broker_limit_message_size_exceeded", resp.Error.Error.Code)
	}
	return nil
}

func TestDecodeWireErrorClassifiesOperationAndCancellation(t *testing.T) {
	unsupported := decodeWireError("req-op", fmt.Errorf("unsupported operation %q", "not_real"))
	if unsupported == nil {
		t.Fatal("decodeWireError returned nil for unsupported operation")
	}
	if unsupported.Error.Code != "broker_validation_operation_invalid" {
		t.Fatalf("unsupported operation code = %q, want broker_validation_operation_invalid", unsupported.Error.Code)
	}
	if unsupported.Error.Message != "operation is not supported" {
		t.Fatalf("unsupported operation message = %q, want sanitized message", unsupported.Error.Message)
	}
	canceled := decodeWireError("req-cancel", context.Canceled)
	if canceled == nil {
		t.Fatal("decodeWireError returned nil for canceled context")
	}
	if canceled.Error.Code != "request_cancelled" {
		t.Fatalf("context canceled code = %q, want request_cancelled", canceled.Error.Code)
	}
	if !canceled.Error.Retryable {
		t.Fatal("request_cancelled should be retryable")
	}
	deadline := decodeWireError("req-deadline", context.DeadlineExceeded)
	if deadline == nil {
		t.Fatal("decodeWireError returned nil for deadline exceeded")
	}
	if deadline.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("deadline code = %q, want broker_timeout_request_deadline_exceeded", deadline.Error.Code)
	}

	structural := decodeWireError("req-structure", fmt.Errorf("message depth exceeds configured maximum"))
	if structural == nil {
		t.Fatal("decodeWireError returned nil for structural complexity error")
	}
	if structural.Error.Code != "broker_limit_structural_complexity_exceeded" {
		t.Fatalf("structural code = %q, want broker_limit_structural_complexity_exceeded", structural.Error.Code)
	}
	if structural.Error.Message != "request exceeds structural complexity limits" {
		t.Fatalf("structural message = %q, want sanitized message", structural.Error.Message)
	}
}

func TestDecodeLocalRPCRequestRejectsTrailingJSON(t *testing.T) {
	line := []byte(`{"operation":"run_list","request":{"schema_id":"runecode.protocol.v0.RunListRequest","schema_version":"0.1.0","request_id":"req-trailing"}} {}`)
	_, err := decodeLocalRPCRequest(line)
	if err == nil {
		t.Fatal("decodeLocalRPCRequest expected trailing JSON rejection")
	}
}
