package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func containsReasonCodeForAnchorTest(codes []string, code string) bool {
	for idx := range codes {
		if codes[idx] == code {
			return true
		}
	}
	return false
}

func assertAnchorFailureAuthoritativePosture(t *testing.T, service *Service, sealDigest trustpolicy.Digest) {
	t.Helper()
	verification := mustAuditVerificationGetForAnchorFailure(t, service, "req-anchor-failure-posture-verification")
	if verification.Report.AnchoringStatus != trustpolicy.AuditVerificationStatusFailed {
		t.Fatalf("verification report anchoring_status = %q, want %q (hard_failures=%v findings=%d)", verification.Report.AnchoringStatus, trustpolicy.AuditVerificationStatusFailed, verification.Report.HardFailures, len(verification.Report.Findings))
	}
	if !containsReasonCodeForAnchorTest(verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("verification report hard_failures = %v, want include %q", verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
	if !hasAnchorFailureFindingForSeal(verification.Report.Findings, sealDigest) {
		sealDigestID := mustDigestIdentityForAnchorTest(sealDigest)
		t.Fatalf("verification report findings missing anchor failure for seal digest %q", sealDigestID)
	}

	timeline := mustAuditTimelineForAnchorFailure(t, service, "req-anchor-failure-posture-timeline")
	if !timelineHasFailedAnchorReason(timeline.Views) {
		t.Fatalf("timeline views missing failed verification posture with reason %q", trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func mustAuditVerificationGetForAnchorFailure(t *testing.T, service *Service, requestID string) AuditVerificationGetResponse {
	t.Helper()
	verification, errResp := service.HandleAuditVerificationGet(context.Background(), AuditVerificationGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditVerificationGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		ViewLimit:     20,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditVerificationGet returned error: %+v", errResp)
	}
	return verification
}

func mustAuditTimelineForAnchorFailure(t *testing.T, service *Service, requestID string) AuditTimelineResponse {
	t.Helper()
	timeline, timelineErr := service.HandleAuditTimeline(context.Background(), AuditTimelineRequest{
		SchemaID:      "runecode.protocol.v0.AuditTimelineRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Limit:         20,
		Order:         "operational_seq_desc",
	}, RequestContext{})
	if timelineErr != nil {
		t.Fatalf("HandleAuditTimeline returned error: %+v", timelineErr)
	}
	return timeline
}

func hasAnchorFailureFindingForSeal(findings []trustpolicy.AuditVerificationFinding, sealDigest trustpolicy.Digest) bool {
	sealDigestID := mustDigestIdentityForAnchorTest(sealDigest)
	for _, finding := range findings {
		if finding.Code != trustpolicy.AuditVerificationReasonAnchorReceiptInvalid {
			continue
		}
		if finding.SubjectRecordDigest == nil || mustDigestIdentityForAnchorTest(*finding.SubjectRecordDigest) != sealDigestID {
			continue
		}
		return true
	}
	return false
}

func timelineHasFailedAnchorReason(views []AuditTimelineViewEntry) bool {
	for _, view := range views {
		if view.VerificationPosture == nil {
			continue
		}
		if view.VerificationPosture.Status != trustpolicy.AuditVerificationStatusFailed {
			continue
		}
		if containsReasonCodeForAnchorTest(view.VerificationPosture.ReasonCodes, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid) {
			return true
		}
	}
	return false
}

func assertAnchorFailureDoesNotMutateAuthoritativePosture(t *testing.T, service *Service) {
	t.Helper()
	verification, errResp := service.HandleAuditVerificationGet(context.Background(), AuditVerificationGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditVerificationGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-no-posture-mutation",
		ViewLimit:     20,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditVerificationGet returned error: %+v", errResp)
	}
	if verification.Report.AnchoringStatus == trustpolicy.AuditVerificationStatusFailed {
		t.Fatalf("verification report anchoring_status = %q, did not want failed", verification.Report.AnchoringStatus)
	}
	if containsReasonCodeForAnchorTest(verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("verification report hard_failures = %v, did not want %q", verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
}
