package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditRouteShowsPagedTimelineAndVerificationReasonCodes(t *testing.T) {
	model := newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAudit})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Audit safety strip",
		"UNANCHORED_OR_DEGRADED_AUDIT",
		"Timeline paging: page=1 entries=1 has_next=yes",
		"anchoring=degraded (unanchored/degraded)",
		"Verification findings (machine-readable):",
		"code=anchor_receipt_missing severity=warning dimension=anchoring",
		"degraded_reason_codes=",
		"posture=degraded reasons=anchor_receipt_missing",
	)

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected next-page load command")
	}
	updated, _ = updated.Update(cmd())
	view = updated.View(120, 40, focusContent)
	if !strings.Contains(view, "page_cursor=page-2") {
		t.Fatalf("expected second page cursor in view, got %q", view)
	}
	if !strings.Contains(view, "posture=failed reasons=anchor_receipt_invalid") {
		t.Fatalf("expected failed anchoring posture on page 2, got %q", view)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected typed record detail load command")
	}
	updated, _ = updated.Update(cmd())
	view = updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Verification posture: degraded (unanchored/degraded)") {
		t.Fatalf("expected record inspector posture rendering, got %q", view)
	}
}

func TestRenderAuditSafetyAlertStripUsesVerifierAnchoringStatusOnly(t *testing.T) {
	strip := renderAuditSafetyAlertStrip(&brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
		AnchoringStatus:   "ok",
		IntegrityStatus:   "ok",
		CurrentlyDegraded: true,
	}})
	if !strings.Contains(strip, "UNANCHORED_OR_DEGRADED_AUDIT") {
		t.Fatalf("expected degraded badge when currently_degraded=true, got %q", strip)
	}

	strip = renderAuditSafetyAlertStrip(&brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
		AnchoringStatus:   "unanchored",
		IntegrityStatus:   "ok",
		CurrentlyDegraded: false,
	}})
	if strings.Contains(strip, "UNANCHORED_OR_DEGRADED_AUDIT") {
		t.Fatalf("unexpected degraded badge for legacy non-schema anchoring status, got %q", strip)
	}
}

func TestStatusRouteExplainsDegradedSubsystemPosture(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Runtime/audit readiness strip",
		"RUNTIME_POSTURE_AUTH_UNAVAILABLE",
		"AUDIT_STORAGE_NOMINAL",
		"Broker ready=true local_only=true",
		"Subsystem posture:",
		"Diagnostics: degraded subsystems=verifier_material=missing",
		"Version posture: product=0.1.0",
		"Protocol posture: bundle=0.9.0",
	)
}

type auditAnchorProbeClient struct {
	fakeBrokerClient
	anchorResp  brokerapi.AuditAnchorSegmentResponse
	anchorErr   error
	lastAnchor  *brokerapi.AuditAnchorSegmentRequest
	includeSeal bool
}

func (c *auditAnchorProbeClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	if !c.includeSeal {
		return c.fakeBrokerClient.AuditTimeline(ctx, limit, cursor)
	}
	_ = ctx
	_ = limit
	_ = cursor
	entry := brokerapi.AuditTimelineViewEntry{
		RecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		EventType:    "audit_receipt",
		Summary:      "anchor receipt recorded",
		LinkedReferences: []brokerapi.AuditRecordLinkedReference{
			{ReferenceKind: "audit_record", ReferenceID: "sha256:" + strings.Repeat("e", 64), Relation: "subject_segment_seal"},
		},
	}
	return brokerapi.AuditTimelineResponse{Views: []brokerapi.AuditTimelineViewEntry{entry}}, nil
}

func (c *auditAnchorProbeClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	_ = ctx
	copyReq := req
	c.lastAnchor = &copyReq
	if c.anchorErr != nil {
		return brokerapi.AuditAnchorSegmentResponse{}, c.anchorErr
	}
	if strings.TrimSpace(c.anchorResp.AnchoringStatus) == "" {
		return c.fakeBrokerClient.AuditAnchorSegment(ctx, req)
	}
	return c.anchorResp, nil
}

func TestAuditRouteAnchorActionDispatchesToBrokerAndRendersSuccess(t *testing.T) {
	client := &auditAnchorProbeClient{includeSeal: true}
	spy := newRecordingBrokerClient(client)
	model := newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAudit})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("expected anchor command")
	}
	updated, _ = updated.Update(cmd())

	if client.lastAnchor == nil {
		t.Fatal("expected AuditAnchorSegment request capture")
	}
	if client.lastAnchor.ExportReceiptCopy {
		t.Fatalf("expected export copy default false, got true")
	}
	gotSeal, err := client.lastAnchor.SealDigest.Identity()
	if err != nil {
		t.Fatalf("expected valid seal digest in request: %v", err)
	}
	if gotSeal != "sha256:"+strings.Repeat("e", 64) {
		t.Fatalf("expected selected/latest seal digest, got %q", gotSeal)
	}
	if !containsCall(spy.Calls(), "AuditAnchorSegment") {
		t.Fatalf("expected AuditAnchorSegment call, got %v", spy.Calls())
	}
	if !containsCall(spy.Calls(), "AuditAnchorPresenceGet") {
		t.Fatalf("expected AuditAnchorPresenceGet call, got %v", spy.Calls())
	}
	if client.lastAnchor.PresenceAttestation == nil {
		t.Fatal("expected broker-owned presence attestation on anchor request")
	}
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Anchor action",
		"Anchor action: ok",
		"receipt=sha256:",
		"export_copy=off",
	)
}

func TestAuditRouteAnchorActionRendersFailureReason(t *testing.T) {
	client := &auditAnchorProbeClient{
		includeSeal: true,
		anchorResp: brokerapi.AuditAnchorSegmentResponse{
			AnchoringStatus: "failed",
			FailureCode:     "approval_required",
		},
	}
	model := newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, client)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAudit})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("expected anchor command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Anchor action: failed",
		"reason=approval_required",
	)
}

func TestAuditRouteAnchorActionWithoutDigestSkipsBrokerCall(t *testing.T) {
	base := &auditAnchorProbeClient{includeSeal: false, anchorErr: errors.New("should not be called")}
	spy := newRecordingBrokerClient(base)
	model := newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAudit})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("expected no anchor command when no seal digest is available")
	}
	if containsCall(spy.Calls(), "AuditAnchorSegment") {
		t.Fatalf("expected AuditAnchorSegment not called, got %v", spy.Calls())
	}

	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"no sealed segment digest available",
		"no broker call made",
	)
}
