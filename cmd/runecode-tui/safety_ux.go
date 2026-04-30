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
		return []string{fmt.Sprintf("provisioning posture=%s (TOFU isolate key provisioning)", valueOrNA(posture)), provisioningDegradedBadge("PROVISIONING_TOFU")}
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
		return []string{fmt.Sprintf("attestation posture=%s (evidence present, verification succeeded)", valueOrNA(posture)), successBadge("ATTESTATION_VALID")}
	case "tofu_only":
		return []string{fmt.Sprintf("attestation posture=%s (session binding only; no verified attestation)", valueOrNA(posture)), provisioningDegradedBadge("ATTESTATION_TOFU_ONLY")}
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
