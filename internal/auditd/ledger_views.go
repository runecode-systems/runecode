package auditd

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) operationalViewsForSegmentLocked(segmentID string, limit int) ([]trustpolicy.AuditOperationalView, error) {
	if segmentID == "" {
		return []trustpolicy.AuditOperationalView{}, nil
	}
	segment, err := l.loadSegment(segmentID)
	if err != nil {
		return nil, err
	}
	start := viewStartIndex(limit, len(segment.Frames))
	views := make([]trustpolicy.AuditOperationalView, 0, len(segment.Frames)-start)
	for idx := start; idx < len(segment.Frames); idx++ {
		view, err := frameOperationalView(segment.Frames[idx])
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func viewStartIndex(limit int, total int) int {
	if limit <= 0 || limit > total {
		limit = total
	}
	start := total - limit
	if start < 0 {
		start = 0
	}
	return start
}

func frameOperationalView(frame trustpolicy.AuditSegmentRecordFrame) (trustpolicy.AuditOperationalView, error) {
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return trustpolicy.AuditOperationalView{}, err
	}
	if err := verifyFrameRecordDigest(frame, envelope); err != nil {
		return trustpolicy.AuditOperationalView{}, err
	}
	view, err := trustpolicy.BuildDefaultOperationalAuditView(envelope)
	if err != nil {
		return trustpolicy.AuditOperationalView{}, fmt.Errorf("build operational view: %w", err)
	}
	return view, nil
}

func verifyFrameRecordDigest(frame trustpolicy.AuditSegmentRecordFrame, envelope trustpolicy.SignedObjectEnvelope) error {
	persisted, err := frame.RecordDigest.Identity()
	if err != nil {
		return fmt.Errorf("invalid persisted frame record_digest: %w", err)
	}
	computedDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return fmt.Errorf("compute frame record digest: %w", err)
	}
	computed, _ := computedDigest.Identity()
	if computed != persisted {
		return fmt.Errorf("frame record_digest mismatch: persisted %q computed %q", persisted, computed)
	}
	return nil
}

func (l *Ledger) LatestVerificationSummaryAndViews(limit int) (trustpolicy.DerivedRunAuditVerificationSummary, []trustpolicy.AuditOperationalView, trustpolicy.AuditVerificationReportPayload, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	report, err := l.latestVerificationReportLocked()
	if err != nil {
		return trustpolicy.DerivedRunAuditVerificationSummary{}, nil, trustpolicy.AuditVerificationReportPayload{}, err
	}
	summary, err := trustpolicy.BuildDerivedRunAuditVerificationSummary(report)
	if err != nil {
		return trustpolicy.DerivedRunAuditVerificationSummary{}, nil, trustpolicy.AuditVerificationReportPayload{}, err
	}
	views, err := l.operationalViewsForSegmentLocked(report.VerificationScope.LastSegmentID, limit)
	if err != nil {
		return trustpolicy.DerivedRunAuditVerificationSummary{}, nil, trustpolicy.AuditVerificationReportPayload{}, err
	}
	return summary, views, report, nil
}
