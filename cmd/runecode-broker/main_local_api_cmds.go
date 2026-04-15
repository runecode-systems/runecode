package main

import (
	"context"
	"flag"
	"io"
	"os/signal"
	"syscall"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func commandRequestContext(parent context.Context) (context.Context, context.CancelFunc) {
	base := parent
	if base == nil {
		base = context.Background()
	}
	return signal.NotifyContext(base, syscall.SIGINT, syscall.SIGTERM)
}

func handleRunList(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("run-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 20, "max run entries")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "run-list usage: runecode-broker run-list [--limit N]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.RunList(ctx, brokerapi.RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		Limit:         *limit,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Runs)
}

func handleRunGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("run-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "run id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "run-get usage: runecode-broker run-get --run-id id"}
	}
	if *runID == "" {
		return &usageError{message: "run-get requires --run-id"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.RunGet(ctx, brokerapi.RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		RunID:         *runID,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Run)
}

func handleRunWatch(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("run-watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	streamID := fs.String("stream-id", "", "stable stream id")
	runID := fs.String("run-id", "", "optional run id filter")
	workspaceID := fs.String("workspace-id", "", "optional workspace id filter")
	lifecycleState := fs.String("lifecycle-state", "", "optional lifecycle state filter")
	follow := fs.Bool("follow", false, "stream follow mode")
	includeSnapshot := fs.Bool("include-snapshot", true, "include initial snapshot event")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "run-watch usage: runecode-broker run-watch [--stream-id id] [--run-id id] [--workspace-id id] [--lifecycle-state state] [--follow] [--include-snapshot]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	requestID := defaultRequestID()
	resolvedStreamID := *streamID
	if resolvedStreamID == "" {
		resolvedStreamID = "run-watch-" + requestID
	}
	events, errResp := api.RunWatch(ctx, brokerapi.RunWatchRequest{
		SchemaID:        "runecode.protocol.v0.RunWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       requestID,
		StreamID:        resolvedStreamID,
		RunID:           *runID,
		WorkspaceID:     *workspaceID,
		LifecycleState:  *lifecycleState,
		Follow:          *follow,
		IncludeSnapshot: *includeSnapshot,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, events)
}

func handleSessionList(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("session-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 20, "max session entries")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "session-list usage: runecode-broker session-list [--limit N]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.SessionList(ctx, brokerapi.SessionListRequest{
		SchemaID:      "runecode.protocol.v0.SessionListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		Limit:         *limit,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Sessions)
}

func handleSessionGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("session-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sessionID := fs.String("session-id", "", "session id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "session-get usage: runecode-broker session-get --session-id id"}
	}
	if *sessionID == "" {
		return &usageError{message: "session-get requires --session-id"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.SessionGet(ctx, brokerapi.SessionGetRequest{
		SchemaID:      "runecode.protocol.v0.SessionGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		SessionID:     *sessionID,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Session)
}

func handleSessionSendMessage(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("session-send-message", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sessionID := fs.String("session-id", "", "session id")
	role := fs.String("role", "user", "message role")
	content := fs.String("content", "", "message content")
	idempotencyKey := fs.String("idempotency-key", "", "optional idempotency key")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "session-send-message usage: runecode-broker session-send-message --session-id id --content text [--role user|assistant|system|tool] [--idempotency-key key]"}
	}
	if *sessionID == "" {
		return &usageError{message: "session-send-message requires --session-id"}
	}
	if *content == "" {
		return &usageError{message: "session-send-message requires --content"}
	}
	if !validSessionMessageRole(*role) {
		return &usageError{message: "session-send-message --role must be one of: user|assistant|system|tool"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.SessionSendMessage(ctx, brokerapi.SessionSendMessageRequest{
		SchemaID:       "runecode.protocol.v0.SessionSendMessageRequest",
		SchemaVersion:  "0.1.0",
		RequestID:      defaultRequestID(),
		SessionID:      *sessionID,
		Role:           *role,
		ContentText:    *content,
		IdempotencyKey: *idempotencyKey,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleSessionWatch(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("session-watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	streamID := fs.String("stream-id", "", "stable stream id")
	sessionID := fs.String("session-id", "", "optional session id filter")
	workspaceID := fs.String("workspace-id", "", "optional workspace id filter")
	status := fs.String("status", "", "optional session status filter")
	lastActivityKind := fs.String("last-activity-kind", "", "optional session activity-kind filter")
	follow := fs.Bool("follow", false, "stream follow mode")
	includeSnapshot := fs.Bool("include-snapshot", true, "include initial snapshot event")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "session-watch usage: runecode-broker session-watch [--stream-id id] [--session-id id] [--workspace-id id] [--status active|completed|archived] [--last-activity-kind kind] [--follow] [--include-snapshot]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	requestID := defaultRequestID()
	resolvedStreamID := *streamID
	if resolvedStreamID == "" {
		resolvedStreamID = "session-watch-" + requestID
	}
	events, errResp := api.SessionWatch(ctx, brokerapi.SessionWatchRequest{
		SchemaID:         "runecode.protocol.v0.SessionWatchRequest",
		SchemaVersion:    "0.1.0",
		RequestID:        requestID,
		StreamID:         resolvedStreamID,
		SessionID:        *sessionID,
		WorkspaceID:      *workspaceID,
		Status:           *status,
		LastActivityKind: *lastActivityKind,
		Follow:           *follow,
		IncludeSnapshot:  *includeSnapshot,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, events)
}

func handleApprovalList(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("approval-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "filter by run id")
	status := fs.String("status", "", "filter by status")
	limit := fs.Int("limit", 20, "max approval entries")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "approval-list usage: runecode-broker approval-list [--run-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--limit N]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ApprovalList(ctx, brokerapi.ApprovalListRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		RunID:         *runID,
		Status:        *status,
		Limit:         *limit,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Approvals)
}

func handleApprovalGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("approval-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	approvalID := fs.String("approval-id", "", "approval id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "approval-get usage: runecode-broker approval-get --approval-id sha256:..."}
	}
	if *approvalID == "" {
		return &usageError{message: "approval-get requires --approval-id"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ApprovalGet(ctx, brokerapi.ApprovalGetRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		ApprovalID:    *approvalID,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleApprovalWatch(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("approval-watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	streamID := fs.String("stream-id", "", "stable stream id")
	approvalID := fs.String("approval-id", "", "optional approval id filter")
	runID := fs.String("run-id", "", "optional run id filter")
	workspaceID := fs.String("workspace-id", "", "optional workspace id filter")
	status := fs.String("status", "", "optional approval status filter")
	follow := fs.Bool("follow", false, "stream follow mode")
	includeSnapshot := fs.Bool("include-snapshot", true, "include initial snapshot event")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "approval-watch usage: runecode-broker approval-watch [--stream-id id] [--approval-id sha256:...] [--run-id id] [--workspace-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--follow] [--include-snapshot]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	requestID := defaultRequestID()
	resolvedStreamID := *streamID
	if resolvedStreamID == "" {
		resolvedStreamID = "approval-watch-" + requestID
	}
	events, errResp := api.ApprovalWatch(ctx, brokerapi.ApprovalWatchRequest{
		SchemaID:        "runecode.protocol.v0.ApprovalWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       requestID,
		StreamID:        resolvedStreamID,
		ApprovalID:      *approvalID,
		RunID:           *runID,
		WorkspaceID:     *workspaceID,
		Status:          *status,
		Follow:          *follow,
		IncludeSnapshot: *includeSnapshot,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, events)
}

func handleVersionInfo(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.VersionInfoGet(ctx, brokerapi.VersionInfoGetRequest{
		SchemaID:      "runecode.protocol.v0.VersionInfoGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.VersionInfo)
}

func handleStreamLogs(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("stream-logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "filter by run id")
	roleInstanceID := fs.String("role-instance-id", "", "filter by role instance id")
	streamID := fs.String("stream-id", "", "stable stream id")
	startCursor := fs.String("start-cursor", "", "cursor to resume from")
	follow := fs.Bool("follow", false, "stream follow mode")
	includeBacklog := fs.Bool("include-backlog", true, "include backlog")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "stream-logs usage: runecode-broker stream-logs [--stream-id id] [--run-id id] [--role-instance-id id] [--start-cursor cursor] [--follow] [--include-backlog]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	requestID := defaultRequestID()
	resolvedStreamID := *streamID
	if resolvedStreamID == "" {
		resolvedStreamID = "log-" + requestID
	}
	events, errResp := api.LogStream(ctx, brokerapi.LogStreamRequest{
		SchemaID:       "runecode.protocol.v0.LogStreamRequest",
		SchemaVersion:  "0.1.0",
		RequestID:      requestID,
		StreamID:       resolvedStreamID,
		RunID:          *runID,
		RoleInstanceID: *roleInstanceID,
		StartCursor:    *startCursor,
		Follow:         *follow,
		IncludeBacklog: *includeBacklog,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, events)
}

func validSessionMessageRole(role string) bool {
	switch role {
	case "user", "assistant", "system", "tool":
		return true
	default:
		return false
	}
}
