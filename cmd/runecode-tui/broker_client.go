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

const localAPISchemaVersion = "0.1.0"
const localAPIFamily = "broker_local_api"

var requestSeq uint64

var localIPCConfigProvider = brokerapi.DefaultLocalIPCConfig

var localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
	return brokerapi.DialLocalRPC(ctx, cfg)
}

var localBrokerClientFactory = func() localBrokerClient {
	return &rpcBrokerClient{}
}

type localBrokerClient interface {
	RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error)
	RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error)
	RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error)
	SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error)
	SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error)
	SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error)
	SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error)
	ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error)
	ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error)
	ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error)
	ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error)
	BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error)
	BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error)
	ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error)
	ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error)
	ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error)
	LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error)
	LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error)
	AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error)
	AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error)
	AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error)
	AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error)
	AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error)
	AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error)
	AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error)
	ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error)
	VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error)
}

type localRPCInvoker interface {
	Invoke(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse
	Close() error
}

type rpcBrokerClient struct{}

func newLocalBrokerClient() localBrokerClient {
	return localBrokerClientFactory()
}

func (c *rpcBrokerClient) RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error) {
	req := brokerapi.RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("run-list"), Limit: limit}
	resp := brokerapi.RunListResponse{}
	return resp, c.invoke(ctx, "run_list", req, &resp)
}

func (c *rpcBrokerClient) RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error) {
	req := brokerapi.RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("run-get"), RunID: runID}
	resp := brokerapi.RunGetResponse{}
	return resp, c.invoke(ctx, "run_get", req, &resp)
}

func (c *rpcBrokerClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.RunWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("run-watch")
	events := []brokerapi.RunWatchEvent{}
	return events, c.invoke(ctx, "run_watch", req, &events)
}

func (c *rpcBrokerClient) SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error) {
	req := brokerapi.SessionListRequest{SchemaID: "runecode.protocol.v0.SessionListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("session-list"), Limit: limit}
	resp := brokerapi.SessionListResponse{}
	return resp, c.invoke(ctx, "session_list", req, &resp)
}

func (c *rpcBrokerClient) SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error) {
	req := brokerapi.SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("session-get"), SessionID: sessionID}
	resp := brokerapi.SessionGetResponse{}
	return resp, c.invoke(ctx, "session_get", req, &resp)
}

func (c *rpcBrokerClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	req.SchemaID = "runecode.protocol.v0.SessionSendMessageRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("session-send")
	resp := brokerapi.SessionSendMessageResponse{}
	return resp, c.invoke(ctx, "session_send_message", req, &resp)
}

func (c *rpcBrokerClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.SessionWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("session-watch")
	events := []brokerapi.SessionWatchEvent{}
	return events, c.invoke(ctx, "session_watch", req, &events)
}

func (c *rpcBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	req := brokerapi.ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("approval-list"), Limit: limit}
	resp := brokerapi.ApprovalListResponse{}
	return resp, c.invoke(ctx, "approval_list", req, &resp)
}

func (c *rpcBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	req := brokerapi.ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("approval-get"), ApprovalID: approvalID}
	resp := brokerapi.ApprovalGetResponse{}
	return resp, c.invoke(ctx, "approval_get", req, &resp)
}

func (c *rpcBrokerClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ApprovalResolveRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("approval-resolve")
	resp := brokerapi.ApprovalResolveResponse{}
	return resp, c.invoke(ctx, "approval_resolve", req, &resp)
}

func (c *rpcBrokerClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.ApprovalWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("approval-watch")
	events := []brokerapi.ApprovalWatchEvent{}
	return events, c.invoke(ctx, "approval_watch", req, &events)
}

func (c *rpcBrokerClient) BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error) {
	req := brokerapi.BackendPostureGetRequest{SchemaID: "runecode.protocol.v0.BackendPostureGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("backend-posture-get")}
	resp := brokerapi.BackendPostureGetResponse{}
	return resp, c.invoke(ctx, "backend_posture_get", req, &resp)
}

func (c *rpcBrokerClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error) {
	req.SchemaID = "runecode.protocol.v0.BackendPostureChangeRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("backend-posture-change")
	resp := brokerapi.BackendPostureChangeResponse{}
	return resp, c.invoke(ctx, "backend_posture_change", req, &resp)
}

func (c *rpcBrokerClient) ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error) {
	req := brokerapi.LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("artifact-list"), Limit: limit, DataClass: dataClass}
	resp := brokerapi.LocalArtifactListResponse{}
	return resp, c.invoke(ctx, "artifact_list", req, &resp)
}

