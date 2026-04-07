package protocolschema

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSchemaManifestMatchesSchemas(t *testing.T) {
	manifest := loadManifest(t)
	assertManifestMetadata(t, manifest)
	assertManifestFileSet(t, manifest.SchemaFiles, "objects", ".schema.json")
	assertManifestRegistryFileSet(t, manifest.Registries)
	assertSchemaManifestEntries(t, manifest)
	assertReservedStatuses(t, manifest)
	assertSchemaVersions(t, manifest)
}

func assertReservedStatuses(t *testing.T, manifest manifestFile) {
	t.Helper()
	assertReservedStatus(t, manifest, "runecode.protocol.v0.WorkflowDefinition")
	assertReservedStatus(t, manifest, "runecode.protocol.v0.ProcessDefinition")
}

func assertSchemaVersions(t *testing.T, manifest manifestFile) {
	t.Helper()
	assertSchemaVersionsCore(t, manifest)
	assertSchemaVersionsLocalBroker(t, manifest)
}

func assertSchemaVersionsCore(t *testing.T, manifest manifestFile) {
	t.Helper()
	versions := map[string]string{
		"runecode.protocol.v0.ArtifactReference":          "0.3.0",
		"runecode.protocol.v0.ArtifactPolicy":             "0.1.0",
		"runecode.protocol.v0.AuditRecordDigest":          "0.1.0",
		"runecode.protocol.v0.AuditEvent":                 "0.5.0",
		"runecode.protocol.v0.AuditEventContractCatalog":  "0.1.0",
		"runecode.protocol.v0.AuditReceipt":               "0.4.0",
		"runecode.protocol.v0.AuditSegmentSeal":           "0.2.0",
		"runecode.protocol.v0.AuditSegmentFile":           "0.1.0",
		"runecode.protocol.v0.AuditVerificationReport":    "0.1.0",
		"runecode.protocol.v0.SignedObjectEnvelope":       "0.2.0",
		"runecode.protocol.v0.ApprovalRequest":            "0.3.0",
		"runecode.protocol.v0.ApprovalDecision":           "0.3.0",
		"runecode.protocol.v0.VerifierRecord":             "0.1.0",
		"runecode.protocol.v0.BrokerArtifactListRequest":  "0.1.0",
		"runecode.protocol.v0.BrokerArtifactListResponse": "0.1.0",
		"runecode.protocol.v0.BrokerArtifactHeadRequest":  "0.1.0",
		"runecode.protocol.v0.BrokerArtifactHeadResponse": "0.1.0",
		"runecode.protocol.v0.BrokerArtifactPutRequest":   "0.1.0",
		"runecode.protocol.v0.BrokerArtifactPutResponse":  "0.1.0",
		"runecode.protocol.v0.BrokerErrorResponse":        "0.1.0",
	}
	for schemaID, version := range versions {
		assertManifestSchemaVersion(t, manifest, schemaID, version)
	}
}

