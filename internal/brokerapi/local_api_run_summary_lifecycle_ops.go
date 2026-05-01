package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func runLifecycleFromStore(status string, pendingApprovals int, hasArtifacts bool, runnerAdvisory artifacts.RunnerAdvisoryState, runtimeFacts launcherbackend.RuntimeFactsSnapshot) string {
	if pendingApprovals > 0 {
		return "blocked"
	}
	if runtimeLifecycle, ok := authoritativeRuntimeLifecycle(runtimeFacts); ok {
		return runtimeLifecycle
	}
	if terminal, ok := terminalLifecycleFromStoreStatus(status); ok {
		return terminal
	}
	if advisoryLifecycle, ok := advisoryRunnableLifecycle(runnerAdvisory); ok {
		return advisoryLifecycle
	}
	if mapped, ok := mappedRunnableStoreLifecycle(status, hasArtifacts); ok {
		return mapped
	}
	if !hasArtifacts {
		return "pending"
	}
	return "active"
}

func authoritativeRuntimeLifecycle(runtimeFacts launcherbackend.RuntimeFactsSnapshot) (string, bool) {
	if runtimeFacts.TerminalReport != nil {
		switch runtimeFacts.TerminalReport.TerminationKind {
		case launcherbackend.BackendTerminationKindCompleted:
			return "completed", true
		case launcherbackend.BackendTerminationKindFailed:
			return "failed", true
		}
	}
	receipt := runtimeFacts.LaunchReceipt.Normalized()
	if strings.TrimSpace(receipt.LaunchFailureReasonCode) != "" {
		return "failed", true
	}
	if !hasAuthoritativeRuntimeLifecycle(receipt) {
		return "", false
	}
	switch receipt.Lifecycle.CurrentState {
	case launcherbackend.BackendLifecycleStatePlanned:
		return "pending", true
	case launcherbackend.BackendLifecycleStateLaunching, launcherbackend.BackendLifecycleStateStarted, launcherbackend.BackendLifecycleStateBinding:
		return "starting", true
	case launcherbackend.BackendLifecycleStateActive, launcherbackend.BackendLifecycleStateTerminating:
		return "active", true
	case launcherbackend.BackendLifecycleStateTerminated:
		return "completed", true
	default:
		return "", false
	}
}

func terminalLifecycleFromStoreStatus(status string) (string, bool) {
	switch status {
	case "completed", "failed", "cancelled":
		return status, true
	case "retained", "closed":
		return "completed", true
	default:
		return "", false
	}
}

func advisoryRunnableLifecycle(runnerAdvisory artifacts.RunnerAdvisoryState) (string, bool) {
	if runnerAdvisory.Lifecycle == nil {
		return "", false
	}
	advisoryLifecycle := strings.TrimSpace(runnerAdvisory.Lifecycle.LifecycleState)
	switch advisoryLifecycle {
	case "pending", "starting", "active", "blocked", "recovering":
		return advisoryLifecycle, true
	default:
		return "", false
	}
}

func mappedRunnableStoreLifecycle(status string, hasArtifacts bool) (string, bool) {
	switch status {
	case "pending", "starting", "active", "blocked", "recovering", "completed", "failed", "cancelled":
		if status == "active" && !hasArtifacts {
			return "starting", true
		}
		return status, true
	default:
		return "", false
	}
}
