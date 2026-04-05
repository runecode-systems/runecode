package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestHelpAndUnknownCommand(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"--help"}, stdout, stderr); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage: runecode-broker") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
	err := run([]string{"not-a-command"}, stdout, stderr)
	if err == nil {
		t.Fatal("expected usage error for unknown command")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("unknown command error type = %T, want *usageError", err)
	}
}

func TestPutListHeadGetArtifactCLI(t *testing.T) {
	root := setBrokerServiceForTest(t)
	payloadPath := writeTempFile(t, "payload.txt", "hello artifact")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ref := putArtifactViaCLI(t, stdout, stderr, payloadPath, "spec_text", testDigest("1"))
	list := listArtifactsViaCLI(t, stdout, stderr)
	if len(list) != 1 {
		t.Fatalf("list-artifacts count = %d, want 1", len(list))
	}
	record := headArtifactViaCLI(t, stdout, stderr, ref.Digest)
	if record.Reference.Digest != ref.Digest {
		t.Fatalf("head digest = %q, want %q", record.Reference.Digest, ref.Digest)
	}
	outputPath := filepath.Join(t.TempDir(), "output.txt")
	getArtifactViaCLI(t, stdout, stderr, ref.Digest, outputPath)
	b, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("read get-artifact output error: %v", readErr)
	}
	if string(b) != "hello artifact" {
		t.Fatalf("get-artifact payload = %q, want hello artifact", string(b))
	}

	if _, err := os.Stat(filepath.Join(root, "state.json")); err != nil {
		t.Fatalf("expected broker state.json: %v", err)
	}
}

func TestPromotionFlowAndCheckFlowCLI(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	err := run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "unapproved_file_excerpts", "--digest", unapproved.Digest, "--egress"}, stdout, stderr)
	if err != artifacts.ErrUnapprovedEgressDenied {
		t.Fatalf("check-flow unapproved egress error = %v, want %v", err, artifacts.ErrUnapprovedEgressDenied)
	}
	approved := promoteViaCLI(t, stdout, stderr, unapproved.Digest)
	err = run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--digest", approved.Digest, "--egress"}, stdout, stderr)
	if err != artifacts.ErrApprovedEgressRequiresManifest {
		t.Fatalf("check-flow approved no opt-in error = %v, want %v", err, artifacts.ErrApprovedEgressRequiresManifest)
	}
	err = run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--digest", approved.Digest, "--egress", "--manifest-opt-in"}, stdout, stderr)
	if err != nil {
		t.Fatalf("check-flow approved with opt-in error: %v", err)
	}
}

func TestGCAndBackupCommands(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	payloadPath := writeTempFile(t, "tmp.txt", "tmp payload")
	err := run([]string{"put-artifact", "--file", payloadPath, "--content-type", "text/plain", "--data-class", "spec_text", "--provenance-hash", testDigest("3"), "--run-id", "run-1"}, stdout, stderr)
	if err != nil {
		t.Fatalf("put-artifact returned error: %v", err)
	}
	err = run([]string{"set-run-status", "--run-id", "run-1", "--status", "closed"}, stdout, stderr)
	if err != nil {
		t.Fatalf("set-run-status returned error: %v", err)
	}
	stdout.Reset()
	err = run([]string{"gc"}, stdout, stderr)
	if err != nil {
		t.Fatalf("gc returned error: %v", err)
	}
	result := artifacts.GCResult{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &result); unmarshalErr != nil {
		t.Fatalf("gc output parse error: %v", unmarshalErr)
	}

	backupPath := filepath.Join(t.TempDir(), "artifact-backup.json")
	err = run([]string{"export-backup", "--path", backupPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("export-backup returned error: %v", err)
	}
	err = run([]string{"restore-backup", "--path", backupPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("restore-backup returned error: %v", err)
	}
}

func writeTempFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp file error: %v", err)
	}
	return path
}

func putArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, path, dataClass, provenance string) artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	err := run([]string{"put-artifact", "--file", path, "--content-type", "text/plain", "--data-class", dataClass, "--provenance-hash", provenance}, stdout, stderr)
	if err != nil {
		t.Fatalf("put-artifact returned error: %v", err)
	}
	ref := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &ref); unmarshalErr != nil {
		t.Fatalf("put-artifact output parse error: %v", unmarshalErr)
	}
	return ref
}

func listArtifactsViaCLI(t *testing.T, stdout, stderr *bytes.Buffer) []artifacts.ArtifactRecord {
	t.Helper()
	stdout.Reset()
	err := run([]string{"list-artifacts"}, stdout, stderr)
	if err != nil {
		t.Fatalf("list-artifacts returned error: %v", err)
	}
	list := []artifacts.ArtifactRecord{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &list); unmarshalErr != nil {
		t.Fatalf("list-artifacts output parse error: %v", unmarshalErr)
	}
	return list
}

func headArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string) artifacts.ArtifactRecord {
	t.Helper()
	stdout.Reset()
	err := run([]string{"head-artifact", "--digest", digest}, stdout, stderr)
	if err != nil {
		t.Fatalf("head-artifact returned error: %v", err)
	}
	record := artifacts.ArtifactRecord{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &record); unmarshalErr != nil {
		t.Fatalf("head-artifact output parse error: %v", unmarshalErr)
	}
	return record
}

func getArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest, out string) {
	t.Helper()
	stdout.Reset()
	err := run([]string{"get-artifact", "--digest", digest, "--out", out}, stdout, stderr)
	if err != nil {
		t.Fatalf("get-artifact returned error: %v", err)
	}
}

func promoteViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string) artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	err := run([]string{"promote-excerpt", "--unapproved-digest", digest, "--approver", "human", "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err != nil {
		t.Fatalf("promote-excerpt returned error: %v", err)
	}
	approved := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &approved); unmarshalErr != nil {
		t.Fatalf("approved parse error: %v", unmarshalErr)
	}
	return approved
}

func setBrokerServiceForTest(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "store")
	brokerServiceFactory = func() (*brokerapi.Service, error) {
		return brokerapi.NewService(root)
	}
	t.Cleanup(func() {
		brokerServiceFactory = brokerService
	})
	return root
}

func TestBrokerServiceUsesTempFallbackWhenUserDirsUnavailable(t *testing.T) {
	originalFactory := brokerServiceFactory
	defer func() { brokerServiceFactory = originalFactory }()

	t.Setenv("HOME", "")
	if err := os.Unsetenv("XDG_CACHE_HOME"); err != nil {
		t.Fatalf("Unsetenv(XDG_CACHE_HOME) error: %v", err)
	}
	if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
		t.Fatalf("Unsetenv(XDG_CONFIG_HOME) error: %v", err)
	}

	root := defaultBrokerStoreRoot()
	if root == "" {
		t.Fatal("defaultBrokerStoreRoot returned empty path")
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("defaultBrokerStoreRoot = %q, want absolute path", root)
	}
	if !strings.Contains(filepath.ToSlash(root), "/runecode/artifact-store") {
		t.Fatalf("defaultBrokerStoreRoot = %q, want path containing runecode/artifact-store", root)
	}
	if _, err := brokerapi.NewService(root); err != nil {
		t.Fatalf("NewService(%q) error: %v", root, err)
	}
}

func testDigest(seed string) string {
	base := strings.Repeat(seed, 64)
	if len(base) > 64 {
		base = base[:64]
	}
	for len(base) < 64 {
		base += "0"
	}
	return "sha256:" + base
}
