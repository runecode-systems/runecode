package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type gitPreparedApprovalBinding struct {
	ApprovalID     string
	RequestDigest  string
	DecisionDigest string
}

type gitRemotePrepareResolvedInput struct {
	runID                    string
	provider                 string
	requestKind              string
	destinationRef           string
	typedRequestHash         trustpolicy.Digest
	typedRequestHashIdentity string
	proof                    gitRuntimeProofPayload
}

func (s *Service) pendingApprovalBindingForDecision(runID, actionRequestHash, policyDecisionHash string) (gitPreparedApprovalBinding, bool) {
	for _, rec := range s.ApprovalList() {
		if !matchesPendingApprovalRun(rec, runID) {
			continue
		}
		if !matchesPendingApprovalHashes(rec, actionRequestHash, policyDecisionHash) {
			continue
		}
		if !isSHA256Digest(strings.TrimSpace(rec.RequestDigest)) {
			continue
		}
		return gitPreparedApprovalBinding{
			ApprovalID:     strings.TrimSpace(rec.ApprovalID),
			RequestDigest:  strings.TrimSpace(rec.RequestDigest),
			DecisionDigest: strings.TrimSpace(rec.DecisionDigest),
		}, true
	}
	return gitPreparedApprovalBinding{}, false
}

func matchesPendingApprovalRun(rec artifacts.ApprovalRecord, runID string) bool {
	return strings.TrimSpace(rec.RunID) == strings.TrimSpace(runID) && strings.TrimSpace(rec.Status) == "pending"
}

func matchesPendingApprovalHashes(rec artifacts.ApprovalRecord, actionRequestHash, policyDecisionHash string) bool {
	return strings.TrimSpace(rec.ActionRequestHash) == strings.TrimSpace(actionRequestHash) &&
		strings.TrimSpace(rec.PolicyDecisionHash) == strings.TrimSpace(policyDecisionHash)
}

func (s *Service) resolveGitRemotePrepareInput(req GitRemoteMutationPrepareRequest, requestID string) (gitRemotePrepareResolvedInput, *ErrorResponse) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
		return gitRemotePrepareResolvedInput{}, &errOut
	}
	provider := normalizeGitProvider(req.Provider)
	requestKind := resolveGitRemoteRequestKind(req.TypedRequest)
	if err := validateGitRemoteRequestKind(requestKind); err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return gitRemotePrepareResolvedInput{}, &errOut
	}
	destinationRef := resolveGitRemoteDestinationRef(req)
	if destinationRef == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "destination_ref is required")
		return gitRemotePrepareResolvedInput{}, &errOut
	}
	typedRequestHash, typedRequestHashIdentity, err := canonicalizeGitRemoteTypedRequest(req.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return gitRemotePrepareResolvedInput{}, &errOut
	}
	proof, err := gitRuntimeProofForPrepare(req.TypedRequest, typedRequestHash, provider)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return gitRemotePrepareResolvedInput{}, &errOut
	}
	return gitRemotePrepareResolvedInput{
		runID:                    runID,
		provider:                 provider,
		requestKind:              requestKind,
		destinationRef:           destinationRef,
		typedRequestHash:         typedRequestHash,
		typedRequestHashIdentity: typedRequestHashIdentity,
		proof:                    proof,
	}, nil
}

func (s *Service) evaluatePreparedGitRemoteMutation(_ context.Context, requestID string, resolved gitRemotePrepareResolvedInput, typedRequest map[string]any) (policyengine.PolicyDecision, gitPreparedApprovalBinding, string, *ErrorResponse) {
	action := gitGatewayPrepareAction(resolved.destinationRef, resolved.requestKind, typedRequest, resolved.typedRequestHash, resolved.proof)
	decision, evalErr := s.EvaluateAction(resolved.runID, action)
	if evalErr != nil {
		errOut := s.errorFromPolicyEvaluation(requestID, evalErr)
		return policyengine.PolicyDecision{}, gitPreparedApprovalBinding{}, "", &errOut
	}
	if decision.DecisionOutcome != policyengine.DecisionRequireHumanApproval {
		errOut := s.makeError(requestID, "broker_limit_policy_rejected", "policy", false, fmt.Sprintf("git remote prepare requires exact approval; decision outcome %q", decision.DecisionOutcome))
		return policyengine.PolicyDecision{}, gitPreparedApprovalBinding{}, "", &errOut
	}
	policyDecisionHash := decisionDigestIdentity(decision)
	if !isSHA256Digest(policyDecisionHash) {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "policy decision hash unavailable")
		return policyengine.PolicyDecision{}, gitPreparedApprovalBinding{}, "", &errOut
	}
	approvalBinding, ok := s.pendingApprovalBindingForDecision(resolved.runID, decision.ActionRequestHash, policyDecisionHash)
	if !ok {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "policy required approval but no pending approval binding was recorded")
		return policyengine.PolicyDecision{}, gitPreparedApprovalBinding{}, "", &errOut
	}
	return decision, approvalBinding, policyDecisionHash, nil
}

