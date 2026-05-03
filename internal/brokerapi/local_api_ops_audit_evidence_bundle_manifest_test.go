package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/secretsd"
)

func TestAuditEvidenceBundleManifestGetBuildsManifest(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	resp, errResp := service.HandleAuditEvidenceBundleManifestGet(context.Background(), validAuditEvidenceBundleManifestGetRequest("req-audit-bundle"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleManifestGet error response: %+v", errResp)
	}
	if err := service.validateResponse(resp, auditEvidenceBundleManifestGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditEvidenceBundleManifestGetResponse) returned error: %v", err)
	}
	assertProjectedAuditEvidenceBundleManifest(t, resp.Manifest)
	if resp.SignedManifest != nil {
		t.Fatal("signed_manifest present without external_sharing_intended")
	}
}

func assertProjectedAuditEvidenceBundleManifest(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if manifest.SchemaID != "runecode.protocol.v0.AuditEvidenceBundleManifest" {
		t.Fatalf("manifest.schema_id = %q, want runecode.protocol.v0.AuditEvidenceBundleManifest", manifest.SchemaID)
	}
	assertAuditEvidenceBundleManifestMaterialized(t, manifest)
	assertAuditEvidenceBundleManifestControlPlane(t, manifest)
}

func assertAuditEvidenceBundleManifestMaterialized(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if len(manifest.IncludedObjects) == 0 {
		t.Fatal("included_objects empty, want projected included objects")
	}
	if len(manifest.SealReferences) == 0 {
		t.Fatal("seal_references empty, want seal references")
	}
	if len(manifest.RootDigests) == 0 {
		t.Fatal("root_digests empty, want root digests")
	}
	if len(manifest.TrustRootDigests) == 0 {
		t.Fatal("trust_root_digests empty, want trust roots")
	}
}

func assertAuditEvidenceBundleManifestControlPlane(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if manifest.ControlPlane == nil {
		t.Fatal("control_plane_provenance missing, want projected control-plane digests")
	}
	if manifest.ControlPlane.ProtocolBundleHash == nil {
		t.Fatal("control_plane_provenance.protocol_bundle_manifest_hash missing, want protocol bundle digest")
	}
	if manifest.ControlPlane.VerifierImplDigest == nil {
		t.Fatal("control_plane_provenance.verifier_implementation_digest missing, want verifier implementation digest")
	}
	if manifest.ControlPlane.TrustPolicyDigest == nil {
		t.Fatal("control_plane_provenance.trust_policy_digest missing, want trust-policy digest")
	}
}

func validAuditEvidenceBundleManifestGetRequest(requestID string) AuditEvidenceBundleManifestGetRequest {
	bundleDigest := digestChar("b")
	return AuditEvidenceBundleManifestGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleManifestGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile: "external_relying_party_minimal",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev", ProtocolBundleManifestHash: &bundleDigest},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "digest_metadata_only",
			SelectiveDisclosureApplied: true,
		},
		Redactions: []AuditEvidenceBundleRedaction{{Path: "artifacts/prompt.txt", ReasonCode: "policy_minimize_sensitive"}},
	}
}

func TestAuditEvidenceBundleManifestGetSignsWhenExternalSharingIntended(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	bundleDigest := digestChar("b")
	secretsRoot := t.TempDir()
	secretsSvc, err := secretsd.Open(secretsRoot)
	if err != nil {
		t.Fatalf("secretsd.Open returned error: %v", err)
	}
	service.secretsSvc = secretsSvc
	resp, errResp := service.HandleAuditEvidenceBundleManifestGet(context.Background(), AuditEvidenceBundleManifestGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleManifestGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-sign",
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile: "external_relying_party_minimal",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev", ProtocolBundleManifestHash: &bundleDigest},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "digest_metadata_only",
			SelectiveDisclosureApplied: true,
		},
		ExternalSharingIntended: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleManifestGet error response: %+v", errResp)
	}
	if resp.SignedManifest == nil {
		t.Fatal("signed_manifest = nil, want envelope")
	}
	if resp.SignedManifest.PayloadSchemaID != "runecode.protocol.v0.AuditEvidenceBundleManifest" {
		t.Fatalf("signed_manifest.payload_schema_id = %q, want runecode.protocol.v0.AuditEvidenceBundleManifest", resp.SignedManifest.PayloadSchemaID)
	}
	if err := service.validateResponse(resp, auditEvidenceBundleManifestGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditEvidenceBundleManifestGetResponse) returned error: %v", err)
	}
}

func TestAuditEvidenceBundleManifestGetExternalSharingRequiresSealedSegment(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	bundleDigest := digestChar("b")
	_, errResp := service.HandleAuditEvidenceBundleManifestGet(context.Background(), AuditEvidenceBundleManifestGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleManifestGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-sign-no-seal",
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile: "external_relying_party_minimal",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev", ProtocolBundleManifestHash: &bundleDigest},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "digest_metadata_only",
			SelectiveDisclosureApplied: true,
		},
		ExternalSharingIntended: true,
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleManifestGet expected error when no sealed segment exists")
	}
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
}

func TestAuditEvidenceBundleManifestGetRejectsUnsupportedScopeKind(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	_, errResp := service.HandleAuditEvidenceBundleManifestGet(context.Background(), AuditEvidenceBundleManifestGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleManifestGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-scope-invalid",
		Scope:         AuditEvidenceBundleScope{ScopeKind: "unsupported"},
		ExportProfile: "external_relying_party_minimal",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "digest_metadata_only",
			SelectiveDisclosureApplied: true,
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleManifestGet expected scope validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}