func assertSchemaVersionsLocalBroker(t *testing.T, manifest manifestFile) {
	t.Helper()
	versions := map[string]string{
		"runecode.protocol.v0.RunSummary":                   "0.1.0",
		"runecode.protocol.v0.RunDetail":                    "0.1.0",
		"runecode.protocol.v0.RunStageSummary":              "0.1.0",
		"runecode.protocol.v0.RunRoleSummary":               "0.1.0",
		"runecode.protocol.v0.RunCoordinationSummary":       "0.1.0",
		"runecode.protocol.v0.ApprovalSummary":              "0.1.0",
		"runecode.protocol.v0.ApprovalBoundScope":           "0.1.0",
		"runecode.protocol.v0.ArtifactSummary":              "0.1.0",
		"runecode.protocol.v0.BrokerReadiness":              "0.1.0",
		"runecode.protocol.v0.BrokerVersionInfo":            "0.1.0",
		"runecode.protocol.v0.RunListRequest":               "0.1.0",
		"runecode.protocol.v0.RunListResponse":              "0.1.0",
		"runecode.protocol.v0.RunGetRequest":                "0.1.0",
		"runecode.protocol.v0.RunGetResponse":               "0.1.0",
		"runecode.protocol.v0.ApprovalListRequest":          "0.1.0",
		"runecode.protocol.v0.ApprovalListResponse":         "0.1.0",
		"runecode.protocol.v0.ApprovalGetRequest":           "0.1.0",
		"runecode.protocol.v0.ApprovalGetResponse":          "0.1.0",
		"runecode.protocol.v0.ApprovalResolveRequest":       "0.1.0",
		"runecode.protocol.v0.ApprovalResolveResponse":      "0.1.0",
		"runecode.protocol.v0.ArtifactListRequest":          "0.1.0",
		"runecode.protocol.v0.ArtifactListResponse":         "0.1.0",
		"runecode.protocol.v0.ArtifactHeadRequest":          "0.1.0",
		"runecode.protocol.v0.ArtifactHeadResponse":         "0.1.0",
		"runecode.protocol.v0.ArtifactReadRequest":          "0.1.0",
		"runecode.protocol.v0.ArtifactStreamEvent":          "0.1.0",
		"runecode.protocol.v0.AuditTimelineRequest":         "0.1.0",
		"runecode.protocol.v0.AuditTimelineResponse":        "0.1.0",
		"runecode.protocol.v0.AuditVerificationGetRequest":  "0.1.0",
		"runecode.protocol.v0.AuditVerificationGetResponse": "0.1.0",
		"runecode.protocol.v0.LogStreamRequest":             "0.1.0",
		"runecode.protocol.v0.LogStreamEvent":               "0.1.0",
		"runecode.protocol.v0.ReadinessGetRequest":          "0.1.0",
		"runecode.protocol.v0.ReadinessGetResponse":         "0.1.0",
		"runecode.protocol.v0.VersionInfoGetRequest":        "0.1.0",
		"runecode.protocol.v0.VersionInfoGetResponse":       "0.1.0",
	}
	for schemaID, version := range versions {
		assertManifestSchemaVersion(t, manifest, schemaID, version)
	}
}

func TestManifestAndRegistryDocumentsValidateAgainstMetaSchemas(t *testing.T) {
	manifest := loadManifest(t)
	compiler := newMetaCompiler(t)

	manifestSchema := mustCompileMetaSchema(t, compiler, metaPath(t, manifestMetaPath))
	if err := manifestSchema.Validate(loadJSONMap(t, schemaPath(t, "manifest.json"))); err != nil {
		t.Fatalf("manifest.json failed meta-schema validation: %v", err)
	}

	registrySchema := mustCompileMetaSchema(t, compiler, metaPath(t, registryMetaPath))
	for _, entry := range manifest.Registries {
		entry := entry
		t.Run(entry.Path, func(t *testing.T) {
			if err := registrySchema.Validate(loadJSONMap(t, schemaPath(t, entry.Path))); err != nil {
				t.Fatalf("%s failed registry meta-schema validation: %v", entry.Path, err)
			}
		})
	}
}

func TestRegistryNamespacesAreSeparate(t *testing.T) {
	manifest := loadManifest(t)
	registryNames, codesByRegistry := collectRegistryData(t, manifest)
	assertRegistryCodeNamespacesSeparate(t, registryNames, codesByRegistry)
	assertErrorRegistryCodes(t)
	assertPolicyRegistryCodes(t)
	assertAuditRegistryCodes(t)
	assertAuditReceiptRegistryCodes(t)
	assertAuditVerificationReasonRegistryCodes(t)
	assertApprovalRegistryCodes(t)
}

func assertRegistryCodeNamespacesSeparate(t *testing.T, registryNames []string, codesByRegistry map[string]map[string]struct{}) {
	t.Helper()

	sort.Strings(registryNames)
	for i := 0; i < len(registryNames); i++ {
		for j := i + 1; j < len(registryNames); j++ {
			assertNoCodeOverlap(t, codesByRegistry, registryNames[i], registryNames[j])
		}
	}
}

