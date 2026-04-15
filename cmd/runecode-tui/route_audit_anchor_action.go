package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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
		if !isSealReference(ref.ReferenceKind, ref.Relation) {
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

func isSealReference(kind string, relation string) bool {
	if strings.ToLower(strings.TrimSpace(kind)) != "audit_record" {
		return false
	}
	return strings.ToLower(strings.TrimSpace(relation)) == "subject_segment_seal"
}

func (m auditRouteModel) anchorSealCmd(sealDigest string, exportCopy bool) tea.Cmd {
	return func() tea.Msg {
		parsed := parseDigestIdentity(sealDigest)
		if _, err := parsed.Identity(); err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: fmt.Errorf("invalid seal digest")}
		}
		ctx, cancel := withLoadTimeout()
		defer cancel()
		preflight, err := m.client.AuditAnchorPreflightGet(ctx, brokerapi.AuditAnchorPreflightGetRequest{})
		if err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: err}
		}
		if err := validateAuditAnchorPreflightForTUI(preflight, parsed); err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: err}
		}
		presenceResp, err := m.client.AuditAnchorPresenceGet(ctx, brokerapi.AuditAnchorPresenceGetRequest{SealDigest: parsed})
		if err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: err}
		}
		if auditAnchorPresenceAttestationRequired(strings.TrimSpace(presenceResp.PresenceMode)) && presenceResp.PresenceAttestation == nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: fmt.Errorf("presence attestation unavailable")}
		}
		resp, err := m.client.AuditAnchorSegment(ctx, brokerapi.AuditAnchorSegmentRequest{SealDigest: parsed, PresenceAttestation: presenceResp.PresenceAttestation, ExportReceiptCopy: exportCopy})
		if err != nil {
			return auditAnchorCompletedMsg{sealDigest: sealDigest, err: err}
		}
		return auditAnchorCompletedMsg{response: &resp, sealDigest: sealDigest}
	}
}

func auditAnchorPresenceAttestationRequired(mode string) bool {
	mode = strings.TrimSpace(mode)
	return mode == "os_confirmation" || mode == "hardware_touch"
}

func validateAuditAnchorPreflightForTUI(preflight brokerapi.AuditAnchorPreflightGetResponse, requested trustpolicy.Digest) error {
	if !preflight.SignerReadiness.Ready {
		if code := strings.TrimSpace(preflight.SignerReadiness.ReasonCode); code != "" {
			return fmt.Errorf("%s", code)
		}
		return fmt.Errorf("anchor signer unavailable")
	}
	if !preflight.VerifierReadiness.Ready {
		if code := strings.TrimSpace(preflight.VerifierReadiness.ReasonCode); code != "" {
			return fmt.Errorf("%s", code)
		}
		return fmt.Errorf("audit verifier unavailable")
	}
	if preflight.PresenceRequirements.Required && !preflight.PresenceRequirements.AttestationReady {
		if code := strings.TrimSpace(preflight.PresenceRequirements.ReasonCode); code != "" {
			return fmt.Errorf("%s", code)
		}
		return fmt.Errorf("presence attestation unavailable")
	}
	if preflight.LatestAnchorableSeal == nil {
		return fmt.Errorf("no latest anchorable seal")
	}
	want, err := requested.Identity()
	if err != nil {
		return fmt.Errorf("invalid requested seal digest")
	}
	got, err := preflight.LatestAnchorableSeal.SealDigest.Identity()
	if err != nil {
		return fmt.Errorf("invalid latest anchorable seal digest")
	}
	if got != want {
		return fmt.Errorf("selected seal is not latest anchorable seal")
	}
	return nil
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
