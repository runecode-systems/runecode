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
	assertSchemaStatus(t, manifest, "runecode.protocol.v0.WorkflowDefinition", "mvp")
	assertSchemaStatus(t, manifest, "runecode.protocol.v0.ProcessDefinition", "mvp")
	assertReservedStatus(t, manifest, "runecode.protocol.v0.ProjectLifecycleOperatorDecisionPath")
}

func TestWorkflowSelectionAndProcessExecutableDefinitionsValidate(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	workflowSchema := mustCompileObjectSchema(t, bundle, "objects/WorkflowDefinition.schema.json")
	processSchema := mustCompileObjectSchema(t, bundle, "objects/ProcessDefinition.schema.json")

	workflow := workflowDefinitionFixtureWithRequiredGates()
	if err := workflowSchema.Validate(workflow); err != nil {
		t.Fatalf("workflow schema validation failed: %v", err)
	}

	process := processDefinitionFixtureWithRequiredGates()
	if err := processSchema.Validate(process); err != nil {
		t.Fatalf("process schema validation failed: %v", err)
	}
}

func TestRunPlanSchemaValidatesCompiledPlanShape(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	runPlanSchema := mustCompileObjectSchema(t, bundle, "objects/RunPlan.schema.json")
	runPlan := runPlanFixtureWithRequiredGates()

	if err := runPlanSchema.Validate(runPlan); err != nil {
		t.Fatalf("run plan schema validation failed: %v", err)
	}
}

func TestWorkflowDefinitionSchemaRejectsLegacyAndIncompleteSelectionBindings(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	workflowSchema := mustCompileObjectSchema(t, bundle, "objects/WorkflowDefinition.schema.json")

	t.Run("missing selected process hash", func(t *testing.T) {
		fixture := cloneFixtureMap(t, workflowDefinitionFixtureWithRequiredGates())
		delete(fixture, "selected_process_definition_hash")
		if err := workflowSchema.Validate(fixture); err == nil {
			t.Fatal("workflow schema validation unexpectedly passed without selected_process_definition_hash")
		}
	})

	t.Run("legacy process_definitions array", func(t *testing.T) {
		fixture := cloneFixtureMap(t, workflowDefinitionFixtureWithRequiredGates())
		fixture["process_definitions"] = []any{map[string]any{"process_id": "process_default"}}
		if err := workflowSchema.Validate(fixture); err == nil {
			t.Fatal("workflow schema validation unexpectedly passed with legacy process_definitions field")
		}
	})
}

func TestProcessDefinitionSchemaRejectsMissingDependencyEdges(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	processSchema := mustCompileObjectSchema(t, bundle, "objects/ProcessDefinition.schema.json")
	fixture := cloneFixtureMap(t, processDefinitionFixtureWithRequiredGates())
	delete(fixture, "dependency_edges")

	if err := processSchema.Validate(fixture); err == nil {
		t.Fatal("process schema validation unexpectedly passed without dependency_edges")
	}
}

func TestRunPlanSchemaRejectsLegacyGateOnlyShape(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	runPlanSchema := mustCompileObjectSchema(t, bundle, "objects/RunPlan.schema.json")
	fixture := cloneFixtureMap(t, runPlanFixtureWithRequiredGates())
	delete(fixture, "entries")

	if err := runPlanSchema.Validate(fixture); err == nil {
		t.Fatal("run plan schema validation unexpectedly passed without entries")
	}
}

func TestRunPlanSchemaRejectsInvalidEntryWaitKindsAndDependencyHandoffRequirements(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	runPlanSchema := mustCompileObjectSchema(t, bundle, "objects/RunPlan.schema.json")

	t.Run("missing waiting_approval", func(t *testing.T) {
		fixture := cloneFixtureMap(t, runPlanFixtureWithRequiredGates())
		entries := fixture["entries"].([]any)
		entry := entries[0].(map[string]any)
		entry["supported_wait_kinds"] = []any{"waiting_operator_input", "waiting_operator_input"}
		if err := runPlanSchema.Validate(fixture); err == nil {
			t.Fatal("run plan schema validation unexpectedly passed with invalid supported_wait_kinds")
		}
	})

	t.Run("dependency handoff required false", func(t *testing.T) {
		fixture := cloneFixtureMap(t, runPlanFixtureWithRequiredGates())
		entries := fixture["entries"].([]any)
		entry := entries[0].(map[string]any)
		entry["dependency_cache_handoffs"] = []any{
			map[string]any{
				"request_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)},
				"consumer_role":  "workspace-edit",
				"required":       false,
			},
		}
		if err := runPlanSchema.Validate(fixture); err == nil {
			t.Fatal("run plan schema validation unexpectedly passed with dependency handoff required=false")
		}
	})
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
	assertZKProofRegistryCodes(t)
	assertApprovalRegistryCodes(t)
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