func assertErrorRegistryCodes(t *testing.T) {
	t.Helper()

	errorRegistry := loadRegistry(t, schemaPath(t, "registries/error.code.registry.json"))
	assertRegistryContainsCodes(t, errorRegistry,
		"unknown_schema_id",
		"unsupported_schema_version",
		"unsupported_hash_algorithm",
		"schema_bundle_version_mismatch",
		"stream_timeout",
		"gateway_failure",
		"request_cancelled",
		"broker_auth_peer_credentials_required",
		"broker_validation_request_id_missing",
		"broker_validation_schema_invalid",
		"broker_validation_payload_base64_invalid",
		"broker_validation_data_class_invalid",
		"broker_not_found_artifact",
		"broker_limit_message_size_exceeded",
		"broker_limit_structural_complexity_exceeded",
		"broker_limit_in_flight_exceeded",
		"broker_limit_policy_rejected",
		"broker_timeout_request_deadline_exceeded",
		"broker_approval_state_invalid",
	)
}

func assertPolicyRegistryCodes(t *testing.T) {
	t.Helper()

	policyRegistry := loadRegistry(t, schemaPath(t, "registries/policy_reason_code.registry.json"))
	assertRegistryContainsCodes(t, policyRegistry,
		"deny_by_default",
		"allow_manifest_opt_in",
		"approval_required",
		"artifact_flow_denied",
		"unapproved_excerpt_egress_denied",
		"approved_excerpt_revoked",
		"artifact_quota_exceeded",
	)
}

func assertAuditRegistryCodes(t *testing.T) {
	t.Helper()

	auditRegistry := loadRegistry(t, schemaPath(t, "registries/audit_event_type.registry.json"))
	assertRegistryContainsCodes(t, auditRegistry,
		"session_open",
		"model_egress",
		"auth_egress",
		"artifact_flow_blocked",
		"artifact_promotion_action",
		"artifact_quota_violation",
		"artifact_retention_action",
		"audit_segment_imported",
		"audit_segment_restored",
		"secrets_lease_acquired",
		"secrets_lease_released",
		"isolate_session_started",
		"isolate_session_bound",
	)
	assertAuditEventContractCatalogCoverage(t, auditRegistry)
}

func assertAuditEventContractCatalogCoverage(t *testing.T, auditRegistry registryFile) {
	t.Helper()

	type auditEventContractCatalogFixture struct {
		Entries []struct {
			AuditEventType string `json:"audit_event_type"`
		} `json:"entries"`
	}

	var catalog auditEventContractCatalogFixture
	loadJSON(t, fixturePath(t, "schema/audit-event-contract-catalog.valid.json"), &catalog)

	if len(catalog.Entries) == 0 {
		t.Fatal("audit event contract catalog fixture must include at least one entry")
	}

	seenCatalogTypes := map[string]struct{}{}
	for _, entry := range catalog.Entries {
		if entry.AuditEventType == "" {
			t.Fatal("audit event contract catalog entry must include audit_event_type")
		}
		if _, exists := seenCatalogTypes[entry.AuditEventType]; exists {
			t.Fatalf("audit event contract catalog reuses audit_event_type %q", entry.AuditEventType)
		}
		seenCatalogTypes[entry.AuditEventType] = struct{}{}
		assertRegistryCode(t, auditRegistry, entry.AuditEventType)
	}

	for _, code := range auditRegistry.Codes {
		if _, ok := seenCatalogTypes[code.Code]; !ok {
			t.Fatalf("audit event contract catalog missing registry code %q", code.Code)
		}
	}
}

func assertApprovalRegistryCodes(t *testing.T) {
	t.Helper()

	approvalRegistry := loadRegistry(t, schemaPath(t, "registries/approval_trigger_code.registry.json"))
	assertRegistryContainsCodes(t, approvalRegistry,
		"stage_sign_off",
		"reduced_assurance_backend",
		"gate_override",
		"gateway_egress_scope_change",
		"out_of_workspace_write",
		"secret_access_lease",
		"dependency_install",
		"system_command_execution",
	)
}

