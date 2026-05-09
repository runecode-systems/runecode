package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type runEvidenceLinks struct {
	SessionID      string
	ApprovalIDs    []string
	ArtifactLabels []string
	AuditLabels    []string
}

func renderRunInspector(detail *brokerapi.RunDetail, presentation contentPresentationMode, document *longFormDocumentState) string {
	if detail == nil {
		return "  Select a run to review workflow status, outcome, and linked evidence."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	summary := detail.Summary
	presentation = normalizePresentationMode(presentation)
	waitingStages := pendingApprovalStageCount(detail.StageSummaries)
	waitingRoles := waitingRoleCount(detail.RoleSummaries)
	links := buildRunEvidenceLinks(detail)
	content := runInspectorContent(summary, detail, waitingStages, waitingRoles, links, presentation)
	contentKind := runInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "run", ID: strings.TrimSpace(summary.RunID), WorkspaceID: strings.TrimSpace(summary.WorkspaceID)}
	document.SetDocument(ref, contentKind, "run details", content)
	return renderInspectorShell(inspectorShellSpec{
		Title:        "Run inspector",
		Summary:      runInspectorSummary(summary, detail),
		Identity:     runInspectorIdentity(summary, detail),
		Status:       runInspectorStatus(summary, detail),
		Badges:       []string{stateBadgeWithLabel("state", summary.LifecycleState), appTheme.InspectorHint.Render("authoritative vs advisory state shown")},
		References:   runInspectorReferences(detail, links),
		LocalActions: runInspectorLocalActions(),
		CopyActions:  runRouteCopyActions(detail),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		Document:     document,
	})
}

func runInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "jump:session", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeChat}}},
		{Label: "jump:approvals", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}},
		{Label: "jump:artifacts", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}},
		{Label: "jump:audit", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}},
		{Label: "copy:run_id"},
		{Label: "copy:session_id"},
	}
}

func runInspectorReferences(detail *brokerapi.RunDetail, links runEvidenceLinks) []inspectorReference {
	if detail == nil {
		return nil
	}
	refs := []inspectorReference{}
	if sessionID := strings.TrimSpace(links.SessionID); sessionID != "" {
		refs = append(refs, inspectorReference{Label: "session", Items: []inspectorReferenceItem{{Label: sessionID, Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "session", RouteID: routeChat, SessionID: sessionID}}}}})
	}
	refs = append(refs,
		inspectorReference{Label: "approvals", Items: mapReferenceIDs(links.ApprovalIDs, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}
		})},
		inspectorReference{Label: "artifacts", Items: mapReferenceIDs(links.ArtifactLabels, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}
		})},
		inspectorReference{Label: "audit", Items: mapReferenceIDs(links.AuditLabels, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}
		})},
	)
	return refs
}

func runInspectorReferenceActions(detail *brokerapi.RunDetail) []routeActionItem {
	if detail == nil {
		return nil
	}
	refs := runInspectorReferences(detail, buildRunEvidenceLinks(detail))
	out := make([]routeActionItem, 0, 12)
	for _, ref := range refs {
		for _, item := range ref.Items {
			if strings.TrimSpace(item.Label) == "" {
				continue
			}
			out = append(out, routeActionItem{Label: ref.Label + ":" + item.Label, Action: item.Action})
		}
	}
	return out
}

func pendingApprovalStageCount(stages []brokerapi.RunStageSummary) int {
	count := 0
	for _, stage := range stages {
		if stage.PendingApprovalCount > 0 {
			count++
		}
	}
	return count
}

func waitingRoleCount(roles []brokerapi.RunRoleSummary) int {
	count := 0
	for _, role := range roles {
		if role.WaitReasonCode != "" {
			count++
		}
	}
	return count
}

func buildRunEvidenceLinks(detail *brokerapi.RunDetail) runEvidenceLinks {
	if detail == nil {
		return runEvidenceLinks{}
	}
	return runEvidenceLinks{
		SessionID:      strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "session_id")),
		ApprovalIDs:    append([]string(nil), detail.PendingApprovalIDs...),
		ArtifactLabels: runArtifactReferenceLabels(detail),
		AuditLabels:    runAuditReferenceLabels(detail),
	}
}

