package auditd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	auditEvidenceBundleOfflineVerificationSchemaID      = "runecode.protocol.v0.AuditEvidenceBundleOfflineVerification"
	auditEvidenceBundleOfflineVerificationSchemaVersion = "0.1.0"
)

func (l *Ledger) OfflineVerifyEvidenceBundle(reader io.Reader, archiveFormat string) (AuditEvidenceBundleOfflineVerification, error) {
	if l == nil {
		return AuditEvidenceBundleOfflineVerification{}, fmt.Errorf("ledger is required")
	}
	format := normalizeEvidenceBundleArchiveFormat(archiveFormat)
	if format != auditEvidenceBundleArchiveFormatTar {
		return AuditEvidenceBundleOfflineVerification{}, fmt.Errorf("unsupported archive format %q", format)
	}
	if reader == nil {
		return AuditEvidenceBundleOfflineVerification{}, fmt.Errorf("bundle reader is required")
	}
	bundle, err := loadAuditEvidenceBundleFromTar(reader)
	if err != nil {
		return AuditEvidenceBundleOfflineVerification{}, err
	}
	verifiedAt := l.nowFn().UTC().Format(time.RFC3339)
	findings, reportPosture := verifyAuditEvidenceBundleContents(bundle)
	status := offlineBundleVerificationStatus(findings, reportPosture)
	return AuditEvidenceBundleOfflineVerification{
		SchemaID:            auditEvidenceBundleOfflineVerificationSchemaID,
		SchemaVersion:       auditEvidenceBundleOfflineVerificationSchemaVersion,
		VerifiedAt:          verifiedAt,
		ArchiveFormat:       format,
		ManifestDigest:      bundle.manifestDigestIdentity,
		BundleID:            bundle.manifest.BundleID,
		ExportProfile:       bundle.manifest.ExportProfile,
		Scope:               bundle.manifest.Scope,
		VerifierIdentity:    bundle.manifest.VerifierIdentity,
		TrustRootDigests:    bundle.manifest.TrustRootDigests,
		VerificationStatus:  status,
		Findings:            findings,
		VerificationReports: reportPosture,
	}, nil
}

func verifyAuditEvidenceBundleContents(bundle offlineBundleSnapshot) ([]AuditEvidenceBundleOfflineFinding, []AuditEvidenceBundleOfflineReportPosture) {
	findings := make([]AuditEvidenceBundleOfflineFinding, 0, 8)
	findings = append(findings, verifyOfflineBundleManifest(bundle)...)
	reports := make([]AuditEvidenceBundleOfflineReportPosture, 0, 4)
	objectFindings, reportPostures := verifyOfflineBundleIncludedObjects(bundle)
	findings = append(findings, objectFindings...)
	reports = append(reports, reportPostures...)
	findings = append(findings, verifyOfflineBundleReportCoverage(bundle, reports)...)
	findings = append(findings, verifyOfflineBundleRecomputation(bundle)...)
	sortOfflineFindings(findings)
	sort.Slice(reports, func(i, j int) bool { return reports[i].Digest < reports[j].Digest })
	return findings, reports
}

func verifyOfflineBundleManifest(bundle offlineBundleSnapshot) []AuditEvidenceBundleOfflineFinding {
	findings := make([]AuditEvidenceBundleOfflineFinding, 0, 4)
	if !bytes.Equal(bundle.manifestCanonicalJSON, bundle.objects["manifest.json"].content) {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "manifest_not_canonical", Severity: "warning", Message: "manifest.json is not canonicalized; verification used canonical projection", ObjectPath: "manifest.json", Digest: bundle.manifestDigestIdentity})
	}
	if strings.TrimSpace(bundle.manifest.VerifierIdentity.KeyIDValue) == "" {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verifier_identity_missing", Severity: "warning", Message: "manifest verifier_identity.key_id_value is missing"})
	}
	if len(bundle.manifest.TrustRootDigests) == 0 {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "trust_root_identity_missing", Severity: "warning", Message: "manifest trust_root_digests is empty"})
	}
	for i := range bundle.manifest.TrustRootDigests {
		if _, err := digestFromIdentity(bundle.manifest.TrustRootDigests[i]); err != nil {
			findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "trust_root_identity_invalid", Severity: "error", Message: fmt.Sprintf("manifest trust_root_digests[%d] invalid: %v", i, err), Digest: strings.TrimSpace(bundle.manifest.TrustRootDigests[i])})
		}
	}
	return findings
}

