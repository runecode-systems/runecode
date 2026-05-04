package auditd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func loadOfflineSignerEvidence(bundle offlineBundleSnapshot) ([]trustpolicy.AuditSignerEvidenceReference, bool) {
	obj, ok := offlineObjectByFamily(bundle, "signer_evidence")
	if !ok {
		return nil, false
	}
	refs := []trustpolicy.AuditSignerEvidenceReference{}
	if err := json.Unmarshal(obj.content, &refs); err != nil {
		return nil, false
	}
	return refs, true
}

func loadOfflineStoragePosture(bundle offlineBundleSnapshot) (*trustpolicy.AuditStoragePostureEvidence, bool) {
	obj, ok := offlineObjectByFamily(bundle, "storage_posture")
	if !ok {
		return nil, false
	}
	posture := trustpolicy.AuditStoragePostureEvidence{}
	if err := json.Unmarshal(obj.content, &posture); err != nil {
		return nil, false
	}
	return &posture, true
}

func loadOfflineExternalAnchorEvidence(bundle offlineBundleSnapshot) ([]trustpolicy.ExternalAnchorEvidencePayload, []trustpolicy.Digest, error) {
	evidence, err := loadOfflineExternalAnchorEvidenceObjects(bundle)
	if err != nil {
		return nil, nil, err
	}
	sidecars, err := loadOfflineExternalAnchorSidecarDigests(bundle)
	if err != nil {
		return nil, nil, err
	}
	return evidence, sidecars, nil
}

