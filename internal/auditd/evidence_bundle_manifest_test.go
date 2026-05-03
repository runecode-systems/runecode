package auditd

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type selectiveDisclosureProfileExpectation struct {
	name                    string
	exportProfile           string
	posture                 string
	wantSelectiveDisclosure bool
	wantSegmentIncluded     bool
	wantReceiptIncluded     bool
	wantDeclaredSegmentRed  bool
	wantDeclaredReceiptRed  bool
}

func TestBuildEvidenceBundleManifestSelectiveDisclosureProfiles(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedMinimalManifestEvidence(t, ledger, fixture)
	for _, tc := range selectiveDisclosureProfileTests() {
		t.Run(tc.name, func(t *testing.T) {
			manifest := mustBuildSelectiveDisclosureManifest(t, ledger, tc.exportProfile, tc.posture)
			assertSelectiveDisclosureProfileManifest(t, manifest, tc)
		})
	}
}

func selectiveDisclosureProfileTests() []selectiveDisclosureProfileExpectation {
	return []selectiveDisclosureProfileExpectation{
		{name: "operator-private-full keeps full object families", exportProfile: "operator_private_full", posture: "operator_private", wantSelectiveDisclosure: false, wantSegmentIncluded: true, wantReceiptIncluded: true},
		{name: "company-internal-audit redacts segments only", exportProfile: "company_internal_audit", posture: "digest_metadata_only", wantSelectiveDisclosure: true, wantReceiptIncluded: true, wantDeclaredSegmentRed: true},
		{name: "external-relying-party-minimal redacts segments and receipts", exportProfile: "external_relying_party_minimal", posture: "digest_metadata_only", wantSelectiveDisclosure: true, wantDeclaredSegmentRed: true, wantDeclaredReceiptRed: true},
		{name: "incident-response-scope keeps evidence families but marks selective disclosure", exportProfile: "incident_response_scope", posture: "operator_private", wantSelectiveDisclosure: true, wantSegmentIncluded: true, wantReceiptIncluded: true},
	}
}

func mustBuildSelectiveDisclosureManifest(t *testing.T, ledger *Ledger, exportProfile string, posture string) AuditEvidenceBundleManifest {
	t.Helper()
	manifest, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile: exportProfile,
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    posture,
			SelectiveDisclosureApplied: false,
		},
	})
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest(%s) returned error: %v", exportProfile, err)
	}
	return manifest
}

func assertSelectiveDisclosureProfileManifest(t *testing.T, manifest AuditEvidenceBundleManifest, tc selectiveDisclosureProfileExpectation) {
	t.Helper()
	assertSelectiveDisclosurePosture(t, manifest, tc.wantSelectiveDisclosure)
	assertSelectiveDisclosureFamilies(t, manifest, tc.wantSegmentIncluded, tc.wantReceiptIncluded, tc.exportProfile)
	assertSelectiveDisclosureRedactions(t, manifest, tc.wantDeclaredSegmentRed, tc.wantDeclaredReceiptRed)
}

func assertSelectiveDisclosurePosture(t *testing.T, manifest AuditEvidenceBundleManifest, wantSelectiveDisclosure bool) {
	t.Helper()
	if manifest.DisclosurePosture.SelectiveDisclosureApplied != wantSelectiveDisclosure {
		t.Fatalf("selective_disclosure_applied = %v, want %v", manifest.DisclosurePosture.SelectiveDisclosureApplied, wantSelectiveDisclosure)
	}
}

func assertSelectiveDisclosureFamilies(t *testing.T, manifest AuditEvidenceBundleManifest, wantSegmentIncluded bool, wantReceiptIncluded bool, exportProfile string) {
	t.Helper()
	if includesObjectFamily(manifest.IncludedObjects, "audit_segment") != wantSegmentIncluded {
		t.Fatalf("included_objects has audit_segment = %v, want %v (profile=%s)", includesObjectFamily(manifest.IncludedObjects, "audit_segment"), wantSegmentIncluded, exportProfile)
	}
	if includesObjectFamily(manifest.IncludedObjects, "audit_receipt") != wantReceiptIncluded {
		t.Fatalf("included_objects has audit_receipt = %v, want %v (profile=%s)", includesObjectFamily(manifest.IncludedObjects, "audit_receipt"), wantReceiptIncluded, exportProfile)
	}
}

