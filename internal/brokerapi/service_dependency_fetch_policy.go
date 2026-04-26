package brokerapi

import (
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *dependencyFetchService) enrichPolicyAndAllowlistLinkage(runID string, req DependencyFetchRequestObject, resolution *dependencyUnitResolution) {
	if resolution == nil {
		return
	}
	resolution.destinationKind = strings.TrimSpace(req.RegistryIdentity.DescriptorKind)
	resolution.destinationRef = destinationRefFromDescriptor(req.RegistryIdentity)
	policyRef, actionHash, ok := s.latestDependencyFetchPolicyDecision(runID, resolution.requestHash)
	if ok {
		resolution.policyDecisionHash = policyRef
		resolution.actionRequestHash = actionHash
	}
	runtime := policyRuntime{service: s.owner}
	compileInput, err := runtime.loadCompileInput(strings.TrimSpace(runID))
	if err != nil {
		return
	}
	payload := gatewayActionPayloadRuntime{
		GatewayRoleKind: "dependency-fetch",
		DestinationKind: resolution.destinationKind,
		DestinationRef:  resolution.destinationRef,
		Operation:       "fetch_dependency",
	}
	_, match, found, _ := findAllowlistEntryForGatewayPayload(compileInput.Allowlists, payload)
	if !found {
		return
	}
	resolution.matchedAllowlistRef = strings.TrimSpace(match.AllowlistRef)
	resolution.matchedAllowlistID = strings.TrimSpace(match.EntryID)
}

func (s *dependencyFetchService) latestDependencyFetchPolicyDecision(runID, requestHash string) (string, string, bool) {
	runID = strings.TrimSpace(runID)
	requestHash = strings.TrimSpace(requestHash)
	if runID == "" || requestHash == "" {
		return "", "", false
	}
	latestRef := ""
	latestAction := ""
	var latestRecordedAt time.Time
	for _, ref := range s.owner.PolicyDecisionRefsForRun(runID) {
		rec, ok := s.owner.PolicyDecisionGet(ref)
		if !ok || !matchesDependencyFetchPolicyDecision(rec, requestHash) {
			continue
		}
		recordedAt := rec.RecordedAt.UTC()
		if !isNewerDependencyFetchPolicyDecision(ref, recordedAt, latestRef, latestRecordedAt) {
			continue
		}
		latestRef = ref
		latestAction = strings.TrimSpace(rec.ActionRequestHash)
		latestRecordedAt = recordedAt
	}
	if latestRef == "" {
		return "", "", false
	}
	return latestRef, latestAction, true
}

func isNewerDependencyFetchPolicyDecision(ref string, recordedAt time.Time, latestRef string, latestRecordedAt time.Time) bool {
	return latestRef == "" || recordedAt.After(latestRecordedAt) || (recordedAt.Equal(latestRecordedAt) && ref > latestRef)
}

func matchesDependencyFetchPolicyDecision(rec artifacts.PolicyDecisionRecord, requestHash string) bool {
	if strings.TrimSpace(rec.ActionRequestHash) == "" {
		return false
	}
	if !containsStringIdentity(rec.RelevantArtifactHashes, requestHash) {
		return false
	}
	if operation, ok := rec.Details["operation"].(string); ok && strings.TrimSpace(operation) != "" && strings.TrimSpace(operation) != "fetch_dependency" {
		return false
	}
	if role, ok := rec.Details["gateway_role_kind"].(string); ok && strings.TrimSpace(role) != "" && strings.TrimSpace(role) != "dependency-fetch" {
		return false
	}
	if kind, ok := rec.Details["destination_kind"].(string); ok && strings.TrimSpace(kind) != "" && strings.TrimSpace(kind) != "package_registry" {
		return false
	}
	return true
}
