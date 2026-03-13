package protocolschema

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"testing"
)

type resolvedSchemaRef struct {
	FilePath string
	Location string
	Node     map[string]any
}

func assertReferencedDefinitions(t *testing.T, currentFile string, node map[string]any, schemaDocs map[string]map[string]any, seen map[string]struct{}) {
	t.Helper()

	assertReferencedRef(t, currentFile, node, schemaDocs, seen)
	assertReferencedProperties(t, currentFile, node, schemaDocs, seen)
	assertReferencedDefs(t, currentFile, node, schemaDocs, seen)
	assertReferencedItems(t, currentFile, node, schemaDocs, seen)
}

func assertReferencedRef(t *testing.T, currentFile string, node map[string]any, schemaDocs map[string]map[string]any, seen map[string]struct{}) {
	t.Helper()

	ref, ok := optionalStringValue(node, "$ref")
	if !ok {
		return
	}

	resolved := resolveSchemaRef(t, currentFile, ref, schemaDocs)
	if _, ok := seen[resolved.Location]; ok {
		return
	}

	seen[resolved.Location] = struct{}{}
	assertSchemaNodeInvariants(t, resolved.Location, resolved.Node, false)
	assertReferencedDefinitions(t, resolved.FilePath, resolved.Node, schemaDocs, seen)
}

func assertReferencedProperties(t *testing.T, currentFile string, node map[string]any, schemaDocs map[string]map[string]any, seen map[string]struct{}) {
	t.Helper()

	properties, ok := optionalObjectValue(node, "properties")
	if !ok {
		return
	}

	for _, key := range sortedKeys(properties) {
		child := objectFromAny(t, currentFile+"."+key, properties[key])
		assertReferencedDefinitions(t, currentFile, child, schemaDocs, seen)
	}
}

func assertReferencedDefs(t *testing.T, currentFile string, node map[string]any, schemaDocs map[string]map[string]any, seen map[string]struct{}) {
	t.Helper()

	defs, ok := optionalObjectValue(node, "$defs")
	if !ok {
		return
	}

	for _, key := range sortedKeys(defs) {
		child := objectFromAny(t, currentFile+".$defs."+key, defs[key])
		assertReferencedDefinitions(t, currentFile, child, schemaDocs, seen)
	}
}

func assertReferencedItems(t *testing.T, currentFile string, node map[string]any, schemaDocs map[string]map[string]any, seen map[string]struct{}) {
	t.Helper()

	items, ok := optionalObjectValue(node, "items")
	if !ok {
		return
	}

	assertReferencedDefinitions(t, currentFile, items, schemaDocs, seen)
}

func resolveSchemaRef(t *testing.T, currentFile string, ref string, schemaDocs map[string]map[string]any) resolvedSchemaRef {
	t.Helper()

	refPath, fragment := splitSchemaRef(currentFile, ref)
	doc, ok := schemaDocs[refPath]
	if !ok {
		t.Fatalf("reference %q from %q resolved to unknown schema file %q", ref, currentFile, refPath)
	}

	return resolveSchemaFragment(t, refPath, fragment, doc, currentFile, ref)
}

func splitSchemaRef(currentFile string, ref string) (string, string) {
	refPath := currentFile
	fragment := ""

	hashIndex := strings.IndexByte(ref, '#')
	if hashIndex < 0 {
		return path.Clean(path.Join(path.Dir(currentFile), ref)), fragment
	}
	if hashIndex > 0 {
		refPath = path.Clean(path.Join(path.Dir(currentFile), ref[:hashIndex]))
	}
	fragment = ref[hashIndex+1:]
	return refPath, fragment
}

func resolveSchemaFragment(t *testing.T, refPath string, fragment string, doc map[string]any, currentFile string, ref string) resolvedSchemaRef {
	t.Helper()

	resolvedNode := any(doc)
	location := refPath
	if fragment != "" {
		resolvedNode = resolveJSONPointer(t, doc, fragment)
		location = fmt.Sprintf("%s#%s", refPath, fragment)
	}

	objectNode, ok := resolvedNode.(map[string]any)
	if !ok {
		t.Fatalf("reference %q from %q resolved to %T, want map[string]any", ref, currentFile, resolvedNode)
	}

	return resolvedSchemaRef{FilePath: refPath, Location: location, Node: objectNode}
}

func resolveJSONPointer(t *testing.T, value any, pointer string) any {
	t.Helper()

	if pointer == "" {
		return value
	}
	if !strings.HasPrefix(pointer, "/") {
		t.Fatalf("json pointer %q must begin with '/'", pointer)
	}

	current := value
	for _, rawToken := range strings.Split(pointer[1:], "/") {
		current = resolveJSONPointerToken(t, current, pointer, rawToken)
	}
	return current
}

func resolveJSONPointerToken(t *testing.T, current any, pointer string, rawToken string) any {
	t.Helper()

	token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
	switch typed := current.(type) {
	case map[string]any:
		next, ok := typed[token]
		if !ok {
			t.Fatalf("json pointer %q segment %q not found", pointer, token)
		}
		return next
	case []any:
		index, err := strconv.Atoi(token)
		if err != nil || index < 0 || index >= len(typed) {
			t.Fatalf("json pointer %q segment %q is not a valid array index", pointer, token)
		}
		return typed[index]
	default:
		t.Fatalf("json pointer %q cannot descend into %T", pointer, current)
		return nil
	}
}
