package protocolschema

import (
	"strconv"
	"testing"
)

func TestSchemasStayWithinMVPAuthoringProfile(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))

	for filePath, schemaDoc := range bundle.SchemaDocs {
		filePath := filePath
		schemaDoc := schemaDoc
		t.Run(filePath, func(t *testing.T) {
			assertNoForbiddenSchemaKeywords(t, filePath, schemaDoc)
		})
	}
}

func assertNoForbiddenSchemaKeywords(t *testing.T, location string, node map[string]any) {
	t.Helper()

	if hasKey(node, "patternProperties") {
		t.Fatalf("%s must not use patternProperties in the MVP schema profile", location)
	}
	if hasKey(node, "propertyNames") {
		t.Fatalf("%s must not use propertyNames in the MVP schema profile", location)
	}
	if schemaType, ok := optionalStringValue(node, "type"); ok && schemaType == "number" {
		t.Fatalf("%s must not use JSON number types in the MVP schema profile; use integer or string", location)
	}
	visitSchemaChild(t, location, node, "properties")
	visitSchemaChild(t, location, node, "$defs")
	visitSchemaChild(t, location, node, "items")
	visitSchemaChild(t, location, node, "additionalProperties")
	visitSchemaAlternatives(t, location, node, "allOf")
	visitSchemaAlternatives(t, location, node, "anyOf")
	visitSchemaAlternatives(t, location, node, "oneOf")
	visitSchemaChild(t, location, node, "if")
	visitSchemaChild(t, location, node, "then")
	visitSchemaChild(t, location, node, "else")
	visitSchemaChild(t, location, node, "not")
}

func visitSchemaChild(t *testing.T, location string, node map[string]any, key string) {
	t.Helper()

	children, ok := optionalObjectValue(node, key)
	if !ok {
		return
	}
	if key == "properties" || key == "$defs" {
		for _, childKey := range sortedKeys(children) {
			assertNoForbiddenSchemaKeywords(t, location+"."+key+"."+childKey, objectFromAny(t, location+"."+key+"."+childKey, children[childKey]))
		}
		return
	}

	assertNoForbiddenSchemaKeywords(t, location+"."+key, children)
}

func visitSchemaAlternatives(t *testing.T, location string, node map[string]any, key string) {
	t.Helper()

	items, ok := node[key].([]any)
	if !ok {
		return
	}
	for index, item := range items {
		childLocation := location + "." + key + "[" + strconv.Itoa(index) + "]"
		assertNoForbiddenSchemaKeywords(t, childLocation, objectFromAny(t, childLocation, item))
	}
}
