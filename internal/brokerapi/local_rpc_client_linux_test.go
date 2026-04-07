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
)

func TestLocalRPCClientInvokeRunList(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
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
	defer client.Close()

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
