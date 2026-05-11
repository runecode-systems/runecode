package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func TestRunAndSessionCommandsRejectPositionalArguments(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		t.Fatalf("unexpected local rpc dispatch for %s", wire.Operation)
		return localRPCResponse{}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })

	for _, tt := range positionalArgRejectionCases() {
		t.Run(tt.name, func(t *testing.T) {
			stdout.Reset()
			stderr.Reset()
			err := run(tt.args, stdout, stderr)
			if err == nil {
				t.Fatalf("%s expected usage error for positional arguments", tt.name)
			}
			usageErr, ok := err.(*usageError)
			if !ok {
				t.Fatalf("%s error type = %T, want *usageError", tt.name, err)
			}
			if usageErr.Error() != tt.wantErr {
				t.Fatalf("%s error = %q, want %q", tt.name, usageErr.Error(), tt.wantErr)
			}
		})
	}
}

type positionalArgRejectionCase struct {
	name    string
	args    []string
	wantErr string
}

func positionalArgRejectionCases() []positionalArgRejectionCase {
	return []positionalArgRejectionCase{
		{name: "run-list", args: []string{"run-list", "--limit", "1", "extra"}, wantErr: "run-list does not accept positional arguments"},
		{name: "run-get", args: []string{"run-get", "--run-id", "run-1", "extra"}, wantErr: "run-get does not accept positional arguments"},
		{name: "run-watch", args: []string{"run-watch", "--follow", "extra"}, wantErr: "run-watch does not accept positional arguments"},
		{name: "session-list", args: []string{"session-list", "--limit", "1", "extra"}, wantErr: "session-list does not accept positional arguments"},
		{name: "session-get", args: []string{"session-get", "--session-id", "sess-1", "extra"}, wantErr: "session-get does not accept positional arguments"},
		{name: "session-send-message", args: []string{"session-send-message", "--session-id", "sess-1", "--content", "hello", "extra"}, wantErr: "session-send-message does not accept positional arguments"},
		{name: "session-execution-trigger", args: []string{"session-execution-trigger", "--session-id", "sess-1", "--trigger-source", "interactive_user", "--requested-operation", "start", "--user-message", "hello", "extra"}, wantErr: "session-execution-trigger does not accept positional arguments"},
		{name: "session-watch", args: []string{"session-watch", "--follow", "extra"}, wantErr: "session-watch does not accept positional arguments"},
	}
}
