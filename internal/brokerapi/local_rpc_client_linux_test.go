//go:build linux

package brokerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestLocalRPCClientInvokeRunList(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	runtimeDir, client, done := setupLocalRPCRunListRoundTrip(t, service)
	defer client.Close()
	assertLocalRPCSocketPath(t, runtimeDir)

	resp := RunListResponse{}
	errResp := client.Invoke(context.Background(), "run_list", RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-local-client",
		Limit:         5,
	}, &resp)
	if errResp != nil {
		t.Fatalf("Invoke returned typed error: %+v", errResp)
	}
	if resp.RequestID != "req-local-client" {
		t.Fatalf("response request_id = %q, want req-local-client", resp.RequestID)
	}
	if err := <-done; err != nil {
		t.Fatalf("local rpc server returned error: %v", err)
	}
}

func TestLocalRPCClientInvokeSessionList(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, service, "run-session-client", "step-1")
	if err := service.RecordRuntimeFacts("run-session-client", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-client", SessionID: "sess-client"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	runtimeDir, client, done := setupLocalRPCSessionListRoundTrip(t, service)
	defer client.Close()
	assertLocalRPCSocketPath(t, runtimeDir)

	resp := SessionListResponse{}
	errResp := client.Invoke(context.Background(), "session_list", SessionListRequest{
		SchemaID:      "runecode.protocol.v0.SessionListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-local-session-list",
		Limit:         5,
	}, &resp)
	if errResp != nil {
		t.Fatalf("Invoke returned typed error: %+v", errResp)
	}
	if len(resp.Sessions) != 1 || resp.Sessions[0].Identity.SessionID != "sess-client" {
		t.Fatalf("session_list response = %+v, want sess-client", resp.Sessions)
	}
	if err := <-done; err != nil {
		t.Fatalf("local rpc server returned error: %v", err)
	}
}

func TestLocalRPCClientInvokeSessionSendMessage(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, service, "run-session-client-send", "step-1")
	if err := service.RecordRuntimeFacts("run-session-client-send", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-client-send", SessionID: "sess-client-send"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	runtimeDir, client, done := setupLocalRPCSessionSendMessageRoundTrip(t, service)
	defer client.Close()
	assertLocalRPCSocketPath(t, runtimeDir)

	resp := SessionSendMessageResponse{}
	errResp := client.Invoke(context.Background(), "session_send_message", SessionSendMessageRequest{
		SchemaID:      "runecode.protocol.v0.SessionSendMessageRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-local-session-send",
		SessionID:     "sess-client-send",
		Role:          "user",
		ContentText:   "hello",
	}, &resp)
	if errResp != nil {
		t.Fatalf("Invoke returned typed error: %+v", errResp)
	}
	if resp.EventType != "session_message_ack" {
		t.Fatalf("event_type = %q, want session_message_ack", resp.EventType)
	}
	if resp.StreamID == "" || resp.Seq < 1 {
		t.Fatalf("invalid stream metadata stream_id=%q seq=%d", resp.StreamID, resp.Seq)
	}
	if resp.Message.ContentText != "hello" {
		t.Fatalf("message content_text = %q, want hello", resp.Message.ContentText)
	}
	if err := <-done; err != nil {
		t.Fatalf("local rpc server returned error: %v", err)
	}
}

func TestLocalRPCClientInvokeSessionExecutionTrigger(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, service, "run-session-client-trigger", "step-1")
	if err := service.RecordRuntimeFacts("run-session-client-trigger", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-client-trigger", SessionID: "sess-client-trigger"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	runtimeDir, client, done := setupLocalRPCSessionExecutionTriggerRoundTrip(t, service)
	defer client.Close()
	assertLocalRPCSocketPath(t, runtimeDir)

	resp := SessionExecutionTriggerResponse{}
	errResp := client.Invoke(context.Background(), "session_execution_trigger", SessionExecutionTriggerRequest{
		SchemaID:               "runecode.protocol.v0.SessionExecutionTriggerRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-local-session-trigger",
		SessionID:              "sess-client-trigger",
		TriggerSource:          "interactive_user",
		RequestedOperation:     "start",
		WorkflowRouting:        &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"},
		UserMessageContentText: "hello",
	}, &resp)
	if errResp != nil {
		t.Fatalf("Invoke returned typed error: %+v", errResp)
	}
	if resp.EventType != "session_execution_trigger_ack" {
		t.Fatalf("event_type = %q, want session_execution_trigger_ack", resp.EventType)
	}
	if resp.TriggerID == "" {
		t.Fatal("trigger_id is empty")
	}
	if err := <-done; err != nil {
		t.Fatalf("local rpc server returned error: %v", err)
	}
}

