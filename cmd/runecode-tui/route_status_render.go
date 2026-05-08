package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderLifecycleOperationCard(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateEmpty, Title: "Managed operation", Message: "Managed-operation posture is unavailable.", Reason: "The broker has not published current attach and lifecycle state for this route yet.", NextAction: "Press r to reload or inspect broker health if the route stays unavailable.", ShortcutCue: "r reload"})
	}
	state := routeLoadStateReady
	message := "Attach and normal managed operation are available."
	reason := fmt.Sprintf("Broker lifecycle posture=%s attach_mode=%s", valueOrNA(posture.LifecyclePosture), valueOrNA(posture.AttachMode))
	nextAction := "Normal managed work can continue; use Chat or Runs for active execution details."
	routeCue := "Chat or Runs"
	if !posture.Attachable {
		state = routeLoadStateBlocked
		message = "RuneCode cannot attach to the current broker lifecycle right now."
		reason = lifecycleReasonSummary(posture)
		nextAction = "Reload the route, then inspect broker status or remediation guidance before retrying."
		routeCue = "Status"
	} else if !posture.NormalOperationAllowed {
		state = routeLoadStateBlocked
		reason = lifecycleReasonSummary(posture)
		nextAction = "Use the project setup guidance below, then reload once the broker posture changes."
		routeCue = "Status"
		if strings.EqualFold(strings.TrimSpace(posture.AttachMode), "diagnostics_only") {
			message = "You can attach for diagnostics and remediation only; managed work stays blocked."
		} else {
			message = "You can attach to inspect current state, but managed work stays blocked."
		}
	}
	return renderStateCardSpec(stateCardSpec{State: state, Title: "Managed operation", Message: message, Reason: reason, NextAction: nextAction, RouteCue: routeCue})
}

func renderLifecycleStatusLine(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return "Managed-operation details: unavailable"
	}
	return fmt.Sprintf("Managed-operation details: instance=%s generation=%s posture=%s attach_mode=%s attachable=%t normal_operation_allowed=%t", valueOrNA(posture.ProductInstanceID), valueOrNA(posture.LifecycleGeneration), valueOrNA(posture.LifecyclePosture), valueOrNA(posture.AttachMode), posture.Attachable, posture.NormalOperationAllowed)
}

func renderLifecycleBlockedReasonLine(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return "Managed-operation blocking reasons: unavailable"
	}
	if len(posture.BlockedReasonCodes) == 0 {
		return "Managed-operation blocking reasons: none"
	}
	return "Managed-operation blocking reasons: " + joinCSV(posture.BlockedReasonCodes)
}

func renderLifecycleDegradedReasonLine(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return "Managed-operation degraded reasons: unavailable"
	}
	if len(posture.DegradedReasonCodes) == 0 {
		return "Managed-operation degraded reasons: none"
	}
	return "Managed-operation degraded reasons: " + joinCSV(posture.DegradedReasonCodes)
}

func lifecycleReasonSummary(posture brokerapi.BrokerProductLifecyclePosture) string {
	parts := []string{}
	if len(posture.BlockedReasonCodes) > 0 {
		parts = append(parts, "blocked="+joinCSV(posture.BlockedReasonCodes))
	}
	if len(posture.DegradedReasonCodes) > 0 {
		parts = append(parts, "degraded="+joinCSV(posture.DegradedReasonCodes))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Broker lifecycle posture=%s attach_mode=%s", valueOrNA(posture.LifecyclePosture), valueOrNA(posture.AttachMode))
	}
	return strings.Join(parts, " • ")
}

func renderBackendPostureLine(posture brokerapi.BackendPostureState) string {
	if strings.TrimSpace(posture.InstanceID) == "" {
		return "Backend posture: unavailable"
	}
	return fmt.Sprintf("Backend posture: instance=%s backend=%s reduced_assurance=%t pending_approval=%t", valueOrNA(posture.InstanceID), valueOrNA(posture.BackendKind), posture.ReducedAssuranceActive, posture.PendingApproval)
}

func renderStatusSafetyStrip(r brokerapi.BrokerReadiness) string {
	parts := []string{tableHeader("Runtime/audit readiness strip")}
	if !r.VerifierMaterialAvailable {
		parts = append(parts, dangerBadge("RUNTIME_POSTURE_AUTH_UNAVAILABLE"))
	} else {
		parts = append(parts, successBadge("RUNTIME_POSTURE_AUTH_AVAILABLE"))
	}
	if !r.RecoveryComplete || !r.AppendPositionStable || !r.CurrentSegmentWritable {
		parts = append(parts, auditDegradedBadge("AUDIT_POSTURE_DEGRADED_OR_UNAVAILABLE"))
	} else {
		parts = append(parts, successBadge("AUDIT_STORAGE_NOMINAL"))
	}
	if !r.Ready {
		parts = append(parts, systemFailureBadge("SYSTEM_FAILURE_BROKER_NOT_READY"))
	}
	return compactLines(parts...)
}

func renderReadinessDiagnostics(r brokerapi.BrokerReadiness) string {
	issues := []string{}
	if !r.RecoveryComplete {
		issues = append(issues, "recovery=incomplete")
	}
	if !r.AppendPositionStable {
		issues = append(issues, "ledger_append=unstable")
	}
	if !r.CurrentSegmentWritable {
		issues = append(issues, "current_segment=read_only_or_unavailable")
	}
	if !r.VerifierMaterialAvailable {
		issues = append(issues, "verifier_material=missing")
	}
	if !r.DerivedIndexCaughtUp {
		issues = append(issues, "derived_index=lagging")
	}
	if len(issues) == 0 {
		return "Diagnostics: all readiness subsystems report nominal posture."
	}
	return fmt.Sprintf("Diagnostics: degraded subsystems=%s", joinCSV(issues))
}

