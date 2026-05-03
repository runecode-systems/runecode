package auditd

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestOfflineVerifyEvidenceBundlePreservesVerifierAndTrustRootIdentity(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	report := validReportFixture("segment-000001")
	report.CurrentlyDegraded = true
	report.AnchoringStatus = trustpolicy.AuditVerificationStatusDegraded
	report.DegradedReasons = []string{trustpolicy.AuditVerificationReasonExternalAnchorDeferredOrUnavailable}
	_ = mustPersistReport(t, ledger, report)

	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "external_relying_party_minimal",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "digest_metadata_only", SelectiveDisclosureApplied: true},
		},
		ArchiveFormat: "tar",
	})

	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(archive), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if verification.VerificationStatus != "degraded" {
		t.Fatalf("verification_status = %q, want degraded", verification.VerificationStatus)
	}
	if verification.VerifierIdentity.KeyIDValue == "" {
		t.Fatal("verifier_identity.key_id_value empty, want preserved verifier identity")
	}
	if len(verification.TrustRootDigests) == 0 {
		t.Fatal("trust_root_digests empty, want preserved trust-root identity")
	}
	if !hasOfflineFindingCode(verification.Findings, "verification_report_degraded_posture") {
		t.Fatalf("findings = %+v, want degraded posture finding", verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleSurfacesMissingVerificationEvidence(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineVerifyBundleEvidence(t, ledger, fixture)
	baseline := mustBuildExternalRelyingPartyManifest(t, ledger, nil)
	verificationReportPath := mustIncludedOfflineObjectPathByFamily(t, baseline.IncludedObjects, "audit_verification_report")
	archive := mustExportExternalRelyingPartyBundle(t, ledger, []AuditEvidenceBundleRedaction{{Path: verificationReportPath, ReasonCode: "policy_minimize_sensitive"}})
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(archive), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if verification.VerificationStatus != "failed" {
		t.Fatalf("verification_status = %q, want failed", verification.VerificationStatus)
	}
	if !hasOfflineFindingCode(verification.Findings, "verification_report_missing") {
		t.Fatalf("findings = %+v, want verification_report_missing", verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleManifestIsNotSubstituteForMissingEvidenceObjects(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineVerifyBundleEvidence(t, ledger, fixture)
	manifestBytes, manifest := mustReadExportedBundleManifest(t, ledger)
	if len(manifest.IncludedObjects) == 0 {
		t.Fatal("manifest included_objects empty, want evidence references")
	}

	manifestOnlyArchive := mustBuildTarFromEntries(t, map[string][]byte{"manifest.json": manifestBytes})
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(manifestOnlyArchive), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle(manifest-only) returned error: %v", err)
	}
	if verification.VerificationStatus != "failed" {
		t.Fatalf("verification_status = %q, want failed when manifest has no underlying evidence objects", verification.VerificationStatus)
	}
	if !hasOfflineFindingCode(verification.Findings, "bundle_object_missing") {
		t.Fatalf("findings = %+v, want bundle_object_missing", verification.Findings)
	}
}

func seedOfflineVerifyBundleEvidence(t *testing.T, ledger *Ledger, fixture auditFixtureKey) {
	t.Helper()
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
}

func mustBuildExternalRelyingPartyManifest(t *testing.T, ledger *Ledger, redactions []AuditEvidenceBundleRedaction) AuditEvidenceBundleManifest {
	t.Helper()
	manifest, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile:     "external_relying_party_minimal",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "digest_metadata_only", SelectiveDisclosureApplied: true},
		Redactions:        redactions,
	})
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest returned error: %v", err)
	}
	return manifest
}

func mustIncludedOfflineObjectPathByFamily(t *testing.T, objects []AuditEvidenceBundleIncludedObject, family string) string {
	t.Helper()
	for i := range objects {
		if objects[i].ObjectFamily == family {
			return objects[i].Path
		}
	}
	t.Fatalf("included_objects = %+v, want %s path", objects, family)
	return ""
}

func mustExportExternalRelyingPartyBundle(t *testing.T, ledger *Ledger, redactions []AuditEvidenceBundleRedaction) []byte {
	t.Helper()
	return mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "external_relying_party_minimal",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "digest_metadata_only", SelectiveDisclosureApplied: true},
			Redactions:        redactions,
		},
		ArchiveFormat: "tar",
	})
}

func mustReadExportedBundleManifest(t *testing.T, ledger *Ledger) ([]byte, AuditEvidenceBundleManifest) {
	t.Helper()
	rawArchive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	entries := readTarEntries(t, rawArchive)
	manifestBytes, ok := entries["manifest.json"]
	if !ok {
		t.Fatal("manifest.json missing from archive")
	}
	manifest := AuditEvidenceBundleManifest{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("Unmarshal(manifest.json) returned error: %v", err)
	}
	return manifestBytes, manifest
}

func mustExportBundleArchiveForOfflineVerify(t *testing.T, ledger *Ledger, req AuditEvidenceBundleExportRequest) []byte {
	t.Helper()
	exported, err := ledger.ExportEvidenceBundle(req)
	if err != nil {
		t.Fatalf("ExportEvidenceBundle returned error: %v", err)
	}
	defer exported.Reader.Close()
	b, err := io.ReadAll(exported.Reader)
	if err != nil {
		t.Fatalf("ReadAll(exported.Reader) returned error: %v", err)
	}
	return b
}

func hasOfflineFindingCode(findings []AuditEvidenceBundleOfflineFinding, code string) bool {
	for i := range findings {
		if findings[i].Code == code {
			return true
		}
	}
	return false
}

func mustBuildTarFromEntries(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	for name, payload := range entries {
		header := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(payload))}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%q) returned error: %v", name, err)
		}
		if _, err := tw.Write(payload); err != nil {
			t.Fatalf("Write(%q) returned error: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar writer close returned error: %v", err)
	}
	return out.Bytes()
}
