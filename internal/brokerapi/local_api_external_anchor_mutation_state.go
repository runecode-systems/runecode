package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func externalAnchorPreparedMutationID(runID, destinationRef, typedRequestHash, actionRequestHash, policyDecisionHash string) (string, error) {
	payload := map[string]any{
		"run_id":               strings.TrimSpace(runID),
		"destination_ref":      strings.TrimSpace(destinationRef),
		"typed_request_hash":   strings.TrimSpace(typedRequestHash),
		"action_request_hash":  strings.TrimSpace(actionRequestHash),
		"policy_decision_hash": strings.TrimSpace(policyDecisionHash),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}

func externalAnchorPreparedStateFromRecord(record artifacts.ExternalAnchorPreparedMutationRecord) (ExternalAnchorMutationPreparedState, error) {
	required, optional, err := externalAnchorPreparedStateDigestsFromRecord(record)
	if err != nil {
		return ExternalAnchorMutationPreparedState{}, err
	}
	state := externalAnchorPreparedStateCore(record)
	primaryTarget, targetSet, err := externalAnchorPreparedTargetStateFromRecord(record)
	if err != nil {
		return ExternalAnchorMutationPreparedState{}, err
	}
	state.PrimaryTarget = primaryTarget
	state.TargetSet = targetSet
	applyExternalAnchorPreparedStateRequiredDigests(&state, required)
	applyExternalAnchorPreparedStateOptionalDigests(&state, optional)
	return state, nil
}

func externalAnchorPreparedStateCore(record artifacts.ExternalAnchorPreparedMutationRecord) ExternalAnchorMutationPreparedState {
	return ExternalAnchorMutationPreparedState{
		SchemaID:                     "runecode.protocol.v0.ExternalAnchorMutationPreparedState",
		SchemaVersion:                "0.1.0",
		PreparedMutationID:           record.PreparedMutationID,
		RunID:                        record.RunID,
		ExecutionPathway:             externalAnchorExecutionPathway(),
		AnchorPosture:                externalAnchorPostureFromRecord(record),
		DestinationRef:               record.DestinationRef,
		RequestKind:                  record.RequestKind,
		TypedRequestSchemaID:         record.TypedRequestSchemaID,
		TypedRequestSchemaVersion:    record.TypedRequestSchemaVer,
		TypedRequest:                 cloneStringAnyMap(record.TypedRequest),
		RequiredApprovalID:           record.RequiredApprovalID,
		LifecycleState:               record.LifecycleState,
		LifecycleReasonCode:          record.LifecycleReasonCode,
		ExecutionState:               record.ExecutionState,
		ExecutionReasonCode:          record.ExecutionReasonCode,
		CreatedAt:                    record.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                    record.UpdatedAt.UTC().Format(time.RFC3339),
		LastPrepareRequestID:         record.LastPrepareRequestID,
		LastGetRequestID:             record.LastGetRequestID,
		LastExecuteRequestID:         record.LastExecuteRequestID,
		LastExecuteTargetAuthLeaseID: record.LastExecuteTargetAuthLeaseID,
		LastExecuteAttemptID:         record.LastExecuteAttemptID,
		LastExecuteSnapshotSegmentID: record.LastExecuteSnapshotSegmentID,
		LastExecuteDeferredPolls:     record.LastExecuteDeferredPolls,
		LastExecuteDeferredClaimID:   record.LastExecuteDeferredClaimID,
	}
}

func externalAnchorPreparedTargetStateFromRecord(record artifacts.ExternalAnchorPreparedMutationRecord) (ExternalAnchorMutationPreparedTarget, []ExternalAnchorMutationPreparedTarget, error) {
	primary, err := externalAnchorPreparedSingleTargetStateFromBinding(record.PrimaryTarget, "primary_target")
	if err != nil {
		return ExternalAnchorMutationPreparedTarget{}, nil, err
	}
	targets := make([]ExternalAnchorMutationPreparedTarget, 0, len(record.TargetSet))
	for i := range record.TargetSet {
		target, err := externalAnchorPreparedSingleTargetStateFromBinding(record.TargetSet[i], fmt.Sprintf("target_set[%d]", i))
		if err != nil {
			return ExternalAnchorMutationPreparedTarget{}, nil, err
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		targets = []ExternalAnchorMutationPreparedTarget{primary}
	}
	return primary, targets, nil
}

func externalAnchorPreparedSingleTargetStateFromBinding(target artifacts.ExternalAnchorPreparedTargetBinding, field string) (ExternalAnchorMutationPreparedTarget, error) {
	digest, err := digestFromIdentity(strings.TrimSpace(target.TargetDescriptorDigest))
	if err != nil {
		return ExternalAnchorMutationPreparedTarget{}, fmt.Errorf("%s.target_descriptor_digest invalid: %w", field, err)
	}
	return ExternalAnchorMutationPreparedTarget{
		TargetKind:             strings.TrimSpace(target.TargetKind),
		TargetRequirement:      target.TargetRequirement,
		TargetDescriptor:       cloneStringAnyMap(target.TargetDescriptor),
		TargetDescriptorDigest: digest,
	}, nil
}

func applyExternalAnchorPreparedStateRequiredDigests(state *ExternalAnchorMutationPreparedState, required externalAnchorPreparedStateRequiredDigests) {
	state.TypedRequestHash = required.TypedRequestHash
	state.ActionRequestHash = required.ActionRequestHash
	state.PolicyDecisionHash = required.PolicyDecisionHash
}

func applyExternalAnchorPreparedStateOptionalDigests(state *ExternalAnchorMutationPreparedState, optional externalAnchorPreparedStateOptionalDigests) {
	state.RequiredApprovalRequestHash = optional.RequiredApprovalRequestHash
	state.RequiredApprovalDecisionHash = optional.RequiredApprovalDecisionHash
	state.LastExecuteAttemptSealDigest = optional.LastExecuteAttemptSealDigest
	state.LastExecuteAttemptTargetID = optional.LastExecuteAttemptTargetDigest
	state.LastExecuteAttemptRequestID = optional.LastExecuteAttemptRequestDigest
	state.LastExecuteSnapshotSealID = optional.LastExecuteSnapshotSealDigest
	state.LastAnchorReceiptDigest = optional.LastAnchorReceiptDigest
	state.LastAnchorEvidenceDigest = optional.LastAnchorEvidenceDigest
	state.LastAnchorVerificationDigest = optional.LastAnchorVerificationDigest
	state.LastAnchorProofDigest = optional.LastAnchorProofDigest
	state.LastAnchorProviderReceipt = optional.LastAnchorProviderReceiptDigest
	state.LastAnchorTranscriptDigest = optional.LastAnchorTranscriptDigest
}

type externalAnchorPreparedStateRequiredDigests struct {
	TypedRequestHash   trustpolicy.Digest
	ActionRequestHash  trustpolicy.Digest
	PolicyDecisionHash trustpolicy.Digest
}

type externalAnchorPreparedStateOptionalDigests struct {
	RequiredApprovalRequestHash     *trustpolicy.Digest
	RequiredApprovalDecisionHash    *trustpolicy.Digest
	LastExecuteAttemptSealDigest    *trustpolicy.Digest
	LastExecuteAttemptTargetDigest  *trustpolicy.Digest
	LastExecuteAttemptRequestDigest *trustpolicy.Digest
	LastExecuteSnapshotSealDigest   *trustpolicy.Digest
	LastAnchorReceiptDigest         *trustpolicy.Digest
	LastAnchorEvidenceDigest        *trustpolicy.Digest
	LastAnchorVerificationDigest    *trustpolicy.Digest
	LastAnchorProofDigest           *trustpolicy.Digest
	LastAnchorProviderReceiptDigest *trustpolicy.Digest
	LastAnchorTranscriptDigest      *trustpolicy.Digest
}

func externalAnchorPreparedStateDigestsFromRecord(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorPreparedStateRequiredDigests, externalAnchorPreparedStateOptionalDigests, error) {
	required, err := externalAnchorPreparedStateRequiredDigestSet(record)
	if err != nil {
		return externalAnchorPreparedStateRequiredDigests{}, externalAnchorPreparedStateOptionalDigests{}, err
	}
	optional, err := externalAnchorPreparedStateOptionalDigestSet(record)
	if err != nil {
		return externalAnchorPreparedStateRequiredDigests{}, externalAnchorPreparedStateOptionalDigests{}, err
	}
	return required, optional, nil
}

func externalAnchorPreparedStateRequiredDigestSet(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorPreparedStateRequiredDigests, error) {
	typedRequestHash, err := digestFromIdentity(record.TypedRequestHash)
	if err != nil {
		return externalAnchorPreparedStateRequiredDigests{}, fmt.Errorf("typed_request_hash invalid: %w", err)
	}
	actionRequestHash, err := digestFromIdentity(record.ActionRequestHash)
	if err != nil {
		return externalAnchorPreparedStateRequiredDigests{}, fmt.Errorf("action_request_hash invalid: %w", err)
	}
	policyDecisionHash, err := digestFromIdentity(record.PolicyDecisionHash)
	if err != nil {
		return externalAnchorPreparedStateRequiredDigests{}, fmt.Errorf("policy_decision_hash invalid: %w", err)
	}
	return externalAnchorPreparedStateRequiredDigests{TypedRequestHash: typedRequestHash, ActionRequestHash: actionRequestHash, PolicyDecisionHash: policyDecisionHash}, nil
}

func externalAnchorPreparedStateOptionalDigestSet(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorPreparedStateOptionalDigests, error) {
	values := externalAnchorPreparedStateOptionalDigests{}
	for _, item := range []struct {
		identity string
		field    string
		target   **trustpolicy.Digest
	}{{record.RequiredApprovalReqHash, "required_approval_request_hash", &values.RequiredApprovalRequestHash}, {record.RequiredApprovalDecHash, "required_approval_decision_hash", &values.RequiredApprovalDecisionHash}, {record.LastExecuteAttemptSealDigest, "last_execute_attempt_seal_digest", &values.LastExecuteAttemptSealDigest}, {record.LastExecuteAttemptTargetID, "last_execute_attempt_target_descriptor_digest", &values.LastExecuteAttemptTargetDigest}, {record.LastExecuteAttemptRequestID, "last_execute_attempt_typed_request_hash", &values.LastExecuteAttemptRequestDigest}, {record.LastExecuteSnapshotSealID, "last_execute_snapshot_seal_digest", &values.LastExecuteSnapshotSealDigest}, {record.LastAnchorReceiptDigest, "last_anchor_receipt_digest", &values.LastAnchorReceiptDigest}, {record.LastAnchorEvidenceDigest, "last_anchor_evidence_digest", &values.LastAnchorEvidenceDigest}, {record.LastAnchorVerificationDigest, "last_anchor_verification_digest", &values.LastAnchorVerificationDigest}, {record.LastAnchorProofDigest, "last_anchor_proof_digest", &values.LastAnchorProofDigest}, {record.LastAnchorProviderReceipt, "last_anchor_provider_receipt_digest", &values.LastAnchorProviderReceiptDigest}, {record.LastAnchorTranscriptDigest, "last_anchor_transcript_digest", &values.LastAnchorTranscriptDigest}} {
		digest, err := optionalDigestFromIdentity(item.identity, item.field)
		if err != nil {
			return externalAnchorPreparedStateOptionalDigests{}, err
		}
		*item.target = digest
	}
	return values, nil
}

func externalAnchorExecutionPathway() string {
	return "non_workspace_gateway"
}

func externalAnchorPostureFromRecord(record artifacts.ExternalAnchorPreparedMutationRecord) string {
	if strings.TrimSpace(record.LifecycleState) == gitRemoteMutationLifecyclePrepared && strings.TrimSpace(record.ExecutionState) == gitRemoteMutationExecutionNotStarted {
		return "external_configured_not_run"
	}
	switch strings.TrimSpace(record.ExecutionState) {
	case gitRemoteMutationExecutionDeferred:
		return "external_execute_deferred"
	case gitRemoteMutationExecutionCompleted:
		return "external_execute_completed"
	case gitRemoteMutationExecutionBlocked:
		return "external_execute_blocked"
	case gitRemoteMutationExecutionFailed:
		return "external_execute_failed"
	default:
		if strings.TrimSpace(record.LifecycleState) == gitRemoteMutationLifecycleExecuting {
			return "external_execute_in_progress"
		}
		return "external_configured_not_run"
	}
}
