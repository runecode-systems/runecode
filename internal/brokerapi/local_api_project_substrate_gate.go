package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) enforceProjectSubstrateGate(requestID, schemaPath string) *ErrorResponse {
	if s == nil {
		return nil
	}
	if isProjectSubstrateDiagnosticsSchema(schemaPath) {
		return nil
	}
	result, err := s.discoverProjectSubstrate()
	if err != nil {
		if !isProjectSubstrateDiscoveryFailureAllowed(schemaPath) {
			errResp := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
			return &errResp
		}
		result = projectsubstrate.DiscoveryResult{}
	}
	compatibility := result.Compatibility
	if compatibility.NormalOperationAllowed {
		return nil
	}
	blocked := strings.Join(compatibility.BlockedReasonCodes, ",")
	if strings.TrimSpace(blocked) == "" {
		blocked = compatibility.Posture
	}
	errResp := s.makeError(requestID, "project_substrate_operation_blocked", "policy", false, "project substrate posture blocks normal operation: "+blocked)
	return &errResp
}

func isProjectSubstrateDiscoveryFailureAllowed(schemaPath string) bool {
	switch strings.TrimSpace(schemaPath) {
	case readinessGetRequestSchemaPath, versionInfoGetRequestSchemaPath:
		return true
	default:
		return false
	}
}

func isProjectSubstrateDiagnosticsSchema(schemaPath string) bool {
	path := strings.TrimSpace(schemaPath)
	return isProjectSubstrateInspectSchema(path) || isProjectSubstrateManagementSchema(path)
}

func isProjectSubstrateInspectSchema(schemaPath string) bool {
	switch schemaPath {
	case readinessGetRequestSchemaPath,
		versionInfoGetRequestSchemaPath,
		zkProofGenerateRequestSchemaPath,
		zkProofVerifyRequestSchemaPath,
		runListRequestSchemaPath,
		runGetRequestSchemaPath,
		approvalListRequestSchemaPath,
		approvalGetRequestSchemaPath,
		artifactListRequestSchemaPath,
		artifactHeadRequestSchemaPath,
		sessionListRequestSchemaPath,
		sessionGetRequestSchemaPath,
		productLifecyclePostureGetRequestSchemaPath,
		projectSubstrateGetRequestSchemaPath,
		projectSubstratePostureGetRequestSchemaPath,
		auditTimelineRequestSchemaPath,
		auditVerificationGetRequestSchemaPath,
		auditRecordGetRequestSchemaPath,
		auditFinalizeVerifyRequestSchemaPath,
		auditAnchorPreflightGetRequestSchemaPath,
		auditAnchorPresenceGetRequestSchemaPath,
		auditAnchorSegmentRequestSchemaPath,
		backendPostureGetRequestSchemaPath,
		providerProfileListRequestSchemaPath,
		providerProfileGetRequestSchemaPath,
		gitSetupGetRequestSchemaPath:
		return true
	default:
		return false
	}
}

func isProjectSubstrateManagementSchema(schemaPath string) bool {
	switch schemaPath {
	case projectSubstrateAdoptRequestSchemaPath,
		projectSubstrateInitPreviewRequestSchemaPath,
		projectSubstrateInitApplyRequestSchemaPath,
		projectSubstrateUpgradePreviewRequestSchemaPath,
		projectSubstrateUpgradeApplyRequestSchemaPath,
		providerSetupSessionBeginRequestSchemaPath,
		providerSetupSecretIngressPrepareRequestSchemaPath,
		providerSetupSecretIngressSubmitRequestSchemaPath,
		providerValidationBeginRequestSchemaPath,
		providerValidationCommitRequestSchemaPath,
		providerCredentialLeaseIssueRequestSchemaPath,
		gitSetupAuthBootstrapRequestSchemaPath,
		gitSetupIdentityUpsertRequestSchemaPath:
		return true
	default:
		return false
	}
}
