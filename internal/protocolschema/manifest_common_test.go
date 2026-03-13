package protocolschema

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	bundleID            = "runecode.protocol.v0"
	runtimeSchemaPrefix = bundleID + "."
	manifestMetaPath    = "meta/manifest.schema.json"
	registryMetaPath    = "meta/registry.schema.json"
)

var (
	allowedDataClasses = map[string]struct{}{
		"public":    {},
		"sensitive": {},
		"secret":    {},
	}
	placeholderSchemaIDs = map[string]struct{}{
		"runecode.protocol.v0.ApprovalRequest":  {},
		"runecode.protocol.v0.ApprovalDecision": {},
		"runecode.protocol.v0.PolicyDecision":   {},
		"runecode.protocol.v0.Error":            {},
	}
)

type manifestFile struct {
	BundleID                   string                `json:"bundle_id"`
	BundleVersion              string                `json:"bundle_version"`
	JSONSchemaDraft            string                `json:"json_schema_draft"`
	RuntimeSchemaPrefix        string                `json:"runtime_schema_prefix"`
	Canonicalization           string                `json:"canonicalization"`
	TopLevelObjectRequirements topLevelRequirements  `json:"top_level_object_requirements"`
	SchemaFiles                []schemaManifestEntry `json:"schema_files"`
	Registries                 []registryManifest    `json:"registries"`
}

type topLevelRequirements struct {
	RequireSchemaID      bool   `json:"require_schema_id"`
	RequireSchemaVersion bool   `json:"require_schema_version"`
	UnknownSchemaPosture string `json:"unknown_schema_posture"`
}

type schemaManifestEntry struct {
	Path          string `json:"path"`
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	Owner         string `json:"owner"`
	Status        string `json:"status"`
	Note          string `json:"note"`
}

type registryManifest struct {
	Path               string `json:"path"`
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	DocumentationOwner string `json:"documentation_owner"`
	Status             string `json:"status"`
}

type registryFile struct {
	RegistryName       string         `json:"registry_name"`
	Namespace          string         `json:"namespace"`
	DocumentationOwner string         `json:"documentation_owner"`
	Status             string         `json:"status"`
	Description        string         `json:"description"`
	Codes              []registryCode `json:"codes"`
}

type registryCode struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
}

type compiledBundle struct {
	Compiler   *jsonschema.Compiler
	SchemaDocs map[string]map[string]any
}

func loadManifest(t *testing.T) manifestFile {
	t.Helper()

	var manifest manifestFile
	loadJSON(t, schemaPath(t, "manifest.json"), &manifest)
	return manifest
}

func loadRegistry(t *testing.T, filePath string) registryFile {
	t.Helper()

	var registry registryFile
	loadJSON(t, filePath, &registry)
	return registry
}

func newCompiledBundle(t *testing.T, manifest manifestFile) compiledBundle {
	t.Helper()

	compiler := jsonschema.NewCompiler()
	schemaDocs := make(map[string]map[string]any, len(manifest.SchemaFiles))

	for _, entry := range manifest.SchemaFiles {
		schemaDoc := loadJSONMap(t, schemaPath(t, entry.Path))
		schemaID := stringValue(t, schemaDoc, "$id")
		if err := compiler.AddResource(schemaID, schemaDoc); err != nil {
			t.Fatalf("AddResource(%q) returned error: %v", schemaID, err)
		}
		schemaDocs[entry.Path] = schemaDoc
	}

	return compiledBundle{Compiler: compiler, SchemaDocs: schemaDocs}
}

func newMetaCompiler(t *testing.T) *jsonschema.Compiler {
	t.Helper()

	compiler := jsonschema.NewCompiler()
	for _, metaFile := range []string{manifestMetaPath, registryMetaPath} {
		doc := loadJSONMap(t, metaPath(t, metaFile))
		docID := stringValue(t, doc, "$id")
		if err := compiler.AddResource(docID, doc); err != nil {
			t.Fatalf("AddResource(%q) returned error: %v", docID, err)
		}
	}

	return compiler
}

func mustCompileMetaSchema(t *testing.T, compiler *jsonschema.Compiler, filePath string) *jsonschema.Schema {
	t.Helper()

	doc := loadJSONMap(t, filePath)
	docID := stringValue(t, doc, "$id")
	schema, err := compiler.Compile(docID)
	if err != nil {
		t.Fatalf("Compile(%q) for %q returned error: %v", docID, filePath, err)
	}
	return schema
}

func mustCompileObjectSchema(t *testing.T, bundle compiledBundle, filePath string) *jsonschema.Schema {
	t.Helper()

	doc, ok := bundle.SchemaDocs[filePath]
	if !ok {
		t.Fatalf("schema document %q not found", filePath)
	}

	objID := stringValue(t, doc, "$id")
	schema, err := bundle.Compiler.Compile(objID)
	if err != nil {
		t.Fatalf("Compile(%q) for %q returned error: %v", objID, filePath, err)
	}
	return schema
}

