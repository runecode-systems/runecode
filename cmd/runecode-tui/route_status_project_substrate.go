package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (m statusRouteModel) beginProjectSubstrateAction(key string) (routeModel, tea.Cmd) {
	actionMsg, cmd := m.projectSubstrateActionForKey(key)
	if cmd == nil {
		return m, nil
	}
	if m.loading || m.changing || m.actioning {
		return m, nil
	}
	m.actioning = true
	m.projectSubstrateActionCard = nil
	m.actionMsg = actionMsg
	m.errText = ""
	m.status = ""
	return m, cmd
}

func (m statusRouteModel) projectSubstrateActionForKey(key string) (string, tea.Cmd) {
	switch key {
	case "a":
		return "Inspecting compatible existing project setup for broker-owned adoption...", m.projectSubstrateAdoptCmd()
	case "i":
		return "Loading broker-owned init preview for project setup...", m.projectSubstrateInitPreviewCmd()
	case "I":
		return "Applying broker-owned init preview for project setup...", m.projectSubstrateInitApplyCmd()
	case "u":
		return "Loading broker-owned upgrade preview for project setup...", m.projectSubstrateUpgradePreviewCmd()
	case "U":
		return "Applying broker-owned upgrade preview for project setup...", m.projectSubstrateUpgradeApplyCmd()
	default:
		return "", nil
	}
}

func (m statusRouteModel) projectSubstrateAdoptCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.ProjectSubstrateAdopt(ctx)
		if err != nil {
			return projectSubstrateActionResultMsg{
				err:        err,
				status:     "Project setup adoption was not completed.",
				actionCard: projectSubstrateActionFailureCard("Compatible adoption", "RuneCode could not confirm compatible existing project setup.", m.data.project, safeUIErrorText(err), "Adoption is read-only recognition only. Reload or inspect current broker posture before trying again."),
			}
		}
		return projectSubstrateActionResultMsg{
			status:     fmt.Sprintf("Project setup adoption refreshed: status=%s", valueOrNA(resp.Adoption.Status)),
			adoption:   &resp,
			actionCard: projectSubstrateAdoptionCard(resp.Adoption, m.data.project),
		}
	}
}

func (m statusRouteModel) projectSubstrateInitPreviewCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.ProjectSubstrateInitPreview(ctx)
		if err != nil {
			return projectSubstrateActionResultMsg{
				err:        err,
				status:     "Project setup init preview was not loaded.",
				actionCard: projectSubstrateActionFailureCard("Init preview", "RuneCode could not load the broker-owned init preview.", m.data.project, safeUIErrorText(err), "Reload or retry init preview before any init apply."),
			}
		}
		return projectSubstrateActionResultMsg{
			status:      fmt.Sprintf("Project setup init preview refreshed: status=%s", valueOrNA(resp.Preview.Status)),
			initPreview: &resp,
			actionCard:  projectSubstrateInitPreviewCard(resp.Preview, m.data.project),
		}
	}
}

func (m statusRouteModel) projectSubstrateInitApplyCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		preview := m.data.project.InitPreview
		token := strings.TrimSpace(preview.PreviewToken)
		if token == "" {
			return projectSubstrateActionResultMsg{
				err:        fmt.Errorf("project setup init apply unavailable: preview handle missing; reload or run init preview first"),
				status:     "Project setup init apply is unavailable.",
				actionCard: projectSubstrateApplyUnavailableCard("Init apply", "Init apply is unavailable because no preview handle is currently published.", m.data.project, "Run init preview first, then apply only after reviewing the planned mutation."),
			}
		}
		// Preview freshness is broker-enforced; stale or superseded tokens must fail closed in ProjectSubstrateInitApply.
		applyResp, err := m.client.ProjectSubstrateInitApply(ctx, token)
		if err != nil {
			return projectSubstrateActionResultMsg{
				err:        err,
				status:     "Project setup init apply failed.",
				actionCard: projectSubstrateActionFailureCard("Init apply", "RuneCode could not apply the broker-owned init preview; the preview handle may be stale or superseded.", m.data.project, safeUIErrorText(err), "Rerun init preview before trying init apply again."),
			}
		}
		return projectSubstrateActionResultMsg{
			status:     fmt.Sprintf("Project setup init apply completed: status=%s", valueOrNA(applyResp.ApplyResult.Status)),
			actionCard: projectSubstrateInitApplyCard(preview, applyResp.ApplyResult, m.data.project),
			reload:     true,
		}
	}
}

func (m statusRouteModel) projectSubstrateUpgradePreviewCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.ProjectSubstrateUpgradePreview(ctx)
		if err != nil {
			return projectSubstrateActionResultMsg{
				err:        err,
				status:     "Project setup upgrade preview was not loaded.",
				actionCard: projectSubstrateActionFailureCard("Upgrade preview", "RuneCode could not load the broker-owned upgrade preview.", m.data.project, safeUIErrorText(err), "Reload or retry upgrade preview before any upgrade apply."),
			}
		}
		return projectSubstrateActionResultMsg{
			status:         fmt.Sprintf("Project setup upgrade preview refreshed: status=%s", valueOrNA(resp.Preview.Status)),
			upgradePreview: &resp,
			actionCard:     projectSubstrateUpgradePreviewCard(resp.Preview, m.data.project),
		}
	}
}