func verifyOfflineBundleIncludedObjects(bundle offlineBundleSnapshot) ([]AuditEvidenceBundleOfflineFinding, []AuditEvidenceBundleOfflineReportPosture) {
	findings := make([]AuditEvidenceBundleOfflineFinding, 0, len(bundle.manifest.IncludedObjects))
	reports := make([]AuditEvidenceBundleOfflineReportPosture, 0, 4)
	for i := range bundle.manifest.IncludedObjects {
		objectFindings, report, ok := verifyOfflineBundleIncludedObject(bundle, bundle.manifest.IncludedObjects[i])
		findings = append(findings, objectFindings...)
		if ok {
			reports = append(reports, report)
		}
	}
	return findings, reports
}

func verifyOfflineBundleIncludedObject(bundle offlineBundleSnapshot, obj AuditEvidenceBundleIncludedObject) ([]AuditEvidenceBundleOfflineFinding, AuditEvidenceBundleOfflineReportPosture, bool) {
	path := strings.TrimSpace(obj.Path)
	rel, ok := bundle.objects[path]
	if !ok {
		findings := []AuditEvidenceBundleOfflineFinding{{Code: "bundle_object_missing", Severity: "error", Message: "manifest listed object is missing from archive", ObjectPath: path, Digest: strings.TrimSpace(obj.Digest)}}
		if obj.ObjectFamily == "audit_verification_report" {
			findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verification_report_missing", Severity: "error", Message: "bundle does not include audit verification report evidence", ObjectPath: path, Digest: strings.TrimSpace(obj.Digest)})
		}
		return findings, AuditEvidenceBundleOfflineReportPosture{}, false
	}
	findings := verifyOfflineBundleObjectContent(obj, rel)
	if obj.ObjectFamily != "audit_verification_report" {
		return findings, AuditEvidenceBundleOfflineReportPosture{}, false
	}
	report, postureFindings, ok := verifyOfflineReportPosture(rel.content, strings.TrimSpace(obj.Digest))
	return append(findings, postureFindings...), report, ok
}

func verifyOfflineBundleObjectContent(obj AuditEvidenceBundleIncludedObject, rel offlineBundleObject) []AuditEvidenceBundleOfflineFinding {
	findings := make([]AuditEvidenceBundleOfflineFinding, 0, 2)
	path := strings.TrimSpace(obj.Path)
	if int64(len(rel.content)) != obj.ByteLength {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "bundle_object_size_mismatch", Severity: "error", Message: "manifest byte_length does not match archive object size", ObjectPath: path, Digest: strings.TrimSpace(obj.Digest)})
	}
	computed, err := offlineBundleObjectDigestIdentity(obj.ObjectFamily, rel.content)
	if err != nil {
		return append(findings, AuditEvidenceBundleOfflineFinding{Code: "bundle_object_digest_unverifiable", Severity: "warning", Message: err.Error(), ObjectPath: path, Digest: strings.TrimSpace(obj.Digest)})
	}
	if strings.TrimSpace(obj.Digest) != computed {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "bundle_object_digest_mismatch", Severity: "error", Message: "manifest digest does not match archive object content", ObjectPath: path, Digest: strings.TrimSpace(obj.Digest)})
	}
	return findings
}

