package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func projectSupportedRuntimeRequirementsState(state map[string]any) {
	delete(state, "supported_runtime_requirement_reason_codes")
	delete(state, "reduced_assurance_approval_backed")
	delete(state, "reduced_assurance_approval_status")

	reasons := supportedRuntimeRequirementReasonCodes(state)
	state["supported_runtime_requirements_satisfied"] = len(reasons) == 0
	if len(reasons) > 0 {
		state["supported_runtime_requirement_reason_codes"] = reasons
	}
	reducedAssuranceApprovalBacked, approvalStatus := reducedAssuranceApprovalStatus(state)
	if runtimePostureDegraded, _ := state["runtime_posture_degraded"].(bool); runtimePostureDegraded {
		state["reduced_assurance_approval_backed"] = reducedAssuranceApprovalBacked
		if approvalStatus != "" {
			state["reduced_assurance_approval_status"] = approvalStatus
		}
	}
}

func supportedRuntimeRequirementReasonCodes(state map[string]any) []string {
	reasons := make([]string, 0, 3)
	if posture, _ := state["attestation_posture"].(string); posture != launcherbackend.AttestationPostureValid {
		reasons = append(reasons, "attestation_posture_not_valid")
	}
	if verifierClass, _ := state["attestation_verifier_class"].(string); verifierClass == "" || verifierClass == launcherbackend.AttestationVerifierClassUnknown {
		reasons = append(reasons, "attestation_verifier_class_unknown")
	} else if verifierClass == launcherbackend.AttestationVerifierClassExternalVerifier {
		reasons = append(reasons, "attestation_verifier_class_unacceptable")
	}
	if runtimePostureDegraded, _ := state["runtime_posture_degraded"].(bool); runtimePostureDegraded {
		approvalBacked, _ := reducedAssuranceApprovalStatus(state)
		if !approvalBacked {
			reasons = append(reasons, "reduced_assurance_selection_evidence_missing")
		}
	}
	return reasons
}

func reducedAssuranceApprovalStatus(state map[string]any) (bool, string) {
	evidence, _ := state["backend_posture_selection_evidence"].(map[string]any)
	if len(evidence) == 0 {
		return false, ""
	}
	approval, _ := evidence["approval"].(map[string]any)
	if len(approval) == 0 {
		return false, ""
	}
	status, _ := approval["status"].(string)
	trimmedStatus := strings.TrimSpace(status)
	return approvalEvidenceSatisfiesReducedAssurance(trimmedStatus), trimmedStatus
}
