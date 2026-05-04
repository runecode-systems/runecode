package brokerapi

import (
	"log"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) maybePersistRuntimeSummaryReceipt() error {
	sealDigest, receipts, ok := s.runtimeSummarySealReceipts()
	if !ok {
		return nil
	}
	counts, runDigests := countRuntimeProvenanceReceipts(receipts)
	approvalCount, approvalSupport, runIDs := runtimeApprovalConsumptionForRunDigests(runDigests, s.listApprovals())
	boundaryCount, boundarySupport := runtimeBoundaryCrossingsForRuns(runIDs, s.List())
	return s.persistRuntimeSummaryReceipts(sealDigest, receipts, counts, approvalCount, approvalSupport, runIDs, boundaryCount, boundarySupport)
}

func (s *Service) runtimeSummarySealReceipts() (trustpolicy.Digest, []trustpolicy.SignedObjectEnvelope, bool) {
	if s == nil || s.auditLedger == nil {
		return trustpolicy.Digest{}, nil, false
	}
	segmentID, sealDigest, err := s.auditLedger.LatestAnchorableSeal()
	if err != nil || strings.TrimSpace(segmentID) == "" {
		return trustpolicy.Digest{}, nil, false
	}
	receipts, err := s.auditLedger.ReceiptsForSealDigest(sealDigest)
	if err != nil {
		return trustpolicy.Digest{}, nil, false
	}
	return sealDigest, receipts, true
}

func (s *Service) persistRuntimeSummaryReceipts(sealDigest trustpolicy.Digest, receipts []trustpolicy.SignedObjectEnvelope, counts runtimeProvenanceCounts, approvalCount int64, approvalSupport string, runIDs []string, boundaryCount int64, boundarySupport string) error {
	if !runtimeSummaryKindPresent(receipts, "runtime_summary") {
		payload := runtimeSummaryReceiptPayloadMap(counts, approvalCount, boundaryCount, boundarySupport)
		if err := s.persistRuntimeProvenanceReceipt(sealDigest, "runtime_summary", trustpolicy.AuditSegmentAnchoringSubjectSeal, "runecode.protocol.audit.receipt.runtime_summary.v0", payload); err != nil {
			log.Printf("brokerapi: runtime summary receipt persistence failed: %v", err)
			return err
		}
	}
	if !runtimeSummaryKindPresent(receipts, "degraded_posture_summary") {
		payload := degradedPostureSummaryPayload(runIDs, s.RunStatuses(), s.listApprovals())
		if err := s.persistRuntimeProvenanceReceipt(sealDigest, "degraded_posture_summary", trustpolicy.AuditSegmentAnchoringSubjectSeal, "runecode.protocol.audit.receipt.degraded_posture_summary.v0", payload); err != nil {
			log.Printf("brokerapi: degraded posture summary receipt persistence failed: %v", err)
			return err
		}
	}
	if !runtimeSummaryKindPresent(receipts, "negative_capability_summary") {
		payload := negativeCapabilitySummaryPayloadMap(counts, approvalCount, approvalSupport, boundaryCount, boundarySupport)
		if err := s.persistRuntimeProvenanceReceipt(sealDigest, "negative_capability_summary", trustpolicy.AuditSegmentAnchoringSubjectSeal, "runecode.protocol.audit.receipt.negative_capability_summary.v0", payload); err != nil {
			log.Printf("brokerapi: negative capability summary receipt persistence failed: %v", err)
			return err
		}
	}
	return nil
}

func runtimeSummaryReceiptPayloadMap(counts runtimeProvenanceCounts, approvalCount int64, boundaryCount int64, boundarySupport string) map[string]any {
	return map[string]any{
		"summary_scope_kind":           runtimeSummaryScopeRun,
		"provider_invocation_count":    counts.authorizedProviderCount,
		"secret_lease_issue_count":     counts.leaseIssueCount,
		"secret_lease_revoke_count":    counts.leaseRevokeCount,
		"network_egress_count":         counts.authorizedProviderCount,
		"no_provider_invocation":       counts.authorizedProviderCount == 0,
		"no_secret_lease_issued":       counts.leaseIssueCount == 0,
		"approval_consumption_count":   approvalCount,
		"no_approval_consumed":         approvalCount == 0,
		"boundary_crossing_count":      boundaryCount,
		"no_artifact_crossed_boundary": boundarySupport == "explicit" && boundaryCount == 0,
		"boundary_route":               runtimeSummaryBoundaryRoute,
		"boundary_crossing_support":    boundarySupport,
	}
}

func negativeCapabilitySummaryPayloadMap(counts runtimeProvenanceCounts, approvalCount int64, approvalSupport string, boundaryCount int64, boundarySupport string) map[string]any {
	return map[string]any{
		"summary_scope_kind":                    runtimeSummaryScopeRun,
		"no_secret_lease_issued":                counts.leaseIssueCount == 0,
		"no_network_egress":                     counts.authorizedProviderCount == 0,
		"no_approval_consumed":                  approvalCount == 0,
		"no_artifact_crossed_boundary":          boundarySupport == "explicit" && boundaryCount == 0,
		"boundary_route":                        runtimeSummaryBoundaryRoute,
		"secret_lease_evidence_support":         "explicit",
		"network_egress_evidence_support":       "explicit",
		"approval_consumption_evidence_support": approvalSupport,
		"boundary_crossing_evidence_support":    boundarySupport,
	}
}
