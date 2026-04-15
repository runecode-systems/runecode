package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (m auditRouteModel) anchorSelectedOrLatestSeal() (routeModel, tea.Cmd) {
	sealDigest, ok := m.resolveAnchorSealDigest()
	if !ok {
		m.statusText = "Anchor action: no sealed segment digest available from verification/audit data; no broker call made."
		return m, nil
	}
	m.anchoring = true
	m.errText = ""
	m.statusText = fmt.Sprintf("Anchor action: requesting broker anchor for seal=%s export_copy=%t", sealDigest, m.exportCopy)
	return m, m.anchorSealCmd(sealDigest, m.exportCopy)
}

func (m auditRouteModel) resolveAnchorSealDigest() (string, bool) {
	if digest, ok := selectedTimelineSealDigest(m.timeline, m.selected); ok {
		return digest, true
	}
	if digest, ok := activeRecordSealDigest(m.active); ok {
		return digest, true
	}
	if digest, ok := latestVerificationSealDigest(m.verify); ok {
		return digest, true
	}
	return "", false
}

func selectedTimelineSealDigest(timeline []brokerapi.AuditTimelineViewEntry, selected int) (string, bool) {
	if selected >= 0 && selected < len(timeline) {
		if digest, ok := sealDigestFromReferences(timeline[selected].LinkedReferences); ok {
			return digest, true
		}
	}
	for _, entry := range timeline {
		if digest, ok := sealDigestFromReferences(entry.LinkedReferences); ok {
			return digest, true
		}
	}
	return "", false
}

func activeRecordSealDigest(active *brokerapi.AuditRecordGetResponse) (string, bool) {
	if active == nil {
		return "", false
	}
	return sealDigestFromReferences(active.Record.LinkedReferences)
}

func latestVerificationSealDigest(verify *brokerapi.AuditVerificationGetResponse) (string, bool) {
	if verify == nil {
		return "", false
	}
	for _, view := range verify.Views {
		if view.Receipt == nil {
			continue
		}
		if strings.TrimSpace(view.Receipt.SubjectFamily) != "audit_segment_seal" {
			continue
		}
		identity, err := view.Receipt.SubjectDigest.Identity()
		if err != nil {
			continue
		}
		return identity, true
	}
	return "", false
}

func sealDigestFromReferences(refs []brokerapi.AuditRecordLinkedReference) (string, bool) {
	for _, ref := range refs {
		if !isSealReferenceKind(ref.ReferenceKind) {
			continue
		}
		digest := parseDigestIdentity(ref.ReferenceID)
		identity, err := digest.Identity()
		if err == nil {
			return identity, true
		}
	}
	return "", false
}

func isSealReferenceKind(kind string) bool {
	norm := strings.ToLower(strings.TrimSpace(kind))
	switch norm {
	case "audit_segment_seal_digest", "segment_seal_digest":
		return true
	default:
		return false
	}
}

func (m auditRouteModel) anchorSealCmd(sealDigest string, exportCopy bool) tea.Cmd {
	return func() tea.Msg {
		parsed := parseDigestIdentity(sealDigest)
		if _, err := parsed.Identity(); err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: fmt.Errorf("invalid seal digest")}
		}
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.AuditAnchorSegment(ctx, brokerapi.AuditAnchorSegmentRequest{SealDigest: parsed, ExportReceiptCopy: exportCopy})
		if err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: err}
		}
		return auditAnchorCompletedMsg{response: &resp, sealDigest: sealDigest}
	}
}

func renderAuditAnchorActionSummary(m auditRouteModel) string {
	if m.anchoring {
		return tableHeader("Anchor action") + " request in flight via broker audit_anchor_segment"
	}
	exportCopy := "off"
	if m.exportCopy {
		exportCopy = "on"
	}
	status := m.statusText
	if strings.TrimSpace(status) == "" {
		status = "idle"
	}
	return fmt.Sprintf("%s export_copy=%s status=%s", tableHeader("Anchor action"), exportCopy, status)
}
