package auditd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func offlineBundleObjectDigestIdentity(family string, payload []byte) (string, error) {
	switch strings.TrimSpace(family) {
	case "audit_segment":
		return digestIdentityFromSegmentPayload(payload)
	case "audit_segment_seal", "audit_receipt":
		return digestIdentityFromSignedEnvelopePayload(payload)
	case "audit_verification_report", "external_anchor_evidence", "external_anchor_sidecar", "event_contract_catalog", "verifier_record_set", "signer_evidence", "storage_posture":
		return digestIdentityFromCanonicalPayload(payload)
	default:
		return "", fmt.Errorf("object family %q has no offline digest verifier in this lane", family)
	}
}

func digestIdentityFromSegmentPayload(payload []byte) (string, error) {
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := json.Unmarshal(payload, &segment); err != nil {
		return "", err
	}
	b, err := json.Marshal(segment)
	if err != nil {
		return "", err
	}
	if _, err := jsoncanonicalizer.Transform(b); err != nil {
		return "", err
	}
	d, err := canonicalDigest(segment)
	if err != nil {
		return "", err
	}
	return d.Identity()
}

func digestIdentityFromSignedEnvelopePayload(payload []byte) (string, error) {
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", err
	}
	d, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return "", err
	}
	return d.Identity()
}

func digestIdentityFromCanonicalPayload(payload []byte) (string, error) {
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return "", err
	}
	d, err := canonicalDigest(value)
	if err != nil {
		return "", err
	}
	return d.Identity()
}

func offlineBundleVerificationStatus(findings []AuditEvidenceBundleOfflineFinding, reports []AuditEvidenceBundleOfflineReportPosture) string {
	failed, degraded := offlineBundleVerificationFlags(findings, reports)
	if failed {
		return "failed"
	}
	if degraded {
		return "degraded"
	}
	return "ok"
}

func offlineBundleVerificationFlags(findings []AuditEvidenceBundleOfflineFinding, reports []AuditEvidenceBundleOfflineReportPosture) (bool, bool) {
	failed, degraded := offlineBundleFindingFlags(findings)
	reportFailed, reportDegraded := offlineBundleReportFlags(reports)
	return failed || reportFailed, degraded || reportDegraded
}

func offlineBundleFindingFlags(findings []AuditEvidenceBundleOfflineFinding) (bool, bool) {
	failed := false
	degraded := false
	for i := range findings {
		if findings[i].Severity == "error" {
			failed = true
		}
		if isOfflineBundleDegradedFinding(findings[i].Code) {
			degraded = true
		}
	}
	return failed, degraded
}

func isOfflineBundleDegradedFinding(code string) bool {
	switch code {
	case "verification_report_degraded_posture", "verification_report_missing", "verification_evidence_redacted", "verification_recompute_inputs_missing", "verification_recompute_unavailable":
		return true
	default:
		return false
	}
}

func offlineBundleReportFlags(reports []AuditEvidenceBundleOfflineReportPosture) (bool, bool) {
	failed := false
	degraded := false
	for i := range reports {
		if reports[i].CurrentlyDegraded || len(reports[i].DegradedReasons) > 0 {
			degraded = true
		}
		if offlineBundleReportFailed(reports[i]) {
			failed = true
		}
	}
	return failed, degraded
}

func offlineBundleReportFailed(report AuditEvidenceBundleOfflineReportPosture) bool {
	return len(report.HardFailures) > 0 || report.IntegrityStatus == trustpolicy.AuditVerificationStatusFailed || report.AnchoringStatus == trustpolicy.AuditVerificationStatusFailed || report.StoragePostureStatus == trustpolicy.AuditVerificationStatusFailed || report.SegmentLifecycleStatus == trustpolicy.AuditVerificationStatusFailed
}
