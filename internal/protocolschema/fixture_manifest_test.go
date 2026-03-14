package protocolschema

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var lowercaseSHA256Hex = regexp.MustCompile(`^[a-f0-9]{64}$`)

func TestFixtureManifestMatchesFixtureFiles(t *testing.T) {
	manifest := loadFixtureManifest(t)
	assertSameStringSet(t, manifestFixturePaths(manifest), listedFixtureFiles(t))
}

func TestFixtureDigestsUseLowercaseSHA256Hex(t *testing.T) {
	for _, rel := range listedFixtureFiles(t) {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			assertFixtureDigestValues(t, rel, loadJSONValue(t, fixturePath(t, rel)))
		})
	}
}

func manifestFixturePaths(manifest fixtureManifestFile) []string {
	paths := make([]string, 0, len(manifest.SchemaFixtures)+len(manifest.StreamSequenceFixtures)+len(manifest.RuntimeFixtures)+(2*len(manifest.CanonicalFixtures)))
	for _, entry := range manifest.SchemaFixtures {
		paths = append(paths, entry.FixturePath)
	}
	for _, entry := range manifest.StreamSequenceFixtures {
		paths = append(paths, entry.FixturePath)
	}
	for _, entry := range manifest.RuntimeFixtures {
		paths = append(paths, entry.FixturePath)
	}
	for _, entry := range manifest.CanonicalFixtures {
		paths = append(paths, entry.PayloadPath)
		if entry.ExpectValid {
			paths = append(paths, entry.CanonicalJSONPath)
		}
	}
	return paths
}

func listedFixtureFiles(t *testing.T) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(fixtureRoot(), func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(filePath) == "manifest.json" || !strings.HasSuffix(filePath, ".json") {
			return nil
		}
		rel, err := filepath.Rel(fixtureRoot(), filePath)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%q) returned error: %v", fixtureRoot(), err)
	}
	sort.Strings(files)
	return files
}

func assertFixtureDigestValues(t *testing.T, location string, value any) {
	t.Helper()

	switch typed := value.(type) {
	case []any:
		assertFixtureDigestArrayValues(t, location, typed)
	case map[string]any:
		assertFixtureDigestObjectValues(t, location, typed)
	}
}

func assertFixtureDigestArrayValues(t *testing.T, location string, values []any) {
	t.Helper()

	for index, item := range values {
		assertFixtureDigestValues(t, location+"["+strconv.Itoa(index)+"]", item)
	}
}

func assertFixtureDigestObjectValues(t *testing.T, location string, object map[string]any) {
	t.Helper()

	assertDigestShape(t, location, object)
	for _, key := range sortedKeys(object) {
		assertFixtureDigestValues(t, location+"."+key, object[key])
	}
}

func assertDigestShape(t *testing.T, location string, object map[string]any) {
	t.Helper()

	hashAlgValue, hasHashAlg := object["hash_alg"]
	hashValue, hasHash := object["hash"]
	if !hasHashAlg && !hasHash {
		return
	}
	hashAlg, ok := hashAlgValue.(string)
	if !ok {
		t.Fatalf("%s hash_alg has type %T, want string", location, hashAlgValue)
	}
	hash, ok := hashValue.(string)
	if !ok {
		t.Fatalf("%s hash has type %T, want string", location, hashValue)
	}
	if hashAlg != "sha256" {
		t.Fatalf("%s hash_alg = %q, want sha256", location, hashAlg)
	}
	if !lowercaseSHA256Hex.MatchString(hash) {
		t.Fatalf("%s hash = %q, want 64 lowercase hex characters", location, hash)
	}
}