func (m statusRouteModel) projectSubstrateUpgradeApplyCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		preview := m.data.project.UpgradePreview
		digest := strings.TrimSpace(preview.PreviewDigest)
		if digest == "" {
			return projectSubstrateActionResultMsg{
				err:        fmt.Errorf("project setup upgrade apply unavailable: preview digest missing; reload or run upgrade preview first"),
				status:     "Project setup upgrade apply is unavailable.",
				actionCard: projectSubstrateApplyUnavailableCard("Upgrade apply", "Upgrade apply is unavailable because no preview digest is currently published.", m.data.project, "Run upgrade preview first, then apply only after reviewing the planned mutation."),
			}
		}
		// Preview freshness is broker-enforced; stale or superseded digests must fail closed in ProjectSubstrateUpgradeApply.
		applyResp, err := m.client.ProjectSubstrateUpgradeApply(ctx, digest)
		if err != nil {
			return projectSubstrateActionResultMsg{
				err:        err,
				status:     "Project setup upgrade apply failed.",
				actionCard: projectSubstrateActionFailureCard("Upgrade apply", "RuneCode could not apply the broker-owned upgrade preview; the preview digest may be stale or superseded.", m.data.project, safeUIErrorText(err), "Rerun upgrade preview before trying upgrade apply again."),
			}
		}
		return projectSubstrateActionResultMsg{
			status:     fmt.Sprintf("Project setup upgrade apply completed: status=%s", valueOrNA(applyResp.ApplyResult.Status)),
			actionCard: projectSubstrateUpgradeApplyCard(preview, applyResp.ApplyResult, m.data.project),
			reload:     true,
		}
	}
}

func projectSubstrateHandleDisplay(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return projectSubstrateHandleAcquiredText
}

func projectSubstrateAdoptionCard(adoption projectsubstrate.AdoptionResult, posture brokerapi.ProjectSubstratePostureGetResponse) *stateCardSpec {
	return &stateCardSpec{
		State:       routeLoadStateReady,
		Title:       "Compatible adoption",
		Message:     "RuneCode found an existing compatible setup it can recognize without changing repository files.",
		Reason:      projectSubstrateActionReason(adoption.ReasonCodes, posture),
		NextAction:  "If managed work is still blocked, use init or upgrade preview next; otherwise continue with normal work.",
		ShortcutCue: "i preview init • u preview upgrade • r reload",
	}
}

func projectSubstrateInitPreviewCard(preview projectsubstrate.InitPreview, posture brokerapi.ProjectSubstratePostureGetResponse) *stateCardSpec {
	return previewCard("Init preview", "Handle", valueOrNA(preview.Status), projectSubstrateHandleDisplay(preview.PreviewToken), projectSubstratePreviewMutationLine(preview.PreviewToken, preview.Status), projectSubstrateActionReason(preview.ReasonCodes, posture), preview.RequiredFollowUp, "Review this preview, then press I only if you want the broker-owned init mutation applied.")
}

func projectSubstrateUpgradePreviewCard(preview projectsubstrate.UpgradePreview, posture brokerapi.ProjectSubstratePostureGetResponse) *stateCardSpec {
	return previewCard("Upgrade preview", "Digest", valueOrNA(preview.Status), projectSubstrateHandleDisplay(preview.PreviewDigest), projectSubstratePreviewMutationLine(preview.PreviewDigest, preview.Status), projectSubstrateActionReason(preview.ReasonCodes, posture), preview.RequiredFollowUp, "Review this preview, then press U only if you want the broker-owned upgrade mutation applied.")
}

func previewCard(title, identityLabel, status, handle, mutation, reason string, followUp []string, next string) *stateCardSpec {
	message := fmt.Sprintf("Preview is %s. %s %s is %s.", projectSubstratePreviewStatusSummary(status, handle), strings.TrimSpace(mutationSentence(mutation)), identityLabel, handle)
	if handle == "n/a" {
		message = fmt.Sprintf("Preview is %s. %s No %s is currently available.", projectSubstratePreviewStatusSummary(status, handle), strings.TrimSpace(mutationSentence(mutation)), strings.ToLower(identityLabel))
	}
	return &stateCardSpec{State: previewCardState(handle, status), Title: title, Message: message, Reason: projectSubstrateReasonWithFollowUp(reason, followUp), NextAction: next, ShortcutCue: "I apply init • U apply upgrade • r reload"}
}