func assertSelectiveDisclosureRedactions(t *testing.T, manifest AuditEvidenceBundleManifest, wantDeclaredSegmentRed bool, wantDeclaredReceiptRed bool) {
	t.Helper()
	hasSegmentRedaction := hasBundleRedaction(manifest.Redactions, "segments/*", "profile_digest_metadata_default")
	if hasSegmentRedaction != wantDeclaredSegmentRed {
		t.Fatalf("has segment profile redaction = %v, want %v (redactions=%+v)", hasSegmentRedaction, wantDeclaredSegmentRed, manifest.Redactions)
	}
	hasReceiptRedaction := hasBundleRedaction(manifest.Redactions, "sidecar/receipts/*", "profile_digest_metadata_default")
	if hasReceiptRedaction != wantDeclaredReceiptRed {
		t.Fatalf("has receipt profile redaction = %v, want %v (redactions=%+v)", hasReceiptRedaction, wantDeclaredReceiptRed, manifest.Redactions)
	}
}

func TestBuildEvidenceBundleManifestIncludesScopeRootsAndDisclosure(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedMinimalManifestEvidence(t, ledger, fixture)
	manifest, err := ledger.BuildEvidenceBundleManifest(externalRelyingPartyManifestRequest())
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest returned error: %v", err)
	}
	assertExternalRelyingPartyManifest(t, manifest)
	assertExternalRelyingPartyDisclosure(t, manifest)
	assertExternalRelyingPartyIncludedObjects(t, manifest)
	assertPortableIncludedObjectPaths(t, manifest.IncludedObjects)
	assertExternalRelyingPartyExcludesRawEvidence(t, manifest)
	assertExternalRelyingPartyRedactions(t, manifest)
}

func seedMinimalManifestEvidence(t *testing.T, ledger *Ledger, fixture auditFixtureKey) {
	t.Helper()
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	receipt := buildAnchorReceiptEnvelope(t, fixture, sealResult.SealEnvelopeDigest)
	_ = mustPersistReceipt(t, ledger, receipt)
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
}

func externalRelyingPartyManifestRequest() AuditEvidenceBundleManifestRequest {
	return AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile:     "external_relying_party_minimal",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev", ProtocolBundleManifestHash: "sha256:" + strings.Repeat("b", 64)},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "digest_metadata_only", SelectiveDisclosureApplied: true},
		Redactions:        []AuditEvidenceBundleRedaction{{Path: "artifacts/prompt.txt", ReasonCode: "policy_minimize_sensitive"}},
	}
}

func assertExternalRelyingPartyManifest(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	assertExternalRelyingPartyManifestHeader(t, manifest)
	assertExternalRelyingPartyManifestScope(t, manifest)
	assertExternalRelyingPartyManifestEvidence(t, manifest)
}

func assertExternalRelyingPartyManifestHeader(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if manifest.SchemaID != auditEvidenceBundleManifestSchemaID || manifest.SchemaVersion != auditEvidenceBundleManifestSchemaVersion {
		t.Fatalf("manifest schema = %q/%q, want %q/%q", manifest.SchemaID, manifest.SchemaVersion, auditEvidenceBundleManifestSchemaID, auditEvidenceBundleManifestSchemaVersion)
	}
	if manifest.ExportProfile != "external_relying_party_minimal" {
		t.Fatalf("export_profile = %q, want external_relying_party_minimal", manifest.ExportProfile)
	}
}

func assertExternalRelyingPartyManifestScope(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if manifest.Scope.ScopeKind != "run" || manifest.Scope.RunID != "run-1" {
		t.Fatalf("manifest scope = %+v, want run scope", manifest.Scope)
	}
	if strings.TrimSpace(manifest.InstanceIdentity) != "" {
		t.Fatalf("instance_identity_digest = %q, want empty when no instance identity evidence exists", manifest.InstanceIdentity)
	}
}

