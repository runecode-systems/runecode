// Command runecode-broker provides a local artifact and policy broker surface.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const localIPCMaxConcurrentConns = 64

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }
func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var u *usageError
		if errors.As(err, &u) {
			fmt.Fprintln(os.Stderr, u.Error())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return writeHelp(stdout)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return writeHelp(stdout)
	}

	handler, ok := commandHandlers()[args[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
	service, err := brokerServiceFactory()
	if err != nil {
		return fmt.Errorf("runecode-broker failed to initialize store: %w", err)
	}
	return handler(args[1:], service, stdout)
}

var brokerServiceFactory = brokerService
var localIPCListen = brokerapi.ListenLocalIPC

type commandHandler func([]string, *brokerapi.Service, io.Writer) error

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"serve-local":             handleServeLocal,
		"list-artifacts":          handleListArtifacts,
		"head-artifact":           handleHeadArtifact,
		"get-artifact":            handleGetArtifact,
		"put-artifact":            handlePutArtifact,
		"check-flow":              handleCheckFlow,
		"promote-excerpt":         handlePromoteExcerpt,
		"revoke-approved-excerpt": handleRevokeApprovedExcerpt,
		"set-run-status":          handleSetRunStatus,
		"gc":                      handleGC,
		"export-backup":           handleExportBackup,
		"restore-backup":          handleRestoreBackup,
		"show-audit":              handleShowAudit,
		"show-policy":             handleShowPolicy,
		"set-reserved-classes":    handleSetReservedClasses,
		"audit-readiness":         handleAuditReadiness,
		"audit-verification":      handleAuditVerification,
	}
}

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

func parseServeLocalArgs(args []string) (brokerapi.LocalIPCConfig, bool, error) {
	defaults, err := brokerapi.DefaultLocalIPCConfig()
	if err != nil {
		return brokerapi.LocalIPCConfig{}, false, err
	}
	fs := flag.NewFlagSet("serve-local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runtimeDir := fs.String("runtime-dir", defaults.RuntimeDir, "runtime directory for local unix socket")
	socketName := fs.String("socket-name", defaults.SocketName, "socket filename")
	once := fs.Bool("once", false, "accept a single connection and exit")
	if err := fs.Parse(args); err != nil {
		return brokerapi.LocalIPCConfig{}, false, &usageError{message: "serve-local usage: runecode-broker serve-local [--runtime-dir dir] [--socket-name broker.sock] [--once]"}
	}
	return brokerapi.LocalIPCConfig{RuntimeDir: *runtimeDir, SocketName: *socketName}, *once, nil
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

type localRPCRequest struct {
	Operation string          `json:"operation"`
	Request   json.RawMessage `json:"request"`
}

type localRPCResponse struct {
	OK       bool                     `json:"ok"`
	Response any                      `json:"response,omitempty"`
	Error    *brokerapi.ErrorResponse `json:"error,omitempty"`
}

type rpcOperationHandler func(json.RawMessage) localRPCResponse

func serveLocalConn(conn io.ReadWriteCloser, service *brokerapi.Service, creds brokerapi.PeerCredentials) error {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	meta := brokerapi.RequestContext{ClientID: fmt.Sprintf("uid:%d/pid:%d", creds.UID, creds.PID), LaneID: "local_ipc"}
	for {
		wire := localRPCRequest{}
		if err := decoder.Decode(&wire); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		resp := dispatchLocalRPC(service, wire, meta)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
}

func dispatchLocalRPC(service *brokerapi.Service, wire localRPCRequest, meta brokerapi.RequestContext) localRPCResponse {
	handler, ok := localRPCHandlers(service, meta)[wire.Operation]
	if !ok {
		return localRPCResponse{OK: false, Error: decodeWireError("", fmt.Errorf("unsupported operation %q", wire.Operation))}
	}
	return handler(wire.Request)
}

func localRPCHandlers(service *brokerapi.Service, meta brokerapi.RequestContext) map[string]rpcOperationHandler {
	handlers := map[string]rpcOperationHandler{}
	mergeRPCHandlers(handlers, runApprovalRPCHandlers(service, meta))
	mergeRPCHandlers(handlers, artifactRPCHandlers(service, meta))
	mergeRPCHandlers(handlers, auditHealthRPCHandlers(service, meta))
	return handlers
}

func mergeRPCHandlers(dst, src map[string]rpcOperationHandler) {
	for key, handler := range src {
		dst[key] = handler
	}
}

func runApprovalRPCHandlers(service *brokerapi.Service, meta brokerapi.RequestContext) map[string]rpcOperationHandler {
	return map[string]rpcOperationHandler{
		"run_list": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.RunListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleRunList(context.Background(), req, meta)
			})
		},
		"run_get": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.RunGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleRunGet(context.Background(), req, meta)
			})
		},
		"approval_list": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ApprovalListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleApprovalList(context.Background(), req, meta)
			})
		},
		"approval_get": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ApprovalGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleApprovalGet(context.Background(), req, meta)
			})
		},
		"approval_resolve": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ApprovalResolveRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleApprovalResolve(context.Background(), req, meta)
			})
		},
	}
}

