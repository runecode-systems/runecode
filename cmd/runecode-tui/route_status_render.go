package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func renderLifecycleOperationCard(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateEmpty, Title: "Managed operation", Message: "Managed-operation posture is unavailable.", Reason: "The broker has not published current attach and lifecycle state for this route yet.", NextAction: "Press r to reload or inspect broker health if the route stays unavailable.", ShortcutCue: "r reload"})
	}
	state := routeLoadStateReady
	message := "Attach and normal managed operation are available."
	reason := lifecycleReasonSummary(posture)
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
		return "Managed-operation summary: unavailable"
	}
	parts := []string{}
	if strings.TrimSpace(posture.AttachMode) != "" {
		parts = append(parts, attachModeSummary(posture.AttachMode))
	}
	if posture.ActiveSessionCount > 0 || posture.ActiveRunCount > 0 {
		parts = append(parts, fmt.Sprintf("Active work: %d sessions • %d runs", posture.ActiveSessionCount, posture.ActiveRunCount))
	}
	if posture.Attachable && posture.NormalOperationAllowed {
		parts = append(parts, "Normal work is available now")
	} else if posture.Attachable {
		parts = append(parts, "Attach is available for inspection and remediation")
	} else {
		parts = append(parts, "Attach is not currently available")
	}
	return "Managed-operation summary: " + strings.Join(parts, " • ")
}

func renderLifecycleBlockedReasonLine(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return "Managed-operation blockers: unavailable"
	}
	if len(posture.BlockedReasonCodes) == 0 {
		return "Managed-operation blockers: none reported"
	}
	return "Managed-operation blockers: " + humanizeStatusCodes(posture.BlockedReasonCodes)
}

func renderLifecycleDegradedReasonLine(posture brokerapi.BrokerProductLifecyclePosture) string {
	if strings.TrimSpace(posture.SchemaID) == "" {
		return "Managed-operation watchouts: unavailable"
	}
	if len(posture.DegradedReasonCodes) == 0 {
		return "Managed-operation watchouts: none reported"
	}
	return "Managed-operation watchouts: " + humanizeStatusCodes(posture.DegradedReasonCodes)
}

func lifecycleReasonSummary(posture brokerapi.BrokerProductLifecyclePosture) string {
	parts := []string{}
	if len(posture.BlockedReasonCodes) > 0 {
		parts = append(parts, "blocked because "+humanizeStatusCodes(posture.BlockedReasonCodes))
	}
	if len(posture.DegradedReasonCodes) > 0 {
		parts = append(parts, "watchouts: "+humanizeStatusCodes(posture.DegradedReasonCodes))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Broker reports %s operation with %s access.", lifecyclePostureSummary(posture.LifecyclePosture), attachModeSummary(posture.AttachMode))
	}
	return strings.Join(parts, " • ")
}

func renderBackendPostureLine(posture brokerapi.BackendPostureState) string {
	if strings.TrimSpace(posture.InstanceID) == "" {
		return "Backend: unavailable"
	}
	parts := []string{fmt.Sprintf("Backend: %s", valueOrNA(posture.BackendKind))}
	if posture.PendingApproval {
		parts = append(parts, "approval pending")
	}
	if posture.ReducedAssuranceActive {
		parts = append(parts, "reduced assurance active")
	}
	return strings.Join(parts, " • ")
}

func renderStatusSafetyStrip(r brokerapi.BrokerReadiness) string {
	parts := []string{tableHeader("System health")}
	if !r.VerifierMaterialAvailable {
		parts = append(parts, dangerBadge("Runtime proof missing"))
	} else {
		parts = append(parts, successBadge("Runtime proof ready"))
	}
	if !r.RecoveryComplete || !r.AppendPositionStable || !r.CurrentSegmentWritable {
		parts = append(parts, auditDegradedBadge("Audit needs review"))
	} else {
		parts = append(parts, successBadge("Audit ready"))
	}
	if !r.Ready {
		parts = append(parts, systemFailureBadge("Broker not ready"))
	}
	return compactLines(parts...)
}

