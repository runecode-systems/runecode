package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

type gitRemoteExecutionAttemptBinding struct {
	AttemptID        string
	TypedRequestHash string
}

type gitRemoteExecutionSnapshot struct {
	SegmentID    string
	SealIdentity string
}

func (s *Service) gitRemoteExecuteAttemptBinding(requestID string, record artifacts.GitRemotePreparedMutationRecord) (gitRemoteExecutionAttemptBinding, *ErrorResponse) {
	typedRequestHash, err := canonicalGitTypedRequestHash(record.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("stored typed request hash invalid: %v", err))
		return gitRemoteExecutionAttemptBinding{}, &errOut
	}
	attemptID, err := gitRemoteExecuteAttemptID(record.PreparedMutationID, typedRequestHash)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("attempt id generation failed: %v", err))
		return gitRemoteExecutionAttemptBinding{}, &errOut
	}
	return gitRemoteExecutionAttemptBinding{AttemptID: attemptID, TypedRequestHash: typedRequestHash}, nil
}

func (s *Service) snapshotGitRemoteExecutionInputs(requestID string) (gitRemoteExecutionSnapshot, *ErrorResponse) {
	if s == nil || s.auditLedger == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
		return gitRemoteExecutionSnapshot{}, &errOut
	}
	segmentID, digest, err := s.auditLedger.LatestAnchorableSeal()
	if err != nil {
		if err == auditd.ErrNoSealedSegment {
			return gitRemoteExecutionSnapshot{SegmentID: "", SealIdentity: ""}, nil
		}
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("latest anchorable seal unavailable: %v", err))
		return gitRemoteExecutionSnapshot{}, &errOut
	}
	identity, err := digest.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "latest anchorable seal digest is invalid")
		return gitRemoteExecutionSnapshot{}, &errOut
	}
	return gitRemoteExecutionSnapshot{SegmentID: segmentID, SealIdentity: identity}, nil
}

func gitRemoteExecuteAttemptID(preparedMutationID, typedRequestHash string) (string, error) {
	payload := map[string]any{
		"prepared_mutation_id":                strings.TrimSpace(preparedMutationID),
		"typed_request_hash":                  strings.TrimSpace(typedRequestHash),
		"attempt_identity_schema_version":     "0.1.0",
		"attempt_identity_source":             "git_remote_execute",
		"attempt_identity_binding_invariants": []string{"typed_request_hash"},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}