func assertLocalRPCSocketPath(t *testing.T, runtimeDir string) {
	t.Helper()
	socketPath := filepath.Join(runtimeDir, "broker.sock")
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("expected socket at %q: %v", socketPath, err)
	}
}

func setupLocalRPCRunListRoundTrip(t *testing.T, service *Service) (string, *LocalRPCClient, chan error) {
	t.Helper()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Listener.Accept()
		if err != nil {
			done <- err
			return
		}
		done <- serveSingleLocalRPCConnForTest(conn, service)
	}()
	client, err := DialLocalRPC(context.Background(), LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("DialLocalRPC returned error: %v", err)
	}
	return runtimeDir, client, done
}

func setupLocalRPCSessionListRoundTrip(t *testing.T, service *Service) (string, *LocalRPCClient, chan error) {
	t.Helper()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Listener.Accept()
		if err != nil {
			done <- err
			return
		}
		done <- serveSingleLocalRPCSessionListConnForTest(conn, service)
	}()
	client, err := DialLocalRPC(context.Background(), LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("DialLocalRPC returned error: %v", err)
	}
	return runtimeDir, client, done
}

func setupLocalRPCSessionSendMessageRoundTrip(t *testing.T, service *Service) (string, *LocalRPCClient, chan error) {
	t.Helper()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Listener.Accept()
		if err != nil {
			done <- err
			return
		}
		done <- serveSingleLocalRPCSessionSendMessageConnForTest(conn, service)
	}()
	client, err := DialLocalRPC(context.Background(), LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("DialLocalRPC returned error: %v", err)
	}
	return runtimeDir, client, done
}

func setupLocalRPCSessionExecutionTriggerRoundTrip(t *testing.T, service *Service) (string, *LocalRPCClient, chan error) {
	t.Helper()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Listener.Accept()
		if err != nil {
			done <- err
			return
		}
		done <- serveSingleLocalRPCSessionExecutionTriggerConnForTest(conn, service)
	}()
	client, err := DialLocalRPC(context.Background(), LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("DialLocalRPC returned error: %v", err)
	}
	return runtimeDir, client, done
}

func TestDialLocalRPCWithLimitsAppliesMessageLimitValidation(t *testing.T) {
	runtimeDir, err := os.MkdirTemp("", "rc-rpc-")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtimeDir) })
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	limits := DefaultLimits()
	limits.MaxMessageBytes = 64
	client, err := DialLocalRPCWithLimits(context.Background(), LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"}, limits)
	if err != nil {
		t.Fatalf("DialLocalRPCWithLimits returned error: %v", err)
	}
	defer client.Close()

	errResp := client.Invoke(context.Background(), "run_list", RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-client-limited",
		Cursor:        "this-makes-the-payload-definitely-larger-than-64-bytes",
	}, nil)
	if errResp == nil {
		t.Fatal("Invoke expected message-size validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func serveSingleLocalRPCConnForTest(conn net.Conn, service *Service) error {
	defer conn.Close()
	meta := RequestContext{ClientID: "test-client", LaneID: "local_ipc"}
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	wire := LocalRPCRequest{}
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if err := ValidateRawMessageLimits(wire.Request, service.APILimits()); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	response := dispatchLocalRPCTest(service, wire, meta)
	return encoder.Encode(response)
}

func dispatchLocalRPCTest(service *Service, wire LocalRPCRequest, meta RequestContext) LocalRPCResponse {
	if wire.Operation != "run_list" {
		return LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(fmt.Errorf("unsupported operation %q", wire.Operation))}
	}
	req := RunListRequest{}
	if err := json.Unmarshal(wire.Request, &req); err != nil {
		return LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)}
	}
	resp, errResp := service.HandleRunList(context.Background(), req, meta)
	if errResp != nil {
		return LocalRPCResponse{OK: false, Error: errResp}
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)}
	}
	return LocalRPCResponse{OK: true, Response: json.RawMessage(raw)}
}

