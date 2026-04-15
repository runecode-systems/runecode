package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) evaluateModelGatewayInvoke(requestID, runID string, binding llmExecutionBinding, outcome string) (policyengine.PolicyDecision, *ErrorResponse) {
	action := llmGatewayEgressAction(binding, outcome)
	decision, err := s.EvaluateAction(runID, action)
	if err != nil {
		errOut := s.errorFromPolicyEvaluation(requestID, err)
		return policyengine.PolicyDecision{}, &errOut
	}
	if decision.DecisionOutcome != policyengine.DecisionAllow {
		errOut := s.makeError(requestID, "broker_limit_policy_rejected", "policy", false, fmt.Sprintf("model invoke decision outcome %q (%s)", decision.DecisionOutcome, decision.PolicyReasonCode))
		return policyengine.PolicyDecision{}, &errOut
	}
	return decision, nil
}

func llmGatewayEgressAction(binding llmExecutionBinding, outcome string) policyengine.ActionRequest {
	return policyengine.NewGatewayEgressAction(policyengine.GatewayEgressActionInput{
		ActionEnvelope:  llmGatewayActionEnvelope(binding),
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  "model.example.com/v1/chat/completions",
		EgressDataClass: "spec_text",
		Operation:       "invoke_model",
		TimeoutSeconds:  llmTimeoutSeconds(),
		PayloadHash:     &binding.RequestHash,
		AuditContext:    llmGatewayAuditContext(binding, outcome),
		QuotaContext:    llmGatewayQuotaContext(),
	})
}

func llmGatewayActionEnvelope(binding llmExecutionBinding) policyengine.ActionEnvelope {
	return policyengine.ActionEnvelope{
		CapabilityID:           "cap_gateway",
		RelevantArtifactHashes: []trustpolicy.Digest{binding.RequestHash},
		Actor:                  policyengine.ActionActor{ActorKind: "role_instance", RoleFamily: "gateway", RoleKind: "model-gateway"},
	}
}

func llmGatewayAuditContext(binding llmExecutionBinding, outcome string) *policyengine.GatewayAuditContextInput {
	return &policyengine.GatewayAuditContextInput{
		OutboundBytes: 128,
		StartedAt:     "2026-04-12T10:00:00Z",
		CompletedAt:   "2026-04-12T10:00:01Z",
		Outcome:       outcome,
		RequestHash:   &binding.RequestHash,
		ResponseHash:  &binding.ResponseHash,
		LeaseID:       binding.LeaseID,
	}
}

func llmGatewayQuotaContext() *policyengine.GatewayQuotaContextInput {
	return &policyengine.GatewayQuotaContextInput{
		QuotaProfileKind:    "hybrid",
		Phase:               "admission",
		EnforceDuringStream: false,
		Meters: policyengine.GatewayQuotaMetersInput{
			RequestUnits:     llmMeterInt64(1),
			InputTokens:      llmMeterInt64(256),
			OutputTokens:     llmMeterInt64(64),
			ConcurrencyUnits: llmMeterInt64(1),
			SpendMicros:      llmMeterInt64(1000),
			EntitlementUnits: llmMeterInt64(1),
		},
	}
}

func llmTimeoutSeconds() *int {
	timeoutSeconds := 30
	return &timeoutSeconds
}

func llmMeterInt64(value int64) *int64 {
	return &value
}

func (s *Service) emitModelGatewayAudit(runID string, decision policyengine.PolicyDecision, outcome string, binding llmExecutionBinding) error {
	policyHash := decisionDigestIdentity(decision)
	return s.gatewayRuntime.emitGatewayAuditEvent(runID, decision, gatewayActionPayloadRuntime{
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  "model.example.com/v1/chat/completions",
		Operation:       "invoke_model",
		PayloadHash:     &binding.RequestHash,
		AuditContext: &gatewayAuditContextPayload{
			Outcome:            outcome,
			RequestHash:        &binding.RequestHash,
			ResponseHash:       &binding.ResponseHash,
			LeaseID:            binding.LeaseID,
			PolicyDecisionHash: digestFromIdentityOrNil(policyHash),
		},
	})
}
