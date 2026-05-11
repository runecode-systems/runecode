package brokerapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func TestApprovedImplementationInputSetFixtureValidatesAgainstSchema(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	mutationDigest := artifacts.DigestBytes([]byte("approved-mutation"))
	payload := approvedImplementationInputSetFixture(t, s, []string{mutationDigest}, []string{mutationDigest}, nil)
	raw, err := artifacts.CanonicalizeJSONBytes(mustJSONMarshalForApprovedImplementationTest(t, payload))
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	if err := artifacts.ValidateObjectPayloadAgainstSchema(raw, "objects/RuneContextApprovedImplementationInputSet.schema.json"); err != nil {
		t.Fatalf("ValidateObjectPayloadAgainstSchema returned error: %v", err)
	}
}

func TestReadArtifactPayloadVerifiedRejectsBlobDigestDrift(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte("approved"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: artifacts.DigestBytes([]byte("approved")), CreatedByRole: "test", TrustedSource: true})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	record, err := s.store.Head(ref.Digest)
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if err := os.WriteFile(record.BlobPath, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err = s.readArtifactPayloadVerified(ref.Digest)
	if err == nil || !strings.Contains(err.Error(), "artifact payload digest drift") {
		t.Fatalf("readArtifactPayloadVerified error = %v, want digest drift", err)
	}
}

func TestValidateApprovedImplementationWriteModeEnforcesCreateAndUpdate(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.md")
	missing := filepath.Join(root, "missing.md")
	if err := os.WriteFile(existing, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := validateApprovedImplementationWriteMode(missing, "create"); err != nil {
		t.Fatalf("create missing target returned error: %v", err)
	}
	if err := validateApprovedImplementationWriteMode(existing, "update"); err != nil {
		t.Fatalf("update existing target returned error: %v", err)
	}
	if err := validateApprovedImplementationWriteMode(existing, "create"); err == nil {
		t.Fatal("create existing target expected error")
	}
	if err := validateApprovedImplementationWriteMode(missing, "update"); err == nil {
		t.Fatal("update missing target expected error")
	}
}

func TestApprovedImplementationWriteIntentRejectsUnexpectedFields(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	payload := map[string]any{
		"target_path":    "runecontext/changes/CHG-test/proposal.md",
		"content":        "body",
		"content_digest": digestObject(artifacts.DigestBytes([]byte("body"))),
		"write_mode":     "create",
		"unexpected":     "value",
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(mustJSONMarshalForApprovedImplementationTest(t, payload))
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	_, _, _, err = s.approvedImplementationWriteIntent(canonical)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("approvedImplementationWriteIntent error = %v, want unsupported field", err)
	}
}

func TestApprovedImplementationWriteIntentLoadsContentArtifactDigest(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	content := []byte("artifact-backed body")
	contentRef, err := s.Put(artifacts.PutRequest{Payload: content, ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: artifacts.DigestBytes(content), CreatedByRole: "test", TrustedSource: true})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	payload := map[string]any{
		"target_path":             "runecontext/changes/CHG-artifact-backed/proposal.md",
		"content_artifact_digest": contentRef.Digest,
		"content_digest":          digestObject(artifacts.DigestBytes(content)),
		"write_mode":              "create",
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(mustJSONMarshalForApprovedImplementationTest(t, payload))
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	targetPath, writeMode, resolvedContent, err := s.approvedImplementationWriteIntent(canonical)
	if err != nil {
		t.Fatalf("approvedImplementationWriteIntent returned error: %v", err)
	}
	if targetPath != "runecontext/changes/CHG-artifact-backed/proposal.md" {
		t.Fatalf("targetPath = %q, want artifact-backed path", targetPath)
	}
	if writeMode != "create" {
		t.Fatalf("writeMode = %q, want create", writeMode)
	}
	if string(resolvedContent) != string(content) {
		t.Fatalf("resolved content = %q, want %q", string(resolvedContent), string(content))
	}
}

func TestApprovedImplementationWriteIntentRejectsContentArtifactDigestDrift(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	content := []byte("artifact-backed drift")
	contentRef, err := s.Put(artifacts.PutRequest{Payload: content, ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: artifacts.DigestBytes(content), CreatedByRole: "test", TrustedSource: true})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	payload := map[string]any{
		"target_path":             "runecontext/changes/CHG-artifact-backed/proposal.md",
		"content_artifact_digest": contentRef.Digest,
		"content_digest":          digestObject(artifacts.DigestBytes([]byte("other"))),
		"write_mode":              "create",
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(mustJSONMarshalForApprovedImplementationTest(t, payload))
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	_, _, _, err = s.approvedImplementationWriteIntent(canonical)
	if err == nil || !strings.Contains(err.Error(), "content_digest drift") {
		t.Fatalf("approvedImplementationWriteIntent error = %v, want content_digest drift", err)
	}
}

func mustJSONMarshalForApprovedImplementationTest(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	return b
}
