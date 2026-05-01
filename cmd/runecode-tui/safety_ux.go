package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderRunSafetyStrip(summary brokerapi.RunSummary, width int) string {
	runtimeDegraded := summary.RuntimePostureDegraded
	parts := []string{
		tableHeader("Safety strip"),
		fmt.Sprintf("backend_kind=%s", valueOrNA(summary.BackendKind)),
	}
	parts = append(parts, runtimeIsolationCueParts(summary.BackendKind, summary.IsolationAssuranceLevel)...)
	parts = append(parts, fmt.Sprintf("runtime_posture_degraded=%t", runtimeDegraded), renderRuntimePostureDegradedBadge(runtimeDegraded))
	parts = append(parts, provisioningPostureCueParts(summary.ProvisioningPosture)...)
	parts = append(parts, auditPostureCueParts(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded)...)
	parts = append(parts, approvalProfileCueParts(summary.ApprovalProfile)...)
	return wrapPartsByWidth(parts, " | ", width)
}

func authoritativeVerifierClassCueParts(state map[string]any) []string {
	class, key := firstAuthoritativeString(state, "attestation_verifier_class")
	if class == "" {
		return []string{"verifier class=n/a", infoBadge("VERIFIER_CLASS_UNREPORTED")}
	}
	return []string{fmt.Sprintf("verifier class=%s (source=%s)", class, key), infoBadge("VERIFIER_CLASS_REPORTED")}
}

func renderAuthoritativeVerifierClassCue(state map[string]any) string {
	return strings.Join(authoritativeVerifierClassCueParts(state), " ")
}

func reducedAssurancePostureCueParts(state map[string]any) []string {
	runtimeDegraded, hasRuntimeDegraded := state["runtime_posture_degraded"].(bool)
	if !hasRuntimeDegraded {
		return []string{"reduced_assurance=n/a", infoBadge("REDUCED_ASSURANCE_UNREPORTED")}
	}
	parts := []string{fmt.Sprintf("reduced_assurance=%t", runtimeDegraded)}
	approvalBacked, _ := state["reduced_assurance_approval_backed"].(bool)
	approvalStatus, _ := state["reduced_assurance_approval_status"].(string)
	if approvalBacked {
		parts = append(parts, fmt.Sprintf("approval_backed=true status=%s", valueOrNA(approvalStatus)))
		parts = append(parts, infoBadge("REDUCED_ASSURANCE_APPROVAL_BACKED"))
		return parts
	}
	if runtimeDegraded {
		parts = append(parts, "approval_backed=false")
	} else {
		parts = append(parts, "approval_backed=n/a")
	}
	if runtimeDegraded {
		parts = append(parts, reducedAssuranceBadge("REDUCED_ASSURANCE_ACTIVE"))
	} else {
		parts = append(parts, successBadge("REDUCED_ASSURANCE_INACTIVE"))
	}
	return parts
}

func renderReducedAssurancePostureCue(state map[string]any) string {
	return strings.Join(reducedAssurancePostureCueParts(state), " ")
}

func supportedRuntimeRequirementsCueParts(state map[string]any) []string {
	satisfied, ok := state["supported_runtime_requirements_satisfied"].(bool)
	if !ok {
		return []string{"supported_runtime_requirements=n/a", infoBadge("SUPPORTED_RUNTIME_UNREPORTED")}
	}
	reasons := authoritativeStringSlice(state, "supported_runtime_requirement_reason_codes")
	if satisfied {
		return []string{"supported_runtime_requirements_satisfied=true", successBadge("SUPPORTED_RUNTIME_REQUIREMENTS_SATISFIED")}
	}
	reasonSuffix := ""
	if len(reasons) > 0 {
		reasonSuffix = " reasons=" + strings.Join(reasons, ",")
	}
	return []string{fmt.Sprintf("supported_runtime_requirements_satisfied=false%s", reasonSuffix), warnBadge("SUPPORTED_RUNTIME_REQUIREMENTS_UNSATISFIED")}
}

func renderSupportedRuntimeRequirementsCue(state map[string]any) string {
	return strings.Join(supportedRuntimeRequirementsCueParts(state), " ")
}

