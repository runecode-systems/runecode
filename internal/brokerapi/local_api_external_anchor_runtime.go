package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type externalAnchorExecutionInput struct {
	AttemptID            string
	PreparedMutationID   string
	RunID                string
	SealDigestIdentity   string
	TypedRequestHash     string
	TargetDigestIdentity string
	TargetAuthLeaseID    string
	RequestID            string
	SnapshotSegmentID    string
	SnapshotSealDigest   string
	Mode                 string
	PollRemaining        int
}

type externalAnchorExecutionOutcome struct {
	ExecutionState      string
	ExecutionReasonCode string
	LifecycleState      string
	LifecycleReasonCode string
}

type externalAnchorExecutionRuntime interface {
	Execute(ctx context.Context, input externalAnchorExecutionInput) externalAnchorExecutionOutcome
}

type externalAnchorExecutionRuntimeFunc func(ctx context.Context, input externalAnchorExecutionInput) externalAnchorExecutionOutcome

func (f externalAnchorExecutionRuntimeFunc) Execute(ctx context.Context, input externalAnchorExecutionInput) externalAnchorExecutionOutcome {
	return f(ctx, input)
}

type externalAnchorExecutionRuntimeDeterministic struct{}

func (externalAnchorExecutionRuntimeDeterministic) Execute(_ context.Context, input externalAnchorExecutionInput) externalAnchorExecutionOutcome {
	switch strings.TrimSpace(input.Mode) {
	case "deferred_poll":
		if input.PollRemaining <= 0 {
			return externalAnchorExecutionOutcome{
				ExecutionState:      gitRemoteMutationExecutionCompleted,
				ExecutionReasonCode: "",
				LifecycleState:      gitRemoteMutationLifecycleExecuted,
				LifecycleReasonCode: "",
			}
		}
		fallthrough
	default:
		return externalAnchorExecutionOutcome{
			ExecutionState:      gitRemoteMutationExecutionDeferred,
			ExecutionReasonCode: "external_anchor_execution_deferred",
			LifecycleState:      gitRemoteMutationLifecyclePrepared,
			LifecycleReasonCode: gitRemoteMutationLifecycleDeferredReason,
		}
	}
}

func externalAnchorDeferredBackoff(pollsRemaining int) time.Duration {
	step := pollsRemaining
	if step < 1 {
		step = 1
	}
	if step > 6 {
		step = 6
	}
	return time.Duration(step) * 5 * time.Millisecond
}

func externalAnchorExecutionInputFromRecord(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorExecutionInput, error) {
	pollRemaining := record.LastExecuteDeferredPolls
	sealIdentity, targetIdentity, typedRequestIdentity, err := externalAnchorAttemptDigestIdentities(record)
	if err != nil {
		return externalAnchorExecutionInput{}, err
	}
	snapshotSealIdentity, err := externalAnchorOptionalDigestIdentity(record.LastExecuteSnapshotSealID, "last_execute_snapshot_seal_digest")
	if err != nil {
		return externalAnchorExecutionInput{}, err
	}
	mode := "deferred"
	if intField(record.TypedRequest, "deferred_poll_count") > 0 {
		mode = "deferred_poll"
	}
	return externalAnchorExecutionInput{
		AttemptID:            strings.TrimSpace(record.LastExecuteAttemptID),
		PreparedMutationID:   strings.TrimSpace(record.PreparedMutationID),
		RunID:                strings.TrimSpace(record.RunID),
		SealDigestIdentity:   sealIdentity,
		TypedRequestHash:     typedRequestIdentity,
		TargetDigestIdentity: targetIdentity,
		TargetAuthLeaseID:    strings.TrimSpace(record.LastExecuteTargetAuthLeaseID),
		RequestID:            strings.TrimSpace(record.LastExecuteRequestID),
		SnapshotSegmentID:    strings.TrimSpace(record.LastExecuteSnapshotSegmentID),
		SnapshotSealDigest:   snapshotSealIdentity,
		Mode:                 mode,
		PollRemaining:        pollRemaining,
	}, nil
}

func externalAnchorAttemptDigestIdentities(record artifacts.ExternalAnchorPreparedMutationRecord) (string, string, string, error) {
	sealIdentity, err := externalAnchorNormalizeDigestIdentity(record.LastExecuteAttemptSealDigest, "last_execute_attempt_seal_digest")
	if err != nil {
		return "", "", "", err
	}
	targetIdentity, err := externalAnchorNormalizeDigestIdentity(record.LastExecuteAttemptTargetID, "last_execute_attempt_target_descriptor_digest")
	if err != nil {
		return "", "", "", err
	}
	typedRequestIdentity, err := externalAnchorNormalizeDigestIdentity(record.LastExecuteAttemptRequestID, "last_execute_attempt_typed_request_hash")
	if err != nil {
		return "", "", "", err
	}
	return sealIdentity, targetIdentity, typedRequestIdentity, nil
}

func externalAnchorOptionalDigestIdentity(identity string, field string) (string, error) {
	trimmed := strings.TrimSpace(identity)
	if trimmed == "" {
		return "", nil
	}
	return externalAnchorNormalizeDigestIdentity(trimmed, field)
}

func setExternalAnchorExecutionOutcome(record *artifacts.ExternalAnchorPreparedMutationRecord, outcome externalAnchorExecutionOutcome) {
	record.ExecutionState = strings.TrimSpace(outcome.ExecutionState)
	record.ExecutionReasonCode = strings.TrimSpace(outcome.ExecutionReasonCode)
	record.LifecycleState = strings.TrimSpace(outcome.LifecycleState)
	record.LifecycleReasonCode = strings.TrimSpace(outcome.LifecycleReasonCode)
	if strings.TrimSpace(record.ExecutionState) != gitRemoteMutationExecutionDeferred {
		record.LastExecuteDeferredPolls = 0
		record.LastExecuteDeferredClaimID = ""
		record.LastExecuteDeferredClaimedAt = nil
	}
}

func normalizeExternalAnchorExecutionOutcome(outcome externalAnchorExecutionOutcome) externalAnchorExecutionOutcome {
	if strings.TrimSpace(outcome.ExecutionState) == "" {
		outcome.ExecutionState = gitRemoteMutationExecutionFailed
	}
	if strings.TrimSpace(outcome.LifecycleState) == "" {
		outcome.LifecycleState = gitRemoteMutationLifecycleFailed
	}
	return outcome
}

func setExternalAnchorDeferredPollRemaining(record *artifacts.ExternalAnchorPreparedMutationRecord, pollRemaining int) {
	if record == nil {
		return
	}
	if pollRemaining < 0 {
		pollRemaining = 0
	}
	record.LastExecuteDeferredPolls = pollRemaining
}

func externalAnchorNormalizeDigestIdentity(identity string, field string) (string, error) {
	trimmed := strings.TrimSpace(identity)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	d, err := digestFromIdentity(trimmed)
	if err != nil {
		return "", fmt.Errorf("%s invalid: %w", field, err)
	}
	normalized, err := d.Identity()
	if err != nil {
		return "", fmt.Errorf("%s invalid: %w", field, err)
	}
	return normalized, nil
}
