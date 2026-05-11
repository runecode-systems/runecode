package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func runEvidenceSummary(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return "approvals 0 • artifact classes 0 • active manifests 0 • policy refs 0"
	}
	parts := []string{fmt.Sprintf("approvals %d", len(detail.PendingApprovalIDs)), fmt.Sprintf("artifact classes %d", len(detail.ArtifactCountsByClass)), fmt.Sprintf("active manifests %d", len(detail.ActiveManifestHashes)), fmt.Sprintf("policy refs %d", len(detail.LatestPolicyDecisionRefs))}
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
		if digest != "" {
			labels = append(labels, "manifest "+shortIdentity(digest))
		}
	}
	for _, digest := range detail.LatestPolicyDecisionRefs {
		digest = strings.TrimSpace(digest)
		if digest != "" {
			labels = append(labels, "policy "+shortIdentity(digest))
		}
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
		return currentEvidence + "; supported runtime evidence is present, but beta attested posture still waits for explicit post-handshake gating integration"
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
		return sanitizeAttestationPostureAndReasons(posture, reasons)
	}
	reasons := make([]string, 0, len(reasonsAny))
	for _, value := range reasonsAny {
		if s, ok := value.(string); ok {
			reasons = append(reasons, s)
		}
	}
	return sanitizeAttestationPostureAndReasons(posture, reasons)
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
	raw := compactLines(fmt.Sprintf("run_id=%s", summary.RunID), fmt.Sprintf("workspace_id=%s", summary.WorkspaceID), fmt.Sprintf("session_id=%s", valueOrNA(sessionID)), fmt.Sprintf("lifecycle=%s", summary.LifecycleState), fmt.Sprintf("backend_kind=%s", summary.BackendKind))
	return compactCopyActions([]routeCopyAction{{ID: "run_id", Label: "run id", Text: summary.RunID}, {ID: "workspace_id", Label: "workspace id", Text: summary.WorkspaceID}, {ID: "session_id", Label: "session id", Text: sessionID}, {ID: "approval_ids", Label: "linked approval ids", Text: strings.Join(detail.PendingApprovalIDs, "\n")}, {ID: "audit_refs", Label: "audit refs", Text: strings.Join(auditRefs, "\n")}, {ID: "raw_block", Label: "raw block", Text: raw}})
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
