package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderRunSafetyStrip(summary brokerapi.RunSummary) string {
	runtimeDegraded := summary.RuntimePostureDegraded || strings.EqualFold(strings.TrimSpace(summary.IsolationAssuranceLevel), "degraded") || strings.EqualFold(strings.TrimSpace(summary.BackendKind), "container")
	parts := []string{
		tableHeader("Safety strip"),
		fmt.Sprintf("backend_kind=%s", valueOrNA(summary.BackendKind)),
		renderRuntimeIsolationCue(summary.BackendKind, summary.IsolationAssuranceLevel),
		fmt.Sprintf("runtime_posture_degraded=%t %s", runtimeDegraded, renderRuntimePostureDegradedBadge(runtimeDegraded)),
		renderProvisioningPostureCue(summary.ProvisioningPosture),
		renderAuditPostureCue(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded),
		renderApprovalProfileCue(summary.ApprovalProfile),
	}
	return strings.Join(parts, " | ")
}

func renderRuntimePostureDegradedBadge(degraded bool) string {
	if degraded {
		return reducedAssuranceBadge("RUNTIME_POSTURE_DEGRADED")
	}
	return successBadge("RUNTIME_POSTURE_NOMINAL")
}

func renderRuntimeIsolationCue(backendKind, isolation string) string {
	nBackend := strings.ToLower(strings.TrimSpace(backendKind))
	nIsolation := strings.ToLower(strings.TrimSpace(isolation))
	if nBackend == "container" || strings.Contains(nIsolation, "container") {
		return "runtime isolation=container (reduced assurance) " + reducedAssuranceBadge("RUNTIME_REDUCED_CONTAINER")
	}
	switch nIsolation {
	case "sandboxed", "isolated", "microvm":
		return fmt.Sprintf("runtime isolation=%s %s", valueOrNA(isolation), successBadge("RUNTIME_ASSURED"))
	case "reduced":
		return fmt.Sprintf("runtime isolation=%s %s", valueOrNA(isolation), reducedAssuranceBadge("RUNTIME_REDUCED"))
	case "degraded", "unknown", "unavailable":
		return fmt.Sprintf("runtime isolation=%s (authoritative posture degraded/unavailable) %s", valueOrNA(isolation), dangerBadge("RUNTIME_POSTURE_DEGRADED"))
	default:
		return fmt.Sprintf("runtime isolation=%s %s", valueOrNA(isolation), infoBadge("RUNTIME_POSTURE_REPORTED"))
	}
}

func renderProvisioningPostureCue(posture string) string {
	n := strings.ToLower(strings.TrimSpace(posture))
	switch n {
	case "ok", "trusted", "bound":
		return fmt.Sprintf("provisioning posture=%s %s", valueOrNA(posture), successBadge("PROVISIONING_OK"))
	case "tofu":
		return fmt.Sprintf("provisioning posture=%s (TOFU isolate key provisioning) %s", valueOrNA(posture), provisioningDegradedBadge("PROVISIONING_TOFU"))
	case "degraded", "unavailable", "unknown":
		return fmt.Sprintf("provisioning posture=%s (degraded) %s", valueOrNA(posture), provisioningDegradedBadge("PROVISIONING_DEGRADED"))
	default:
		return fmt.Sprintf("provisioning posture=%s %s", valueOrNA(posture), infoBadge("PROVISIONING_REPORTED"))
	}
}

func renderAuditPostureCue(integrity, anchoring string, degraded bool) string {
	nAnchoring := strings.ToLower(strings.TrimSpace(anchoring))
	nIntegrity := strings.ToLower(strings.TrimSpace(integrity))
	if nAnchoring == "failed" || nIntegrity == "failed" || nIntegrity == "invalid" {
		return fmt.Sprintf("audit posture=%s/%s (invalid/failed anchoring) %s", valueOrNA(integrity), valueOrNA(anchoring), dangerBadge("AUDIT_FAILED"))
	}
	if degraded || nAnchoring == "degraded" {
		return fmt.Sprintf("audit posture=%s/%s (unanchored/degraded) %s", valueOrNA(integrity), valueOrNA(anchoring), auditDegradedBadge("AUDIT_UNANCHORED_OR_DEGRADED"))
	}
	return fmt.Sprintf("audit posture=%s/%s %s", valueOrNA(integrity), valueOrNA(anchoring), successBadge("AUDIT_ANCHORED"))
}

func renderApprovalProfileCue(profile string) string {
	n := strings.ToLower(strings.TrimSpace(profile))
	if n == "" || n == "unknown" {
		return fmt.Sprintf("approval_profile=%s %s", valueOrNA(profile), warnBadge("APPROVAL_PROFILE_UNKNOWN"))
	}
	return fmt.Sprintf("approval_profile=%s %s", profile, infoBadge("APPROVAL_PROFILE_ACTIVE"))
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
