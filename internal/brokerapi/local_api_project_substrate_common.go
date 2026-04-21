package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) prepareProjectSubstrateRequest(ctx context.Context, reqID, fallbackReqID string, admissionErr error, req any, schemaPath string, meta RequestContext) (string, context.Context, func(), *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(reqID, fallbackReqID, admissionErr, req, schemaPath)
	if errResp != nil {
		return "", nil, nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	cleanup := func() {
		cancel()
		release()
	}
	if err := requestCtx.Err(); err != nil {
		cleanup()
		errOut := s.errorFromContext(requestID, err)
		return "", nil, nil, &errOut
	}
	return requestID, requestCtx, cleanup, nil
}

func (s *Service) projectSubstrateOperationInput() (string, projectsubstrate.RepoRootAuthority) {
	return s.projectSubstrate.RepositoryRoot, s.projectSubstrateAuthority()
}

func (s *Service) refreshProjectSubstrateDiscovery(requestID string) (projectsubstrate.DiscoveryResult, *ErrorResponse) {
	result, err := s.discoverProjectSubstrate()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return projectsubstrate.DiscoveryResult{}, &errOut
	}
	return result, nil
}

func (s *Service) projectSubstrateAuthority() projectsubstrate.RepoRootAuthority {
	if s.projectSubstrate.Contract.RepoRootAuthority == string(projectsubstrate.RepoRootAuthorityProcessWorkingDirectory) {
		return projectsubstrate.RepoRootAuthorityProcessWorkingDirectory
	}
	return projectsubstrate.RepoRootAuthorityExplicitConfig
}

func projectSubstrateBlockedProjection(summary ProjectSubstratePostureSummary, initPreview projectsubstrate.InitPreview, upgradePreview projectsubstrate.UpgradePreview) (string, []string) {
	if summary.NormalOperationAllowed {
		advisory := []string{}
		if summary.CompatibilityPosture == projectsubstrate.CompatibilityPostureSupportedWithUpgrade {
			advisory = append(advisory, "upgrade_preview_available")
			advisory = append(advisory, normalizeGuidanceCodes(upgradePreview.RequiredFollowUp)...)
		}
		return "", advisory
	}
	guidance := []string{"inspect_project_substrate_posture"}
	guidance = append(guidance, normalizeGuidanceCodes(initPreview.RequiredFollowUp)...)
	guidance = append(guidance, normalizeGuidanceCodes(upgradePreview.RequiredFollowUp)...)
	if summary.CompatibilityPosture == projectsubstrate.CompatibilityPostureMissing {
		guidance = append(guidance, "initialize_canonical_runecontext_substrate")
	}
	explanation := "normal operation blocked by project substrate posture"
	if len(summary.BlockedReasonCodes) > 0 {
		explanation += ": " + strings.Join(summary.BlockedReasonCodes, ",")
	}
	return explanation, normalizeGuidanceCodes(guidance)
}

func normalizeGuidanceCodes(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