func serveSingleLocalRPCSessionListConnForTest(conn net.Conn, service *Service) error {
	defer conn.Close()
	meta := RequestContext{ClientID: "test-client", LaneID: "local_ipc"}
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	wire := LocalRPCRequest{}
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if err := ValidateRawMessageLimits(wire.Request, service.APILimits()); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	if wire.Operation != "session_list" {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(fmt.Errorf("unsupported operation %q", wire.Operation))})
	}
	req := SessionListRequest{}
	if err := json.Unmarshal(wire.Request, &req); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	resp, errResp := service.HandleSessionList(context.Background(), req, meta)
	if errResp != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: errResp})
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	return encoder.Encode(LocalRPCResponse{OK: true, Response: json.RawMessage(raw)})
}

func serveSingleLocalRPCSessionSendMessageConnForTest(conn net.Conn, service *Service) error {
	defer conn.Close()
	meta := RequestContext{ClientID: "test-client", LaneID: "local_ipc"}
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	wire := LocalRPCRequest{}
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if err := ValidateRawMessageLimits(wire.Request, service.APILimits()); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	if wire.Operation != "session_send_message" {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(fmt.Errorf("unsupported operation %q", wire.Operation))})
	}
	req := SessionSendMessageRequest{}
	if err := json.Unmarshal(wire.Request, &req); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	resp, errResp := service.HandleSessionSendMessage(context.Background(), req, meta)
	if errResp != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: errResp})
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	return encoder.Encode(LocalRPCResponse{OK: true, Response: json.RawMessage(raw)})
}

func serveSingleLocalRPCSessionExecutionTriggerConnForTest(conn net.Conn, service *Service) error {
	defer conn.Close()
	meta := RequestContext{ClientID: "test-client", LaneID: "local_ipc"}
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	wire := LocalRPCRequest{}
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if err := ValidateRawMessageLimits(wire.Request, service.APILimits()); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	if wire.Operation != "session_execution_trigger" {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(fmt.Errorf("unsupported operation %q", wire.Operation))})
	}
	req := SessionExecutionTriggerRequest{}
	if err := json.Unmarshal(wire.Request, &req); err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	resp, errResp := service.HandleSessionExecutionTrigger(context.Background(), req, meta)
	if errResp != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: errResp})
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return encoder.Encode(LocalRPCResponse{OK: false, Error: decodeLocalRPCTestError(err)})
	}
	return encoder.Encode(LocalRPCResponse{OK: true, Response: json.RawMessage(raw)})
}

func decodeLocalRPCTestError(err error) *ErrorResponse {
	if err == nil {
		err = fmt.Errorf("unknown local rpc error")
	}
	resp := toErrorResponse(defaultRequestIDFallback, "broker_validation_schema_invalid", "validation", false, err.Error())
	return &resp
}

func TestValidateRawMessageLimitsRejectsLargePayload(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxMessageBytes = 32
	raw := json.RawMessage(`{"schema_id":"runecode.protocol.v0.RunListRequest","schema_version":"0.1.0","request_id":"req-1","limit":100}`)
	err := ValidateRawMessageLimits(raw, limits)
	if err == nil {
		t.Fatal("ValidateRawMessageLimits expected size failure")
	}
}

func TestLocalRPCClientInvokeRespectsContextDeadline(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	client, err := DialLocalRPC(context.Background(), LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("DialLocalRPC returned error: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Millisecond))
	defer cancel()
	errResp := client.Invoke(ctx, "run_list", RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-expired"}, nil)
	if errResp == nil {
		t.Fatal("Invoke expected deadline typed error")
	}
	if errResp.Error.Code != "request_cancelled" {
		t.Fatalf("error code = %q, want request_cancelled", errResp.Error.Code)
	}
}