func assertAuditReceiptRegistryCodes(t *testing.T) {
	t.Helper()

	auditReceiptRegistry := loadRegistry(t, schemaPath(t, "registries/audit_receipt_kind.registry.json"))
	assertRegistryContainsCodes(t, auditReceiptRegistry,
		"anchor",
		"import",
		"restore",
		"reconciliation",
	)
}

func assertAuditVerificationReasonRegistryCodes(t *testing.T) {
	t.Helper()

	auditVerificationRegistry := loadRegistry(t, schemaPath(t, "registries/audit_verification_reason_code.registry.json"))
	assertRegistryContainsCodes(t, auditVerificationRegistry,
		"segment_frame_digest_mismatch",
		"segment_frame_byte_length_mismatch",
		"segment_file_hash_mismatch",
		"segment_merkle_root_mismatch",
		"segment_seal_invalid",
		"segment_seal_chain_mismatch",
		"stream_sequence_gap",
		"stream_sequence_rollback_or_duplicate",
		"stream_previous_hash_mismatch",
		"detached_signature_invalid",
		"signer_evidence_missing",
		"signer_evidence_invalid",
		"signer_historically_inadmissible",
		"signer_currently_revoked_or_compromised",
		"event_contract_mismatch",
		"event_contract_missing",
		"import_restore_provenance_inconsistent",
		"receipt_invalid",
		"anchor_receipt_missing",
		"anchor_receipt_invalid",
		"segment_lifecycle_inconsistent",
		"storage_posture_degraded",
		"storage_posture_invalid",
	)
}

func assertRegistryContainsCodes(t *testing.T, registry registryFile, codes ...string) {
	t.Helper()

	for _, code := range codes {
		assertRegistryCode(t, registry, code)
	}
}

func assertManifestMetadata(t *testing.T, manifest manifestFile) {
	t.Helper()

	if manifest.BundleID != bundleID {
		t.Fatalf("bundle_id = %q, want %q", manifest.BundleID, bundleID)
	}
	if manifest.BundleVersion == "" {
		t.Fatal("bundle_version must be non-empty")
	}
	if manifest.JSONSchemaDraft != "2020-12" {
		t.Fatalf("json_schema_draft = %q, want 2020-12", manifest.JSONSchemaDraft)
	}
	if manifest.RuntimeSchemaPrefix != runtimeSchemaPrefix {
		t.Fatalf("runtime_schema_prefix = %q, want %q", manifest.RuntimeSchemaPrefix, runtimeSchemaPrefix)
	}
	if manifest.Canonicalization != "RFC8785-JCS" {
		t.Fatalf("canonicalization = %q, want RFC8785-JCS", manifest.Canonicalization)
	}

	reqs := manifest.TopLevelObjectRequirements
	if !reqs.RequireSchemaID {
		t.Fatal("top-level objects must require schema_id")
	}
	if !reqs.RequireSchemaVersion {
		t.Fatal("top-level objects must require schema_version")
	}
	if reqs.UnknownSchemaPosture != "fail_closed" {
		t.Fatalf("unknown_schema_posture = %q, want fail_closed", reqs.UnknownSchemaPosture)
	}
}

func assertSchemaManifestEntries(t *testing.T, manifest manifestFile) {
	t.Helper()

	seenIDs := map[string]string{}
	for _, entry := range manifest.SchemaFiles {
		entry := entry
		t.Run(entry.Path, func(t *testing.T) {
			assertUniqueSchemaID(t, seenIDs, entry)
			assertSchemaManifestEntry(t, manifest, entry)
		})
	}
}

func assertUniqueSchemaID(t *testing.T, seenIDs map[string]string, entry schemaManifestEntry) {
	t.Helper()

	if previous, ok := seenIDs[entry.SchemaID]; ok {
		t.Fatalf("duplicate schema_id %q in %q and %q", entry.SchemaID, previous, entry.Path)
	}
	seenIDs[entry.SchemaID] = entry.Path
}

