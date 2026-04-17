package policyengine

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSchemaBundlePrecompilesSchemas(t *testing.T) {
	bundle, err := schemaBundle()
	if err != nil {
		t.Fatalf("schemaBundle returned error: %v", err)
	}
	if len(bundle.compiledSchemas) == 0 {
		t.Fatal("schemaBundle returned no compiled schemas")
	}
	if _, ok := bundle.compiledSchemas[roleManifestSchemaPath]; !ok {
		t.Fatalf("schemaBundle missing compiled schema for %q", roleManifestSchemaPath)
	}
	if len(bundle.compiledSchemas) != len(bundle.schemaDocs) {
		t.Fatalf("compiled schema count mismatch: got %d compiled schemas for %d schema docs", len(bundle.compiledSchemas), len(bundle.schemaDocs))
	}
}

func TestValidateObjectPayloadAgainstSchemaConcurrentAccess(t *testing.T) {
	payload := testManifestInput(t, validRoleManifestPayload(), "").Payload

	const workers = 64
	const iterations = 50

	start := make(chan struct{})
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				if err := validateObjectPayloadAgainstSchema(payload, roleManifestSchemaPath); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("validateObjectPayloadAgainstSchema returned error: %v", err)
	}
}

func TestSchemaBundleRetriesAfterTransientLoadFailure(t *testing.T) {
	schemaBundleMu.Lock()
	loadedBundle = compiledSchemaBundle{}
	bundleLoaded = false
	schemaBundleMu.Unlock()

	root := os.Getenv("RUNE_REPO_ROOT")
	t.Cleanup(func() {
		_ = os.Setenv("RUNE_REPO_ROOT", root)
		schemaBundleMu.Lock()
		loadedBundle = compiledSchemaBundle{}
		bundleLoaded = false
		schemaBundleMu.Unlock()
	})

	badRoot := filepath.Join(t.TempDir(), "missing-repo")
	if err := os.Setenv("RUNE_REPO_ROOT", badRoot); err != nil {
		t.Fatalf("Setenv returned error: %v", err)
	}
	if _, err := schemaBundle(); err == nil {
		t.Fatal("expected schemaBundle to fail for invalid repo root")
	}

	if err := os.Setenv("RUNE_REPO_ROOT", "/home/zeb/code/runecode-systems/runecode"); err != nil {
		t.Fatalf("Setenv returned error: %v", err)
	}
	if _, err := schemaBundle(); err != nil {
		t.Fatalf("expected schemaBundle retry to recover after transient failure, got %v", err)
	}
}
