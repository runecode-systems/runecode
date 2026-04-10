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

	"github.com/runecode-ai/runecode/internal/brokerapi"
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
	err := run([]string{"not-a-command"}, stdout, stderr)
	if err == nil {
		t.Fatal("expected usage error for unknown command")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("unknown command error type = %T, want *usageError", err)
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
