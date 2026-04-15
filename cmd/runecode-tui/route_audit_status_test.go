package main

import (
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