func loadJSON(t *testing.T, filePath string, target any) {
	t.Helper()

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", filePath, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("Unmarshal(%q) returned error: %v", filePath, err)
	}
}

func loadJSONMap(t *testing.T, filePath string) map[string]any {
	t.Helper()

	var value map[string]any
	loadJSON(t, filePath, &value)
	return value
}

func schemaRoot() string {
	return filepath.Join("..", "..", "protocol", "schemas")
}

func metaPath(t *testing.T, rel string) string {
	t.Helper()

	return rootedSchemaPath(t, schemaRoot(), rel, "protocol/schemas")
}

func schemaPath(t *testing.T, rel string) string {
	t.Helper()

	return rootedSchemaPath(t, schemaRoot(), rel, "protocol/schemas")
}

func rootedSchemaPath(t *testing.T, root string, rel string, label string) string {
	t.Helper()

	if rel == "" {
		t.Fatalf("%s path must be non-empty", label)
	}

	if filepath.IsAbs(rel) || path.IsAbs(rel) {
		t.Fatalf("%s path %q must be relative", label, rel)
	}

	cleaned := path.Clean(rel)
	if cleaned != rel {
		t.Fatalf("%s path %q must already be clean; got %q", label, rel, cleaned)
	}

	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		t.Fatalf("%s path %q escapes %s", label, rel, label)
	}

	absPath := filepath.Join(root, filepath.FromSlash(cleaned))
	relToRoot, err := filepath.Rel(root, absPath)
	if err != nil {
		t.Fatalf("Rel(%q) returned error: %v", rel, err)
	}

	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		t.Fatalf("%s path %q escapes %s", label, rel, label)
	}

	return absPath
}

func stringValue(t *testing.T, object map[string]any, key string) string {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}

	stringValue, ok := value.(string)
	if !ok {
		t.Fatalf("key %q has type %T, want string", key, value)
	}

	return stringValue
}

func optionalStringValue(object map[string]any, key string) (string, bool) {
	value, ok := object[key]
	if !ok {
		return "", false
	}

	stringValue, ok := value.(string)
	return stringValue, ok
}

func boolValue(t *testing.T, object map[string]any, key string) bool {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}

	boolValue, ok := value.(bool)
	if !ok {
		t.Fatalf("key %q has type %T, want bool", key, value)
	}

	return boolValue
}

func stringSliceValue(t *testing.T, object map[string]any, key string) []string {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}

	items, ok := value.([]any)
	if !ok {
		t.Fatalf("key %q has type %T, want []any", key, value)
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		stringItem, ok := item.(string)
		if !ok {
			t.Fatalf("key %q has non-string item type %T", key, item)
		}
		result = append(result, stringItem)
	}

	return result
}

func objectValue(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}

	child, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("key %q has type %T, want map[string]any", key, value)
	}

	return child
}

func optionalObjectValue(object map[string]any, key string) (map[string]any, bool) {
	value, ok := object[key]
	if !ok {
		return nil, false
	}

	child, ok := value.(map[string]any)
	return child, ok
}

func objectFromAny(t *testing.T, location string, value any) map[string]any {
	t.Helper()

	child, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map[string]any", location, value)
	}

	return child
}

func hasKey(object map[string]any, key string) bool {
	_, ok := object[key]
	return ok
}

func hasNumber(object map[string]any, key string) bool {
	value, ok := object[key]
	if !ok {
		return false
	}
	_, ok = value.(float64)
	return ok
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()

	for _, value := range values {
		if value == want {
			return
		}
	}

	t.Fatalf("%q not found in %v", want, values)
}

func assertConst(t *testing.T, properties map[string]any, key string, want string) {
	t.Helper()

	property := objectValue(t, properties, key)
	if got := stringValue(t, property, "const"); got != want {
		t.Fatalf("const for property %q = %q, want %q", key, got, want)
	}
}

func assertReservedStatus(t *testing.T, manifest manifestFile, schemaID string) {
	t.Helper()

	for _, entry := range manifest.SchemaFiles {
		if entry.SchemaID == schemaID {
			if entry.Status != "reserved" {
				t.Fatalf("status for %q = %q, want reserved", schemaID, entry.Status)
			}
			return
		}
	}

	t.Fatalf("schema_id %q not found in manifest", schemaID)
}

func assertRegistryCode(t *testing.T, registry registryFile, want string) {
	t.Helper()

	for _, code := range registry.Codes {
		if code.Code == want {
			return
		}
	}

	t.Fatalf("registry %q missing code %q", registry.RegistryName, want)
}

func assertNoCodeOverlap(t *testing.T, codesByRegistry map[string]map[string]struct{}, left string, right string) {
	t.Helper()

	for code := range codesByRegistry[left] {
		if _, ok := codesByRegistry[right][code]; ok {
			t.Fatalf("registry code %q must not appear in both %q and %q", code, left, right)
		}
	}
}

func requiresPlaceholderNote(schemaID string) bool {
	_, ok := placeholderSchemaIDs[schemaID]
	return ok
}
