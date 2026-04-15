package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"strings"
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

func handleBackendPostureGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("backend-posture-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "backend-posture-get usage: runecode-broker backend-posture-get"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.BackendPostureGet(ctx, brokerapi.BackendPostureGetRequest{
		SchemaID:      "runecode.protocol.v0.BackendPostureGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Posture)
}

func handleBackendPostureChange(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("backend-posture-change", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	targetInstanceID := fs.String("target-instance-id", "", "target launcher instance id")
	targetBackendKind := fs.String("target-backend-kind", "", "target backend kind (microvm|container)")
	selectionMode := fs.String("selection-mode", "explicit_selection", "selection mode")
	changeKind := fs.String("change-kind", "select_backend", "change kind")
	assuranceChangeKind := fs.String("assurance-change-kind", "reduce_assurance", "assurance change kind")
	optInKind := fs.String("opt-in-kind", "exact_action_approval", "opt-in kind")
	reducedAssuranceAcknowledged := fs.Bool("reduced-assurance-acknowledged", false, "operator acknowledged reduced assurance semantics")
	reason := fs.String("reason", "operator_requested_reduced_assurance_backend_opt_in", "change reason")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "backend-posture-change usage: runecode-broker backend-posture-change --target-backend-kind microvm|container [--target-instance-id id] [--selection-mode explicit_selection|automatic_fallback_attempt] [--change-kind select_backend] [--assurance-change-kind reduce_assurance|maintain_assurance] [--opt-in-kind exact_action_approval|none] [--reduced-assurance-acknowledged] [--reason text]"}
	}
	if strings.TrimSpace(*targetBackendKind) == "" {
		return &usageError{message: "backend-posture-change requires --target-backend-kind"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resolvedInstanceID := strings.TrimSpace(*targetInstanceID)
	if resolvedInstanceID == "" {
		postureResp, errResp := api.BackendPostureGet(ctx, brokerapi.BackendPostureGetRequest{
			SchemaID:      "runecode.protocol.v0.BackendPostureGetRequest",
			SchemaVersion: "0.1.0",
			RequestID:     defaultRequestID(),
		})
		if errResp != nil {
			return fmt.Errorf("backend-posture-change requires --target-instance-id when backend posture cannot be queried: %w", localAPIError(errResp))
		}
		resolvedInstanceID = strings.TrimSpace(postureResp.Posture.InstanceID)
	}
	if resolvedInstanceID == "" {
		return &usageError{message: "backend-posture-change requires --target-instance-id (could not infer active instance)"}
	}
	resp, errResp := api.BackendPostureChange(ctx, brokerapi.BackendPostureChangeRequest{
		SchemaID:                     "runecode.protocol.v0.BackendPostureChangeRequest",
		SchemaVersion:                "0.1.0",
		RequestID:                    defaultRequestID(),
		TargetInstanceID:             resolvedInstanceID,
		TargetBackendKind:            strings.ToLower(strings.TrimSpace(*targetBackendKind)),
		SelectionMode:                strings.TrimSpace(*selectionMode),
		ChangeKind:                   strings.TrimSpace(*changeKind),
		AssuranceChangeKind:          strings.TrimSpace(*assuranceChangeKind),
		OptInKind:                    strings.TrimSpace(*optInKind),
		ReducedAssuranceAcknowledged: *reducedAssuranceAcknowledged,
		Reason:                       strings.TrimSpace(*reason),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
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

func handleLLMInvoke(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("llm-invoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "run id")
	requestFile := fs.String("request-file", "", "path to LLMRequest JSON")
	requestDigest := fs.String("request-digest", "", "optional canonical digest identity")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "llm-invoke usage: runecode-broker llm-invoke --run-id id --request-file path [--request-digest sha256:...]"}
	}
	if *runID == "" {
		return &usageError{message: "llm-invoke requires --run-id"}
	}
	if *requestFile == "" {
		return &usageError{message: "llm-invoke requires --request-file"}
	}
	parsedRequestDigest, err := parseOptionalDigestFlag(*requestDigest, "--request-digest")
	if err != nil {
		return &usageError{message: err.Error()}
	}
	requestValue, err := loadJSONValue(*requestFile)
	if err != nil {
		return err
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.LLMInvoke(ctx, brokerapi.LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), RunID: *runID, LLMRequest: requestValue, RequestDigest: parsedRequestDigest})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleLLMStream(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("llm-stream", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "run id")
	requestFile := fs.String("request-file", "", "path to LLMRequest JSON")
	requestDigest := fs.String("request-digest", "", "optional canonical digest identity")
	streamID := fs.String("stream-id", "", "stable stream id")
	follow := fs.Bool("follow", false, "stream follow mode")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "llm-stream usage: runecode-broker llm-stream --run-id id --request-file path [--request-digest sha256:...] [--stream-id id] [--follow]"}
	}
	if *runID == "" {
		return &usageError{message: "llm-stream requires --run-id"}
	}
	if *requestFile == "" {
		return &usageError{message: "llm-stream requires --request-file"}
	}
	parsedRequestDigest, err := parseOptionalDigestFlag(*requestDigest, "--request-digest")
	if err != nil {
		return &usageError{message: err.Error()}
	}
	requestValue, err := loadJSONValue(*requestFile)
	if err != nil {
		return err
	}
	resolvedStreamID := *streamID
	if resolvedStreamID == "" {
		resolvedStreamID = "llm-stream-" + defaultRequestID()
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.LLMStream(ctx, brokerapi.LLMStreamRequest{SchemaID: "runecode.protocol.v0.LLMStreamRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), RunID: *runID, StreamID: resolvedStreamID, LLMRequest: requestValue, RequestDigest: parsedRequestDigest, Follow: *follow})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func loadJSONValue(path string) (any, error) {
	value := map[string]any{}
	if err := loadStrictJSONFileValue(path, &value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
