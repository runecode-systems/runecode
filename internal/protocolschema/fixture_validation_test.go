package protocolschema

import (
	"os"
	"strings"
	"testing"
)

func TestSchemaFixturesValidateAgainstManifestDefinedSchemas(t *testing.T) {
	manifest := loadFixtureManifest(t)
	bundle := newCompiledBundle(t, loadManifest(t))

	for _, entry := range manifest.SchemaFixtures {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			schema := mustCompileObjectSchema(t, bundle, entry.SchemaPath)
			fixture := loadJSONMap(t, fixturePath(t, entry.FixturePath))
			err := schema.Validate(fixture)
			assertValidationOutcome(t, err, !entry.ExpectValid)
		})
	}
}

func TestStreamSequenceFixturesValidateSchemaAndRuntimeRules(t *testing.T) {
	manifest := loadFixtureManifest(t)
	bundle := newCompiledBundle(t, loadManifest(t))

	for _, entry := range manifest.StreamSequenceFixtures {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			assertStreamSequenceFixture(t, bundle, entry)
		})
	}
}

func assertStreamSequenceFixture(t *testing.T, bundle compiledBundle, entry streamSequenceFixtureEntry) {
	t.Helper()

	schema := mustCompileObjectSchema(t, bundle, entry.EventSchemaPath)
	events := loadJSONArray(t, fixturePath(t, entry.FixturePath))
	schemaErr := validateStreamEventSchemas(schema, events)
	runtimeErr := validateStreamSequence(events)
	hasErr := schemaErr != nil || runtimeErr != nil

	if entry.ExpectValid && hasErr {
		t.Fatalf("fixture failed: schemaErr=%v runtimeErr=%v", schemaErr, runtimeErr)
	}
	if !entry.ExpectValid && !hasErr {
		t.Fatal("fixture unexpectedly passed stream validation")
	}
}

func validateStreamEventSchemas(schema interface{ Validate(any) error }, events []any) error {
	for _, item := range events {
		if err := schema.Validate(item); err != nil {
			return err
		}
	}
	return nil
}

func TestRuntimeInvariantFixturesValidateFailClosed(t *testing.T) {
	manifest := loadFixtureManifest(t)
	bundle := newCompiledBundle(t, loadManifest(t))

	for _, entry := range manifest.RuntimeFixtures {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			schema := mustCompileObjectSchema(t, bundle, entry.SchemaPath)
			fixture := loadJSONMap(t, fixturePath(t, entry.FixturePath))
			if err := schema.Validate(fixture); err != nil {
				t.Fatalf("fixture must be schema-valid before runtime checks: %v", err)
			}

			err := validateLLMRuntimeInvariant(entry.Rule, fixture)
			assertValidationOutcome(t, err, !entry.ExpectValid)
		})
	}
}

func TestCanonicalizationFixturesMatchGoldenBytesAndHashes(t *testing.T) {
	manifest := loadFixtureManifest(t)

	for _, entry := range manifest.CanonicalFixtures {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			assertCanonicalFixture(t, entry)
		})
	}
}

func assertCanonicalFixture(t *testing.T, entry canonicalFixtureEntry) {
	t.Helper()

	payload := loadJSONValue(t, fixturePath(t, entry.PayloadPath))
	canonical, err := canonicalizeJSONValue(payload)
	if !entry.ExpectValid {
		if err == nil {
			t.Fatal("canonicalizeJSONValue returned nil error, want failure")
		}
		return
	}
	if err != nil {
		t.Fatalf("canonicalizeJSONValue returned error: %v", err)
	}
	golden := loadCanonicalGolden(t, entry.CanonicalJSONPath)
	if canonical != golden {
		t.Fatalf("canonical JSON mismatch\n got: %s\nwant: %s", canonical, golden)
	}
	if got := canonicalSHA256Hex(canonical); got != entry.SHA256 {
		t.Fatalf("sha256 = %s, want %s", got, entry.SHA256)
	}
}

func loadCanonicalGolden(t *testing.T, rel string) string {
	t.Helper()

	goldenBytes, err := os.ReadFile(fixturePath(t, rel))
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", rel, err)
	}
	return strings.TrimSuffix(string(goldenBytes), "\n")
}
