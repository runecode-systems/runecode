package main

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestAuditAnchorFailureReasonPrefersFailureCode(t *testing.T) {
	resp := brokerapi.AuditAnchorSegmentResponse{
		FailureCode:    "external_anchor_deferred_or_unavailable",
		FailureMessage: "external anchor confirmation is deferred",
	}
	if got := auditAnchorFailureReason(resp); got != "external_anchor_deferred_or_unavailable" {
		t.Fatalf("auditAnchorFailureReason() = %q, want external_anchor_deferred_or_unavailable", got)
	}
}

func TestAuditAnchorFailureReasonFallsBackToFailureMessage(t *testing.T) {
	resp := brokerapi.AuditAnchorSegmentResponse{FailureMessage: "external anchor confirmation is deferred"}
	if got := auditAnchorFailureReason(resp); got != "external anchor confirmation is deferred" {
		t.Fatalf("auditAnchorFailureReason() = %q, want external anchor confirmation is deferred", got)
	}
}
