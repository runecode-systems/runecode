package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const (
	localIPCMaxConcurrentConns = 64
	localIPCMaxRequestBytes    = 1 << 20
	localIPCReadIdleTimeout    = 10 * time.Second
	localIPCWriteTimeout       = 10 * time.Second
)

func handleServeLocal(args []string, service *brokerapi.Service, stdout io.Writer) error {
	cfg, once, err := parseServeLocalArgs(args)
	if err != nil {
		return err
	}
	listener, err := localIPCListen(cfg)
	if err != nil {
		return fmt.Errorf("local ipc startup failed: %w", err)
	}
	return serveLocalLoop(listener, service, stdout, once)
}

func serveLocalLoop(listener *brokerapi.LocalIPCListener, service *brokerapi.Service, stdout io.Writer, once bool) error {
	defer listener.Close()
	if _, err := fmt.Fprintln(stdout, "local broker api listening"); err != nil {
		return err
	}
	connSlots := make(chan struct{}, localIPCMaxConcurrentConns)
	for {
		conn, err := listener.Listener.Accept()
		if err != nil {
			return err
		}
		done, err := handleAcceptedLocalConn(conn, service, connSlots, once)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

func handleAcceptedLocalConn(conn net.Conn, service *brokerapi.Service, connSlots chan struct{}, once bool) (bool, error) {
	creds, authErr := brokerapi.AuthenticateLocalPeer(conn, brokerapi.DefaultAdmissionPolicy())
	if authErr != nil {
		_ = conn.Close()
		if once {
			return false, authErr
		}
		return false, nil
	}
	if once {
		return true, serveLocalConn(conn, service, creds)
	}
	connSlots <- struct{}{}
	go func() {
		defer func() { <-connSlots }()
		if err := serveLocalConn(conn, service, creds); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "serve-local connection error: %v\n", err)
		}
	}()
	return false, nil
}

type localRPCRequest = brokerapi.LocalRPCRequest

type localRPCResponse = brokerapi.LocalRPCResponse

type rpcOperationHandler func(json.RawMessage) localRPCResponse

type rpcOperation struct {
	requestSchemaPath string
	handle            rpcOperationHandler
}

func serveLocalConn(conn net.Conn, service *brokerapi.Service, creds brokerapi.PeerCredentials) error {
	defer conn.Close()
	connCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	limits := service.APILimits()
	requestBytesLimit := limits.MaxMessageBytes
	if requestBytesLimit <= 0 {
		requestBytesLimit = localIPCMaxRequestBytes
	}
	readIdleTimeout := limits.StreamIdleTimeout
	if readIdleTimeout <= 0 {
		readIdleTimeout = localIPCReadIdleTimeout
	}
	encoder := json.NewEncoder(conn)
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 8<<10), requestBytesLimit)
	meta := brokerapi.RequestContext{ClientID: fmt.Sprintf("uid:%d/pid:%d", creds.UID, creds.PID), LaneID: "local_ipc"}
	for {
		wire, done, err := readLocalRPCRequest(conn, scanner, encoder, requestBytesLimit, readIdleTimeout)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if wire == nil {
			continue
		}
		resp := localRPCDispatch(service, connCtx, *wire, meta)
		if err := encodeLocalRPCResponse(conn, encoder, resp); err != nil {
			return err
		}
	}
}

var localRPCDispatch = dispatchLocalRPC

func readLocalRPCRequest(conn net.Conn, scanner *bufio.Scanner, encoder *json.Encoder, requestBytesLimit int, readIdleTimeout time.Duration) (*localRPCRequest, bool, error) {
	if err := conn.SetReadDeadline(time.Now().Add(readIdleTimeout)); err != nil {
		return nil, false, err
	}
	if !scanner.Scan() {
		err := scanner.Err()
		if err == nil {
			return nil, true, nil
		}
		if errors.Is(err, bufio.ErrTooLong) {
			resp := localRPCResponse{OK: false, Error: decodeWireError("", fmt.Errorf("request exceeds local IPC message size limit"))}
			if encodeErr := encodeLocalRPCResponse(conn, encoder, resp); encodeErr != nil {
				return nil, true, nil
			}
			return nil, true, nil
		}
		return nil, false, err
	}
	wire, err := decodeLocalRPCRequest(scanner.Bytes())
	if err != nil {
		resp := localRPCResponse{OK: false, Error: decodeWireError("", err)}
		if encodeErr := encodeLocalRPCResponse(conn, encoder, resp); encodeErr != nil {
			return nil, false, encodeErr
		}
		return nil, false, nil
	}
	if len(wire.Request) > requestBytesLimit {
		resp := localRPCResponse{OK: false, Error: decodeWireError("", fmt.Errorf("request exceeds local IPC message size limit"))}
		if encodeErr := encodeLocalRPCResponse(conn, encoder, resp); encodeErr != nil {
			return nil, true, nil
		}
		return nil, true, nil
	}
	return &wire, false, nil
}