func artifactRPCHandlers(service *brokerapi.Service, meta brokerapi.RequestContext) map[string]rpcOperationHandler {
	return map[string]rpcOperationHandler{
		"artifact_list": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.LocalArtifactListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleArtifactListV0(context.Background(), req, meta)
			})
		},
		"artifact_head": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.LocalArtifactHeadRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleArtifactHeadV0(context.Background(), req, meta)
			})
		},
		"artifact_read": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleArtifactRead(service, raw, meta)
		},
		"log_stream": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleLogStream(service, raw, meta)
		},
	}
}

func auditHealthRPCHandlers(service *brokerapi.Service, meta brokerapi.RequestContext) map[string]rpcOperationHandler {
	return map[string]rpcOperationHandler{
		"audit_timeline": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditTimelineRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditTimeline(context.Background(), req, meta)
			})
		},
		"audit_verification_get": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditVerificationGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditVerificationGet(context.Background(), req, meta)
			})
		},
		"readiness_get": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ReadinessGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleReadinessGet(context.Background(), req, meta)
			})
		},
		"version_info_get": func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.VersionInfoGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleVersionInfoGet(context.Background(), req, meta)
			})
		},
	}
}

func decodeAndHandle[T any](raw json.RawMessage, handle func(T) (any, *brokerapi.ErrorResponse)) localRPCResponse {
	var req T
	if err := json.Unmarshal(raw, &req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	resp, errResp := handle(req)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	return localRPCResponse{OK: true, Response: resp}
}

func decodeAndHandleArtifactRead(service *brokerapi.Service, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.ArtifactReadRequest{}
	if err := json.Unmarshal(raw, &req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	handle, errResp := service.HandleArtifactRead(context.Background(), req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamArtifactReadEvents(handle)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(req.RequestID, err)}
	}
	return localRPCResponse{OK: true, Response: events}
}

func decodeAndHandleLogStream(service *brokerapi.Service, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.LogStreamRequest{}
	if err := json.Unmarshal(raw, &req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, errResp := service.HandleLogStreamRequest(context.Background(), req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamLogEvents(ack)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(req.RequestID, err)}
	}
	return localRPCResponse{OK: true, Response: events}
}

func decodeWireError(requestID string, err error) *brokerapi.ErrorResponse {
	if requestID == "" {
		requestID = "invalid_request"
	}
	return &brokerapi.ErrorResponse{
		SchemaID:      "runecode.protocol.v0.BrokerErrorResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Error: brokerapi.ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: "0.3.0",
			Code:          "broker_validation_schema_invalid",
			Category:      "validation",
			Retryable:     false,
			Message:       err.Error(),
		},
	}
}

func handleListArtifacts(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	resp, errResp := service.HandleArtifactList(context.Background(), brokerapi.DefaultArtifactListRequest(defaultRequestID()), brokerapi.RequestContext{})
	if errResp != nil {
		return fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
	}
	return writeJSON(stdout, resp.Artifacts)
}

func handleHeadArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("head-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "artifact digest")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "head-artifact usage: runecode-broker head-artifact --digest sha256:..."}
	}
	if *digest == "" {
		return &usageError{message: "head-artifact requires --digest"}
	}
	resp, errResp := service.HandleArtifactHead(
		context.Background(),
		brokerapi.DefaultArtifactHeadRequest(defaultRequestID(), *digest),
		brokerapi.RequestContext{},
	)
	if errResp != nil {
		return fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
	}
	return writeJSON(stdout, resp.Artifact)
}

func handleGetArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	opts, err := parseGetArtifactArgs(args)
	if err != nil {
		return err
	}
	handle, errResp := service.HandleArtifactRead(context.Background(), opts.toRequest(), brokerapi.RequestContext{})
	if errResp != nil {
		return fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
	}
	events, err := service.StreamArtifactReadEvents(handle)
	if err != nil {
		return err
	}
	written, err := writeArtifactEventsToFile(events, opts.out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote %d bytes to %s\n", written, opts.out)
	return err
}

type getArtifactOptions struct {
	digest        string
	producer      string
	consumer      string
	manifestOptIn bool
	dataClass     string
	out           string
}

func parseGetArtifactArgs(args []string) (getArtifactOptions, error) {
	fs := flag.NewFlagSet("get-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "artifact digest")
	producer := fs.String("producer", "", "producer role for flow check")
	consumer := fs.String("consumer", "", "consumer role for flow check")
	manifestOptIn := fs.Bool("manifest-opt-in", false, "manifest opt-in posture for approved excerpts")
	dataClass := fs.String("data-class", "", "optional expected data class")
	out := fs.String("out", "", "output file path")
	if err := fs.Parse(args); err != nil {
		return getArtifactOptions{}, &usageError{message: "get-artifact usage: runecode-broker get-artifact --digest sha256:... --producer role --consumer role [--manifest-opt-in] [--data-class class] --out path"}
	}
	if *digest == "" || *producer == "" || *consumer == "" || *out == "" {
		return getArtifactOptions{}, &usageError{message: "get-artifact requires --digest --producer --consumer and --out"}
	}
	return getArtifactOptions{
		digest:        *digest,
		producer:      *producer,
		consumer:      *consumer,
		manifestOptIn: *manifestOptIn,
		dataClass:     *dataClass,
		out:           *out,
	}, nil
}

func (o getArtifactOptions) toRequest() brokerapi.ArtifactReadRequest {
	return brokerapi.ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		Digest:        o.digest,
		ProducerRole:  o.producer,
		ConsumerRole:  o.consumer,
		ManifestOptIn: o.manifestOptIn,
		DataClass:     o.dataClass,
	}
}

func writeArtifactEventsToFile(events []brokerapi.ArtifactStreamEvent, outPath string) (int64, error) {
	tmpPath := outPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmpPath)
	var written int64
	for _, event := range events {
		n, processErr := processArtifactFileEvent(f, event)
		if processErr != nil {
			_ = f.Close()
			return 0, processErr
		}
		written += int64(n)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := replaceFile(tmpPath, outPath); err != nil {
		return 0, err
	}
	return written, nil
}

func processArtifactFileEvent(f *os.File, event brokerapi.ArtifactStreamEvent) (int, error) {
	switch event.EventType {
	case "artifact_stream_chunk":
		chunk, err := base64.StdEncoding.DecodeString(event.ChunkBase64)
		if err != nil {
			return 0, err
		}
		n, err := f.Write(chunk)
		if err != nil {
			return 0, err
		}
		return n, nil
	case "artifact_stream_terminal":
		if event.Error != nil {
			return 0, fmt.Errorf("%s: %s", event.Error.Code, event.Error.Message)
		}
	}
	return 0, nil
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if removeErr := os.Remove(dst); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return os.Rename(src, dst)
}

func handlePutArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("put-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "artifact payload file")
	contentType := fs.String("content-type", "application/octet-stream", "artifact content type")
	dataClass := fs.String("data-class", string(artifacts.DataClassSpecText), "artifact data class")
	provenance := fs.String("provenance-hash", "", "provenance receipt hash")
	role := fs.String("role", "workspace", "producer role")
	runID := fs.String("run-id", "", "run id")
	stepID := fs.String("step-id", "", "step id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "put-artifact usage: runecode-broker put-artifact --file path --content-type text/plain --data-class spec_text --provenance-hash sha256:..."}
	}
	if *filePath == "" || *provenance == "" {
		return &usageError{message: "put-artifact requires --file and --provenance-hash"}
	}
	payload, err := os.ReadFile(*filePath)
	if err != nil {
		return err
	}
	request := brokerapi.DefaultArtifactPutRequest(defaultRequestID(), payload, *contentType, *dataClass, *provenance, *role, *runID, *stepID)
	resp, errResp := service.HandleArtifactPut(context.Background(), request, brokerapi.RequestContext{})
	if errResp != nil {
		return fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
	}
	return writeJSON(stdout, resp.Artifact)
}

func handleCheckFlow(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("check-flow", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	producer := fs.String("producer", "", "producer role")
	consumer := fs.String("consumer", "", "consumer role")
	dataClass := fs.String("data-class", "", "data class")
	digest := fs.String("digest", "", "digest")
	isEgress := fs.Bool("egress", false, "egress flow")
	manifestOptIn := fs.Bool("manifest-opt-in", false, "manifest opted in for approved excerpts")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "check-flow usage: runecode-broker check-flow --producer workspace --consumer model_gateway --data-class spec_text --digest sha256:... [--egress] [--manifest-opt-in]"}
	}
	if *producer == "" || *consumer == "" || *dataClass == "" || *digest == "" {
		return &usageError{message: "check-flow requires --producer --consumer --data-class --digest"}
	}
	class, err := brokerapi.ParseDataClass(*dataClass)
	if err != nil {
		return &usageError{message: err.Error()}
	}
	if err := service.CheckFlow(artifacts.FlowCheckRequest{ProducerRole: *producer, ConsumerRole: *consumer, DataClass: class, Digest: *digest, IsEgress: *isEgress, ManifestOptIn: *manifestOptIn}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "allowed")
	return err
}

func handlePromoteExcerpt(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("promote-excerpt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	unapprovedDigest := fs.String("unapproved-digest", "", "source unapproved artifact digest")
	approver := fs.String("approver", "", "human approver id")
	approvalRequestPath := fs.String("approval-request", "", "path to signed approval request envelope JSON")
	approvalEnvelopePath := fs.String("approval-envelope", "", "path to signed approval decision envelope JSON")
	repoPath := fs.String("repo-path", "", "repo path")
	commit := fs.String("commit", "", "commit hash")
	extractorVersion := fs.String("extractor-version", "", "extractor tool version")
	fullContentVisible := fs.Bool("full-content-visible", false, "approval view showed full content")
	explicitViewFull := fs.Bool("explicit-view-full", false, "explicit view-full affordance used")
	bulk := fs.Bool("bulk", false, "bulk promotion request")
	bulkApproved := fs.Bool("bulk-approved", false, "separate bulk approval confirmed")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "promote-excerpt usage: runecode-broker promote-excerpt --unapproved-digest sha256:... --approver user --approval-request approval-request.json --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible"}
	}
	if *unapprovedDigest == "" {
		return &usageError{message: "promote-excerpt requires --unapproved-digest"}
	}
	approvalRequest, err := loadSignedApprovalEnvelope(*approvalRequestPath)
	if err != nil {
		return &usageError{message: fmt.Sprintf("invalid --approval-request: %v", err)}
	}
	approvalEnvelope, err := loadSignedApprovalEnvelope(*approvalEnvelopePath)
	if err != nil {
		return &usageError{message: fmt.Sprintf("invalid --approval-envelope: %v", err)}
	}
	ref, err := service.PromoteApprovedExcerpt(artifacts.PromotionRequest{
		UnapprovedDigest:      *unapprovedDigest,
		Approver:              *approver,
		ApprovalRequest:       approvalRequest,
		ApprovalDecision:      approvalEnvelope,
		RepoPath:              *repoPath,
		Commit:                *commit,
		ExtractorToolVersion:  *extractorVersion,
		FullContentVisible:    *fullContentVisible,
		ExplicitViewFull:      *explicitViewFull,
		BulkRequest:           *bulk,
		BulkApprovalConfirmed: *bulkApproved,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, ref)
}

