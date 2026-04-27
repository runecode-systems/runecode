package brokerapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const dependencyFetchCapabilityID = "cap_gateway"

var errDependencyFetchPolicyDenied = errors.New("dependency fetch denied by policy")

type dependencyFetchAuthorization struct {
	destinationKind     string
	destinationRef      string
	actionRequestHash   string
	policyDecisionHash  string
	matchedAllowlistRef string
	matchedAllowlistID  string
	maxResponseBytes    int64
}

func (s *dependencyFetchService) authorizeDependencyFetch(runID string, req DependencyFetchRequestObject, requestHash string) (dependencyFetchAuthorization, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return dependencyFetchAuthorization{}, fmt.Errorf("%w: dependency fetch run_id is required", errPolicyContextUnavailable)
	}
	action, err := policyActionForDependencyFetch(req, requestHash)
	if err != nil {
		return dependencyFetchAuthorization{}, err
	}
	decision, err := s.evaluateDependencyFetchPolicy(runID, action)
	if err != nil {
		return dependencyFetchAuthorization{}, err
	}
	return s.allowlistAuthorizationForDependencyFetch(runID, req, decision)
}

func (s *dependencyFetchService) evaluateDependencyFetchPolicy(runID string, action policyengine.ActionRequest) (policyengine.PolicyDecision, error) {
	decision, evalErr := s.owner.EvaluateAction(runID, action)
	if evalErr != nil {
		if errors.Is(evalErr, errPolicyContextUnavailable) {
			return policyengine.PolicyDecision{}, fmt.Errorf("%w: %w: %v", errPolicyContextUnavailable, errDependencyFetchPolicyDenied, evalErr)
		}
		return policyengine.PolicyDecision{}, evalErr
	}
	if decision.DecisionOutcome != policyengine.DecisionAllow {
		return policyengine.PolicyDecision{}, dependencyFetchPolicyDeniedError(decision)
	}
	return decision, nil
}

func (s *dependencyFetchService) allowlistAuthorizationForDependencyFetch(runID string, req DependencyFetchRequestObject, decision policyengine.PolicyDecision) (dependencyFetchAuthorization, error) {
	runtime := policyRuntime{service: s.owner}
	compileInput, err := runtime.loadCompileInput(runID)
	if err != nil {
		return dependencyFetchAuthorization{}, err
	}
	payload := gatewayActionPayloadRuntime{
		GatewayRoleKind: "dependency-fetch",
		DestinationKind: strings.TrimSpace(req.RegistryIdentity.DescriptorKind),
		DestinationRef:  destinationRefFromDescriptor(req.RegistryIdentity),
		Operation:       "fetch_dependency",
	}
	entry, match, found, reason := findAllowlistEntryForGatewayPayload(compileInput.Allowlists, payload)
	if !found {
		if strings.TrimSpace(reason) == "" {
			reason = "runtime_gateway_destination_not_allowlisted"
		}
		return dependencyFetchAuthorization{}, fmt.Errorf("dependency fetch allowlist context unavailable: %s", reason)
	}
	maxResponseBytes := int64(gatewayRuntimeMaxResponseBytes)
	if entry.MaxResponseBytes != nil && *entry.MaxResponseBytes > 0 {
		maxResponseBytes = int64(*entry.MaxResponseBytes)
	}
	return dependencyFetchAuthorization{
		destinationKind:     payload.DestinationKind,
		destinationRef:      payload.DestinationRef,
		actionRequestHash:   strings.TrimSpace(decision.ActionRequestHash),
		policyDecisionHash:  decisionDigestIdentity(decision),
		matchedAllowlistRef: strings.TrimSpace(match.AllowlistRef),
		matchedAllowlistID:  strings.TrimSpace(match.EntryID),
		maxResponseBytes:    maxResponseBytes,
	}, nil
}

func dependencyFetchPolicyDeniedError(decision policyengine.PolicyDecision) error {
	reason := strings.TrimSpace(policyDecisionInvariantReason(decision))
	if reason == "" {
		reason = "policy_decision_non_allow"
	}
	return fmt.Errorf("%w: decision outcome %q (%s) reason=%s", errDependencyFetchPolicyDenied, decision.DecisionOutcome, decision.PolicyReasonCode, reason)
}

func policyActionForDependencyFetch(req DependencyFetchRequestObject, requestHashIdentity string) (policyengine.ActionRequest, error) {
	requestHash, err := digestFromIdentity(requestHashIdentity)
	if err != nil {
		return policyengine.ActionRequest{}, err
	}
	timeoutSeconds := 30
	return policyengine.NewDependencyFetchAction(policyengine.GatewayEgressActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID:           dependencyFetchCapabilityID,
			RelevantArtifactHashes: []trustpolicy.Digest{requestHash},
			Actor:                  policyengine.ActionActor{ActorKind: "role_instance", RoleFamily: "gateway", RoleKind: "dependency-fetch"},
		},
		GatewayRoleKind: "dependency-fetch",
		DestinationKind: strings.TrimSpace(req.RegistryIdentity.DescriptorKind),
		DestinationRef:  destinationRefFromDescriptor(req.RegistryIdentity),
		EgressDataClass: "dependency_resolved_payload",
		Operation:       "fetch_dependency",
		TimeoutSeconds:  &timeoutSeconds,
		PayloadHash:     &requestHash,
		AuditContext:    dependencyFetchAdmissionAuditContext(requestHash),
		QuotaContext:    dependencyFetchAdmissionQuotaContext(),
		DependencyRequest: &policyengine.DependencyFetchRequestInput{
			RegistryIdentity: req.RegistryIdentity,
			Ecosystem:        strings.TrimSpace(req.Ecosystem),
			PackageName:      strings.TrimSpace(req.PackageName),
			PackageVersion:   strings.TrimSpace(req.PackageVersion),
		},
	}), nil
}

func dependencyFetchAdmissionAuditContext(requestHash trustpolicy.Digest) *policyengine.GatewayAuditContextInput {
	now := time.Now().UTC()
	completed := now.Add(time.Millisecond)
	return &policyengine.GatewayAuditContextInput{
		OutboundBytes: 0,
		StartedAt:     now.Format(time.RFC3339),
		CompletedAt:   completed.Format(time.RFC3339),
		Outcome:       "admission_allowed",
		RequestHash:   &requestHash,
	}
}

func dependencyFetchAdmissionQuotaContext() *policyengine.GatewayQuotaContextInput {
	requestUnits := int64(1)
	inputTokens := int64(1)
	return &policyengine.GatewayQuotaContextInput{
		QuotaProfileKind:    "hybrid",
		Phase:               "admission",
		EnforceDuringStream: false,
		Meters: policyengine.GatewayQuotaMetersInput{
			RequestUnits: &requestUnits,
			InputTokens:  &inputTokens,
		},
	}
}

func policyDecisionInvariantReason(decision policyengine.PolicyDecision) string {
	if decision.Details == nil {
		return ""
	}
	reason, _ := decision.Details["reason"].(string)
	return reason
}
