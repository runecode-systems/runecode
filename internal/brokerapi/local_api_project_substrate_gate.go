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
	switch strings.TrimSpace(schemaPath) {
	case readinessGetRequestSchemaPath,
		versionInfoGetRequestSchemaPath,
		projectSubstrateGetRequestSchemaPath,
		projectSubstratePostureGetRequestSchemaPath,
		projectSubstrateAdoptRequestSchemaPath,
		projectSubstrateInitPreviewRequestSchemaPath,
		projectSubstrateInitApplyRequestSchemaPath,
		projectSubstrateUpgradePreviewRequestSchemaPath,
		projectSubstrateUpgradeApplyRequestSchemaPath,
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
		providerSetupSessionBeginRequestSchemaPath,
		providerSetupSecretIngressPrepareRequestSchemaPath,
		providerSetupSecretIngressSubmitRequestSchemaPath,
		providerValidationBeginRequestSchemaPath,
		providerValidationCommitRequestSchemaPath,
		providerCredentialLeaseIssueRequestSchemaPath,
		gitSetupGetRequestSchemaPath,
		gitSetupAuthBootstrapRequestSchemaPath,
		gitSetupIdentityUpsertRequestSchemaPath:
		return true
	default:
		return false
	}
}
