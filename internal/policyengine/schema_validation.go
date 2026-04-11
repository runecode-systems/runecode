package policyengine

import (
	"encoding/json"
	"fmt"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	compiledSchemaCache   = map[string]*jsonschema.Schema{}
	compiledSchemaCacheMu sync.RWMutex
)

func validateObjectPayloadAgainstSchema(payload []byte, schemaPath string) error {
	schema, err := compiledObjectSchema(schemaPath)
	if err != nil {
		return err
	}
	value := map[string]any{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate payload against %q: %w", schemaPath, err)
	}
	return nil
}

func ValidateObjectPayloadAgainstSchema(payload []byte, schemaPath string) error {
	return validateObjectPayloadAgainstSchema(payload, schemaPath)
}

func compiledObjectSchema(schemaPath string) (*jsonschema.Schema, error) {
	compiledSchemaCacheMu.RLock()
	if schema, ok := compiledSchemaCache[schemaPath]; ok {
		compiledSchemaCacheMu.RUnlock()
		return schema, nil
	}
	compiledSchemaCacheMu.RUnlock()
	bundle, err := schemaBundle()
	if err != nil {
		return nil, err
	}
	doc, ok := bundle.schemaDocs[schemaPath]
	if !ok {
		return nil, fmt.Errorf("schema document %q not found", schemaPath)
	}
	id, err := stringFieldValue(doc, "$id")
	if err != nil {
		return nil, fmt.Errorf("schema document %q: %w", schemaPath, err)
	}
	schema, err := bundle.compiler.Compile(id)
	if err != nil {
		return nil, fmt.Errorf("compile schema %q: %w", schemaPath, err)
	}
	compiledSchemaCacheMu.Lock()
	compiledSchemaCache[schemaPath] = schema
	compiledSchemaCacheMu.Unlock()
	return schema, nil
}
