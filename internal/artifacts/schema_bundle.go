package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaRoot = "protocol/schemas"

type schemaBundleManifest struct {
	SchemaFiles []schemaBundleManifestEntry `json:"schema_files"`
}

type schemaBundleManifestEntry struct {
	Path string `json:"path"`
}

type compiledSchemaBundle struct {
	schemaDocs      map[string]map[string]any
	compiledSchemas map[string]*jsonschema.Schema
}

var (
	schemaBundleMu sync.RWMutex
	loadedBundle   compiledSchemaBundle
	bundleLoaded   bool
)

func schemaBundle() (compiledSchemaBundle, error) {
	schemaBundleMu.RLock()
	if bundleLoaded {
		bundle := loadedBundle
		schemaBundleMu.RUnlock()
		return bundle, nil
	}
	schemaBundleMu.RUnlock()

	schemaBundleMu.Lock()
	defer schemaBundleMu.Unlock()
	if bundleLoaded {
		return loadedBundle, nil
	}
	bundle, err := loadSchemaBundle()
	if err != nil {
		return compiledSchemaBundle{}, err
	}
	loadedBundle = bundle
	bundleLoaded = true
	return loadedBundle, nil
}

func loadSchemaBundle() (compiledSchemaBundle, error) {
	manifest := schemaBundleManifest{}
	manifestPath, err := schemaAbsolutePath("manifest.json")
	if err != nil {
		return compiledSchemaBundle{}, err
	}
	if err := loadJSONFile(manifestPath, &manifest); err != nil {
		return compiledSchemaBundle{}, err
	}
	schemaDocs, compiledSchemas, err := compileSchemaDocs(manifest.SchemaFiles)
	if err != nil {
		return compiledSchemaBundle{}, err
	}
	return compiledSchemaBundle{schemaDocs: schemaDocs, compiledSchemas: compiledSchemas}, nil
}

func compileSchemaDocs(entries []schemaBundleManifestEntry) (map[string]map[string]any, map[string]*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	schemaDocs, err := loadAndRegisterSchemaDocs(entries, compiler)
	if err != nil {
		return nil, nil, err
	}
	compiledSchemas, err := compileRegisteredSchemas(entries, schemaDocs, compiler)
	if err != nil {
		return nil, nil, err
	}
	return schemaDocs, compiledSchemas, nil
}

func loadAndRegisterSchemaDocs(entries []schemaBundleManifestEntry, compiler *jsonschema.Compiler) (map[string]map[string]any, error) {
	schemaDocs := map[string]map[string]any{}
	for _, entry := range entries {
		schemaDoc, err := loadSchemaDoc(entry.Path)
		if err != nil {
			return nil, err
		}
		if err := addSchemaDocResource(compiler, entry.Path, schemaDoc); err != nil {
			return nil, err
		}
		schemaDocs[entry.Path] = schemaDoc
	}
	return schemaDocs, nil
}

func compileRegisteredSchemas(entries []schemaBundleManifestEntry, schemaDocs map[string]map[string]any, compiler *jsonschema.Compiler) (map[string]*jsonschema.Schema, error) {
	compiledSchemas := map[string]*jsonschema.Schema{}
	for _, entry := range entries {
		schema, err := compileRegisteredSchema(entry.Path, schemaDocs, compiler)
		if err != nil {
			return nil, err
		}
		compiledSchemas[entry.Path] = schema
	}
	return compiledSchemas, nil
}

func compileRegisteredSchema(schemaPath string, schemaDocs map[string]map[string]any, compiler *jsonschema.Compiler) (*jsonschema.Schema, error) {
	doc, ok := schemaDocs[schemaPath]
	if !ok {
		return nil, fmt.Errorf("schema document %q not found", schemaPath)
	}
	id, err := stringFieldValue(doc, "$id")
	if err != nil {
		return nil, fmt.Errorf("schema document %q: %w", schemaPath, err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		return nil, fmt.Errorf("compile schema %q: %w", schemaPath, err)
	}
	return schema, nil
}

func loadSchemaDoc(schemaRelPath string) (map[string]any, error) {
	schemaPath, err := schemaAbsolutePath(schemaRelPath)
	if err != nil {
		return nil, err
	}
	schemaDoc := map[string]any{}
	if err := loadJSONFile(schemaPath, &schemaDoc); err != nil {
		return nil, err
	}
	return schemaDoc, nil
}

func addSchemaDocResource(compiler *jsonschema.Compiler, schemaPath string, schemaDoc map[string]any) error {
	schemaID, err := stringFieldValue(schemaDoc, "$id")
	if err != nil {
		return fmt.Errorf("schema document %q: %w", schemaPath, err)
	}
	if err := compiler.AddResource(schemaID, schemaDoc); err != nil {
		return fmt.Errorf("add schema resource %q: %w", schemaID, err)
	}
	return nil
}

func schemaAbsolutePath(rel string) (string, error) {
	if root := os.Getenv("RUNE_REPO_ROOT"); root != "" {
		return filepath.Abs(filepath.Join(root, schemaRoot, rel))
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(cwd, schemaRoot, rel)
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
		next := filepath.Dir(cwd)
		if next == cwd {
			break
		}
		cwd = next
	}
	return "", fmt.Errorf("unable to locate %s/%s from current directory", schemaRoot, rel)
}

func loadJSONFile(filePath string, target any) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %q: %w", filePath, err)
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("decode %q: %w", filePath, err)
	}
	return nil
}

func stringFieldValue(value map[string]any, key string) (string, error) {
	raw, ok := value[key]
	if !ok {
		return "", fmt.Errorf("missing key %q", key)
	}
	text, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("key %q has type %T, want string", key, raw)
	}
	return text, nil
}