func runInspectorContent(summary brokerapi.RunSummary, detail *brokerapi.RunDetail, waitingStages int, waitingRoles int, links runEvidenceLinks, presentation contentPresentationMode) string {
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("structured run detail: run=%s workflow=%s status=%s", summary.RunID, valueOrNA(summary.WorkflowKind), valueOrNA(summary.LifecycleState)),
			fmt.Sprintf("counts: authoritative=%d advisory=%d stages=%d roles=%d pending_approvals=%d artifact_classes=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState), len(detail.StageSummaries), len(detail.RoleSummaries), len(detail.PendingApprovalIDs), len(detail.ArtifactCountsByClass)),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			fmt.Sprintf("raw summary run_id=%s lifecycle=%s backend_kind=%s workflow_kind=%s", summary.RunID, summary.LifecycleState, summary.BackendKind, valueOrNA(summary.WorkflowKind)),
			fmt.Sprintf("raw coordination blocked=%t wait_reason=%s mode=%s", detail.Coordination.Blocked, valueOrNA(detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.CoordinationMode)),
			fmt.Sprintf("raw maps authoritative_keys=%d advisory_keys=%d artifact_classes=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState), len(detail.ArtifactCountsByClass)),
		)
	}
	attestationPosture, attestationReasons := attestationPostureFromState(detail.AuthoritativeState)
	return compactLines(
		runOutcomeHeadline(summary, detail),
		fmt.Sprintf("Workflow operation: %s", runWorkflowOperation(summary, detail)),
		fmt.Sprintf("Runtime backend: backend_kind=%s", valueOrNA(summary.BackendKind)),
		muted("Structured/raw modes expose workflow hashes, coordination internals, approval profile, manifests, policy refs, and broker map counts."),
		fmt.Sprintf("Plan authority: %s", runPlanAuthoritySummary(summary, detail)),
		fmt.Sprintf("Runner/reporting posture: %s", runReportingPostureSummary(detail)),
		fmt.Sprintf("Blocked/failure reason: %s", runBlockingReasonSummary(summary, detail)),
		fmt.Sprintf("Evidence links: %s", runEvidenceSummary(detail)),
		fmt.Sprintf("Linked approvals: %s", runJoinedOrNA(links.ApprovalIDs)),
		fmt.Sprintf("Artifacts trail: %s", runJoinedOrNA(links.ArtifactLabels)),
		fmt.Sprintf("Audit evidence: %s", runJoinedOrNA(links.AuditLabels)),
		fmt.Sprintf("Navigation cues: session=%s approvals=%d artifacts=%d audit=%d", valueOrNA(links.SessionID), len(links.ApprovalIDs), runArtifactCount(detail), len(links.AuditLabels)),
		"Runtime assurance: "+renderRuntimeIsolationCue(summary.BackendKind, summary.IsolationAssuranceLevel),
		"Provisioning posture: "+renderProvisioningPostureCue(summary.ProvisioningPosture),
		"Attestation truth: "+renderAttestationPostureCue(attestationPosture, attestationReasons)+"; "+renderRuntimeAttestationTruthfulnessCue(detail.AuthoritativeState),
		"Audit posture: "+renderAuditPostureCue(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded),
		fmt.Sprintf("Work summary: %d stage(s), %d with approval waits, %d waiting role(s), %d pending approval(s).", len(detail.StageSummaries), waitingStages, waitingRoles, len(detail.PendingApprovalIDs)),
	)
}

func runInspectorSummary(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("Run %s is %s with %d pending approval(s).", summary.RunID, valueOrNA(summary.LifecycleState), runPendingApprovalCount(summary, detail))
}

func runInspectorIdentity(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("run=%s workspace=%s session=%s", summary.RunID, valueOrNA(summary.WorkspaceID), valueOrNA(authoritativeString(detail.AuthoritativeState, "session_id")))
}

func runInspectorStatus(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("lifecycle=%s workflow=%s blocked=%t reason=%s", valueOrNA(summary.LifecycleState), runWorkflowOperation(summary, detail), detail.Coordination.Blocked, valueOrNA(runBlockingReasonCode(summary, detail)))
}

func runOutcomeHeadline(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("Outcome summary: run %s is %s.", summary.RunID, valueOrNA(summary.LifecycleState))
}

