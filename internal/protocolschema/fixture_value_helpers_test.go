package protocolschema

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

func requiredArrayValue(object map[string]any, key string) ([]any, error) {
	value, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q", key)
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want []any", key, value)
	}
	return items, nil
}

func optionalArrayValue(object map[string]any, key string) ([]any, error) {
	value, ok := object[key]
	if !ok {
		return []any{}, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want []any", key, value)
	}
	return items, nil
}

func objectFromFixtureValue(value any, location string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s has type %T, want map[string]any", location, value)
	}
	return object, nil
}

func requiredObjectField(object map[string]any, key string) (map[string]any, error) {
	value, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q", key)
	}
	child, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want map[string]any", key, value)
	}
	return child, nil
}

func stringField(object map[string]any, key string) (string, error) {
	value, ok := object[key]
	if !ok {
		return "", fmt.Errorf("missing key %q", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("key %q has type %T, want string", key, value)
	}
	return text, nil
}

func integerField(object map[string]any, key string) (int64, error) {
	value, ok := object[key]
	if !ok {
		return 0, fmt.Errorf("missing key %q", key)
	}
	return integerValue(value, key)
}

func boolField(object map[string]any, key string) (bool, error) {
	value, ok := object[key]
	if !ok {
		return false, fmt.Errorf("missing key %q", key)
	}
	flag, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("key %q has type %T, want bool", key, value)
	}
	return flag, nil
}

func integerValue(value any, location string) (int64, error) {
	switch typed := value.(type) {
	case json.Number:
		return canonicalIntegerFromText(typed.String(), location)
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return 0, fmt.Errorf("%s must be a finite integer", location)
		}
		if typed != float64(int64(typed)) {
			return 0, fmt.Errorf("%s must be an integer", location)
		}
		text := strconv.FormatInt(int64(typed), 10)
		return canonicalIntegerFromText(text, location)
	default:
		return 0, fmt.Errorf("%s has type %T, want integer", location, value)
	}
}

func canonicalIntegerFromText(text string, location string) (int64, error) {
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s = %q is not a supported integer: %w", location, text, err)
	}
	return parsed, nil
}

func digestIdentityField(object map[string]any, key string) (string, error) {
	digest, err := requiredObjectField(object, key)
	if err != nil {
		return "", err
	}
	return digestIdentity(digest)
}

func digestIdentity(digest map[string]any) (string, error) {
	hashAlg, err := stringField(digest, "hash_alg")
	if err != nil {
		return "", err
	}
	hash, err := stringField(digest, "hash")
	if err != nil {
		return "", err
	}
	return hashAlg + ":" + hash, nil
}

func toolIdentity(tool map[string]any) (string, error) {
	toolName, err := stringField(tool, "tool_name")
	if err != nil {
		return "", err
	}
	argumentsSchemaID, err := stringField(tool, "arguments_schema_id")
	if err != nil {
		return "", err
	}
	argumentsSchemaVersion, err := stringField(tool, "arguments_schema_version")
	if err != nil {
		return "", err
	}
	return toolName + "|" + argumentsSchemaID + "|" + argumentsSchemaVersion, nil
}
