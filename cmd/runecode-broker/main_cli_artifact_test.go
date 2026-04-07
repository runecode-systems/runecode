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
	if record.Digest != ref.Digest {
		t.Fatalf("head digest = %q, want %q", record.Digest, ref.Digest)
	}
	outputPath := filepath.Join(t.TempDir(), "output.txt")
	getArtifactViaCLI(t, stdout, stderr, ref.Digest, "workspace", "model_gateway", "", false, outputPath)
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
	approvalRequestPath, approvalEnvelopePath, verifierRecords := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	seedTrustedVerifierForBrokerCLITest(t, verifierRecords)
	err := run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "unapproved_file_excerpts", "--digest", unapproved.Digest, "--egress"}, stdout, stderr)
	if err != artifacts.ErrUnapprovedEgressDenied {
		t.Fatalf("check-flow unapproved egress error = %v, want %v", err, artifacts.ErrUnapprovedEgressDenied)
	}
	approved := promoteViaCLI(t, stdout, stderr, unapproved.Digest, approvalRequestPath, approvalEnvelopePath)
	err = run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--digest", approved.Digest, "--egress"}, stdout, stderr)
	if err != artifacts.ErrApprovedEgressRequiresManifest {
		t.Fatalf("check-flow approved no opt-in error = %v, want %v", err, artifacts.ErrApprovedEgressRequiresManifest)
	}
	err = run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--digest", approved.Digest, "--egress", "--manifest-opt-in"}, stdout, stderr)
	if err != nil {
		t.Fatalf("check-flow approved with opt-in error: %v", err)
	}
}

func TestGetArtifactCLIRejectsMissingProducerConsumer(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"get-artifact", "--digest", testDigest("a"), "--out", filepath.Join(t.TempDir(), "out.txt")}, stdout, stderr)
	if err == nil {
		t.Fatal("get-artifact expected usage error when producer/consumer missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestGetArtifactCLIApprovedExcerptRequiresManifestOptIn(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	approvalRequestPath, approvalEnvelopePath, verifierRecords := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	seedTrustedVerifierForBrokerCLITest(t, verifierRecords)
	approved := promoteViaCLI(t, stdout, stderr, unapproved.Digest, approvalRequestPath, approvalEnvelopePath)

	outPath := filepath.Join(t.TempDir(), "approved.txt")
	err := run([]string{"get-artifact", "--digest", approved.Digest, "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--out", outPath}, stdout, stderr)
	if err == nil {
		t.Fatal("get-artifact expected manifest-opt-in policy rejection")
	}
	if !strings.Contains(err.Error(), "broker_limit_policy_rejected") {
		t.Fatalf("error = %q, want typed policy rejection code", err.Error())
	}

	err = run([]string{"get-artifact", "--digest", approved.Digest, "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--manifest-opt-in", "--out", outPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("get-artifact with manifest opt-in returned error: %v", err)
	}
	b, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("read approved artifact output error: %v", readErr)
	}
	if string(b) != "approved:\nprivate excerpt" {
		t.Fatalf("approved get-artifact payload = %q, want approved payload", string(b))
	}
}

func TestWriteArtifactEventsToFileRejectsCancelledTerminal(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "cancelled.txt")
	events := []brokerapi.ArtifactStreamEvent{
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_start", Digest: testDigest("1"), DataClass: "spec_text"},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "artifact_stream_chunk", Digest: testDigest("1"), DataClass: "spec_text", ChunkBase64: "aGVsbG8=", ChunkBytes: 5},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 3, EventType: "artifact_stream_terminal", Digest: testDigest("1"), DataClass: "spec_text", Terminal: true, TerminalStatus: "cancelled"},
	}

	_, err := writeArtifactEventsToFile(events, outputPath)
	if err == nil {
		t.Fatal("writeArtifactEventsToFile expected cancelled terminal failure")
	}
	if !strings.Contains(err.Error(), "terminal status") {
		t.Fatalf("error = %q, want terminal status failure", err.Error())
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output file should not exist after cancelled stream, statErr=%v", statErr)
	}
}

func TestWriteArtifactEventsToFileRequiresCompletedTerminal(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "missing-terminal.txt")
	events := []brokerapi.ArtifactStreamEvent{
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_start", Digest: testDigest("1"), DataClass: "spec_text"},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "artifact_stream_chunk", Digest: testDigest("1"), DataClass: "spec_text", ChunkBase64: "aGVsbG8=", ChunkBytes: 5},
	}

	_, err := writeArtifactEventsToFile(events, outputPath)
	if err == nil {
		t.Fatal("writeArtifactEventsToFile expected missing terminal failure")
	}
	if !strings.Contains(err.Error(), "did not complete successfully") {
		t.Fatalf("error = %q, want incomplete stream failure", err.Error())
	}
}

func TestWriteArtifactEventsToFileSucceedsOnCompletedTerminal(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "completed.txt")
	events := []brokerapi.ArtifactStreamEvent{
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_start", Digest: testDigest("1"), DataClass: "spec_text"},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "artifact_stream_chunk", Digest: testDigest("1"), DataClass: "spec_text", ChunkBase64: "aGVsbG8=", ChunkBytes: 5},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 3, EventType: "artifact_stream_terminal", Digest: testDigest("1"), DataClass: "spec_text", Terminal: true, TerminalStatus: "completed"},
	}

	written, err := writeArtifactEventsToFile(events, outputPath)
	if err != nil {
		t.Fatalf("writeArtifactEventsToFile returned error: %v", err)
	}
	if written != 5 {
		t.Fatalf("written bytes = %d, want 5", written)
	}
	b, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("ReadFile returned error: %v", readErr)
	}
	if string(b) != "hello" {
		t.Fatalf("output payload = %q, want hello", string(b))
	}
}

func TestHeadArtifactReturnsTypedValidationCodeForInvalidDigest(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"head-artifact", "--digest", "invalid"}, stdout, stderr)
	if err == nil {
		t.Fatal("head-artifact expected validation error for invalid digest")
	}
	if !strings.Contains(err.Error(), "broker_validation_schema_invalid") {
		t.Fatalf("error = %q, want typed broker validation code", err.Error())
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
