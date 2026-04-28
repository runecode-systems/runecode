package main

import (
	"context"
	"flag"
	"io"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

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

func handleSessionExecutionTrigger(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("session-execution-trigger", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sessionID := fs.String("session-id", "", "session id")
	turnID := fs.String("turn-id", "", "optional target turn id for continue")
	triggerSource := fs.String("trigger-source", "interactive_user", "trigger source classification")
	requestedOperation := fs.String("requested-operation", "start", "requested execution operation")
	workflowFamily := fs.String("workflow-family", "runecontext", "workflow pack family")
	workflowOperation := fs.String("workflow-operation", "draft_promote_apply", "workflow pack operation")
	userMessage := fs.String("user-message", "", "optional user message content")
	idempotencyKey := fs.String("idempotency-key", "", "optional idempotency key")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "session-execution-trigger usage: runecode-broker session-execution-trigger --session-id id [--turn-id id] [--trigger-source interactive_user|autonomous_background|resume_follow_up] [--requested-operation start|continue] [--workflow-family runecontext] [--workflow-operation change_draft|spec_draft|draft_promote_apply|approved_change_implementation] [--user-message text] [--idempotency-key key]"}
	}
	if *sessionID == "" {
		return &usageError{message: "session-execution-trigger requires --session-id"}
	}
	if !validSessionExecutionTriggerSource(*triggerSource) {
		return &usageError{message: "session-execution-trigger --trigger-source must be one of: interactive_user|autonomous_background|resume_follow_up"}
	}
	if !validSessionExecutionRequestedOperation(*requestedOperation) {
		return &usageError{message: "session-execution-trigger --requested-operation must be one of: start|continue"}
	}
	if *triggerSource == "interactive_user" && *userMessage == "" {
		return &usageError{message: "session-execution-trigger --user-message is required for interactive_user trigger source"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.SessionExecutionTrigger(ctx, brokerapi.SessionExecutionTriggerRequest{
		SchemaID:               "runecode.protocol.v0.SessionExecutionTriggerRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              defaultRequestID(),
		SessionID:              *sessionID,
		TurnID:                 *turnID,
		TriggerSource:          *triggerSource,
		RequestedOperation:     *requestedOperation,
		WorkflowRouting:        sessionWorkflowRoutingForCLI(*requestedOperation, *workflowFamily, *workflowOperation),
		UserMessageContentText: *userMessage,
		IdempotencyKey:         *idempotencyKey,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func sessionWorkflowRoutingForCLI(requestedOperation, family, operation string) *brokerapi.SessionWorkflowPackRouting {
	if requestedOperation == "continue" {
		return nil
	}
	return &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: family, WorkflowOperation: operation}
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

func validSessionMessageRole(role string) bool {
	switch role {
	case "user", "assistant", "system", "tool":
		return true
	default:
		return false
	}
}

func validSessionExecutionTriggerSource(source string) bool {
	switch source {
	case "interactive_user", "autonomous_background", "resume_follow_up":
		return true
	default:
		return false
	}
}

func validSessionExecutionRequestedOperation(operation string) bool {
	switch operation {
	case "start", "continue":
		return true
	default:
		return false
	}
}