func loadOfflineExternalAnchorEvidenceObjects(bundle offlineBundleSnapshot) ([]trustpolicy.ExternalAnchorEvidencePayload, error) {
	objects := offlineBundleObjectsByFamily(bundle.manifest.IncludedObjects, "external_anchor_evidence")
	out := make([]trustpolicy.ExternalAnchorEvidencePayload, 0, len(objects))
	for i := range objects {
		raw, ok := bundle.objects[objects[i].Path]
		if !ok {
			continue
		}
		rec := trustpolicy.ExternalAnchorEvidencePayload{}
		if err := json.Unmarshal(raw.content, &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadOfflineExternalAnchorSidecarDigests(bundle offlineBundleSnapshot) ([]trustpolicy.Digest, error) {
	objects := offlineBundleObjectsByFamily(bundle.manifest.IncludedObjects, "external_anchor_sidecar")
	out := make([]trustpolicy.Digest, 0, len(objects))
	for i := range objects {
		d, err := digestFromIdentity(objects[i].Digest)
		if err != nil {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func compareVerificationConclusions(expected trustpolicy.AuditVerificationReportPayload, got trustpolicy.AuditVerificationReportPayload) string {
	expected = normalizeOfflineRecomputedReportComparison(expected)
	got = normalizeOfflineRecomputedReportComparison(got)
	if verificationStatusesMismatch(expected, got) {
		return fmt.Sprintf(
			"recomputed verification conclusion mismatch: expected statuses=%s/%s/%s/%s degraded=%t cryptographically_valid=%t historically_admissible=%t got=%s/%s/%s/%s degraded=%t cryptographically_valid=%t historically_admissible=%t",
			expected.IntegrityStatus,
			expected.AnchoringStatus,
			expected.StoragePostureStatus,
			expected.SegmentLifecycleStatus,
			expected.CurrentlyDegraded,
			expected.CryptographicallyValid,
			expected.HistoricallyAdmissible,
			got.IntegrityStatus,
			got.AnchoringStatus,
			got.StoragePostureStatus,
			got.SegmentLifecycleStatus,
			got.CurrentlyDegraded,
			got.CryptographicallyValid,
			got.HistoricallyAdmissible,
		)
	}
	if !sameStringSet(expected.HardFailures, got.HardFailures) {
		return fmt.Sprintf("recomputed hard_failures mismatch: expected=%v got=%v", normalizeStringList(expected.HardFailures), normalizeStringList(got.HardFailures))
	}
	if !sameStringSet(expected.DegradedReasons, got.DegradedReasons) {
		return fmt.Sprintf("recomputed degraded_reasons mismatch: expected=%v got=%v", normalizeStringList(expected.DegradedReasons), normalizeStringList(got.DegradedReasons))
	}
	return ""
}

func normalizeOfflineRecomputedReportComparison(report trustpolicy.AuditVerificationReportPayload) trustpolicy.AuditVerificationReportPayload {
	hadOnlyExportIncomplete := report.IntegrityStatus == trustpolicy.AuditVerificationStatusDegraded && hasOnlyDroppedReason(report.DegradedReasons, trustpolicy.AuditVerificationReasonEvidenceExportIncomplete)
	report.DegradedReasons = filteredReasonCodes(report.DegradedReasons, trustpolicy.AuditVerificationReasonEvidenceExportIncomplete)
	report.Findings = filteredFindings(report.Findings, trustpolicy.AuditVerificationReasonEvidenceExportIncomplete)
	report.CurrentlyDegraded = len(report.DegradedReasons) > 0
	if hadOnlyExportIncomplete || (report.IntegrityStatus == trustpolicy.AuditVerificationStatusDegraded && len(report.DegradedReasons) == 0) {
		report.IntegrityStatus = trustpolicy.AuditVerificationStatusOK
	}
	return report
}

func hasOnlyDroppedReason(codes []string, drop string) bool {
	hasDrop := false
	for i := range codes {
		if strings.TrimSpace(codes[i]) == drop {
			hasDrop = true
			continue
		}
		return false
	}
	return hasDrop
}

func filteredReasonCodes(codes []string, drop string) []string {
	out := make([]string, 0, len(codes))
	for i := range codes {
		if strings.TrimSpace(codes[i]) == drop {
			continue
		}
		out = append(out, codes[i])
	}
	return out
}

func filteredFindings(findings []trustpolicy.AuditVerificationFinding, drop string) []trustpolicy.AuditVerificationFinding {
	out := make([]trustpolicy.AuditVerificationFinding, 0, len(findings))
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) == drop {
			continue
		}
		out = append(out, findings[i])
	}
	return out
}

func verificationStatusesMismatch(expected trustpolicy.AuditVerificationReportPayload, got trustpolicy.AuditVerificationReportPayload) bool {
	return expected.IntegrityStatus != got.IntegrityStatus ||
		expected.AnchoringStatus != got.AnchoringStatus ||
		expected.StoragePostureStatus != got.StoragePostureStatus ||
		expected.SegmentLifecycleStatus != got.SegmentLifecycleStatus ||
		expected.CurrentlyDegraded != got.CurrentlyDegraded ||
		expected.CryptographicallyValid != got.CryptographicallyValid ||
		expected.HistoricallyAdmissible != got.HistoricallyAdmissible
}

func sameStringSet(a []string, b []string) bool {
	na := normalizeStringList(a)
	nb := normalizeStringList(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func offlineBundleObjectsByFamily(objects []AuditEvidenceBundleIncludedObject, family string) []AuditEvidenceBundleIncludedObject {
	out := []AuditEvidenceBundleIncludedObject{}
	for i := range objects {
		if strings.TrimSpace(objects[i].ObjectFamily) == strings.TrimSpace(family) {
			out = append(out, objects[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Digest < out[j].Digest })
	return out
}

func offlineObjectByFamily(bundle offlineBundleSnapshot, family string) (offlineBundleObject, bool) {
	objects := offlineBundleObjectsByFamily(bundle.manifest.IncludedObjects, family)
	if len(objects) == 0 {
		return offlineBundleObject{}, false
	}
	obj, ok := bundle.objects[objects[0].Path]
	return obj, ok
}

func offlineObjectByFamilyAndDigest(bundle offlineBundleSnapshot, family string, digest string) (offlineBundleObject, bool) {
	for i := range bundle.manifest.IncludedObjects {
		obj := bundle.manifest.IncludedObjects[i]
		if strings.TrimSpace(obj.ObjectFamily) != strings.TrimSpace(family) {
			continue
		}
		if strings.TrimSpace(obj.Digest) != strings.TrimSpace(digest) {
			continue
		}
		raw, ok := bundle.objects[strings.TrimSpace(obj.Path)]
		return raw, ok
	}
	return offlineBundleObject{}, false
}
