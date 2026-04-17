package artifacts

import (
	"bytes"
	"encoding/json"
	"fmt"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func validateObjectPayloadAgainstSchema(payload []byte, schemaPath string) error {
	return ValidateObjectPayloadAgainstSchema(payload, schemaPath)
}

func ValidateObjectPayloadAgainstSchema(payload []byte, schemaPath string) error {
	schema, err := compiledObjectSchema(schemaPath)
	if err != nil {
		return err
	}
	value := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate payload against %q: %w", schemaPath, err)
	}
	return nil
}

func compiledObjectSchema(schemaPath string) (*jsonschema.Schema, error) {
	bundle, err := schemaBundle()
	if err != nil {
		return nil, err
	}
	schema, ok := bundle.compiledSchemas[schemaPath]
	if !ok {
		return nil, fmt.Errorf("schema document %q not found", schemaPath)
	}
	return schema, nil
}