func runWorkflowOperation(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	workflow := strings.TrimSpace(summary.WorkflowKind)
	if workflow == "" {
		workflow = strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_kind"))
	}
	operation := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_operation"))
	if operation == "" {
		operation = strings.TrimSpace(summary.CurrentStageID)
	}
	if workflow != "" && operation != "" {
		return fmt.Sprintf("%s • stage=%s", workflow, operation)
	}
	if workflow != "" {
		return workflow
	}
	if operation != "" {
		return "stage=" + operation
	}
	return "n/a"
}

func runPendingApprovalCount(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) int {
	if detail != nil && len(detail.PendingApprovalIDs) > 0 {
		return len(detail.PendingApprovalIDs)
	}
	return summary.PendingApprovalCount
}

func runPlanAuthoritySummary(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	reason := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_projection_reason"))
	if strings.TrimSpace(summary.WorkflowDefinitionHash) != "" {
		return fmt.Sprintf("workflow definition hash %s with projection=%s", summary.WorkflowDefinitionHash, valueOrNA(reason))
	}
	if reason != "" {
		return fmt.Sprintf("projection=%s", reason)
	}
	return "plan authority not reported"
}

func runReportingPostureSummary(detail *brokerapi.RunDetail) string {
	parts := []string{}
	if runner := strings.TrimSpace(authoritativeString(detail.AdvisoryState, "runner")); runner != "" {
		parts = append(parts, "runner="+runner)
	}
	if checkpoint := strings.TrimSpace(runCheckpointCode(detail)); checkpoint != "" {
		parts = append(parts, "last checkpoint="+checkpoint)
	}
	if len(parts) == 0 {
		return "runner reporting has not produced advisory progress yet"
	}
	return strings.Join(parts, " • ")
}

func runJoinedOrNA(items []string) string {
	if len(items) == 0 {
		return "n/a"
	}
	return strings.Join(items, " • ")
}

func runBlockingReasonCode(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	if reason := strings.TrimSpace(summary.BlockingReasonCode); reason != "" {
		return reason
	}
	return strings.TrimSpace(detail.Coordination.WaitReasonCode)
}

func runBlockingReasonSummary(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	reason := runBlockingReasonCode(summary, detail)
	if reason == "" && !detail.Coordination.Blocked && !strings.Contains(strings.ToLower(strings.TrimSpace(summary.LifecycleState)), "fail") {
		return "no blocking or failure reason reported"
	}
	if reason == "" {
		return "state reported without a reason code"
	}
	return humanizeExecutionToken(reason)
}

func runEvidenceSummary(detail *brokerapi.RunDetail) string {
	parts := []string{
		fmt.Sprintf("approvals %d", len(detail.PendingApprovalIDs)),
		fmt.Sprintf("artifact classes %d", len(detail.ArtifactCountsByClass)),
		fmt.Sprintf("active manifests %d", len(detail.ActiveManifestHashes)),
		fmt.Sprintf("policy refs %d", len(detail.LatestPolicyDecisionRefs)),
	}
	return strings.Join(parts, " • ")
}

func runArtifactReferenceLabels(detail *brokerapi.RunDetail) []string {
	if detail == nil || len(detail.ArtifactCountsByClass) == 0 {
		return nil
	}
	keys := make([]string, 0, len(detail.ArtifactCountsByClass))
	for class := range detail.ArtifactCountsByClass {
		keys = append(keys, class)
	}
	sort.Strings(keys)
	labels := make([]string, 0, len(keys))
	for _, class := range keys {
		labels = append(labels, fmt.Sprintf("%s (%d)", class, detail.ArtifactCountsByClass[class]))
	}
	return labels
}

func runAuditReferenceLabels(detail *brokerapi.RunDetail) []string {
	if detail == nil {
		return nil
	}
	labels := []string{}
	for _, digest := range detail.ActiveManifestHashes {
		digest = strings.TrimSpace(digest)
		if digest == "" {
			continue
		}
		labels = append(labels, "manifest "+shortIdentity(digest))
	}
	for _, digest := range detail.LatestPolicyDecisionRefs {
		digest = strings.TrimSpace(digest)
		if digest == "" {
			continue
		}
		labels = append(labels, "policy "+shortIdentity(digest))
	}
	if detail.AuditSummary.FindingCount > 0 || detail.AuditSummary.ErrorFindingCount > 0 || detail.AuditSummary.WarningFindingCount > 0 {
		labels = append(labels, fmt.Sprintf("audit findings (%d)", detail.AuditSummary.FindingCount))
	}
	return labels
}