func (c *rpcBrokerClient) ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error) {
	req := brokerapi.LocalArtifactHeadRequest{SchemaID: "runecode.protocol.v0.ArtifactHeadRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("artifact-head"), Digest: digest}
	resp := brokerapi.LocalArtifactHeadResponse{}
	return resp, c.invoke(ctx, "artifact_head", req, &resp)
}

func (c *rpcBrokerClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error) {
	req.SchemaID = "runecode.protocol.v0.ArtifactReadRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("artifact-read")
	events := []brokerapi.ArtifactStreamEvent{}
	return events, c.invoke(ctx, "artifact_read", req, &events)
}

func (c *rpcBrokerClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error) {
	req.SchemaID = "runecode.protocol.v0.LLMInvokeRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("llm-invoke")
	resp := brokerapi.LLMInvokeResponse{}
	return resp, c.invoke(ctx, "llm_invoke", req, &resp)
}

func (c *rpcBrokerClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error) {
	req.SchemaID = "runecode.protocol.v0.LLMStreamRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("llm-stream")
	resp := brokerapi.LLMStreamEnvelope{}
	return resp, c.invoke(ctx, "llm_stream", req, &resp)
}

func (c *rpcBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	req := brokerapi.AuditTimelineRequest{SchemaID: "runecode.protocol.v0.AuditTimelineRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-timeline"), Limit: limit, Cursor: cursor}
	resp := brokerapi.AuditTimelineResponse{}
	return resp, c.invoke(ctx, "audit_timeline", req, &resp)
}

func (c *rpcBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	req := brokerapi.AuditVerificationGetRequest{SchemaID: "runecode.protocol.v0.AuditVerificationGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-verification"), ViewLimit: viewLimit}
	resp := brokerapi.AuditVerificationGetResponse{}
	return resp, c.invoke(ctx, "audit_verification_get", req, &resp)
}

func (c *rpcBrokerClient) AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error) {
	req := brokerapi.AuditFinalizeVerifyRequest{SchemaID: "runecode.protocol.v0.AuditFinalizeVerifyRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-finalize-verify")}
	resp := brokerapi.AuditFinalizeVerifyResponse{}
	return resp, c.invoke(ctx, "audit_finalize_verify", req, &resp)
}

func (c *rpcBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	req := brokerapi.AuditRecordGetRequest{SchemaID: "runecode.protocol.v0.AuditRecordGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-record"), RecordDigest: parseDigestIdentity(digest)}
	resp := brokerapi.AuditRecordGetResponse{}
	return resp, c.invoke(ctx, "audit_record_get", req, &resp)
}

func (c *rpcBrokerClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error) {
	req.SchemaID = "runecode.protocol.v0.AuditAnchorPreflightGetRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("audit-anchor-preflight")
	resp := brokerapi.AuditAnchorPreflightGetResponse{}
	return resp, c.invoke(ctx, "audit_anchor_preflight_get", req, &resp)
}

func (c *rpcBrokerClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	req.SchemaID = "runecode.protocol.v0.AuditAnchorPresenceGetRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("audit-anchor-presence")
	resp := brokerapi.AuditAnchorPresenceGetResponse{}
	return resp, c.invoke(ctx, "audit_anchor_presence_get", req, &resp)
}

func (c *rpcBrokerClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	req.SchemaID = "runecode.protocol.v0.AuditAnchorSegmentRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("audit-anchor")
	resp := brokerapi.AuditAnchorSegmentResponse{}
	return resp, c.invoke(ctx, "audit_anchor_segment", req, &resp)
}

func (c *rpcBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	req := brokerapi.ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("readiness")}
	resp := brokerapi.ReadinessGetResponse{}
	return resp, c.invoke(ctx, "readiness_get", req, &resp)
}

func (c *rpcBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	req := brokerapi.VersionInfoGetRequest{SchemaID: "runecode.protocol.v0.VersionInfoGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("version")}
	resp := brokerapi.VersionInfoGetResponse{}
	return resp, c.invoke(ctx, "version_info_get", req, &resp)
}

func (c *rpcBrokerClient) invoke(ctx context.Context, operation string, req any, out any) error {
	cfg, err := localIPCConfigProvider()
	if err != nil {
		return localIPCSetupError("local_ipc_config_error", err)
	}
	client, err := localRPCDialer(ctx, cfg)
	if err != nil {
		return localIPCSetupError("local_ipc_dial_error", err)
	}
	if client == nil {
		return fmt.Errorf("local_ipc_dial_error")
	}
	defer client.Close()
	if errResp := client.Invoke(ctx, operation, req, out); errResp != nil {
		code := strings.TrimSpace(errResp.Error.Code)
		if code == "" {
			code = "broker_rpc_error"
		}
		message := sanitizeUIText(errResp.Error.Message)
		if message == "" {
			return fmt.Errorf("%s", code)
		}
		return fmt.Errorf("%s: %s", code, message)
	}
	return nil
}

func localIPCSetupError(fallback string, err error) error {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fmt.Errorf("%s", fallback)
	}
	if errors.Is(err, brokerapi.ErrPeerCredentialsUnavailable) {
		return fmt.Errorf("%s", message)
	}
	lower := strings.ToLower(message)
	if strings.Contains(lower, "linux-only") || strings.Contains(lower, "runtime directory is required") || strings.Contains(lower, "socket name is required") || strings.Contains(lower, "peer credentials unavailable") {
		return fmt.Errorf("%s", message)
	}
	return fmt.Errorf("%s", fallback)
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
