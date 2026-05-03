package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const llmOutcomeAdmitted = "admission_allowed"

func (s *Service) evaluateModelGatewayInvokeAdmission(requestID, runID string, binding llmExecutionBinding) (policyengine.PolicyDecision, *ErrorResponse) {
	if err := validateLLMExecutionBindingForPolicy(binding); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return policyengine.PolicyDecision{}, &errOut
	}
	action := llmGatewayEgressAction(binding, llmOutcomeAdmitted)
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
		DestinationRef:  binding.DestinationRef,
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
	startedAt := ""
	completedAt := ""
	if !binding.StartedAt.IsZero() {
		startedAt = binding.StartedAt.UTC().Format(time.RFC3339)
	}
	if !binding.CompletedAt.IsZero() {
		completedAt = binding.CompletedAt.UTC().Format(time.RFC3339)
	}
	return &policyengine.GatewayAuditContextInput{
		OutboundBytes: binding.OutboundBytes,
		StartedAt:     startedAt,
		CompletedAt:   completedAt,
		Outcome:       outcome,
		RequestHash:   &binding.RequestHash,
		ResponseHash:  optionalDigestPointer(binding.ResponseHash),
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

func llmGatewayRuntimePayload(binding llmExecutionBinding, outcome string) gatewayActionPayloadRuntime {
	return gatewayActionPayloadRuntime{
		GatewayRoleKind:   "model-gateway",
		DestinationKind:   "model_endpoint",
		DestinationRef:    binding.DestinationRef,
		ProviderProfileID: binding.ProviderID,
		ModelID:           binding.ModelID,
		EndpointIdentity:  binding.DestinationRef,
		EgressDataClass:   "spec_text",
		Operation:         "invoke_model",
		TimeoutSeconds:    llmTimeoutSeconds(),
		PayloadHash:       &binding.RequestHash,
		AuditContext: &gatewayAuditContextPayload{
			OutboundBytes: binding.OutboundBytes,
			StartedAt:     binding.StartedAt.UTC().Format(time.RFC3339),
			CompletedAt:   binding.CompletedAt.UTC().Format(time.RFC3339),
			Outcome:       outcome,
			RequestHash:   &binding.RequestHash,
			ResponseHash:  optionalDigestPointer(binding.ResponseHash),
			LeaseID:       binding.LeaseID,
		},
		QuotaContext: &gatewayQuotaContextPayload{
			QuotaProfileKind:    "hybrid",
			Phase:               "admission",
			EnforceDuringStream: false,
			Meters: gatewayQuotaMetersPayload{
				RequestUnits:     llmMeterInt64(1),
				InputTokens:      llmMeterInt64(256),
				OutputTokens:     llmMeterInt64(64),
				ConcurrencyUnits: llmMeterInt64(1),
				SpendMicros:      llmMeterInt64(1000),
				EntitlementUnits: llmMeterInt64(1),
			},
		},
	}
}

func (s *Service) emitModelGatewayTerminalAudit(runID string, decision policyengine.PolicyDecision, outcome string, binding llmExecutionBinding) error {
	if err := validateLLMExecutionBindingForAudit(binding); err != nil {
		return fmt.Errorf("llm execution metadata unavailable: %w", err)
	}
	runtime := policyRuntime{service: s}
	compileInput, err := runtime.loadCompileInput(strings.TrimSpace(runID))
	if err != nil {
		return fmt.Errorf("llm execution metadata unavailable: %w", err)
	}
	payload := llmGatewayRuntimePayload(binding, outcome)
	_, match, found, reason := findAllowlistEntryForGatewayPayload(compileInput.Allowlists, payload)
	if !found {
		if reason == "" {
			reason = "runtime_gateway_destination_not_allowlisted"
		}
		return fmt.Errorf("llm execution metadata unavailable: %s", reason)
	}
	policyHash := decisionDigestIdentity(decision)
	policyDecisionHash, err := digestFromIdentityOrNil(policyHash)
	if err != nil {
		s.gatewayRuntime.releaseQuotaUsage(runID, payload)
		return fmt.Errorf("llm execution metadata unavailable: invalid policy decision hash: %w", err)
	}
	payload.AuditContext.PolicyDecisionHash = policyDecisionHash
	if err := s.gatewayRuntime.emitGatewayAuditEvent(runID, decision, payload, match); err != nil {
		s.gatewayRuntime.releaseQuotaUsage(runID, payload)
		return err
	}
	s.persistProviderInvocationReceipt(runID, string(decision.DecisionOutcome), "", payload, match)
	s.gatewayRuntime.releaseQuotaUsage(runID, payload)
	return nil
}

func findAllowlistEntryForGatewayPayload(allowlists []policyengine.ManifestInput, payload gatewayActionPayloadRuntime) (policyengine.GatewayScopeRule, gatewayAllowlistMatch, bool, string) {
	for _, allowlistInput := range allowlists {
		expectedHash, err := validateTrustedAllowlistInputHash(allowlistInput)
		if err != nil {
			continue
		}
		decoded := policyengine.PolicyAllowlist{}
		if err := json.Unmarshal(allowlistInput.Payload, &decoded); err != nil {
			continue
		}
		for _, entry := range decoded.Entries {
			if gatewayAllowlistEntryMatchesRuntimePayload(entry, payload) {
				return entry, gatewayAllowlistMatch{AllowlistRef: expectedHash, EntryID: entry.EntryID}, true, ""
			}
		}
	}
	return policyengine.GatewayScopeRule{}, gatewayAllowlistMatch{}, false, "runtime_gateway_destination_not_allowlisted"
}

func optionalDigestPointer(d trustpolicy.Digest) *trustpolicy.Digest {
	if _, err := d.Identity(); err != nil {
		return nil
	}
	return &d
}

func validateLLMExecutionBindingForPolicy(binding llmExecutionBinding) error {
	if err := validateLLMExecutionBindingCore(binding); err != nil {
		return err
	}
	if strings.TrimSpace(binding.LeaseID) == "" {
		return fmt.Errorf("lease_id missing")
	}
	if binding.LeaseID == llmLeaseIDUnavailableSentinel {
		return fmt.Errorf("lease_id unavailable")
	}
	return nil
}

func validateLLMExecutionBindingForAudit(binding llmExecutionBinding) error {
	if err := validateLLMExecutionBindingForPolicy(binding); err != nil {
		return err
	}
	if binding.StartedAt.IsZero() || binding.CompletedAt.IsZero() {
		return fmt.Errorf("timing metadata missing")
	}
	if !binding.CompletedAt.After(binding.StartedAt) {
		return fmt.Errorf("timing metadata invalid")
	}
	if binding.OutboundBytes <= 0 {
		return fmt.Errorf("outbound byte count missing")
	}
	return nil
}

func validateLLMExecutionBindingCore(binding llmExecutionBinding) error {
	if _, err := binding.RequestHash.Identity(); err != nil {
		return fmt.Errorf("request_hash missing")
	}
	if strings.TrimSpace(binding.DestinationRef) == "" {
		return fmt.Errorf("destination_ref missing")
	}
	return nil
}

// Backward-compatible wrappers retained for tests.
func (s *Service) evaluateModelGatewayInvoke(requestID, runID string, binding llmExecutionBinding, _ string) (policyengine.PolicyDecision, *ErrorResponse) {
	return s.evaluateModelGatewayInvokeAdmission(requestID, runID, binding)
}

func (s *Service) emitModelGatewayAudit(runID string, decision policyengine.PolicyDecision, outcome string, binding llmExecutionBinding) error {
	return s.emitModelGatewayTerminalAudit(runID, decision, outcome, binding)
}
