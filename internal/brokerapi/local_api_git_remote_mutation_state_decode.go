package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type gitPreparedStateOptionalDigests struct {
	RequiredApprovalRequestHash   *trustpolicy.Digest
	RequiredApprovalDecisionHash  *trustpolicy.Digest
	LastExecuteAttemptRequestHash *trustpolicy.Digest
	LastExecuteSnapshotSealDigest *trustpolicy.Digest
}

func decodeGitPreparedStateRecord(record artifacts.GitRemotePreparedMutationRecord) (trustpolicy.Digest, trustpolicy.Digest, trustpolicy.Digest, gitPreparedStateOptionalDigests, GitRemoteMutationDerivedSummary, error) {
	typedRequestHash, actionRequestHash, policyDecisionHash, err := decodeGitPreparedStateRequiredDigests(record)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, gitPreparedStateOptionalDigests{}, GitRemoteMutationDerivedSummary{}, err
	}
	optional, err := decodeGitPreparedStateOptionalDigests(record)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, gitPreparedStateOptionalDigests{}, GitRemoteMutationDerivedSummary{}, err
	}
	derivedSummary := GitRemoteMutationDerivedSummary{}
	if err := remarshalValue(record.DerivedSummary, &derivedSummary); err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, gitPreparedStateOptionalDigests{}, GitRemoteMutationDerivedSummary{}, fmt.Errorf("derived_summary invalid: %w", err)
	}
	if derivedSummary.SchemaID == "" {
		derivedSummary.SchemaID = "runecode.protocol.v0.GitRemoteMutationDerivedSummary"
	}
	if derivedSummary.SchemaVersion == "" {
		derivedSummary.SchemaVersion = "0.1.0"
	}
	return typedRequestHash, actionRequestHash, policyDecisionHash, optional, derivedSummary, nil
}

func decodeGitPreparedStateRequiredDigests(record artifacts.GitRemotePreparedMutationRecord) (trustpolicy.Digest, trustpolicy.Digest, trustpolicy.Digest, error) {
	typedRequestHash, err := digestFromIdentity(record.TypedRequestHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, fmt.Errorf("typed_request_hash invalid: %w", err)
	}
	actionRequestHash, err := digestFromIdentity(record.ActionRequestHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, fmt.Errorf("action_request_hash invalid: %w", err)
	}
	policyDecisionHash, err := digestFromIdentity(record.PolicyDecisionHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, fmt.Errorf("policy_decision_hash invalid: %w", err)
	}
	return typedRequestHash, actionRequestHash, policyDecisionHash, nil
}

func decodeGitPreparedStateOptionalDigests(record artifacts.GitRemotePreparedMutationRecord) (gitPreparedStateOptionalDigests, error) {
	optional := gitPreparedStateOptionalDigests{}
	var err error
	optional.RequiredApprovalRequestHash, err = optionalDigestFromIdentity(record.RequiredApprovalReqHash, "required_approval_request_hash")
	if err != nil {
		return gitPreparedStateOptionalDigests{}, err
	}
	optional.RequiredApprovalDecisionHash, err = optionalDigestFromIdentity(record.RequiredApprovalDecHash, "required_approval_decision_hash")
	if err != nil {
		return gitPreparedStateOptionalDigests{}, err
	}
	optional.LastExecuteAttemptRequestHash, err = optionalDigestFromIdentity(record.LastExecuteAttemptReqID, "last_execute_attempt_typed_request_hash")
	if err != nil {
		return gitPreparedStateOptionalDigests{}, err
	}
	optional.LastExecuteSnapshotSealDigest, err = optionalDigestFromIdentity(record.LastExecuteSnapshotSeal, "last_execute_snapshot_seal_digest")
	if err != nil {
		return gitPreparedStateOptionalDigests{}, err
	}
	return optional, nil
}