func renderProjectSubstrateStatusLine(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return "Project setup details: unavailable"
	}
	return fmt.Sprintf("Project setup details: validation=%s compatibility=%s normal_operation_allowed=%t", valueOrNA(summary.ValidationState), valueOrNA(summary.CompatibilityPosture), summary.NormalOperationAllowed)
}

func renderProjectSubstrateStatusCard(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateEmpty, Title: "Project setup", Message: "Project setup posture is unavailable.", Reason: "The broker has not returned current inspect/adopt/init/upgrade posture for this repository yet.", NextAction: "Press r to reload. If this persists, inspect broker health before attempting setup changes.", ShortcutCue: "r reload"})
	}
	state := routeLoadStateReady
	message := "Project setup supports normal managed work."
	reason := fmt.Sprintf("validation=%s compatibility=%s", valueOrNA(summary.ValidationState), valueOrNA(summary.CompatibilityPosture))
	nextAction := "Review upgrade guidance if recommended; otherwise continue normal work."
	if !summary.NormalOperationAllowed {
		state = routeLoadStateBlocked
		message = "Project setup needs attention before managed work can continue."
		reason = projectSubstrateReasonSummary(posture)
		nextAction = "Inspect the guided flow below, then adopt, preview, or apply the broker-owned remediation that fits this repository posture."
	} else if strings.Contains(strings.ToLower(strings.TrimSpace(summary.CompatibilityPosture)), "upgrade") {
		state = routeLoadStateDegraded
		message = "Project setup is usable, but a broker-owned upgrade is available."
		reason = projectSubstrateReasonSummary(posture)
		nextAction = "Review the upgrade preview before choosing whether to apply it."
	}
	return renderStateCardSpec(stateCardSpec{State: state, Title: "Project setup", Message: message, Reason: reason, NextAction: nextAction, RouteCue: "Status"})
}

func renderProjectSubstrateGuidance(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	parts := []string{tableHeader("Guided setup/remediation flow")}
	parts = append(parts, fmt.Sprintf("Inspect current posture: validation=%s compatibility=%s normal_operation_allowed=%t", valueOrNA(summary.ValidationState), valueOrNA(summary.CompatibilityPosture), summary.NormalOperationAllowed))
	parts = append(parts, fmt.Sprintf("Compatible adoption (a): no mutation; broker only records compatible existing substrate status=%s", projectSubstrateStepStatus(posture.Adoption.Status)))
	parts = append(parts, fmt.Sprintf("Init preview/apply (i/I): preview status=%s mutation=%s handle=%s", projectSubstrateStepStatus(posture.InitPreview.Status), projectSubstrateMutationLabel(posture.InitPreview.Status, posture.InitPreview.PreviewToken), projectSubstrateHandleDisplay(posture.InitPreview.PreviewToken)))
	parts = append(parts, fmt.Sprintf("Upgrade preview/apply (u/U): preview status=%s mutation=%s digest=%s", projectSubstrateStepStatus(posture.UpgradePreview.Status), projectSubstrateMutationLabel(posture.UpgradePreview.Status, posture.UpgradePreview.PreviewDigest), projectSubstrateHandleDisplay(posture.UpgradePreview.PreviewDigest)))
	if strings.TrimSpace(posture.BlockedExplanation) != "" {
		parts = append(parts, "What blocks normal work: "+sanitizeUIText(posture.BlockedExplanation))
	}
	if len(posture.RemediationGuidance) > 0 {
		parts = append(parts, "Broker guidance: "+joinCSV(posture.RemediationGuidance))
	}
	if !summary.NormalOperationAllowed {
		parts = append(parts, "Next action: use adopt if the repository is already compatible; otherwise run preview before any apply, then reload to validate resulting posture.")
	} else if strings.Contains(strings.ToLower(strings.TrimSpace(summary.CompatibilityPosture)), "upgrade") {
		parts = append(parts, "Next action: managed work is allowed, but review the broker-owned upgrade preview and revalidate after any apply.")
	} else {
		parts = append(parts, "Next action: no setup remediation is required right now; reload after any external change if you want fresh broker validation.")
	}
	if strings.TrimSpace(posture.InitPreview.Status) == "" && strings.TrimSpace(posture.UpgradePreview.Status) == "" && strings.TrimSpace(posture.BlockedExplanation) == "" && len(posture.RemediationGuidance) == 0 {
		parts = append(parts, "No preview or remediation details are currently published.")
	}
	return compactLines(parts...)
}

func projectSubstrateReasonSummary(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	parts := []string{}
	if len(summary.BlockedReasonCodes) > 0 {
		parts = append(parts, "blocked="+joinCSV(summary.BlockedReasonCodes))
	}
	if len(summary.ReasonCodes) > 0 {
		parts = append(parts, "reasons="+joinCSV(summary.ReasonCodes))
	}
	if explanation := strings.TrimSpace(posture.BlockedExplanation); explanation != "" {
		parts = append(parts, sanitizeUIText(explanation))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("validation=%s compatibility=%s", valueOrNA(summary.ValidationState), valueOrNA(summary.CompatibilityPosture))
	}
	return strings.Join(parts, " • ")
}

func projectSubstrateStepStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "unavailable"
	}
	return status
}

func projectSubstrateMutationLabel(status string, handle string) string {
	if strings.TrimSpace(handle) == "" {
		return "no mutation until broker preview is available"
	}
	if strings.EqualFold(strings.TrimSpace(status), "ready_for_apply") {
		return "apply will mutate the repository through the broker-owned flow"
	}
	return "preview only; no mutation yet"
}