func verifyOfflineBundleReportCoverage(bundle offlineBundleSnapshot, reports []AuditEvidenceBundleOfflineReportPosture) []AuditEvidenceBundleOfflineFinding {
	if len(reports) != 0 {
		return nil
	}
	if offlineBundleHasReferencedVerificationReport(bundle) {
		return nil
	}
	findings := []AuditEvidenceBundleOfflineFinding{{Code: "verification_report_missing", Severity: "error", Message: "bundle does not include audit verification report evidence"}}
	for i := range bundle.manifest.Redactions {
		if strings.TrimSpace(bundle.manifest.Redactions[i].Path) == "" {
			continue
		}
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verification_evidence_redacted", Severity: "warning", Message: "manifest redactions omit evidence object", ObjectPath: bundle.manifest.Redactions[i].Path})
	}
	return findings
}

func offlineBundleHasReferencedVerificationReport(bundle offlineBundleSnapshot) bool {
	for i := range bundle.manifest.IncludedObjects {
		if strings.TrimSpace(bundle.manifest.IncludedObjects[i].ObjectFamily) == "audit_verification_report" {
			return true
		}
	}
	return false
}

func verifyOfflineReportPosture(raw []byte, digestIdentity string) (AuditEvidenceBundleOfflineReportPosture, []AuditEvidenceBundleOfflineFinding, bool) {
	report := trustpolicy.AuditVerificationReportPayload{}
	if err := json.Unmarshal(raw, &report); err != nil {
		return AuditEvidenceBundleOfflineReportPosture{}, []AuditEvidenceBundleOfflineFinding{{Code: "verification_report_invalid", Severity: "error", Message: fmt.Sprintf("verification report decode failed: %v", err), Digest: digestIdentity}}, false
	}
	computed, err := canonicalDigest(report)
	if err != nil {
		return AuditEvidenceBundleOfflineReportPosture{}, []AuditEvidenceBundleOfflineFinding{{Code: "verification_report_invalid", Severity: "error", Message: fmt.Sprintf("verification report digest failed: %v", err), Digest: digestIdentity}}, false
	}
	computedIdentity, _ := computed.Identity()
	findings := []AuditEvidenceBundleOfflineFinding{}
	if strings.TrimSpace(digestIdentity) != computedIdentity {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verification_report_digest_mismatch", Severity: "error", Message: "verification report digest does not match payload", Digest: digestIdentity})
	}
	if err := trustpolicy.ValidateAuditVerificationReportPayload(report); err != nil {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verification_report_invalid", Severity: "error", Message: fmt.Sprintf("verification report invalid: %v", err), Digest: digestIdentity})
		return AuditEvidenceBundleOfflineReportPosture{}, findings, false
	}
	if report.CurrentlyDegraded || len(report.DegradedReasons) > 0 {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verification_report_degraded_posture", Severity: "warning", Message: "verification report indicates degraded posture", Digest: computedIdentity})
	}
	if len(report.HardFailures) > 0 {
		findings = append(findings, AuditEvidenceBundleOfflineFinding{Code: "verification_report_hard_failures", Severity: "error", Message: "verification report includes hard failures", Digest: computedIdentity})
	}
	return AuditEvidenceBundleOfflineReportPosture{
		Digest:                 computedIdentity,
		IntegrityStatus:        report.IntegrityStatus,
		AnchoringStatus:        report.AnchoringStatus,
		StoragePostureStatus:   report.StoragePostureStatus,
		SegmentLifecycleStatus: report.SegmentLifecycleStatus,
		CurrentlyDegraded:      report.CurrentlyDegraded,
		DegradedReasons:        report.DegradedReasons,
		HardFailures:           report.HardFailures,
	}, findings, true
}

func sortOfflineFindings(findings []AuditEvidenceBundleOfflineFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity == findings[j].Severity {
			if findings[i].Code == findings[j].Code {
				if findings[i].ObjectPath == findings[j].ObjectPath {
					return findings[i].Digest < findings[j].Digest
				}
				return findings[i].ObjectPath < findings[j].ObjectPath
			}
			return findings[i].Code < findings[j].Code
		}
		return offlineSeverityRank(findings[i].Severity) < offlineSeverityRank(findings[j].Severity)
	})
}

func offlineSeverityRank(severity string) int {
	switch severity {
	case "error":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}