func encodeLocalRPCResponse(conn net.Conn, encoder *json.Encoder, resp localRPCResponse) error {
	if err := conn.SetWriteDeadline(time.Now().Add(localIPCWriteTimeout)); err != nil {
		return err
	}
	return encoder.Encode(resp)
}

func decodeLocalRPCRequest(line []byte) (localRPCRequest, error) {
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	request := localRPCRequest{}
	if err := decoder.Decode(&request); err != nil {
		return localRPCRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return localRPCRequest{}, errors.New("request must contain exactly one JSON object")
		}
		return localRPCRequest{}, err
	}
	if request.Operation == "" {
		return localRPCRequest{}, errors.New("operation is required")
	}
	if len(request.Request) == 0 {
		return localRPCRequest{}, errors.New("request is required")
	}
	return request, nil
}

func dispatchLocalRPC(service *brokerapi.Service, ctx context.Context, wire localRPCRequest, meta brokerapi.RequestContext) localRPCResponse {
	operation, ok := localRPCOperations(service, ctx, meta)[wire.Operation]
	if !ok {
		return localRPCResponse{OK: false, Error: decodeWireError("", fmt.Errorf("unsupported operation %q", wire.Operation))}
	}
	if err := validateRawRPCPayload(wire.Request, operation.requestSchemaPath, service.APILimits()); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	return operation.handle(wire.Request)
}

func validateRawRPCPayload(raw json.RawMessage, schemaPath string, limits brokerapi.Limits) error {
	if err := brokerapi.ValidateRawMessageLimits(raw, limits); err != nil {
		return err
	}
	if err := artifacts.ValidateObjectPayloadAgainstSchema(raw, schemaPath); err != nil {
		return err
	}
	return nil
}

func decodeAndHandle[T any](raw json.RawMessage, handle func(T) (any, *brokerapi.ErrorResponse)) localRPCResponse {
	var req T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	resp, errResp := handle(req)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	return localRPCOKResponse(resp)
}

func decodeAndHandleArtifactRead(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.ArtifactReadRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	handle, errResp := service.HandleArtifactRead(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamArtifactReadEvents(handle)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(req.RequestID, err)}
	}
	return localRPCOKResponse(events)
}

func decodeAndHandleLogStream(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.LogStreamRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, errResp := service.HandleLogStreamRequest(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamLogEvents(ack)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(req.RequestID, err)}
	}
	return localRPCOKResponse(events)
}

func decodeAndHandleLLMInvoke(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.LLMInvokeRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	resp, errResp := service.HandleLLMInvoke(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	return localRPCOKResponse(resp)
}

func decodeAndHandleLLMStream(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.LLMStreamRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, binding, inputRef, errResp := service.HandleLLMStreamRequest(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	resp, err := service.StreamLLMEvents(ack, binding, inputRef)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(req.RequestID, err)}
	}
	return localRPCOKResponse(resp)
}

func localRPCOKResponse(value any) localRPCResponse {
	raw, err := json.Marshal(value)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	return localRPCResponse{OK: true, Response: json.RawMessage(raw)}
}

func decodeWireError(requestID string, err error) *brokerapi.ErrorResponse {
	if requestID == "" {
		requestID = "invalid_request"
	}
	code := "broker_validation_schema_invalid"
	category := "validation"
	message := "request validation failed"
	retryable := false
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return &brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: requestID, Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "request_cancelled", Category: "transport", Retryable: true, Message: "request cancelled"}}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return &brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: requestID, Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "broker_timeout_request_deadline_exceeded", Category: "timeout", Retryable: false, Message: "request deadline exceeded"}}
		}
		errText := err.Error()
		if strings.Contains(errText, "unsupported operation") {
			code = "broker_validation_operation_invalid"
			category = "validation"
			message = "operation is not supported"
		}
		if strings.Contains(errText, "message size") {
			code = "broker_limit_message_size_exceeded"
			category = "transport"
			message = "request exceeds message size limit"
		}
		if strings.Contains(errText, "message depth") || strings.Contains(errText, "array length") || strings.Contains(errText, "object property count") {
			code = "broker_limit_structural_complexity_exceeded"
			category = "transport"
			message = "request exceeds structural complexity limits"
		}
	}
	return &brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: requestID, Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: code, Category: category, Retryable: retryable, Message: message}}
}
