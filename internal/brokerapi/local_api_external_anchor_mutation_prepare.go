package brokerapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type externalAnchorPrepareResolvedInput struct {
	runID                       string
	requestKind                 string
	targetKind                  string
	sealDigest                  trustpolicy.Digest
	sealDigestIdentity          string
	typedRequestHash            trustpolicy.Digest
	typedRequestHashIdentity    string
	targetDescriptorDigest      trustpolicy.Digest
	targetDescriptorDigestIdent string
	outboundPayloadDigest       trustpolicy.Digest
	outboundPayloadDigestIdent  string
	destinationRef              string
}

func (s *Service) resolveExternalAnchorPrepareInput(req ExternalAnchorMutationPrepareRequest, requestID string) (externalAnchorPrepareResolvedInput, *ErrorResponse) {
	runID, requestKind, targetKind, errResp := s.resolveExternalAnchorPrepareIdentity(requestID, req)
	if errResp != nil {
		return externalAnchorPrepareResolvedInput{}, errResp
	}
	sealDigest, sealDigestIdentity, typedRequestHash, typedRequestHashIdentity, targetDescriptorDigest, targetDescriptorDigestIdentity, outboundPayloadDigest, outboundPayloadDigestIdentity, errResp := s.resolveExternalAnchorPrepareDigests(requestID, req.TypedRequest)
	if errResp != nil {
		return externalAnchorPrepareResolvedInput{}, errResp
	}
	destinationRef := externalAnchorDestinationRefFromTargetDescriptorDigest(targetDescriptorDigestIdentity)
	if destinationRef == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "typed_request.target_descriptor_digest is required")
		return externalAnchorPrepareResolvedInput{}, &errOut
	}
	return externalAnchorPrepareResolvedInput{
		runID:                       runID,
		requestKind:                 requestKind,
		targetKind:                  targetKind,
		sealDigest:                  sealDigest,
		sealDigestIdentity:          sealDigestIdentity,
		typedRequestHash:            typedRequestHash,
		typedRequestHashIdentity:    typedRequestHashIdentity,
		targetDescriptorDigest:      targetDescriptorDigest,
		targetDescriptorDigestIdent: targetDescriptorDigestIdentity,
		outboundPayloadDigest:       outboundPayloadDigest,
		outboundPayloadDigestIdent:  outboundPayloadDigestIdentity,
		destinationRef:              destinationRef,
	}, nil
}

func (s *Service) resolveExternalAnchorPrepareIdentity(requestID string, req ExternalAnchorMutationPrepareRequest) (string, string, string, *ErrorResponse) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
		return "", "", "", &errOut
	}
	requestKind := resolveExternalAnchorRequestKind(req.TypedRequest)
	if err := validateExternalAnchorRequestKind(requestKind); err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return "", "", "", &errOut
	}
	targetKind := externalAnchorTargetKind(req.TypedRequest)
	if targetKind == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "typed_request.target_kind is required")
		return "", "", "", &errOut
	}
	return runID, requestKind, targetKind, nil
}

func (s *Service) resolveExternalAnchorPrepareDigests(requestID string, typedRequest map[string]any) (trustpolicy.Digest, string, trustpolicy.Digest, string, trustpolicy.Digest, string, trustpolicy.Digest, string, *ErrorResponse) {
	sealDigest, sealDigestIdentity, err := externalAnchorSealDigest(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", s.externalAnchorPrepareValidationError(requestID, err)
	}
	typedRequestHash, typedRequestHashIdentity, err := canonicalizeExternalAnchorTypedRequest(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", s.externalAnchorPrepareValidationError(requestID, err)
	}
	targetDescriptorDigest, targetDescriptorDigestIdentity, err := externalAnchorCanonicalTargetDigest(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", s.externalAnchorPrepareValidationError(requestID, err)
	}
	outboundPayloadDigest, outboundPayloadDigestIdentity, err := externalAnchorPayloadDigest(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", trustpolicy.Digest{}, "", s.externalAnchorPrepareValidationError(requestID, err)
	}
	return sealDigest, sealDigestIdentity, typedRequestHash, typedRequestHashIdentity, targetDescriptorDigest, targetDescriptorDigestIdentity, outboundPayloadDigest, outboundPayloadDigestIdentity, nil
}

func (s *Service) externalAnchorPrepareValidationError(requestID string, err error) *ErrorResponse {
	errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
	return &errOut
}

