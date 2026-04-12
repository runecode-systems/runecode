package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

func (s *Service) projectAuditRecordDetail(recordIdentity string, envelope trustpolicy.SignedObjectEnvelope) (AuditRecordDetail, error) {
	view, viewDigest, detail, err := baseProjectedAuditRecordDetail(recordIdentity, envelope)
	if err != nil {
		return AuditRecordDetail{}, err
	}
	if err := projectAuditRecordFamilyDetail(&detail, envelope, view); err != nil {
		return AuditRecordDetail{}, err
	}
	reasons, posture := s.deriveRecordVerificationPosture(viewDigest)
	if posture != nil {
		detail.VerificationPosture = posture
		detail.LinkedReferences = append(detail.LinkedReferences, verificationReasonRefs(reasons)...)
	}
	detail.LinkedReferences = dedupeAuditRecordReferences(detail.LinkedReferences)
	return detail, nil
}