func renderReadinessDiagnostics(r brokerapi.BrokerReadiness) string {
	issues := []string{}
	if !r.RecoveryComplete {
		issues = append(issues, "recovery still in progress")
	}
	if !r.AppendPositionStable {
		issues = append(issues, "audit ledger append is unstable")
	}
	if !r.CurrentSegmentWritable {
		issues = append(issues, "current audit segment is not writable")
	}
	if !r.VerifierMaterialAvailable {
		issues = append(issues, "runtime verification material is unavailable")
	}
	if !r.DerivedIndexCaughtUp {
		issues = append(issues, "derived index still catching up")
	}
	if len(issues) == 0 {
		return "Broker health: ready for normal work."
	}
	return fmt.Sprintf("Broker health: %s.", strings.Join(issues, "; "))
}

func renderProjectSubstrateStatusLine(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return "Project setup summary: unavailable"
	}
	parts := []string{projectSetupValidationSummary(summary.ValidationState), projectSetupCompatibilitySummary(summary.CompatibilityPosture)}
	if summary.NormalOperationAllowed {
		parts = append(parts, "Normal work can continue")
	} else {
		parts = append(parts, "Normal work stays blocked until setup is fixed")
	}
	return "Project setup summary: " + strings.Join(parts, " • ")
}

func renderProjectSubstrateActionCard(spec *stateCardSpec) string {
	if spec == nil {
		return ""
	}
	return renderStateCardSpec(*spec)
}

func renderProjectSubstrateStatusCard(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateEmpty, Title: "Project setup", Message: "Project setup posture is unavailable.", Reason: "The broker has not returned current inspect/adopt/init/upgrade posture for this repository yet.", NextAction: "Press r to reload. If this persists, inspect broker health before attempting setup changes.", ShortcutCue: "r reload"})
	}
	state := routeLoadStateReady
	message := "Project setup supports normal managed work."
	reason := projectSubstrateReasonSummary(posture)
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
	parts := []string{tableHeader("Project setup guidance")}
	parts = append(parts, renderProjectSubstrateActionSummaryLine("Adopt (a)", posture.Adoption.Status, "Use when RuneCode should recognize an already-compatible setup without changing files."))
	parts = append(parts, renderProjectSubstratePreviewSummaryLine("Init (i/I)", posture.InitPreview.Status, posture.InitPreview.PreviewToken, "Preview setup creation, then apply only if you want RuneCode to make that change."))
	parts = append(parts, renderProjectSubstratePreviewSummaryLine("Upgrade (u/U)", posture.UpgradePreview.Status, posture.UpgradePreview.PreviewDigest, "Preview the recommended upgrade, then apply only after review."))
	if strings.TrimSpace(posture.BlockedExplanation) != "" {
		parts = append(parts, "What blocks normal work: "+sanitizeUIText(posture.BlockedExplanation))
	}
	if len(posture.RemediationGuidance) > 0 {
		parts = append(parts, "Broker guidance: "+humanizeStatusCodes(posture.RemediationGuidance))
	}
	if !summary.NormalOperationAllowed {
		parts = append(parts, "Next action: choose the lightest broker-owned fix that matches this repository, then reload to confirm managed work is restored.")
	} else if strings.Contains(strings.ToLower(strings.TrimSpace(summary.CompatibilityPosture)), "upgrade") {
		parts = append(parts, "Next action: managed work can continue, but review the upgrade preview when you are ready to bring setup back to the recommended posture.")
	} else {
		parts = append(parts, "Next action: no setup remediation is required right now.")
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
		parts = append(parts, "blocked because "+humanizeStatusCodes(summary.BlockedReasonCodes))
	}
	if len(summary.ReasonCodes) > 0 {
		parts = append(parts, "notes: "+humanizeStatusCodes(summary.ReasonCodes))
	}
	if explanation := strings.TrimSpace(posture.BlockedExplanation); explanation != "" {
		parts = append(parts, sanitizeUIText(explanation))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s • %s", projectSetupValidationSummary(summary.ValidationState), projectSetupCompatibilitySummary(summary.CompatibilityPosture))
	}
	return strings.Join(parts, " • ")
}

func renderStatusOverviewLine(r brokerapi.BrokerReadiness, lifecycle brokerapi.BrokerProductLifecyclePosture, posture brokerapi.ProjectSubstratePostureGetResponse) string {
	parts := []string{"Overview:"}
	if r.Ready {
		parts = append(parts, "broker connected")
	} else {
		parts = append(parts, "broker not ready")
	}
	if lifecycle.NormalOperationAllowed && posture.PostureSummary.NormalOperationAllowed {
		parts = append(parts, "normal work available")
	} else {
		parts = append(parts, "normal work blocked")
	}
	if r.LocalOnly {
		parts = append(parts, "local workspace mode")
	}
	return strings.Join(parts, " • ")
}

