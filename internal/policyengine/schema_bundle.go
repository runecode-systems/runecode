package policyengine

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
	compiler   *jsonschema.Compiler
	schemaDocs map[string]map[string]any
}

var (
	schemaBundleOnce sync.Once
	loadedBundle     compiledSchemaBundle
	bundleErr        error
)

func schemaBundle() (compiledSchemaBundle, error) {
	schemaBundleOnce.Do(func() {
		loadedBundle, bundleErr = loadSchemaBundle()
	})
	if bundleErr != nil {
		return compiledSchemaBundle{}, bundleErr
	}
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
	compiler, schemaDocs, err := compileSchemaDocs(manifest.SchemaFiles)
	if err != nil {
		return compiledSchemaBundle{}, err
	}
	return compiledSchemaBundle{compiler: compiler, schemaDocs: schemaDocs}, nil
}

func compileSchemaDocs(entries []schemaBundleManifestEntry) (*jsonschema.Compiler, map[string]map[string]any, error) {
	compiler := jsonschema.NewCompiler()
	schemaDocs := map[string]map[string]any{}
	for _, entry := range entries {
		schemaDoc, err := loadSchemaDoc(entry.Path)
		if err != nil {
			return nil, nil, err
		}
		if err := addSchemaDocResource(compiler, entry.Path, schemaDoc); err != nil {
			return nil, nil, err
		}
		schemaDocs[entry.Path] = schemaDoc
	}
	return compiler, schemaDocs, nil
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