func projectSubstrateInitApplyCard(preview projectsubstrate.InitPreview, result projectsubstrate.InitApplyResult, posture brokerapi.ProjectSubstratePostureGetResponse) *stateCardSpec {
	return &stateCardSpec{State: routeLoadStateCompleted, Title: "Init apply", Message: "RuneCode applied the broker-owned project setup initialization and is now rechecking the resulting posture.", Reason: fmt.Sprintf("Preview was %s • %s", projectSubstratePreviewStatusSummary(preview.Status, preview.PreviewToken), projectSubstrateActionReason(result.ReasonCodes, posture)), NextAction: "RuneCode is reloading broker posture now to validate the resulting project setup before normal work resumes."}
}

func projectSubstrateUpgradeApplyCard(preview projectsubstrate.UpgradePreview, result projectsubstrate.UpgradeApplyResult, posture brokerapi.ProjectSubstratePostureGetResponse) *stateCardSpec {
	return &stateCardSpec{State: routeLoadStateCompleted, Title: "Upgrade apply", Message: "RuneCode applied the broker-owned project setup upgrade and is now rechecking the resulting posture.", Reason: fmt.Sprintf("Preview was %s • %s", projectSubstratePreviewStatusSummary(preview.Status, preview.PreviewDigest), projectSubstrateActionReason(result.ReasonCodes, posture)), NextAction: "RuneCode is reloading broker posture now to validate the resulting project setup before normal work resumes."}
}

func projectSubstrateApplyUnavailableCard(title, message string, posture brokerapi.ProjectSubstratePostureGetResponse, next string) *stateCardSpec {
	return &stateCardSpec{State: routeLoadStateBlocked, Title: title, Message: message, Reason: projectSubstrateBlockingReason(posture), NextAction: next, ShortcutCue: "i preview init • u preview upgrade • r reload"}
}

func projectSubstrateActionFailureCard(title, message string, posture brokerapi.ProjectSubstratePostureGetResponse, failure string, next string) *stateCardSpec {
	reason := projectSubstrateBlockingReason(posture)
	if strings.TrimSpace(failure) != "" {
		reason = reason + " • broker_response=" + sanitizeUIText(failure)
	}
	return &stateCardSpec{State: routeLoadStateError, Title: title, Message: message, Reason: reason, NextAction: next, ShortcutCue: "r reload"}
}

func projectSubstrateValidationFailureCard(posture brokerapi.ProjectSubstratePostureGetResponse, failure string) *stateCardSpec {
	reason := projectSubstrateBlockingReason(posture)
	if strings.TrimSpace(failure) != "" {
		reason = reason + " • validation_refresh=" + sanitizeUIText(failure)
	}
	return &stateCardSpec{State: routeLoadStateError, Title: "Post-apply validation", Message: "Project setup apply finished, but refreshed validation status is currently unavailable.", Reason: reason, NextAction: "Press r to retry validation refresh, then confirm managed-operation and setup posture before continuing."}
}

func projectSubstratePreviewMutationLine(handle, status string) string {
	if strings.TrimSpace(handle) == "" {
		return "No changes are available yet because the broker has not published a preview"
	}
	if strings.EqualFold(strings.TrimSpace(status), "ready_for_apply") {
		return "No changes have been made yet; apply is available if you explicitly choose it"
	}
	return "No changes have been made yet; this preview stays advisory until you explicitly apply it"
}

func previewCardState(handle, status string) routeLoadState {
	if strings.TrimSpace(handle) == "" {
		return routeLoadStateBlocked
	}
	if strings.EqualFold(strings.TrimSpace(status), "ready_for_apply") {
		return routeLoadStateReady
	}
	return routeLoadStateWaiting
}

func projectSubstrateReasonWithFollowUp(reason string, followUp []string) string {
	if len(followUp) == 0 {
		return reason
	}
	return reason + " • next broker steps=" + joinCSV(followUp)
}

func projectSubstrateActionReason(reasons []string, posture brokerapi.ProjectSubstratePostureGetResponse) string {
	parts := []string{}
	if len(reasons) > 0 {
		parts = append(parts, "broker reasons="+joinCSV(reasons))
	}
	if blocking := strings.TrimSpace(projectSubstrateBlockingReason(posture)); blocking != "" {
		parts = append(parts, blocking)
	}
	return strings.Join(parts, " • ")
}

func projectSubstrateBlockingReason(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	if explanation := strings.TrimSpace(posture.BlockedExplanation); explanation != "" {
		return "normal work blocked: " + sanitizeUIText(explanation)
	}
	if len(posture.PostureSummary.BlockedReasonCodes) > 0 {
		return "normal work blocked: " + joinCSV(posture.PostureSummary.BlockedReasonCodes)
	}
	if len(posture.PostureSummary.ReasonCodes) > 0 {
		return "broker guidance: " + joinCSV(posture.PostureSummary.ReasonCodes)
	}
	return "normal work blocked: not reported"
}

func mutationSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.HasSuffix(text, ".") {
		return text
	}
	return text + "."
}
