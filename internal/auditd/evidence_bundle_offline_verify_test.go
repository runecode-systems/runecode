package auditd

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestOfflineVerifyEvidenceBundleRecomputesWhenInputsPresent(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineRecomputeComparableBundleEvidence(t, ledger, fixture)
	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(archive), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if hasOfflineFindingCode(verification.Findings, "verification_recompute_inputs_missing") {
		t.Fatalf("findings = %+v, want recomputation inputs present for operator_private_full", verification.Findings)
	}
	if hasOfflineFindingCode(verification.Findings, "verification_recompute_mismatch") {
		t.Fatalf("findings = %+v, want recomputation to match included report conclusion", verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleExportsSegmentPayloadForRecompute(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineRecomputeComparableBundleEvidence(t, ledger, fixture)
	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	entries := readTarEntries(t, archive)
	segmentPayload, ok := entries["segments/segment-000001.json"]
	if !ok {
		t.Fatal("segments/segment-000001.json missing from archive")
	}
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := json.Unmarshal(segmentPayload, &segment); err != nil {
		t.Fatalf("segment payload should decode as AuditSegmentFilePayload: %v", err)
	}
}

func TestOfflineVerifyEvidenceBundleDegradesWhenRecomputeInputsMissing(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineRecomputeComparableBundleEvidence(t, ledger, fixture)
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
	if !hasOfflineFindingCode(verification.Findings, "verification_recompute_inputs_missing") {
		t.Fatalf("findings = %+v, want explicit recomputation input gap", verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleDegradesWhenVerifierIdentityMissing(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineVerifyBundleEvidence(t, ledger, fixture)
	archive, manifest := mustExportAndReadExternalRelyingPartyManifest(t, ledger)
	manifest.VerifierIdentity.KeyIDValue = ""
	manifest.TrustRootDigests = []string{"sha256:" + strings.Repeat("a", 64)}
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(rewriteBundleManifest(t, archive, manifest)), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if verification.VerificationStatus != "degraded" {
		t.Fatalf("verification_status = %q, want degraded", verification.VerificationStatus)
	}
	if !hasOfflineFindingCode(verification.Findings, "verifier_identity_missing") {
		t.Fatalf("findings = %+v, want verifier_identity_missing", verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleDegradesWhenTrustRootDigestsMissing(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineVerifyBundleEvidence(t, ledger, fixture)
	archive, manifest := mustExportAndReadExternalRelyingPartyManifest(t, ledger)
	manifest.VerifierIdentity.KeyIDValue = strings.Repeat("a", 64)
	manifest.TrustRootDigests = nil
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(rewriteBundleManifest(t, archive, manifest)), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if verification.VerificationStatus != "degraded" {
		t.Fatalf("verification_status = %q, want degraded", verification.VerificationStatus)
	}
	if !hasOfflineFindingCode(verification.Findings, "trust_root_identity_missing") {
		t.Fatalf("findings = %+v, want trust_root_identity_missing", verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleUsesVerifierClockForRecompute(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineRecomputeComparableBundleEvidence(t, ledger, fixture)
	wantNow := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)
	ledger.nowFn = func() time.Time { return wantNow }
	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	bundle, err := loadAuditEvidenceBundleFromTar(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("loadAuditEvidenceBundleFromTar returned error: %v", err)
	}
	bundle.verifiedAt = wantNow
	reportObject := offlineBundleObjectsByFamily(bundle.manifest.IncludedObjects, "audit_verification_report")[0]
	report, _, err := decodeOfflineVerificationReport(bundle, reportObject)
	if err != nil {
		t.Fatalf("decodeOfflineVerificationReport returned error: %v", err)
	}
	report.VerifiedAt = "1999-01-01T00:00:00Z"
	input, missing, err := offlineRecomputeInput(bundle, report)
	if err != nil {
		t.Fatalf("offlineRecomputeInput returned error: %v", err)
	}
	if len(missing) > 0 {
		t.Fatalf("offlineRecomputeInput missing=%v, want complete inputs", missing)
	}
	if !input.Now.Equal(wantNow) {
		t.Fatalf("input.Now = %s, want %s", input.Now.Format(time.RFC3339), wantNow.Format(time.RFC3339))
	}
}

func TestOfflineVerifyEvidenceBundleFlagsMissingReferencedVerificationReport(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineRecomputeComparableBundleEvidence(t, ledger, fixture)
	report := validReportFixture("segment-000001")
	missingDigest := mustPersistReport(t, ledger, report)
	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	entries := readTarEntries(t, archive)
	missingID, _ := missingDigest.Identity()
	missingPath := filepath.ToSlash("sidecar/verification-reports/" + strings.TrimPrefix(missingID, "sha256:") + ".json")
	delete(entries, missingPath)
	modified := mustBuildTarFromEntries(t, entries)
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(modified), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if !hasOfflineFindingCode(verification.Findings, "verification_report_missing") {
		t.Fatalf("findings = %+v, want verification_report_missing", verification.Findings)
	}
	if got := countOfflineFindingCode(verification.Findings, "verification_report_missing"); got != 1 {
		t.Fatalf("verification_report_missing count = %d, want 1 (findings=%+v)", got, verification.Findings)
	}
}

func TestOfflineVerifyEvidenceBundleIgnoresUnrelatedMetaAuditReceiptDrift(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedOfflineRecomputeComparableBundleEvidence(t, ledger, fixture)
	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	entries := readTarEntries(t, archive)
	for path := range entries {
		if strings.HasPrefix(path, "sidecar/receipts/") {
			delete(entries, path)
		}
	}
	modified := mustBuildTarFromEntries(t, entries)
	verification, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(modified), "tar")
	if err != nil {
		t.Fatalf("OfflineVerifyEvidenceBundle returned error: %v", err)
	}
	if !hasOfflineFindingCode(verification.Findings, "bundle_object_missing") {
		t.Fatalf("findings = %+v, want explicit missing-object finding for removed receipts", verification.Findings)
	}
}

func seedOfflineRecomputeComparableBundleEvidence(t *testing.T, ledger *Ledger, fixture auditFixtureKey) {
	t.Helper()
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	seedOfflineComparableReport(t, ledger)
}

func seedOfflineComparableReport(t *testing.T, ledger *Ledger) {
	t.Helper()
	seedArchive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	bundle, err := loadAuditEvidenceBundleFromTar(bytes.NewReader(seedArchive))
	if err != nil {
		t.Fatalf("loadAuditEvidenceBundleFromTar returned error: %v", err)
	}
	bundle.verifiedAt = time.Now().UTC()
	if len(bundle.manifest.SealReferences) == 0 {
		t.Fatal("bundle seal_references empty, want sealed segment for recomputation")
	}
	segmentID := bundle.manifest.SealReferences[0].SegmentID
	input, missing, err := offlineRecomputeInput(bundle, trustpolicy.AuditVerificationReportPayload{VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: segmentID}})
	if err != nil {
		t.Fatalf("offlineRecomputeInput returned error: %v", err)
	}
	if len(missing) > 0 {
		t.Fatalf("offlineRecomputeInput missing=%v, want complete recompute inputs for operator_private_full", missing)
	}
	report, err := trustpolicy.VerifyAuditEvidence(input)
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	_ = mustPersistReport(t, ledger, report)
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

func mustExportAndReadBundleManifest(t *testing.T, ledger *Ledger) ([]byte, AuditEvidenceBundleManifest) {
	t.Helper()
	archive := mustExportBundleArchiveForOfflineVerify(t, ledger, AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	entries := readTarEntries(t, archive)
	manifestBytes, ok := entries["manifest.json"]
	if !ok {
		t.Fatal("manifest.json missing from archive")
	}
	manifest := AuditEvidenceBundleManifest{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("Unmarshal(manifest.json) returned error: %v", err)
	}
	return archive, manifest
}

func mustExportAndReadExternalRelyingPartyManifest(t *testing.T, ledger *Ledger) ([]byte, AuditEvidenceBundleManifest) {
	t.Helper()
	archive := mustExportExternalRelyingPartyBundle(t, ledger, nil)
	entries := readTarEntries(t, archive)
	manifestBytes, ok := entries["manifest.json"]
	if !ok {
		t.Fatal("manifest.json missing from archive")
	}
	manifest := AuditEvidenceBundleManifest{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("Unmarshal(manifest.json) returned error: %v", err)
	}
	return archive, manifest
}

func rewriteBundleManifest(t *testing.T, archive []byte, manifest AuditEvidenceBundleManifest) []byte {
	t.Helper()
	entries := readTarEntries(t, archive)
	manifestBytes, err := evidenceBundleCanonicalBytes(manifest)
	if err != nil {
		t.Fatalf("evidenceBundleCanonicalBytes returned error: %v", err)
	}
	entries["manifest.json"] = manifestBytes
	return mustBuildTarFromEntries(t, entries)
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

func countOfflineFindingCode(findings []AuditEvidenceBundleOfflineFinding, code string) int {
	count := 0
	for i := range findings {
		if findings[i].Code == code {
			count++
		}
	}
	return count
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