func firstAuthoritativeString(state map[string]any, keys ...string) (string, string) {
	for _, key := range keys {
		value, _ := state[key].(string)
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed, key
		}
	}
	return "", ""
}

func authoritativeStringSlice(state map[string]any, key string) []string {
	if values, ok := state[key].([]string); ok {
		return append([]string{}, values...)
	}
	valuesAny, ok := state[key].([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(valuesAny))
	for _, value := range valuesAny {
		stringValue, ok := value.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(stringValue)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func wrapPartsByWidth(parts []string, separator string, width int) string {
	if len(parts) == 0 {
		return ""
	}
	if width <= 0 {
		return strings.Join(parts, separator)
	}
	lines := []string{parts[0]}
	for _, part := range parts[1:] {
		candidate := lines[len(lines)-1] + separator + part
		if lipgloss.Width(candidate) <= width {
			lines[len(lines)-1] = candidate
			continue
		}
		lines = append(lines, part)
	}
	return strings.Join(lines, "\n")
}

func renderRuntimePostureDegradedBadge(degraded bool) string {
	if degraded {
		return reducedAssuranceBadge("RUNTIME_POSTURE_DEGRADED")
	}
	return successBadge("RUNTIME_POSTURE_NOMINAL")
}

func runtimeIsolationCueParts(backendKind, isolation string) []string {
	nBackend := strings.ToLower(strings.TrimSpace(backendKind))
	nIsolation := strings.ToLower(strings.TrimSpace(isolation))
	if nBackend == "container" || strings.Contains(nIsolation, "container") {
		return []string{"runtime isolation=container (reduced assurance)", reducedAssuranceBadge("RUNTIME_REDUCED_CONTAINER")}
	}
	switch nIsolation {
	case "sandboxed", "isolated", "microvm":
		return []string{fmt.Sprintf("runtime isolation=%s", valueOrNA(isolation)), successBadge("RUNTIME_ASSURED")}
	case "reduced":
		return []string{fmt.Sprintf("runtime isolation=%s", valueOrNA(isolation)), reducedAssuranceBadge("RUNTIME_REDUCED")}
	case "degraded", "unknown", "unavailable":
		return []string{fmt.Sprintf("runtime isolation=%s (authoritative posture degraded/unavailable)", valueOrNA(isolation)), dangerBadge("RUNTIME_POSTURE_DEGRADED")}
	default:
		return []string{fmt.Sprintf("runtime isolation=%s", valueOrNA(isolation)), infoBadge("RUNTIME_POSTURE_REPORTED")}
	}
}

func renderRuntimeIsolationCue(backendKind, isolation string) string {
	return strings.Join(runtimeIsolationCueParts(backendKind, isolation), " ")
}

func provisioningPostureCueParts(posture string) []string {
	n := strings.ToLower(strings.TrimSpace(posture))
	switch n {
	case "ok", "trusted", "bound", "attested":
		return []string{fmt.Sprintf("provisioning posture=%s", valueOrNA(posture)), successBadge("PROVISIONING_OK")}
	case "tofu":
		return []string{fmt.Sprintf("provisioning posture=%s (unsupported legacy TOFU posture)", valueOrNA(posture)), dangerBadge("PROVISIONING_TOFU_UNSUPPORTED")}
	case "degraded", "unavailable", "unknown":
		return []string{fmt.Sprintf("provisioning posture=%s (degraded)", valueOrNA(posture)), provisioningDegradedBadge("PROVISIONING_DEGRADED")}
	default:
		return []string{fmt.Sprintf("provisioning posture=%s", valueOrNA(posture)), infoBadge("PROVISIONING_REPORTED")}
	}
}

func renderProvisioningPostureCue(posture string) string {
	return strings.Join(provisioningPostureCueParts(posture), " ")
}

func attestationPostureCueParts(posture string, reasonCodes []string) []string {
	n := strings.ToLower(strings.TrimSpace(posture))
	reason := ""
	if len(reasonCodes) > 0 {
		reason = " reasons=" + strings.Join(reasonCodes, ",")
	}
	switch n {
	case "valid":
		return []string{fmt.Sprintf("attestation posture=%s (evidence present, verification succeeded; isolation assurance varies by runtime posture)", valueOrNA(posture)), successBadge("ATTESTATION_VALID")}
	case "tofu_only":
		return []string{fmt.Sprintf("attestation posture=%s (unsupported legacy session-binding-only posture)", valueOrNA(posture)), dangerBadge("ATTESTATION_TOFU_ONLY_UNSUPPORTED")}
	case "unavailable":
		return []string{fmt.Sprintf("attestation posture=%s (evidence/verification unavailable%s)", valueOrNA(posture), reason), warnBadge("ATTESTATION_UNAVAILABLE")}
	case "invalid":
		return []string{fmt.Sprintf("attestation posture=%s (evidence rejected%s)", valueOrNA(posture), reason), dangerBadge("ATTESTATION_INVALID")}
	case "not_applicable":
		return []string{fmt.Sprintf("attestation posture=%s", valueOrNA(posture)), infoBadge("ATTESTATION_NA")}
	default:
		return []string{fmt.Sprintf("attestation posture=%s%s", valueOrNA(posture), reason), infoBadge("ATTESTATION_REPORTED")}
	}
}

func renderAttestationPostureCue(posture string, reasonCodes []string) string {
	return strings.Join(attestationPostureCueParts(posture, reasonCodes), " ")
}

func auditPostureCueParts(integrity, anchoring string, degraded bool) []string {
	nAnchoring := strings.ToLower(strings.TrimSpace(anchoring))
	nIntegrity := strings.ToLower(strings.TrimSpace(integrity))
	if nAnchoring == "failed" || nIntegrity == "failed" || nIntegrity == "invalid" {
		return []string{fmt.Sprintf("audit posture=%s/%s (invalid/failed anchoring)", valueOrNA(integrity), valueOrNA(anchoring)), dangerBadge("AUDIT_FAILED")}
	}
	if degraded || nAnchoring == "degraded" {
		return []string{fmt.Sprintf("audit posture=%s/%s (unanchored/degraded)", valueOrNA(integrity), valueOrNA(anchoring)), auditDegradedBadge("AUDIT_UNANCHORED_OR_DEGRADED")}
	}
	return []string{fmt.Sprintf("audit posture=%s/%s", valueOrNA(integrity), valueOrNA(anchoring)), successBadge("AUDIT_ANCHORED")}
}

func renderAuditPostureCue(integrity, anchoring string, degraded bool) string {
	return strings.Join(auditPostureCueParts(integrity, anchoring, degraded), " ")
}

func approvalProfileCueParts(profile string) []string {
	n := strings.ToLower(strings.TrimSpace(profile))
	if n == "" || n == "unknown" {
		return []string{fmt.Sprintf("approval_profile=%s", valueOrNA(profile)), warnBadge("APPROVAL_PROFILE_UNKNOWN")}
	}
	return []string{fmt.Sprintf("approval_profile=%s", profile), infoBadge("APPROVAL_PROFILE_ACTIVE")}
}

func renderApprovalProfileCue(profile string) string {
	return strings.Join(approvalProfileCueParts(profile), " ")
}

func renderBlockingStateCue(blocked bool, reasonCode string) string {
	if !blocked {
		return successBadge("NOT_BLOCKING")
	}
	n := strings.ToLower(strings.TrimSpace(reasonCode))
	switch {
	case strings.Contains(n, "gate_override") || strings.Contains(n, "overrid"):
		return gateOverrideBadge("GATE_OVERRIDE")
	case strings.Contains(n, "approval"):
		return approvalRequiredBadge("APPROVAL_REQUIRED")
	case strings.Contains(n, "gate_failed") || strings.Contains(n, "failed"):
		return dangerBadge("GATE_FAILURE")
	case strings.Contains(n, "system") || strings.Contains(n, "error") || strings.Contains(n, "unavailable"):
		return systemFailureBadge("SYSTEM_FAILURE")
	default:
		return blockingBadge("BLOCKING")
	}
}

func renderAdvisoryStateCue(state map[string]any) string {
	if len(state) == 0 {
		return neutralBadge("ADVISORY_EMPTY")
	}
	return advisoryBadge(fmt.Sprintf("ADVISORY_KEYS_%d", len(state)))
}