func (s *Service) buildPreparedGitRemoteMutationRecord(req GitRemoteMutationPrepareRequest, requestID string, resolved gitRemotePrepareResolvedInput, decision policyengine.PolicyDecision, approvalBinding gitPreparedApprovalBinding, policyDecisionHash string) (artifacts.GitRemotePreparedMutationRecord, *ErrorResponse) {
	derivedSummary, err := deriveGitRemoteMutationSummary(req.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return artifacts.GitRemotePreparedMutationRecord{}, &errOut
	}
	summaryMap, err := mapFromValue(derivedSummary)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("derived_summary encode failed: %v", err))
		return artifacts.GitRemotePreparedMutationRecord{}, &errOut
	}
	preparedMutationID, err := gitPreparedMutationID(resolved.runID, resolved.provider, resolved.destinationRef, resolved.typedRequestHashIdentity, decision.ActionRequestHash, policyDecisionHash)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("prepared mutation id generation failed: %v", err))
		return artifacts.GitRemotePreparedMutationRecord{}, &errOut
	}
	return artifacts.GitRemotePreparedMutationRecord{
		PreparedMutationID:      preparedMutationID,
		RunID:                   resolved.runID,
		Provider:                resolved.provider,
		DestinationRef:          resolved.destinationRef,
		RequestKind:             resolved.requestKind,
		TypedRequestSchemaID:    strings.TrimSpace(stringField(req.TypedRequest, "schema_id")),
		TypedRequestSchemaVer:   strings.TrimSpace(stringField(req.TypedRequest, "schema_version")),
		TypedRequest:            cloneStringAnyMap(req.TypedRequest),
		TypedRequestHash:        resolved.typedRequestHashIdentity,
		ActionRequestHash:       strings.TrimSpace(decision.ActionRequestHash),
		PolicyDecisionHash:      policyDecisionHash,
		RequiredApprovalID:      approvalBinding.ApprovalID,
		RequiredApprovalReqHash: approvalBinding.RequestDigest,
		LifecycleState:          gitRemoteMutationLifecyclePrepared,
		ExecutionState:          gitRemoteMutationExecutionNotStarted,
		DerivedSummary:          summaryMap,
		LastPrepareRequestID:    requestID,
	}, nil
}

func gitRuntimeProofForPrepare(typedRequest map[string]any, typedRequestHash trustpolicy.Digest, provider string) (gitRuntimeProofPayload, error) {
	summary, requestKind, err := gitRemoteMutationBaseSummary(typedRequest)
	if err != nil {
		return gitRuntimeProofPayload{}, err
	}
	proof := gitRuntimeProofBase(typedRequestHash, provider, summary.ReferencedPatchArtifactHashes, summary.ExpectedResultTreeHash)
	switch requestKind {
	case "git_ref_update":
		return gitRuntimeProofForRefUpdate(typedRequest, proof)
	case "git_pull_request_create":
		return gitRuntimeProofForPullRequest(proof), nil
	default:
		return gitRuntimeProofPayload{}, fmt.Errorf("typed_request.request_kind is required")
	}
}

func deriveGitRemoteMutationSummary(typedRequest map[string]any) (GitRemoteMutationDerivedSummary, error) {
	summary, requestKind, err := gitRemoteMutationBaseSummary(typedRequest)
	if err != nil {
		return GitRemoteMutationDerivedSummary{}, err
	}
	switch requestKind {
	case "git_ref_update":
		summary, err = gitRemoteSummaryForRefUpdate(typedRequest, summary)
	case "git_pull_request_create":
		summary, err = gitRemoteSummaryForPullRequest(typedRequest, summary)
	default:
		err = fmt.Errorf("typed_request.request_kind must be git_ref_update or git_pull_request_create")
	}
	if err != nil {
		return GitRemoteMutationDerivedSummary{}, err
	}
	if err := requireGitRemoteSummaryFields(summary); err != nil {
		return GitRemoteMutationDerivedSummary{}, err
	}
	return summary, nil
}

func gitGatewayPrepareAction(destinationRef, requestKind string, typedRequest map[string]any, typedRequestHash trustpolicy.Digest, proof gitRuntimeProofPayload) policyengine.ActionRequest {
	now := time.Now().UTC()
	auditContext := map[string]any{
		"schema_id":      "runecode.protocol.v0.GatewayAuditContext",
		"schema_version": "0.1.0",
		"outbound_bytes": 1,
		"started_at":     now.Format(time.RFC3339),
		"completed_at":   now.Add(time.Second).Format(time.RFC3339),
		"outcome":        "admission_allowed",
		"request_hash":   typedRequestHash,
	}
	actionPayload := map[string]any{
		"schema_id":         "runecode.protocol.v0.ActionPayloadGatewayEgress",
		"schema_version":    "0.1.0",
		"gateway_role_kind": "git-gateway",
		"destination_kind":  "git_remote",
		"destination_ref":   destinationRef,
		"egress_data_class": "diffs",
		"operation":         requestKind,
		"payload_hash":      typedRequestHash,
		"audit_context":     auditContext,
		"git_request":       cloneStringAnyMap(typedRequest),
		"git_runtime_proof": proof,
	}
	return policyengine.ActionRequest{
		SchemaID:               "runecode.protocol.v0.ActionRequest",
		SchemaVersion:          "0.1.0",
		ActionKind:             policyengine.ActionKindGatewayEgress,
		CapabilityID:           "cap_gateway",
		RelevantArtifactHashes: []trustpolicy.Digest{typedRequestHash},
		ActionPayloadSchemaID:  "runecode.protocol.v0.ActionPayloadGatewayEgress",
		ActionPayload:          actionPayload,
		ActorKind:              "role_instance",
		RoleFamily:             "gateway",
		RoleKind:               "git-gateway",
		AllowlistRefs:          []string{},
	}
}