func assertExternalRelyingPartyManifestEvidence(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if len(manifest.IncludedObjects) == 0 {
		t.Fatal("included_objects empty, want bundle object list")
	}
	if len(manifest.RootDigests) == 0 {
		t.Fatal("root_digests empty, want root digests")
	}
	if len(manifest.SealReferences) == 0 {
		t.Fatal("seal_references empty, want at least one seal reference")
	}
	if manifest.VerifierIdentity.KeyIDValue == "" {
		t.Fatal("verifier_identity.key_id_value empty, want verifier identity")
	}
	if len(manifest.TrustRootDigests) == 0 {
		t.Fatal("trust_root_digests empty, want trust roots")
	}
	if manifest.ControlPlane == nil {
		t.Fatal("control_plane_provenance missing, want trusted control-plane digests")
	}
	if strings.TrimSpace(manifest.ControlPlane.ProtocolBundleHash) == "" {
		t.Fatal("control_plane_provenance.protocol_bundle_manifest_hash empty, want protocol bundle identity")
	}
	if strings.TrimSpace(manifest.ControlPlane.VerifierImplDigest) == "" {
		t.Fatal("control_plane_provenance.verifier_implementation_digest empty, want verifier implementation identity")
	}
	if strings.TrimSpace(manifest.ControlPlane.TrustPolicyDigest) == "" {
		t.Fatal("control_plane_provenance.trust_policy_digest empty, want trust-policy identity")
	}
}

func assertExternalRelyingPartyDisclosure(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if !manifest.DisclosurePosture.SelectiveDisclosureApplied {
		t.Fatal("disclosure_posture.selective_disclosure_applied = false, want true")
	}
	if len(manifest.Redactions) < 3 {
		t.Fatalf("redactions = %+v, want profile redaction declarations and request redactions", manifest.Redactions)
	}
}

func assertExternalRelyingPartyRedactions(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if !hasBundleRedaction(manifest.Redactions, "segments/*", "profile_digest_metadata_default") {
		t.Fatalf("redactions = %+v, want profile declaration for segments", manifest.Redactions)
	}
	if !hasBundleRedaction(manifest.Redactions, "sidecar/receipts/*", "profile_digest_metadata_default") {
		t.Fatalf("redactions = %+v, want profile declaration for receipts", manifest.Redactions)
	}
	if !hasBundleRedaction(manifest.Redactions, "artifacts/prompt.txt", "policy_minimize_sensitive") {
		t.Fatalf("redactions = %+v, want requested redaction preserved", manifest.Redactions)
	}
}

func assertExternalRelyingPartyIncludedObjects(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if len(manifest.IncludedObjects) == 0 {
		t.Fatal("included_objects empty, want bundle object list")
	}
	if len(manifest.RootDigests) == 0 || len(manifest.SealReferences) == 0 {
		t.Fatalf("manifest roots/seals missing: root_digests=%+v seal_references=%+v", manifest.RootDigests, manifest.SealReferences)
	}
}

func assertPortableIncludedObjectPaths(t *testing.T, objects []AuditEvidenceBundleIncludedObject) {
	t.Helper()
	if len(objects) > 0 {
		if !strings.Contains(objects[0].Path, "/") {
			t.Fatalf("included path %q, want portable slash path", objects[0].Path)
		}
	}
}

func assertExternalRelyingPartyExcludesRawEvidence(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if includesObjectFamily(manifest.IncludedObjects, "audit_segment") {
		t.Fatalf("included_objects = %+v, want no raw segments for external_relying_party_minimal", manifest.IncludedObjects)
	}
	if includesObjectFamily(manifest.IncludedObjects, "audit_receipt") {
		t.Fatalf("included_objects = %+v, want no receipts for external_relying_party_minimal", manifest.IncludedObjects)
	}
}

