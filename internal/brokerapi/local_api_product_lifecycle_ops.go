package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) HandleProductLifecyclePostureGet(ctx context.Context, req ProductLifecyclePostureGetRequest, meta RequestContext) (ProductLifecyclePostureGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, productLifecyclePostureGetRequestSchemaPath)
	if errResp != nil {
		return ProductLifecyclePostureGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ProductLifecyclePostureGetResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ProductLifecyclePostureGetResponse{}, &errOut
	}
	project, err := s.discoverProjectSubstrate()
	if err != nil {
		project = s.syntheticProjectSubstrateDiscoveryFailure()
	}
	posture := s.projectProductLifecyclePosture(project)
	resp := ProductLifecyclePostureGetResponse{
		SchemaID:         "runecode.protocol.v0.ProductLifecyclePostureGetResponse",
		SchemaVersion:    "0.1.0",
		RequestID:        requestID,
		ProductLifecycle: posture,
	}
	if err := s.validateResponse(resp, productLifecyclePostureGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProductLifecyclePostureGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) projectProductLifecyclePosture(project projectsubstrate.DiscoveryResult) BrokerProductLifecyclePosture {
	normalAllowed := project.Compatibility.NormalOperationAllowed
	attachMode, lifecyclePosture, blockedReasons, degradedReasons := projectLifecycleSemantics(project.Compatibility, normalAllowed)
	activeSessions := s.countActiveSessions()
	activeRuns := s.countActiveRuns()
	return BrokerProductLifecyclePosture{
		SchemaID:                     "runecode.protocol.v0.BrokerProductLifecyclePosture",
		SchemaVersion:                "0.1.0",
		ProductInstanceID:            strings.TrimSpace(s.productInstanceID),
		LifecycleGeneration:          strings.TrimSpace(s.lifecycleGeneration),
		AttachMode:                   attachMode,
		LifecyclePosture:             lifecyclePosture,
		Attachable:                   true,
		NormalOperationAllowed:       normalAllowed,
		BlockedReasonCodes:           blockedReasons,
		DegradedReasonCodes:          degradedReasons,
		RepositoryRoot:               strings.TrimSpace(project.RepositoryRoot),
		ProjectContextIdentityDigest: strings.TrimSpace(project.Snapshot.ProjectContextIdentityDigest),
		ActiveSessionCount:           activeSessions,
		ActiveRunCount:               activeRuns,
	}
}

func projectLifecycleSemantics(compatibility projectsubstrate.CompatibilityAssessment, normalAllowed bool) (string, string, []string, []string) {
	attachMode := "full"
	lifecyclePosture := "ready"
	blockedReasons := append([]string{}, compatibility.BlockedReasonCodes...)
	degradedReasons := []string{}
	if !normalAllowed {
		attachMode = "diagnostics_only"
		lifecyclePosture = "blocked"
	}
	if compatibility.Posture == "supported_with_upgrade_available" {
		degradedReasons = append(degradedReasons, "project_substrate_upgrade_available")
		if lifecyclePosture == "ready" {
			lifecyclePosture = "degraded"
		}
	}
	if len(blockedReasons) == 0 && !normalAllowed {
		posture := strings.TrimSpace(compatibility.Posture)
		if posture != "" {
			blockedReasons = append(blockedReasons, "project_substrate_"+posture)
		}
	}
	return attachMode, lifecyclePosture, blockedReasons, degradedReasons
}

func (s *Service) syntheticProjectSubstrateDiscoveryFailure() projectsubstrate.DiscoveryResult {
	repoRoot := strings.TrimSpace(s.projectSubstrate.RepositoryRoot)
	if repoRoot == "" {
		repoRoot = strings.TrimSpace(s.apiConfig.RepositoryRoot)
	}
	return projectsubstrate.DiscoveryResult{
		RepositoryRoot: repoRoot,
		Snapshot: projectsubstrate.ValidationSnapshot{
			SchemaID:      "runecode.protocol.v0.ProjectSubstrateValidationSnapshot",
			SchemaVersion: "0.1.0",
		},
		Compatibility: projectsubstrate.CompatibilityAssessment{
			Posture:                "invalid",
			NormalOperationAllowed: false,
			BlockedReasonCodes:     []string{"project_substrate_discovery_failed"},
		},
	}
}

func (s *Service) countActiveSessions() int {
	states := s.store.SessionDurableStates()
	count := 0
	for _, state := range states {
		status := strings.TrimSpace(state.Status)
		if status == "" || status == "active" || status == "open" {
			count++
		}
	}
	return count
}

func (s *Service) countActiveRuns() int {
	runStatus := s.RunStatuses()
	count := 0
	for _, status := range runStatus {
		if !isTerminalRunStatus(strings.TrimSpace(status)) {
			count++
		}
	}
	return count
}
