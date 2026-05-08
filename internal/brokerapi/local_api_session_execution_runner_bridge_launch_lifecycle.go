package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func sessionExecutionRunnerSubprocessContext(ctx context.Context) (context.Context, context.CancelCauseFunc) {
	return context.WithCancelCause(context.WithoutCancel(ctx))
}

func sessionExecutionRunnerRequestLifecycleErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		if err := context.Cause(ctx); err != nil {
			return err
		}
		return ctx.Err()
	default:
		return nil
	}
}

func waitForSessionExecutionRunner(requestCtx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec, runnerCtx context.Context, stopRunner context.CancelCauseFunc, cmd *exec.Cmd, stdin io.WriteCloser, stdout io.Reader, stderrBytes *boundedSessionExecutionRunnerStderrCapture, stderrDone <-chan struct{}) error {
	transportDone := make(chan struct{})
	requestLifecycleErr, requestDone := watchSessionExecutionRunnerRequestLifecycle(requestCtx, stopRunner, transportDone)
	handleErr := make(chan error, 1)
	go func() {
		err := s.proxyRunnerTransport(runnerCtx, spec.requestID, spec.runID, stdin, stdout)
		_ = stdin.Close()
		close(transportDone)
		handleErr <- err
	}()
	transportErr := <-handleErr
	waitErr := cmd.Wait()
	<-stderrDone
	<-requestDone
	if err := terminalRequestLifecycleErr(requestLifecycleErr); err != nil {
		return err
	}
	if transportErr != nil {
		return runnerTransportFailure(transportErr, waitErr, stderrBytes)
	}
	if waitErr != nil {
		return fmt.Errorf("runner subprocess failed: %v (stderr: %s)", waitErr, summarizeRunnerStderr(stderrBytes.String(), stderrBytes.Truncated()))
	}
	return nil
}

func watchSessionExecutionRunnerRequestLifecycle(requestCtx context.Context, stopRunner context.CancelCauseFunc, transportDone <-chan struct{}) (<-chan error, <-chan struct{}) {
	requestLifecycleErr := make(chan error, 1)
	requestDone := make(chan struct{})
	go func() {
		select {
		case <-requestCtx.Done():
			err := sessionExecutionRunnerRequestLifecycleErr(requestCtx)
			if err != nil {
				stopRunner(err)
				select {
				case requestLifecycleErr <- err:
				default:
				}
			}
		case <-transportDone:
		}
		close(requestDone)
	}()
	return requestLifecycleErr, requestDone
}

func terminalRequestLifecycleErr(requestLifecycleErr <-chan error) error {
	select {
	case err := <-requestLifecycleErr:
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
	default:
	}
	return nil
}

func runnerTransportFailure(transportErr, waitErr error, stderrBytes *boundedSessionExecutionRunnerStderrCapture) error {
	if waitErr != nil {
		return fmt.Errorf("runner transport failed: %v (runner exit: %v; stderr: %s)", transportErr, waitErr, summarizeRunnerStderr(stderrBytes.String(), stderrBytes.Truncated()))
	}
	return fmt.Errorf("runner transport failed: %v (stderr: %s)", transportErr, summarizeRunnerStderr(stderrBytes.String(), stderrBytes.Truncated()))
}