func renderStatusSetupLine(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return "Setup: unavailable"
	}
	return fmt.Sprintf("Setup: %s • %s", projectSetupValidationSummary(summary.ValidationState), projectSetupCompatibilitySummary(summary.CompatibilityPosture))
}

func renderVersionPostureLine(v brokerapi.BrokerVersionInfo) string {
	parts := []string{fmt.Sprintf("Version: RuneCode %s", valueOrNA(v.ProductVersion))}
	if strings.TrimSpace(v.ProtocolBundleVersion) != "" {
		parts = append(parts, fmt.Sprintf("protocol %s", sanitizeUIText(v.ProtocolBundleVersion)))
	}
	if strings.TrimSpace(v.APIFamily) != "" || strings.TrimSpace(v.APIVersion) != "" {
		parts = append(parts, fmt.Sprintf("broker API ready (%s)", versionAPIShortLabel(v.APIFamily, v.APIVersion)))
	}
	return strings.Join(parts, " • ")
}

func humanizeStatusCodes(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parts = append(parts, humanizeExecutionToken(value))
	}
	if len(parts) == 0 {
		return "n/a"
	}
	return strings.Join(parts, ", ")
}

func renderProjectSubstrateActionSummaryLine(label, status, guidance string) string {
	return fmt.Sprintf("%s: %s. %s", label, projectSubstrateActionStatusSummary(status), guidance)
}

func renderProjectSubstratePreviewSummaryLine(label, status, handle, guidance string) string {
	summary := projectSubstratePreviewStatusSummary(status, handle)
	if strings.TrimSpace(handle) != "" {
		summary += fmt.Sprintf(" (%s ready)", projectSubstrateHandleLabel(handle))
	}
	return fmt.Sprintf("%s: %s. %s", label, summary, guidance)
}

func lifecyclePostureSummary(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ready", "normal":
		return "normal"
	case "blocked":
		return "blocked"
	case "degraded":
		return "degraded"
	default:
		return valueOrNA(value)
	}
}

func attachModeSummary(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "managed", "full":
		return "full managed access"
	case "diagnostics_only":
		return "diagnostics-only access"
	case "":
		return "access posture not reported"
	default:
		return sanitizeUIText(value)
	}
}

func projectSetupValidationSummary(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "valid":
		return "setup validated"
	case "missing":
		return "setup missing"
	case "incompatible":
		return "setup incompatible"
	case "":
		return "validation not reported"
	default:
		return "validation: " + sanitizeUIText(value)
	}
}

func projectSetupCompatibilitySummary(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "supported":
		return "fully supported"
	case "supported_with_upgrade_available":
		return "upgrade available"
	case "missing":
		return "no compatible setup detected"
	case "unsupported":
		return "not supported for normal work"
	case "":
		return "compatibility not reported"
	default:
		return "compatibility: " + sanitizeUIText(value)
	}
}

func projectSubstrateActionStatusSummary(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "compatible_existing":
		return "compatible existing setup can be recognized without changes"
	case "unavailable", "":
		return "not currently available"
	default:
		return "broker reports " + sanitizeUIText(status)
	}
}

func projectSubstratePreviewStatusSummary(status, handle string) string {
	if strings.TrimSpace(handle) == "" {
		return "not ready yet"
	}
	if strings.EqualFold(strings.TrimSpace(status), "ready_for_apply") {
		return "preview ready for review and optional apply"
	}
	if strings.TrimSpace(status) == "" {
		return "not currently available"
	}
	return "broker reports " + sanitizeUIText(status)
}

func projectSubstrateHandleLabel(handle string) string {
	if strings.TrimSpace(handle) == "" {
		return "preview"
	}
	return "preview handle"
}

func versionAPIShortLabel(family, version string) string {
	family = strings.TrimSpace(family)
	version = strings.TrimSpace(version)
	switch {
	case family == "" && version == "":
		return "not reported"
	case family == "":
		return sanitizeUIText(version)
	case version == "":
		return sanitizeUIText(family)
	default:
		return sanitizeUIText(family) + " " + sanitizeUIText(version)
	}
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
