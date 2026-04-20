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
		return "Submitting project substrate adoption through broker local API...", m.projectSubstrateAdoptCmd()
	case "i":
		return "Loading project substrate init preview through broker local API...", m.projectSubstrateInitPreviewCmd()
	case "I":
		return "Applying project substrate init with broker preview token...", m.projectSubstrateInitApplyCmd()
	case "u":
		return "Loading project substrate upgrade preview through broker local API...", m.projectSubstrateUpgradePreviewCmd()
	case "U":
		return "Applying project substrate upgrade with broker preview digest...", m.projectSubstrateUpgradeApplyCmd()
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
		status := fmt.Sprintf("Project substrate adopt status=%s", valueOrNA(resp.Adoption.Status))
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
		status := fmt.Sprintf("Project substrate init preview status=%s token=%s", valueOrNA(resp.Preview.Status), projectSubstrateHandleDisplay(resp.Preview.PreviewToken))
		if len(resp.Preview.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(resp.Preview.ReasonCodes)
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
			return projectSubstrateActionResultMsg{err: fmt.Errorf("project substrate init preview token unavailable; reload or run preview first")}
		}
		applyResp, err := m.client.ProjectSubstrateInitApply(ctx, token)
		if err != nil {
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project substrate init apply status=%s preview_status=%s token=%s", valueOrNA(applyResp.ApplyResult.Status), valueOrNA(preview.Status), projectSubstrateHandleDisplay(token))
		if len(applyResp.ApplyResult.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(applyResp.ApplyResult.ReasonCodes)
		}
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
		status := fmt.Sprintf("Project substrate upgrade preview status=%s digest=%s", valueOrNA(resp.Preview.Status), projectSubstrateHandleDisplay(resp.Preview.PreviewDigest))
		if len(resp.Preview.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(resp.Preview.ReasonCodes)
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
			return projectSubstrateActionResultMsg{err: fmt.Errorf("project substrate upgrade preview digest unavailable; reload or run preview first")}
		}
		applyResp, err := m.client.ProjectSubstrateUpgradeApply(ctx, digest)
		if err != nil {
			return projectSubstrateActionResultMsg{err: err}
		}
		status := fmt.Sprintf("Project substrate upgrade apply status=%s preview_status=%s digest=%s", valueOrNA(applyResp.ApplyResult.Status), valueOrNA(preview.Status), projectSubstrateHandleDisplay(digest))
		if len(applyResp.ApplyResult.ReasonCodes) > 0 {
			status += " reasons=" + joinCSV(applyResp.ApplyResult.ReasonCodes)
		}
		return projectSubstrateActionResultMsg{status: status}
	}
}

func projectSubstrateHandleDisplay(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return projectSubstrateHandleAcquiredText
}