func handleRevokeApprovedExcerpt(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("revoke-approved-excerpt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "approved digest")
	actor := fs.String("actor", "", "actor")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "revoke-approved-excerpt usage: runecode-broker revoke-approved-excerpt --digest sha256:... --actor system"}
	}
	if *digest == "" || *actor == "" {
		return &usageError{message: "revoke-approved-excerpt requires --digest and --actor"}
	}
	return service.RevokeApprovedExcerpt(*digest, *actor)
}

func handleSetRunStatus(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("set-run-status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "run id")
	status := fs.String("status", "", "active|retained|closed")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "set-run-status usage: runecode-broker set-run-status --run-id run-1 --status retained"}
	}
	if *runID == "" || *status == "" {
		return &usageError{message: "set-run-status requires --run-id and --status"}
	}
	return service.SetRunStatus(*runID, *status)
}

func handleGC(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	result, err := service.GarbageCollect()
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func handleExportBackup(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("export-backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "output backup path")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "export-backup usage: runecode-broker export-backup --path backup.json"}
	}
	if *path == "" {
		return &usageError{message: "export-backup requires --path"}
	}
	return service.ExportBackup(*path)
}

func handleRestoreBackup(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("restore-backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "backup path")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "restore-backup usage: runecode-broker restore-backup --path backup.json"}
	}
	if *path == "" {
		return &usageError{message: "restore-backup requires --path"}
	}
	return service.RestoreBackup(*path)
}

func handleShowAudit(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	events, err := service.ReadAuditEvents()
	if err != nil {
		return err
	}
	return writeJSON(stdout, events)
}

func handleShowPolicy(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	return writeJSON(stdout, service.Policy())
}

func handleSetReservedClasses(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("set-reserved-classes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	enabled := fs.Bool("enabled", false, "enable reserved web_* data classes")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "set-reserved-classes usage: runecode-broker set-reserved-classes --enabled=true"}
	}
	policy := service.Policy()
	policy.ReservedClassesEnabled = *enabled
	return service.SetPolicy(policy)
}

func handleAuditReadiness(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	readiness, err := service.AuditReadiness()
	if err != nil {
		return err
	}
	return writeJSON(stdout, readiness)
}

func handleAuditVerification(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-verification", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 20, "max operational view entries")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-verification usage: runecode-broker audit-verification [--limit N]"}
	}
	surface, err := service.LatestAuditVerificationSurface(*limit)
	if err != nil {
		return err
	}
	return writeJSON(stdout, surface)
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-broker <command> [flags]

Commands:
  serve-local [--runtime-dir dir] [--socket-name broker.sock] [--once]
  list-artifacts
  head-artifact --digest sha256:...
  get-artifact --digest sha256:... --producer role --consumer role [--manifest-opt-in] [--data-class class] --out path
  put-artifact --file path --content-type type --data-class class --provenance-hash sha256:...
  check-flow --producer role --consumer role --data-class class --digest sha256:... [--egress] [--manifest-opt-in]
  promote-excerpt --unapproved-digest sha256:... --approver user --approval-request approval-request.json --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible
  revoke-approved-excerpt --digest sha256:... --actor user
  set-run-status --run-id id --status active|retained|closed
  gc
  export-backup --path backup.json
  restore-backup --path backup.json
  show-audit
  show-policy
  set-reserved-classes --enabled=true|false
  audit-readiness
  audit-verification [--limit N]`)
	return err
}

func brokerService() (*brokerapi.Service, error) {
	return brokerapi.NewService(defaultBrokerStoreRoot(), auditd.DefaultLedgerRoot())
}
