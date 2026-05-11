package brokerapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/launcherbackend"
)

type sessionExecutionRunnerLaunchFunc func(context.Context, *Service, sessionExecutionRunnerLaunchSpec) error

type sessionExecutionRunnerLaunchSpec struct {
	requestID  string
	runID      string
	planID     string
	planPath   string
	sessionID  string
	runnerRoot string
}

func (s *Service) bridgeSessionExecutionTriggerToRun(ctx context.Context, requestID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) error {
	if ctx == nil {
		return fmt.Errorf("session execution bridge context is required")
	}
	runID := strings.TrimSpace(authority.runID)
	if runID == "" {
		return fmt.Errorf("trusted run id missing for session execution bridge")
	}
	planDigest := strings.TrimSpace(authority.runPlanDigest)
	if planDigest == "" {
		return fmt.Errorf("trusted run plan digest missing for run %q", runID)
	}
	planPath, err := s.exportSessionExecutionRunPlan(planDigest, authority.planID)
	if err != nil {
		return err
	}
	defer os.Remove(planPath)
	defer os.RemoveAll(filepath.Dir(planPath))
	if err := s.markSessionExecutionRunnerLaunching(runID, result.Trigger.SessionID); err != nil {
		return err
	}
	runnerRepoRoot := strings.TrimSpace(s.projectSubstrate.RepositoryRoot)
	if runnerRepoRoot == "" {
		runnerRepoRoot = strings.TrimSpace(s.apiConfig.RepositoryRoot)
	}
	if err := s.sessionExecutionRunner(ctx, s, sessionExecutionRunnerLaunchSpec{
		requestID:  requestID,
		runID:      runID,
		planID:     authority.planID,
		planPath:   planPath,
		sessionID:  strings.TrimSpace(result.Trigger.SessionID),
		runnerRoot: runnerRepoRoot,
	}); err != nil {
		if markErr := s.markSessionExecutionRunnerLaunchFailed(runID, result.Trigger.SessionID, err); markErr != nil {
			return fmt.Errorf("%v; additionally failed to persist runner launch failure: %w", err, markErr)
		}
		return err
	}
	return nil
}

func (s *Service) exportSessionExecutionRunPlan(planDigest, planID string) (string, error) {
	payload, err := s.readArtifactPayloadVerified(planDigest)
	if err != nil {
		return "", fmt.Errorf("read trusted run plan %q: %w", strings.TrimSpace(planDigest), err)
	}
	parentDir, err := os.MkdirTemp("", "runecode-plan-root-")
	if err != nil {
		return "", fmt.Errorf("create trusted run plan root: %w", err)
	}
	name := "runplan-*.json"
	if trimmed := strings.TrimSpace(planID); trimmed != "" {
		name = fmt.Sprintf("runplan-%s-*.json", sessionExecutionIdentifierToken(trimmed))
	}
	path, err := writeSessionExecutionTemporaryFile(parentDir, name, payload)
	if err != nil {
		_ = os.RemoveAll(parentDir)
		return "", fmt.Errorf("persist trusted run plan payload: %w", err)
	}
	return path, nil
}

func writeSessionExecutionTemporaryFile(parentDir, pattern string, payload []byte) (string, error) {
	file, err := os.CreateTemp(parentDir, pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(payload); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func (s *Service) markSessionExecutionRunnerLaunching(runID, sessionID string) error {
	if err := s.SetRunStatus(runID, "starting"); err != nil {
		return err
	}
	return s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                   runID,
		SessionID:               strings.TrimSpace(sessionID),
		BackendKind:             launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel: launcherbackend.IsolationAssuranceUnknown,
		Lifecycle:               &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateLaunching},
	}})
}

func (s *Service) markSessionExecutionRunnerLaunchFailed(runID, sessionID string, launchErr error) error {
	if err := s.SetRunStatus(runID, "failed"); err != nil {
		return err
	}
	return s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                   runID,
		SessionID:               strings.TrimSpace(sessionID),
		BackendKind:             launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel: launcherbackend.IsolationAssuranceUnknown,
		LaunchFailureReasonCode: "runner_stdio_bridge_failed",
		Lifecycle:               &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateTerminated},
	}, TerminalReport: &launcherbackend.BackendTerminalReport{
		RunID:             runID,
		SessionID:         strings.TrimSpace(sessionID),
		TerminationKind:   launcherbackend.BackendTerminationKindFailed,
		FailureReasonCode: "runner_stdio_bridge_failed",
		FailClosed:        true,
		FallbackPosture:   launcherbackend.BackendFallbackPostureNoAutomaticFallback,
	}})
}

func requestIDForRunnerTransport(requestID, runID, kind string, messageIndex int) string {
	base := strings.TrimSpace(requestID)
	if base == "" {
		base = "runner-bridge"
	}
	return fmt.Sprintf("%s:%s:%s:%d", base, strings.TrimSpace(runID), kind, messageIndex)
}
