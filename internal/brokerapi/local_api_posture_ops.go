package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) HandleBackendPostureGet(ctx context.Context, req BackendPostureGetRequest, meta RequestContext) (BackendPostureGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, backendPostureGetRequestSchemaPath)
	if errResp != nil {
		return BackendPostureGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return BackendPostureGetResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return BackendPostureGetResponse{}, &errOut
	}
	posture := s.currentBackendPostureState()
	resp := BackendPostureGetResponse{
		SchemaID:      "runecode.protocol.v0.BackendPostureGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Posture:       posture,
	}
	if err := s.validateResponse(resp, backendPostureGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return BackendPostureGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleBackendPostureChange(ctx context.Context, req BackendPostureChangeRequest, meta RequestContext) (BackendPostureChangeResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.prepareBackendPostureChangeRequest(ctx, req, meta)
	if errResp != nil {
		return BackendPostureChangeResponse{}, errResp
	}
	defer cleanup()
	posture, out, errResp := s.evaluateBackendPostureChange(requestID, req)
	if errResp != nil {
		return BackendPostureChangeResponse{}, errResp
	}
	resp := BackendPostureChangeResponse{
		SchemaID:      "runecode.protocol.v0.BackendPostureChangeResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Outcome:       out,
		Posture:       posture,
	}
	if err := s.validateResponse(resp, backendPostureChangeResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return BackendPostureChangeResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) prepareBackendPostureChangeRequest(ctx context.Context, req BackendPostureChangeRequest, meta RequestContext) (string, func(), *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, backendPostureChangeRequestSchemaPath)
	if errResp != nil {
		return "", nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	cleanup := func() {
		cancel()
		release()
	}
	if err := requestCtx.Err(); err != nil {
		cleanup()
		errOut := s.errorFromContext(requestID, err)
		return "", nil, &errOut
	}
	return requestID, cleanup, nil
}

func (s *Service) evaluateBackendPostureChange(requestID string, req BackendPostureChangeRequest) (BackendPostureState, BackendPostureChangeOutcome, *ErrorResponse) {
	action := newBackendPostureChangeAction(req)
	decision, err := s.EvaluateInstanceControlAction(action)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return BackendPostureState{}, BackendPostureChangeOutcome{}, &errOut
	}
	return s.backendPostureChangeOutcomeFromDecision(requestID, req, action, decision)
}

func newBackendPostureChangeAction(req BackendPostureChangeRequest) policyengine.ActionRequest {
	return policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: "cap_backend",
			Actor:        policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"},
		},
		RunID:                        backendPostureActionRunID(req),
		TargetInstanceID:             strings.TrimSpace(req.TargetInstanceID),
		TargetBackendKind:            strings.TrimSpace(req.TargetBackendKind),
		SelectionMode:                strings.TrimSpace(req.SelectionMode),
		ChangeKind:                   strings.TrimSpace(req.ChangeKind),
		AssuranceChangeKind:          strings.TrimSpace(req.AssuranceChangeKind),
		OptInKind:                    strings.TrimSpace(req.OptInKind),
		ReducedAssuranceAcknowledged: req.ReducedAssuranceAcknowledged,
		Reason:                       strings.TrimSpace(req.Reason),
	})
}

func backendPostureActionRunID(req BackendPostureChangeRequest) string {
	target := strings.TrimSpace(req.TargetInstanceID)
	if target == "" {
		return "instance-control:active"
	}
	return "instance-control:" + target
}

func (s *Service) backendPostureChangeOutcomeFromDecision(requestID string, req BackendPostureChangeRequest, action policyengine.ActionRequest, decision policyengine.PolicyDecision) (BackendPostureState, BackendPostureChangeOutcome, *ErrorResponse) {
	out := BackendPostureChangeOutcome{
		SchemaID:           "runecode.protocol.v0.BackendPostureChangeOutcome",
		SchemaVersion:      "0.1.0",
		PolicyDecisionHash: decisionDigestIdentity(decision),
		ActionRequestHash:  strings.TrimSpace(decision.ActionRequestHash),
	}
	switch decision.DecisionOutcome {
	case policyengine.DecisionAllow:
		if err := s.applyInstanceBackendPosture(context.Background(), strings.TrimSpace(req.TargetInstanceID), strings.TrimSpace(req.TargetBackendKind)); err != nil {
			errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
			return BackendPostureState{}, BackendPostureChangeOutcome{}, &errOut
		}
		out.Outcome = "applied"
		out.OutcomeReasonCode = "policy_allow"
		return s.currentBackendPostureState(), out, nil
	case policyengine.DecisionDeny:
		out.Outcome = "rejected"
		out.OutcomeReasonCode = strings.TrimSpace(decision.PolicyReasonCode)
		if out.OutcomeReasonCode == "" {
			out.OutcomeReasonCode = "policy_denied"
		}
		return s.currentBackendPostureState(), out, nil
	case policyengine.DecisionRequireHumanApproval:
		approvalID, err := s.recordPendingBackendPostureApproval(decision, req, action)
		if err != nil {
			errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
			return BackendPostureState{}, BackendPostureChangeOutcome{}, &errOut
		}
		out.Outcome = "approval_required"
		out.OutcomeReasonCode = "approval_required"
		out.ApprovalID = approvalID
		return s.currentBackendPostureState(), out, nil
	default:
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "unsupported backend posture decision outcome")
		return BackendPostureState{}, BackendPostureChangeOutcome{}, &errOut
	}
}
