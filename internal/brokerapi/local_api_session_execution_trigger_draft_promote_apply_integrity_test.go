package brokerapi

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func TestDraftPromoteApplyRejectsTamperedBoundDraftArtifactBlob(t *testing.T) {
	_, s := newSessionExecutionTriggerWorkflowService(t, "run-draft-promote-tamper", "sess-draft-promote-tamper")
	draft := runChangeDraftForPromoteApply(t, s, "sess-draft-promote-tamper", "req-draft-promote-tamper-draft", "Draft promote tamper")
	tamperArtifactBlobForTest(t, s, draft.digest, []byte(`{"tampered":true}`))
	_, err := s.loadDraftPromoteDecodedArtifact(artifacts.SessionWorkflowPackBoundInputArtifactDurableState{ArtifactRef: "change_draft_artifact", ArtifactDigest: draft.digest})
	if err == nil || !strings.Contains(err.Error(), "digest drift") {
		t.Fatalf("loadDraftPromoteDecodedArtifact error = %v, want digest drift", err)
	}
}

func TestDraftPromoteApplyRejectsTamperedDraftTextArtifactBlob(t *testing.T) {
	_, s := newSessionExecutionTriggerWorkflowService(t, "run-draft-text-tamper", "sess-draft-text-tamper")
	draft := runChangeDraftForPromoteApply(t, s, "sess-draft-text-tamper", "req-draft-text-tamper-draft", "Draft text tamper")
	payload, err := s.readArtifactPayloadVerified(draft.digest)
	if err != nil {
		t.Fatalf("readArtifactPayloadVerified returned error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	draftTextDigest := draftPromoteDigestObjectValueFromMap(decoded, "artifact_digest")
	tamperArtifactBlobForTest(t, s, draftTextDigest, []byte("tampered text"))
	_, _, err = s.loadDraftPromoteText(decoded, draft.digest)
	if err == nil || !strings.Contains(err.Error(), "digest drift") {
		t.Fatalf("loadDraftPromoteText error = %v, want digest drift", err)
	}
}

func TestValidateDraftPromoteProjectDigestRequiresArtifactBindingWhenBound(t *testing.T) {
	bound := "sha256:" + strings.Repeat("a", 64)
	if err := validateDraftPromoteProjectDigest("", bound); err == nil {
		t.Fatal("validateDraftPromoteProjectDigest expected missing artifact binding error")
	}
	if err := validateDraftPromoteProjectDigest(bound, bound); err != nil {
		t.Fatalf("validateDraftPromoteProjectDigest returned error: %v", err)
	}
}

func tamperArtifactBlobForTest(t *testing.T, s *Service, digest string, payload []byte) {
	t.Helper()
	record, err := s.store.Head(digest)
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if err := os.WriteFile(record.BlobPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
