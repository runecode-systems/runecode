package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func runLifecycleFromStore(status string, pendingApprovals int, hasArtifacts bool, runnerAdvisory artifacts.RunnerAdvisoryState) string {
	if pendingApprovals > 0 {
		return "blocked"
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