func assertSchemaManifestEntry(t *testing.T, manifest manifestFile, entry schemaManifestEntry) {
	t.Helper()

	if !strings.HasPrefix(entry.SchemaID, manifest.RuntimeSchemaPrefix) {
		t.Fatalf("schema_id %q does not use runtime prefix %q", entry.SchemaID, manifest.RuntimeSchemaPrefix)
	}
	if entry.SchemaVersion == "" {
		t.Fatalf("schema_version for %q must be non-empty", entry.Path)
	}
	if entry.Owner != "protocol" {
		t.Fatalf("owner for %q = %q, want protocol", entry.Path, entry.Owner)
	}
	if entry.Status != "mvp" && entry.Status != "reserved" {
		t.Fatalf("status for %q = %q, want mvp or reserved", entry.Path, entry.Status)
	}
	if requiresPlaceholderNote(entry.SchemaID) && strings.TrimSpace(entry.Note) == "" {
		t.Fatalf("schema %q must carry a manifest note explaining its placeholder scope", entry.SchemaID)
	}

	schema := loadJSONMap(t, schemaPath(t, entry.Path))
	assertTopLevelSchemaDocument(t, entry, schema)
}

func assertTopLevelSchemaDocument(t *testing.T, entry schemaManifestEntry, schema map[string]any) {
	t.Helper()

	if got := stringValue(t, schema, "$schema"); got != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("$schema for %q = %q, want draft 2020-12", entry.Path, got)
	}
	if got := stringValue(t, schema, "$id"); got == "" {
		t.Fatalf("$id for %q must be non-empty", entry.Path)
	}
	if got := stringValue(t, schema, "type"); got != "object" {
		t.Fatalf("type for %q = %q, want object", entry.Path, got)
	}
	if boolValue(t, schema, "additionalProperties") {
		t.Fatalf("additionalProperties for %q must be false", entry.Path)
	}
	if !hasNumber(schema, "maxProperties") {
		t.Fatalf("schema %q must declare maxProperties", entry.Path)
	}

	required := stringSliceValue(t, schema, "required")
	assertContains(t, required, "schema_id")
	assertContains(t, required, "schema_version")

	properties := objectValue(t, schema, "properties")
	assertConst(t, properties, "schema_id", entry.SchemaID)
	assertConst(t, properties, "schema_version", entry.SchemaVersion)
}

func collectRegistryData(t *testing.T, manifest manifestFile) ([]string, map[string]map[string]struct{}) {
	t.Helper()

	seenNames := map[string]struct{}{}
	seenNamespaces := map[string]struct{}{}
	registryNames := make([]string, 0, len(manifest.Registries))
	codesByRegistry := map[string]map[string]struct{}{}

	for _, entry := range manifest.Registries {
		entry := entry
		t.Run(entry.Path, func(t *testing.T) {
			assertUniqueRegistryManifest(t, entry, seenNames, seenNamespaces)
			registryNames = append(registryNames, entry.Name)
			codesByRegistry[entry.Name] = assertRegistryManifestEntry(t, entry)
		})
	}

	return registryNames, codesByRegistry
}

func assertUniqueRegistryManifest(t *testing.T, entry registryManifest, seenNames map[string]struct{}, seenNamespaces map[string]struct{}) {
	t.Helper()

	if _, ok := seenNames[entry.Name]; ok {
		t.Fatalf("duplicate registry name %q", entry.Name)
	}
	seenNames[entry.Name] = struct{}{}

	if _, ok := seenNamespaces[entry.Namespace]; ok {
		t.Fatalf("duplicate registry namespace %q", entry.Namespace)
	}
	seenNamespaces[entry.Namespace] = struct{}{}
}

func assertRegistryManifestEntry(t *testing.T, entry registryManifest) map[string]struct{} {
	t.Helper()

	if entry.DocumentationOwner != "protocol" {
		t.Fatalf("documentation_owner for %q = %q, want protocol", entry.Path, entry.DocumentationOwner)
	}
	if entry.Status != "mvp" {
		t.Fatalf("status for %q = %q, want mvp", entry.Path, entry.Status)
	}

	registry := loadRegistry(t, schemaPath(t, entry.Path))
	assertRegistryDocumentMetadata(t, entry, registry)
	return assertRegistryCodes(t, entry.Name, registry)
}

