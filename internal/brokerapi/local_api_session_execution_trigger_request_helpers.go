package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

type sessionExecutionTriggerControlValues struct {
	approvalProfile string
	autonomyPosture string
}

func sessionExecutionTriggerControls(req SessionExecutionTriggerRequest) sessionExecutionTriggerControlValues {
	return sessionExecutionTriggerControlValues{
		approvalProfile: normalizeSessionTriggerApprovalProfile(req.ApprovalProfile),
		autonomyPosture: normalizeSessionTriggerAutonomyPosture(req.AutonomyPosture),
	}
}

func sessionExecutionBoundDigest(project projectsubstrate.DiscoveryResult) string {
	boundDigest := strings.TrimSpace(project.Snapshot.ValidatedSnapshotDigest)
	if boundDigest == "" {
		boundDigest = strings.TrimSpace(project.Snapshot.ProjectContextIdentityDigest)
	}
	return boundDigest
}

func sessionExecutionInitialState(triggerSource, autonomyPosture string) (string, string, string) {
	if triggerSource != "autonomous_background" {
		return "running", "", ""
	}
	if autonomyPosture == "operator_guided" {
		return "waiting", "operator_input", "waiting_operator_input"
	}
	return "waiting", "approval", "waiting_approval"
}
