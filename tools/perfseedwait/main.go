package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type config struct {
	runtimeDir string
	socketName string
	sessionID  string
}

func main() {
	cfg := parseConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := brokerapi.DialLocalRPC(ctx, brokerapi.LocalIPCConfig{RuntimeDir: cfg.runtimeDir, SocketName: cfg.socketName})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial local rpc: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	seedWaitSession(ctx, client, cfg)
	printSession(ctx, client, cfg.sessionID)
}

func parseConfig() config {
	runtimeDir := flag.String("runtime-dir", "", "runtime directory")
	socketName := flag.String("socket-name", "perf.sock", "socket name")
	sessionID := flag.String("session-id", "", "session id")
	flag.Parse()
	if *runtimeDir == "" {
		fmt.Fprintln(os.Stderr, "--runtime-dir is required")
		os.Exit(2)
	}
	if *sessionID == "" {
		fmt.Fprintln(os.Stderr, "--session-id is required")
		os.Exit(2)
	}
	return config{runtimeDir: *runtimeDir, socketName: *socketName, sessionID: *sessionID}
}

func seedWaitSession(ctx context.Context, client *brokerapi.LocalRPCClient, cfg config) {
	for i, msg := range []string{"first", "second"} {
		resp := brokerapi.SessionExecutionTriggerResponse{}
		errResp := client.Invoke(ctx, "session_execution_trigger", brokerapi.SessionExecutionTriggerRequest{
			SchemaID:               "runecode.protocol.v0.SessionExecutionTriggerRequest",
			SchemaVersion:          "0.1.0",
			RequestID:              fmt.Sprintf("perf-seed-%d", i+1),
			SessionID:              cfg.sessionID,
			TriggerSource:          "autonomous_background",
			RequestedOperation:     "start",
			WorkflowRouting:        &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "approved_change_implementation"},
			AutonomyPosture:        "operator_guided",
			UserMessageContentText: msg,
		}, &resp)
		if errResp != nil {
			fmt.Fprintf(os.Stderr, "session_execution_trigger %d: %s: %s\n", i+1, errResp.Error.Code, errResp.Error.Message)
			os.Exit(1)
		}
	}
}

func printSession(ctx context.Context, client *brokerapi.LocalRPCClient, sessionID string) {
	result := brokerapi.SessionGetResponse{}
	errResp := client.Invoke(ctx, "session_get", brokerapi.SessionGetRequest{
		SchemaID:      "runecode.protocol.v0.SessionGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "perf-seed-get",
		SessionID:     sessionID,
	}, &result)
	if errResp != nil {
		fmt.Fprintf(os.Stderr, "session_get: %s: %s\n", errResp.Error.Code, errResp.Error.Message)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result.Session); err != nil {
		fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
		os.Exit(1)
	}
}