func runAuditReferenceCount(detail *brokerapi.RunDetail) int {
	return len(runAuditReferenceLabels(detail))
}

func renderRuntimeAttestationTruthfulnessCue(state map[string]any) string {
	attestationPosture, reasons := attestationPostureFromState(state)
	verificationSucceeded, _ := state["attestation_verification_succeeded"].(bool)
	sessionBindingPresent, _ := state["session_binding_present"].(bool)
	attestationEvidencePresent, _ := state["attestation_evidence_present"].(bool)
	supportedRuntimeSatisfied, _ := state["supported_runtime_requirements_satisfied"].(bool)

	currentEvidence := "launch-only evidence"
	switch {
	case verificationSucceeded:
		currentEvidence = "post-handshake verification succeeded"
	case attestationEvidencePresent:
		currentEvidence = "post-handshake evidence collected but not yet supportable"
	case sessionBindingPresent:
		currentEvidence = "secure session bound without verified attestation"
	}

	if supportedRuntimeSatisfied && attestationPosture == "valid" {
		return currentEvidence + "; supported attested posture earned from verified post-handshake evidence"
	}
	if len(reasons) > 0 {
		return currentEvidence + "; beta attested story still gated by post-handshake verification; reasons=" + strings.Join(reasons, ",")
	}
	return currentEvidence + "; beta attested story still gated by post-handshake verification"
}

func attestationPostureFromState(state map[string]any) (string, []string) {
	posture, _ := state["attestation_posture"].(string)
	reasonsAny, ok := state["attestation_reason_codes"].([]any)
	if !ok {
		reasons, _ := state["attestation_reason_codes"].([]string)
		return posture, reasons
	}
	reasons := make([]string, 0, len(reasonsAny))
	for _, value := range reasonsAny {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			reasons = append(reasons, s)
		}
	}
	return posture, reasons
}

func runInspectorContentKind(presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	return inspectorContentStructured
}

func runRouteCopyActions(detail *brokerapi.RunDetail) []routeCopyAction {
	if detail == nil {
		return nil
	}
	summary := detail.Summary
	sessionID := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "session_id"))
	auditRefs := append([]string{}, detail.ActiveManifestHashes...)
	auditRefs = append(auditRefs, detail.LatestPolicyDecisionRefs...)
	raw := compactLines(
		fmt.Sprintf("run_id=%s", summary.RunID),
		fmt.Sprintf("workspace_id=%s", summary.WorkspaceID),
		fmt.Sprintf("session_id=%s", valueOrNA(sessionID)),
		fmt.Sprintf("lifecycle=%s", summary.LifecycleState),
		fmt.Sprintf("backend_kind=%s", summary.BackendKind),
	)
	return compactCopyActions([]routeCopyAction{
		{ID: "run_id", Label: "run id", Text: summary.RunID},
		{ID: "workspace_id", Label: "workspace id", Text: summary.WorkspaceID},
		{ID: "session_id", Label: "session id", Text: sessionID},
		{ID: "approval_ids", Label: "linked approval ids", Text: strings.Join(detail.PendingApprovalIDs, "\n")},
		{ID: "audit_refs", Label: "audit refs", Text: strings.Join(auditRefs, "\n")},
		{ID: "raw_block", Label: "raw block", Text: raw},
	})
}

func (m runsRouteModel) activeSummary() brokerapi.RunSummary {
	if m.active != nil {
		return m.active.Summary
	}
	if len(m.runs) > 0 {
		return m.runs[m.selected]
	}
	return brokerapi.RunSummary{}
}

func (m *runsRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "run", ID: "none"}, inspectorContentStructured, "run details", "")
		return
	}
	summary := m.active.Summary
	presentation := normalizePresentationMode(m.presentation)
	waitingStages := pendingApprovalStageCount(m.active.StageSummaries)
	waitingRoles := waitingRoleCount(m.active.RoleSummaries)
	content := runInspectorContent(summary, m.active, waitingStages, waitingRoles, buildRunEvidenceLinks(m.active), presentation)
	kind := runInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "run", ID: strings.TrimSpace(summary.RunID), WorkspaceID: strings.TrimSpace(summary.WorkspaceID)}
	m.detailDoc.SetDocument(ref, kind, "run details", content)
}
