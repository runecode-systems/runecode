//go:build linux

package brokerapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type LocalRPCClient struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
	limits  Limits
}

func DialLocalRPC(ctx context.Context, cfg LocalIPCConfig) (*LocalRPCClient, error) {
	return DialLocalRPCWithLimits(ctx, cfg, Limits{})
}

func DialLocalRPCWithLimits(ctx context.Context, cfg LocalIPCConfig, limits Limits) (*LocalRPCClient, error) {
	resolved := cfg.withDefaults()
	socketPath, err := resolved.socketPath()
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &LocalRPCClient{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(bufio.NewReader(conn)),
		limits:  APIConfig{Limits: limits}.withDefaults().Limits,
	}, nil
}

func (c *LocalRPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *LocalRPCClient) Invoke(ctx context.Context, operation string, request any, out any) *ErrorResponse {
	if c == nil || c.conn == nil {
		err := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, "local rpc client is not connected")
		return &err
	}
	if operation == "" {
		err := toErrorResponse(defaultRequestIDFallback, "broker_validation_schema_invalid", "validation", false, "operation is required")
		return &err
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		errResp := toErrorResponse(defaultRequestIDFallback, "broker_validation_schema_invalid", "validation", false, "request validation failed")
		return &errResp
	}
	if err := ValidateRawMessageLimits(requestBytes, c.limits); err != nil {
		errResp := toErrorResponse(defaultRequestIDFallback, "broker_validation_schema_invalid", "validation", false, err.Error())
		return &errResp
	}
	wire := LocalRPCRequest{Operation: operation, Request: json.RawMessage(requestBytes)}
	if err := c.setDeadlineFromContext(ctx); err != nil {
		errResp := toErrorResponse(defaultRequestIDFallback, "request_cancelled", "transport", true, err.Error())
		return &errResp
	}
	if err := c.encoder.Encode(wire); err != nil {
		errResp := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, err.Error())
		return &errResp
	}
	response := LocalRPCResponse{}
	if err := c.decoder.Decode(&response); err != nil {
		errResp := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, err.Error())
		return &errResp
	}
	if response.Error != nil {
		return response.Error
	}
	if !response.OK {
		errResp := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, "local rpc request failed without typed error")
		return &errResp
	}
	if out == nil || len(response.Response) == 0 {
		return nil
	}
	if err := json.Unmarshal(response.Response, out); err != nil {
		errResp := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, err.Error())
		return &errResp
	}
	return nil
}

func (c *LocalRPCClient) setDeadlineFromContext(ctx context.Context) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("not connected")
	}
	if ctx == nil {
		return c.conn.SetDeadline(time.Now().Add(c.limits.DefaultRequestDeadline))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		return c.conn.SetDeadline(deadline)
	}
	return c.conn.SetDeadline(time.Now().Add(c.limits.DefaultRequestDeadline))
}
