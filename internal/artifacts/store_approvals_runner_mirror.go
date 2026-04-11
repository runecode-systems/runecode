package artifacts

import (
	"strings"
	"time"
)

func runnerApprovalFromCanonicalRecord(stored ApprovalRecord, nowFn func() time.Time) (*RunnerApproval, error) {
	if strings.TrimSpace(stored.RunID) == "" {
		return nil, nil
	}
	approvalType, actionHash, stageHash := approvalBindingForRunnerHint(stored)
	occurredAt := stored.RequestedAt
	if occurredAt.IsZero() {
		occurredAt = nowFn().UTC()
	}
	resolvedAt := stored.DecidedAt
	if strings.TrimSpace(stored.Status) == "consumed" && stored.ConsumedAt != nil {
		resolvedAt = stored.ConsumedAt
	}
	var resolvedCopy *time.Time
	if resolvedAt != nil {
		t := resolvedAt.UTC()
		resolvedCopy = &t
	}
	approval := RunnerApproval{
		ApprovalID:            stored.ApprovalID,
		RunID:                 stored.RunID,
		StageID:               stored.StageID,
		StepID:                stored.StepID,
		RoleInstanceID:        stored.RoleInstanceID,
		Status:                stored.Status,
		ApprovalType:          approvalType,
		BoundActionHash:       actionHash,
		BoundStageSummaryHash: stageHash,
		OccurredAt:            occurredAt.UTC(),
		ResolvedAt:            resolvedCopy,
		SupersededByApproval:  stored.SupersededByApprovalID,
	}
	return &approval, nil
}

func approvalBindingForRunnerHint(stored ApprovalRecord) (string, string, string) {
	if strings.TrimSpace(stored.ActionKind) == "stage_summary_sign_off" {
		return "stage_sign_off", "", stageSummaryHashForRunnerHint(stored)
	}
	return "exact_action", stored.ActionRequestHash, ""
}

func stageSummaryHashForRunnerHint(stored ApprovalRecord) string {
	if stored.RequestEnvelope == nil {
		return stored.ManifestHash
	}
	payload, err := decodeObjectPayload(stored.RequestEnvelope.Payload)
	if err != nil {
		return stored.ManifestHash
	}
	details, _ := payload["details"].(map[string]any)
	if details == nil {
		return stored.ManifestHash
	}
	digest, err := digestIdentityField(details, "stage_summary_hash")
	if err != nil {
		return stored.ManifestHash
	}
	return digest
}