func assertRegistryDocumentMetadata(t *testing.T, entry registryManifest, registry registryFile) {
	t.Helper()

	if registry.RegistryName != entry.Name {
		t.Fatalf("registry_name for %q = %q, want %q", entry.Path, registry.RegistryName, entry.Name)
	}
	if registry.Namespace != entry.Namespace {
		t.Fatalf("namespace for %q = %q, want %q", entry.Path, registry.Namespace, entry.Namespace)
	}
	if registry.DocumentationOwner != entry.DocumentationOwner {
		t.Fatalf("documentation_owner for %q = %q, want %q", entry.Path, registry.DocumentationOwner, entry.DocumentationOwner)
	}
	if registry.Status != entry.Status {
		t.Fatalf("status for %q = %q, want %q", entry.Path, registry.Status, entry.Status)
	}
	if strings.TrimSpace(registry.Description) == "" {
		t.Fatalf("registry %q must have a non-empty description", entry.Name)
	}
}

func assertRegistryCodes(t *testing.T, registryName string, registry registryFile) map[string]struct{} {
	t.Helper()

	seenCodes := map[string]struct{}{}
	for _, code := range registry.Codes {
		if code.Code == "" {
			t.Fatalf("registry %q has empty code", registryName)
		}
		if _, ok := seenCodes[code.Code]; ok {
			t.Fatalf("registry %q reuses code %q", registryName, code.Code)
		}
		if strings.TrimSpace(code.Summary) == "" {
			t.Fatalf("registry %q code %q must have a non-empty summary", registryName, code.Code)
		}
		seenCodes[code.Code] = struct{}{}
	}
	return seenCodes
}

func assertManifestFileSet(t *testing.T, entries []schemaManifestEntry, dir string, suffix string) {
	t.Helper()

	manifestPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Path, dir+"/") {
			t.Fatalf("manifest path %q must stay under %s/", entry.Path, dir)
		}
		if !strings.HasSuffix(entry.Path, suffix) {
			t.Fatalf("manifest path %q must end with %q", entry.Path, suffix)
		}
		_ = schemaPath(t, entry.Path)
		manifestPaths = append(manifestPaths, entry.Path)
	}

	actualPaths := listedFiles(t, schemaRoot(), dir, suffix)
	assertSameStringSet(t, manifestPaths, actualPaths)
}

func assertManifestRegistryFileSet(t *testing.T, entries []registryManifest) {
	t.Helper()

	manifestPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Path, "registries/") {
			t.Fatalf("registry path %q must stay under registries/", entry.Path)
		}
		if !strings.HasSuffix(entry.Path, ".registry.json") {
			t.Fatalf("registry path %q must end with .registry.json", entry.Path)
		}
		_ = schemaPath(t, entry.Path)
		manifestPaths = append(manifestPaths, entry.Path)
	}

	actualPaths := listedFiles(t, schemaRoot(), "registries", ".registry.json")
	assertSameStringSet(t, manifestPaths, actualPaths)
}

func listedFiles(t *testing.T, root string, dir string, suffix string) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		t.Fatalf("ReadDir(%q) returned error: %v", filepath.Join(root, dir), err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), suffix) {
			files = append(files, path.Join(dir, entry.Name()))
		}
	}

	return files
}

func assertSameStringSet(t *testing.T, got []string, want []string) {
	t.Helper()

	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("set size mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("set mismatch: got %v, want %v", got, want)
		}
	}
}

func assertManifestSchemaVersion(t *testing.T, manifest manifestFile, schemaID string, wantVersion string) {
	t.Helper()

	for _, entry := range manifest.SchemaFiles {
		if entry.SchemaID == schemaID {
			if entry.SchemaVersion != wantVersion {
				t.Fatalf("schema_version for %q = %q, want %q", schemaID, entry.SchemaVersion, wantVersion)
			}
			return
		}
	}

	t.Fatalf("schema_id %q not found in manifest", schemaID)
}
