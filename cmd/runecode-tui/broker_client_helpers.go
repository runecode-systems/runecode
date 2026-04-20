package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (c *rpcBrokerClient) invoke(ctx context.Context, operation string, req any, out any) error {
	return c.invokeWithRPC(ctx, operation, req, nil, out)
}

func (c *rpcBrokerClient) invokeWithSecret(ctx context.Context, operation string, req any, secret []byte, out any) error {
	return c.invokeWithRPC(ctx, operation, req, secret, out)
}

func (c *rpcBrokerClient) invokeWithRPC(ctx context.Context, operation string, req any, secret []byte, out any) error {
	client, err := dialLocalRPCClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if errResp := invokeRPCOperation(ctx, client, operation, req, secret, out); errResp != nil {
		return formatRPCError(errResp)
	}
	return nil
}

func dialLocalRPCClient(ctx context.Context) (localRPCInvoker, error) {
	cfg, err := localIPCConfigProvider()
	if err != nil {
		return nil, localIPCSetupError("local_ipc_config_error", err)
	}
	client, err := localRPCDialer(ctx, cfg)
	if err != nil {
		return nil, localIPCSetupError("local_ipc_dial_error", err)
	}
	if client == nil {
		return nil, fmt.Errorf("local_ipc_dial_error")
	}
	return client, nil
}

func invokeRPCOperation(ctx context.Context, client localRPCInvoker, operation string, req any, secret []byte, out any) *brokerapi.ErrorResponse {
	if secret == nil {
		return client.Invoke(ctx, operation, req, out)
	}
	return client.InvokeSecretIngress(ctx, operation, req, secret, out)
}

func formatRPCError(errResp *brokerapi.ErrorResponse) error {
	code := strings.TrimSpace(errResp.Error.Code)
	if code == "" {
		code = "broker_rpc_error"
	}
	message := sanitizeUIText(errResp.Error.Message)
	if message == "" || message == code {
		return fmt.Errorf("%s", code)
	}
	return fmt.Errorf("%s: %s", code, message)
}

func localIPCSetupError(fallback string, err error) error {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fmt.Errorf("%s", fallback)
	}
	if errors.Is(err, brokerapi.ErrPeerCredentialsUnavailable) || shouldExposeLocalIPCMessage(message) {
		return fmt.Errorf("%s", message)
	}
	return fmt.Errorf("%s", fallback)
}

func shouldExposeLocalIPCMessage(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "linux-only") ||
		strings.Contains(lower, "runtime directory is required") ||
		strings.Contains(lower, "socket name is required") ||
		strings.Contains(lower, "peer credentials unavailable")
}

func localBrokerBoundaryPosture() string {
	return "Local broker API only via broker local IPC; OS peer auth is broker-enforced where supported"
}

func newRequestID(prefix string) string {
	seq := atomic.AddUint64(&requestSeq, 1)
	return prefix + "-" + strconv.FormatUint(seq, 10)
}

func withLoadTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func parseDigestIdentity(identity string) trustpolicy.Digest {
	parts := strings.SplitN(strings.TrimSpace(identity), ":", 2)
	if len(parts) != 2 {
		return trustpolicy.Digest{}
	}
	return trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}
}

func decodeArtifactStream(events []brokerapi.ArtifactStreamEvent) (string, error) {
	var b strings.Builder
	hasTerminal := false
	for _, event := range events {
		if event.EventType == "artifact_stream_terminal" {
			hasTerminal = true
		}
		if err := applyArtifactEvent(&b, event); err != nil {
			return "", err
		}
	}
	if !hasTerminal {
		return "", fmt.Errorf("artifact_stream_incomplete")
	}
	return b.String(), nil
}

func applyArtifactEvent(out *strings.Builder, event brokerapi.ArtifactStreamEvent) error {
	switch event.EventType {
	case "artifact_stream_chunk":
		return appendArtifactChunk(out, event.ChunkBase64)
	case "artifact_stream_terminal":
		return validateArtifactTerminal(event)
	default:
		return nil
	}
}

func appendArtifactChunk(out *strings.Builder, chunkBase64 string) error {
	chunk, err := base64.StdEncoding.DecodeString(chunkBase64)
	if err != nil {
		return err
	}
	if _, err := out.Write(chunk); err != nil {
		return err
	}
	return nil
}

func validateArtifactTerminal(event brokerapi.ArtifactStreamEvent) error {
	if event.Error != nil {
		code := strings.TrimSpace(event.Error.Code)
		if code == "" {
			code = "artifact_stream_error"
		}
		return fmt.Errorf("%s", code)
	}
	if event.TerminalStatus != "completed" {
		return fmt.Errorf("artifact stream terminal status %q", event.TerminalStatus)
	}
	return nil
}