func TestBuildEvidenceBundleManifestIncludesStableInstanceIdentityWhenPresent(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
	instanceDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("9", 64)}
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)}
	if err := persistExternalAnchorEvidenceForSealWithProjectContext(t, ledger, seal.SealEnvelopeDigest, targetDigest, instanceDigest); err != nil {
		t.Fatalf("persistExternalAnchorEvidenceForSealWithProjectContext returned error: %v", err)
	}
	manifest, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile:     "operator_private_full",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest returned error: %v", err)
	}
	if strings.TrimSpace(manifest.InstanceIdentity) == "" {
		t.Fatal("instance_identity_digest empty, want preserved stable instance identity")
	}
	if manifest.InstanceIdentity != "sha256:"+strings.Repeat("9", 64) {
		t.Fatalf("instance_identity_digest = %q, want seeded project context identity digest", manifest.InstanceIdentity)
	}
}

func TestBuildEvidenceBundleManifestAppliesRequestedRedactionsToIncludedObjects(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seedMinimalManifestEvidence(t, ledger, fixture)
	withoutRedaction := mustBuildOperatorPrivateManifest(t, ledger, nil)
	targetPath := mustIncludedObjectPathByFamily(t, withoutRedaction, "audit_receipt")
	withRedaction := mustBuildOperatorPrivateManifest(t, ledger, []AuditEvidenceBundleRedaction{{Path: targetPath, ReasonCode: "policy_minimize_sensitive"}})
	if includesObjectPath(withRedaction.IncludedObjects, targetPath) {
		t.Fatalf("included_objects contains redacted path %q: %+v", targetPath, withRedaction.IncludedObjects)
	}
	if !withRedaction.DisclosurePosture.SelectiveDisclosureApplied {
		t.Fatal("disclosure_posture.selective_disclosure_applied=false, want true when redactions are applied")
	}
	if !hasBundleRedaction(withRedaction.Redactions, targetPath, "policy_minimize_sensitive") {
		t.Fatalf("redactions = %+v, want requested redaction entry", withRedaction.Redactions)
	}
}

func mustBuildOperatorPrivateManifest(t *testing.T, ledger *Ledger, redactions []AuditEvidenceBundleRedaction) AuditEvidenceBundleManifest {
	t.Helper()
	manifest, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile:     "operator_private_full",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		Redactions:        redactions,
	})
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest(operator private) returned error: %v", err)
	}
	return manifest
}

func mustIncludedObjectPathByFamily(t *testing.T, manifest AuditEvidenceBundleManifest, family string) string {
	t.Helper()
	for i := range manifest.IncludedObjects {
		if manifest.IncludedObjects[i].ObjectFamily == family {
			return manifest.IncludedObjects[i].Path
		}
	}
	t.Fatalf("included_objects = %+v, want at least one %s path", manifest.IncludedObjects, family)
	return ""
}

func TestBuildEvidenceBundleManifestRejectsInvalidRequest(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	_, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest expected validation error")
	}
}

func TestBuildEvidenceBundleManifestRejectsDisclosurePostureProfileMismatch(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	_, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile:     "external_relying_party_minimal",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest expected disclosure posture/profile mismatch error")
	}
}

func hasBundleRedaction(redactions []AuditEvidenceBundleRedaction, path string, reason string) bool {
	for i := range redactions {
		if redactions[i].Path == path && redactions[i].ReasonCode == reason {
			return true
		}
	}
	return false
}

func includesObjectFamily(objects []AuditEvidenceBundleIncludedObject, family string) bool {
	for i := range objects {
		if objects[i].ObjectFamily == family {
			return true
		}
	}
	return false
}

func includesObjectPath(objects []AuditEvidenceBundleIncludedObject, targetPath string) bool {
	for i := range objects {
		if objects[i].Path == targetPath {
			return true
		}
	}
	return false
}

func TestBuildEvidenceBundleManifestRejectsUnsupportedScopeAndProfile(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	_, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "unknown"},
		ExportProfile:     "operator_private_full",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest expected scope validation error")
	}
	_, err = ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile:     "unknown_profile",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest expected export profile validation error")
	}
}
