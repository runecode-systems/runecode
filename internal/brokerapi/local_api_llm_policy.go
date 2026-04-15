package brokerapi

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) evaluateModelGatewayInvoke(requestID, runID string, binding llmExecutionBinding, outcome string) (policyengine.PolicyDecision, *ErrorResponse) {
	if err := validateLLMExecutionBindingForPolicy(binding); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, llmExecutionUnavailableMessage)
		return policyengine.PolicyDecision{}, &errOut
	}
	destinationRef, err := s.trustedLLMDestinationRefForRun(runID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, llmExecutionUnavailableMessage)
		return policyengine.PolicyDecision{}, &errOut
	}
	action := llmGatewayEgressAction(binding, outcome, destinationRef)
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

func llmGatewayEgressAction(binding llmExecutionBinding, outcome, destinationRef string) policyengine.ActionRequest {
	return policyengine.NewGatewayEgressAction(policyengine.GatewayEgressActionInput{
		ActionEnvelope:  llmGatewayActionEnvelope(binding),
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  destinationRef,
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
	if err := validateLLMExecutionBindingForAudit(binding); err != nil {
		return fmt.Errorf("llm execution metadata unavailable: %w", err)
	}
	destinationRef, err := s.trustedLLMDestinationRefForRun(runID)
	if err != nil {
		return fmt.Errorf("llm execution metadata unavailable: %w", err)
	}
	policyHash := decisionDigestIdentity(decision)
	policyDecisionHash, err := digestFromIdentityOrNil(policyHash)
	if err != nil {
		return fmt.Errorf("llm execution metadata unavailable: invalid policy decision hash: %w", err)
	}
	return s.gatewayRuntime.emitGatewayAuditEvent(runID, decision, gatewayActionPayloadRuntime{
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  destinationRef,
		Operation:       "invoke_model",
		PayloadHash:     &binding.RequestHash,
		AuditContext: &gatewayAuditContextPayload{
			Outcome:            outcome,
			RequestHash:        &binding.RequestHash,
			ResponseHash:       &binding.ResponseHash,
			LeaseID:            binding.LeaseID,
			PolicyDecisionHash: policyDecisionHash,
		},
	})
}

func (s *Service) trustedLLMDestinationRefForRun(runID string) (string, error) {
	runtime := policyRuntime{service: s}
	compileInput, err := runtime.loadCompileInput(strings.TrimSpace(runID))
	if err != nil {
		return "", err
	}
	return resolveLLMDestinationRefFromAllowlists(compileInput.Allowlists)
}

func resolveLLMDestinationRefFromAllowlists(allowlists []policyengine.ManifestInput) (string, error) {
	for _, allowlistInput := range allowlists {
		allowlist := policyengine.PolicyAllowlist{}
		if err := json.Unmarshal(allowlistInput.Payload, &allowlist); err != nil {
			return "", fmt.Errorf("decode trusted allowlist payload: %w", err)
		}
		for _, entry := range allowlist.Entries {
			if !entrySupportsLLMInvoke(entry) {
				continue
			}
			return destinationRefFromDescriptor(entry.Destination), nil
		}
	}
	return "", fmt.Errorf("trusted model gateway destination unavailable")
}

func entrySupportsLLMInvoke(entry policyengine.GatewayScopeRule) bool {
	if entry.ScopeKind != "gateway_destination" {
		return false
	}
	if !isHardenedModelDestination(entry.Destination) {
		return false
	}
	roleKind := strings.TrimSpace(entry.GatewayRoleKind)
	if roleKind != "" && roleKind != "model-gateway" {
		return false
	}
	for _, operation := range entry.PermittedOperations {
		if operation == "invoke_model" {
			return true
		}
	}
	return false
}

func isHardenedModelDestination(destination policyengine.DestinationDescriptor) bool {
	if destination.DescriptorKind != "model_endpoint" {
		return false
	}
	if strings.TrimSpace(destination.CanonicalHost) == "" {
		return false
	}
	if strings.Contains(destination.CanonicalPathPrefix, "..") {
		return false
	}
	if !destination.TLSRequired {
		return false
	}
	if destination.PrivateRangeBlocking != "enforced" {
		return false
	}
	if destination.DNSRebindingProtection != "enforced" {
		return false
	}
	return true
}

func destinationRefFromDescriptor(descriptor policyengine.DestinationDescriptor) string {
	ref := strings.TrimSpace(descriptor.CanonicalHost)
	if descriptor.CanonicalPort != nil && *descriptor.CanonicalPort != 443 {
		ref = fmt.Sprintf("%s:%d", ref, *descriptor.CanonicalPort)
	}
	return ref + normalizeDestinationPathPrefix(descriptor.CanonicalPathPrefix)
}

func normalizeDestinationPathPrefix(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	normalized := path.Clean(trimmed)
	if normalized == "." {
		return "/"
	}
	return normalized
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
	return nil
}