func (s *Service) evaluatePreparedExternalAnchorMutation(_ context.Context, requestID string, resolved externalAnchorPrepareResolvedInput, typedRequest map[string]any) (policyengine.PolicyDecision, gitPreparedApprovalBinding, string, *ErrorResponse) {
	action := externalAnchorGatewayPrepareAction(resolved, typedRequest)
	decision, evalErr := s.EvaluateAction(resolved.runID, action)
	if evalErr != nil {
		errOut := s.errorFromPolicyEvaluation(requestID, evalErr)
		return policyengine.PolicyDecision{}, gitPreparedApprovalBinding{}, "", &errOut
	}
	if decision.DecisionOutcome != policyengine.DecisionRequireHumanApproval {
		errOut := s.makeError(requestID, "broker_limit_policy_rejected", "policy", false, fmt.Sprintf("external anchor prepare requires exact approval; decision outcome %q", decision.DecisionOutcome))
		return policyengine.PolicyDecision{}, gitPreparedApprovalBinding{}, "", &errOut
	}
	if trigger, _ := decision.RequiredApproval["approval_trigger_code"].(string); strings.TrimSpace(trigger) != "external_anchor_opt_in" {
		errOut := s.makeError(requestID, "broker_limit_policy_rejected", "policy", false, "external anchor prepare requires explicit signed-manifest opt-in approval trigger")
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

func (s *Service) buildPreparedExternalAnchorMutationRecord(req ExternalAnchorMutationPrepareRequest, requestID string, resolved externalAnchorPrepareResolvedInput, decision policyengine.PolicyDecision, approvalBinding gitPreparedApprovalBinding, policyDecisionHash string) (artifacts.ExternalAnchorPreparedMutationRecord, *ErrorResponse) {
	preparedMutationID, err := externalAnchorPreparedMutationID(
		resolved.runID,
		resolved.destinationRef,
		resolved.typedRequestHashIdentity,
		decision.ActionRequestHash,
		policyDecisionHash,
	)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("prepared mutation id generation failed: %v", err))
		return artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
	}
	return artifacts.ExternalAnchorPreparedMutationRecord{
		PreparedMutationID:      preparedMutationID,
		RunID:                   resolved.runID,
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
		LastPrepareRequestID:    requestID,
	}, nil
}

func externalAnchorGatewayPrepareAction(resolved externalAnchorPrepareResolvedInput, typedRequest map[string]any) policyengine.ActionRequest {
	now := time.Now().UTC()
	auditContext := map[string]any{
		"schema_id":      "runecode.protocol.v0.GatewayAuditContext",
		"schema_version": "0.1.0",
		"outbound_bytes": 1,
		"started_at":     now.Format(time.RFC3339),
		"completed_at":   now.Add(time.Second).Format(time.RFC3339),
		"outcome":        "admission_allowed",
		"request_hash":   resolved.typedRequestHash,
	}
	actionPayload := map[string]any{
		"schema_id":               "runecode.protocol.v0.ActionPayloadGatewayEgress",
		"schema_version":          "0.1.0",
		"gateway_role_kind":       "git-gateway",
		"destination_kind":        "git_remote",
		"destination_ref":         resolved.destinationRef,
		"egress_data_class":       "audit_events",
		"operation":               "external_anchor_submit",
		"payload_hash":            resolved.typedRequestHash,
		"audit_context":           auditContext,
		"external_anchor_request": cloneStringAnyMap(typedRequest),
	}
	relevantHashes := uniqueDigestIdentities([]trustpolicy.Digest{resolved.typedRequestHash, resolved.targetDescriptorDigest, resolved.sealDigest, resolved.outboundPayloadDigest})
	return policyengine.ActionRequest{
		SchemaID:               "runecode.protocol.v0.ActionRequest",
		SchemaVersion:          "0.1.0",
		ActionKind:             policyengine.ActionKindGatewayEgress,
		CapabilityID:           "cap_external_anchor",
		RelevantArtifactHashes: relevantHashes,
		ActionPayloadSchemaID:  "runecode.protocol.v0.ActionPayloadGatewayEgress",
		ActionPayload:          actionPayload,
		ActorKind:              "role_instance",
		RoleFamily:             "gateway",
		RoleKind:               "git-gateway",
		AllowlistRefs:          []string{},
	}
}

func uniqueDigestIdentities(values []trustpolicy.Digest) []trustpolicy.Digest {
	set := map[string]trustpolicy.Digest{}
	for _, value := range values {
		identity, err := value.Identity()
		if err != nil {
			continue
		}
		set[identity] = value
	}
	if len(set) == 0 {
		return nil
	}
	identities := make([]string, 0, len(set))
	for identity := range set {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	out := make([]trustpolicy.Digest, 0, len(identities))
	for _, identity := range identities {
		out = append(out, set[identity])
	}
	return out
}
