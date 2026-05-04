package auditd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func verifyOfflineBundleRecomputation(bundle offlineBundleSnapshot) []AuditEvidenceBundleOfflineFinding {
	findings := []AuditEvidenceBundleOfflineFinding{}
	reportObjects := offlineBundleObjectsByFamily(bundle.manifest.IncludedObjects, "audit_verification_report")
	for i := range reportObjects {
		if finding, hasFinding := verifyOfflineReportObject(bundle, reportObjects[i]); hasFinding {
			findings = append(findings, finding)
		}
	}
	return findings
}

func verifyOfflineReportObject(bundle offlineBundleSnapshot, reportObject AuditEvidenceBundleIncludedObject) (AuditEvidenceBundleOfflineFinding, bool) {
	reportPayload, reportDigest, err := decodeOfflineVerificationReport(bundle, reportObject)
	if err != nil {
		return offlineRecomputeFinding("verification_recompute_report_invalid", "error", err.Error(), reportObject.Path, strings.TrimSpace(reportObject.Digest)), true
	}
	recomputeInput, missing, err := offlineRecomputeInput(bundle, reportPayload)
	if err != nil {
		return offlineRecomputeFinding("verification_recompute_unavailable", "warning", err.Error(), reportObject.Path, reportDigest), true
	}
	if len(missing) > 0 {
		message := "offline recomputation omitted due to missing verification inputs: " + strings.Join(missing, ", ")
		return offlineRecomputeFinding("verification_recompute_inputs_missing", "warning", message, reportObject.Path, reportDigest), true
	}
	recomputed, err := trustpolicy.VerifyAuditEvidence(recomputeInput)
	if err != nil {
		return offlineRecomputeFinding("verification_recompute_failed", "error", fmt.Sprintf("offline recomputation failed: %v", err), reportObject.Path, reportDigest), true
	}
	if mismatch := compareVerificationConclusions(reportPayload, recomputed); mismatch != "" {
		return offlineRecomputeFinding("verification_recompute_mismatch", "error", mismatch, reportObject.Path, reportDigest), true
	}
	return AuditEvidenceBundleOfflineFinding{}, false
}

func offlineRecomputeFinding(code string, severity string, message string, objectPath string, digest string) AuditEvidenceBundleOfflineFinding {
	return AuditEvidenceBundleOfflineFinding{
		Code:       code,
		Severity:   severity,
		Message:    message,
		ObjectPath: objectPath,
		Digest:     digest,
	}
}

func decodeOfflineVerificationReport(bundle offlineBundleSnapshot, reportObject AuditEvidenceBundleIncludedObject) (trustpolicy.AuditVerificationReportPayload, string, error) {
	raw, ok := bundle.objects[strings.TrimSpace(reportObject.Path)]
	if !ok {
		return trustpolicy.AuditVerificationReportPayload{}, "", fmt.Errorf("verification report object missing from archive")
	}
	report := trustpolicy.AuditVerificationReportPayload{}
	if err := json.Unmarshal(raw.content, &report); err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, "", fmt.Errorf("verification report decode failed: %w", err)
	}
	digest, err := canonicalDigest(report)
	if err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, "", err
	}
	identity, err := digest.Identity()
	if err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, "", err
	}
	return report, identity, nil
}

func offlineRecomputeInput(bundle offlineBundleSnapshot, report trustpolicy.AuditVerificationReportPayload) (trustpolicy.AuditVerificationInput, []string, error) {
	segmentID := strings.TrimSpace(report.VerificationScope.LastSegmentID)
	if segmentID == "" {
		return trustpolicy.AuditVerificationInput{}, []string{"report.verification_scope.last_segment_id"}, nil
	}
	inputs, missing, err := loadOfflineRequiredRecomputeInputs(bundle, segmentID)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, nil, err
	}
	if len(missing) > 0 {
		return trustpolicy.AuditVerificationInput{}, normalizeStringList(missing), nil
	}
	receipts, err := loadOfflineReceiptsForSeal(bundle, inputs.sealDigest)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, nil, err
	}
	knownSeals, err := offlineKnownSealDigests(bundle.manifest.SealReferences)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, nil, err
	}
	externalEvidence, externalSidecars, err := loadOfflineExternalAnchorEvidence(bundle)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, nil, err
	}
	return buildOfflineAuditVerificationInput(bundle, report, inputs, receipts, knownSeals, externalEvidence, externalSidecars), nil, nil
}

func buildOfflineAuditVerificationInput(
	bundle offlineBundleSnapshot,
	report trustpolicy.AuditVerificationReportPayload,
	inputs offlineRequiredRecomputeInputs,
	receipts []trustpolicy.SignedObjectEnvelope,
	knownSeals []trustpolicy.Digest,
	externalEvidence []trustpolicy.ExternalAnchorEvidencePayload,
	externalSidecars []trustpolicy.Digest,
) trustpolicy.AuditVerificationInput {
	signerEvidence, _ := loadOfflineSignerEvidence(bundle)
	storagePosture, _ := loadOfflineStoragePosture(bundle)
	return trustpolicy.AuditVerificationInput{
		Scope:                   report.VerificationScope,
		Segment:                 inputs.segment,
		RawFramedSegmentBytes:   inputs.segmentRaw,
		SegmentSealEnvelope:     inputs.sealEnvelope,
		KnownSealDigests:        knownSeals,
		ReceiptEnvelopes:        receipts,
		VerifierRecords:         inputs.verifierRecords,
		EventContractCatalog:    inputs.eventCatalog,
		SignerEvidence:          signerEvidence,
		StoragePostureEvidence:  storagePosture,
		ExternalAnchorEvidence:  externalEvidence,
		ExternalAnchorSidecars:  externalSidecars,
		ExternalAnchorTargetSet: []trustpolicy.ExternalAnchorVerificationTarget{},
		Now:                     bundleVerificationNowUTC(bundle),
	}
}

func bundleVerificationNowUTC(bundle offlineBundleSnapshot) time.Time {
	if !bundle.verifiedAt.IsZero() {
		return bundle.verifiedAt.UTC()
	}
	return time.Now().UTC()
}
