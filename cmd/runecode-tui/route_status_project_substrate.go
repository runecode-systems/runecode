package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project setup adoption: status=%s mutation=none", valueOrNA(resp.Adoption.Status))
		if len(resp.Adoption.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(resp.Adoption.ReasonCodes)
		}
		return projectSubstrateActionResultMsg{status: status}
	}
}

func (m statusRouteModel) projectSubstrateInitPreviewCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.ProjectSubstrateInitPreview(ctx)
		if err != nil {
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project setup init preview: status=%s mutation=preview_only handle=%s", valueOrNA(resp.Preview.Status), projectSubstrateHandleDisplay(resp.Preview.PreviewToken))
		if len(resp.Preview.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(resp.Preview.ReasonCodes)
		}
		if strings.TrimSpace(resp.Preview.PreviewToken) == "" {
			status += " next=reload_or_retry_preview"
		}
		return projectSubstrateActionResultMsg{status: status}
	}
}

func (m statusRouteModel) projectSubstrateInitApplyCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		preview := m.data.project.InitPreview
		token := strings.TrimSpace(preview.PreviewToken)
		if token == "" {
			return projectSubstrateActionResultMsg{err: fmt.Errorf("project setup init apply unavailable: preview handle missing; reload or run init preview first")}
		}
		applyResp, err := m.client.ProjectSubstrateInitApply(ctx, token)
		if err != nil {
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project setup init apply: status=%s preview_status=%s mutation=applied handle=%s", valueOrNA(applyResp.ApplyResult.Status), valueOrNA(preview.Status), projectSubstrateHandleDisplay(token))
		if len(applyResp.ApplyResult.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(applyResp.ApplyResult.ReasonCodes)
		}
		status += " next=reload_validation_status"
		return projectSubstrateActionResultMsg{status: status}
	}
}

func (m statusRouteModel) projectSubstrateUpgradePreviewCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.ProjectSubstrateUpgradePreview(ctx)
		if err != nil {
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project setup upgrade preview: status=%s mutation=preview_only digest=%s", valueOrNA(resp.Preview.Status), projectSubstrateHandleDisplay(resp.Preview.PreviewDigest))
		if len(resp.Preview.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(resp.Preview.ReasonCodes)
		}
		if strings.TrimSpace(resp.Preview.PreviewDigest) == "" {
			status += " next=reload_or_retry_preview"
		}
		return projectSubstrateActionResultMsg{status: status}
	}
}

func (m statusRouteModel) projectSubstrateUpgradeApplyCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		preview := m.data.project.UpgradePreview
		digest := strings.TrimSpace(preview.PreviewDigest)
		if digest == "" {
			return projectSubstrateActionResultMsg{err: fmt.Errorf("project setup upgrade apply unavailable: preview digest missing; reload or run upgrade preview first")}
		}
		applyResp, err := m.client.ProjectSubstrateUpgradeApply(ctx, digest)
		if err != nil {
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project setup upgrade apply: status=%s preview_status=%s mutation=applied digest=%s", valueOrNA(applyResp.ApplyResult.Status), valueOrNA(preview.Status), projectSubstrateHandleDisplay(digest))
		if len(applyResp.ApplyResult.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(applyResp.ApplyResult.ReasonCodes)
		}
		status += " next=reload_validation_status"
		return projectSubstrateActionResultMsg{status: status}
	}
}

func projectSubstrateHandleDisplay(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return projectSubstrateHandleAcquiredText
}
