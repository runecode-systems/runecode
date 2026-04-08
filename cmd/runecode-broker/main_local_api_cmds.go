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
