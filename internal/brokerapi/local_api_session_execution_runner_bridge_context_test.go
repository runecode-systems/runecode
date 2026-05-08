package brokerapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBridgeSessionExecutionTriggerToRunPreservesContextCancellation(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-session-bridge-cancelled"
	if err := s.SetRunStatus(runID, "starting"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	result := sessionExecutionPlanAuthorityAppendResult(s, runID)
	authority, err := s.ensureSessionExecutionRunPlanAuthority(result)
	if err != nil {
		t.Fatalf("ensureSessionExecutionRunPlanAuthority returned error: %v", err)
	}
	observedCancellation := false
	s.sessionExecutionRunner = func(ctx context.Context, _ *Service, _ sessionExecutionRunnerLaunchSpec) error {
		observedCancellation = errors.Is(ctx.Err(), context.Canceled)
		return ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.bridgeSessionExecutionTriggerToRun(ctx, "req-session-bridge-cancelled", result, authority)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("bridgeSessionExecutionTriggerToRun error = %v, want context canceled", err)
	}
	if !observedCancellation {
		t.Fatal("sessionExecutionRunner did not observe canceled context")
	}
	if status := s.RunStatuses()[runID]; status != "failed" {
		t.Fatalf("run status = %q, want failed", status)
	}
	runtimeFacts := s.RuntimeFacts(runID)
	if runtimeFacts.TerminalReport == nil || !runtimeFacts.TerminalReport.FailClosed {
		t.Fatalf("runtime terminal report = %+v, want fail_closed true", runtimeFacts.TerminalReport)
	}
}

func TestBridgeSessionExecutionTriggerToRunPreservesContextDeadline(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-session-bridge-deadline"
	if err := s.SetRunStatus(runID, "starting"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	result := sessionExecutionPlanAuthorityAppendResult(s, runID)
	authority, err := s.ensureSessionExecutionRunPlanAuthority(result)
	if err != nil {
		t.Fatalf("ensureSessionExecutionRunPlanAuthority returned error: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second).UTC().Round(0)
	observedDeadline := time.Time{}
	s.sessionExecutionRunner = func(ctx context.Context, _ *Service, _ sessionExecutionRunnerLaunchSpec) error {
		var ok bool
		observedDeadline, ok = ctx.Deadline()
		if !ok {
			t.Fatal("sessionExecutionRunner context missing deadline")
		}
		return nil
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	if err := s.bridgeSessionExecutionTriggerToRun(ctx, "req-session-bridge-deadline", result, authority); err != nil {
		t.Fatalf("bridgeSessionExecutionTriggerToRun returned error: %v", err)
	}
	if !observedDeadline.Equal(deadline) {
		t.Fatalf("observed deadline = %s, want %s", observedDeadline.Format(time.RFC3339Nano), deadline.Format(time.RFC3339Nano))
	}
}

func TestSessionExecutionRunnerSubprocessContextIgnoresCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runnerCtx, stopRunner := sessionExecutionRunnerSubprocessContext(ctx)
	defer stopRunner(nil)
	if err := runnerCtx.Err(); err != nil {
		t.Fatalf("runnerCtx.Err() = %v, want nil", err)
	}
	if _, ok := runnerCtx.Deadline(); ok {
		t.Fatal("runnerCtx unexpectedly preserved canceled caller deadline")
	}
}

func TestSessionExecutionRunnerSubprocessContextAllowsIntentionalShutdown(t *testing.T) {
	runnerCtx, stopRunner := sessionExecutionRunnerSubprocessContext(context.Background())
	stopRunner(context.Canceled)
	if err := runnerCtx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("runnerCtx.Err() = %v, want context canceled", err)
	}
	if cause := context.Cause(runnerCtx); !errors.Is(cause, context.Canceled) {
		t.Fatalf("context.Cause(runnerCtx) = %v, want context canceled", cause)
	}
}

func TestLaunchSessionExecutionRunnerSubprocessRejectsAlreadyCanceledRequestContext(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := launchSessionExecutionRunnerSubprocess(ctx, s, sessionExecutionRunnerLaunchSpec{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("launchSessionExecutionRunnerSubprocess error = %v, want context canceled", err)
	}
}

func TestLaunchSessionExecutionRunnerSubprocessRejectsExpiredRequestContext(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := launchSessionExecutionRunnerSubprocess(ctx, s, sessionExecutionRunnerLaunchSpec{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("launchSessionExecutionRunnerSubprocess error = %v, want deadline exceeded", err)
	}
}

func TestBridgeSessionExecutionTriggerToRunRejectsNilContext(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-session-bridge-nil-context"
	result := sessionExecutionPlanAuthorityAppendResult(s, runID)
	authority, err := s.ensureSessionExecutionRunPlanAuthority(result)
	if err != nil {
		t.Fatalf("ensureSessionExecutionRunPlanAuthority returned error: %v", err)
	}
	err = s.bridgeSessionExecutionTriggerToRun(nil, "req-session-bridge-nil-context", result, authority)
	if err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("bridgeSessionExecutionTriggerToRun error = %v, want context required detail", err)
	}
}

func TestBridgeSessionExecutionTriggerToRunRejectsMissingAuthorityRunID(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	result := sessionExecutionPlanAuthorityAppendResult(s, "run-session-bridge-missing-authority")
	authority, err := s.ensureSessionExecutionRunPlanAuthority(result)
	if err != nil {
		t.Fatalf("ensureSessionExecutionRunPlanAuthority returned error: %v", err)
	}
	authority.runID = ""
	err = s.bridgeSessionExecutionTriggerToRun(context.Background(), "req-session-bridge-missing-authority", result, authority)
	if err == nil || !strings.Contains(err.Error(), "trusted run id missing") {
		t.Fatalf("bridgeSessionExecutionTriggerToRun error = %v, want trusted run id missing detail", err)
	}
}
