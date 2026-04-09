package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestArtifactReadRequiresManifestOptInForApprovedExcerpt(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	approved := setupApprovedExcerptArtifactForReadTests(t, s)

	t.Run("reject without manifest opt-in", func(t *testing.T) {
		_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: "req-approved-deny", Digest: approved.Digest, ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: string(artifacts.DataClassApprovedFileExcerpts)}, RequestContext{})
		if errResp == nil {
			t.Fatal("HandleArtifactRead expected manifest opt-in denial")
		}
		if errResp.Error.Code != "broker_limit_policy_rejected" {
			t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
		}
	})

	t.Run("allow with manifest opt-in", func(t *testing.T) {
		handle, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: "req-approved-allow", Digest: approved.Digest, ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: string(artifacts.DataClassApprovedFileExcerpts), ManifestOptIn: true}, RequestContext{})
		if errResp != nil {
			t.Fatalf("HandleArtifactRead with manifest opt-in error response: %+v", errResp)
		}
		events, streamErr := s.StreamArtifactReadEvents(handle)
		if streamErr != nil {
			t.Fatalf("StreamArtifactReadEvents returned error: %v", streamErr)
		}
		assertArtifactStreamDecodedPayload(t, events, "approved:\nprivate excerpt")
	})
}

func setupApprovedExcerptArtifactForReadTests(t *testing.T, s *Service) artifacts.ArtifactReference {
	t.Helper()
	unapproved, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put unapproved returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTests(t, "human", unapproved.Digest)
	for _, verifier := range verifiers {
		if putErr := putTrustedVerifierRecordForService(s, verifier); putErr != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", putErr)
		}
	}
	approved, err := s.PromoteApprovedExcerpt(artifacts.PromotionRequest{UnapprovedDigest: unapproved.Digest, Approver: "human", ApprovalRequest: requestEnv, ApprovalDecision: decisionEnv, RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true})
	if err != nil {
		t.Fatalf("PromoteApprovedExcerpt returned error: %v", err)
	}
	return approved
}

func parseRFC3339OrNow(payload map[string]any, key string) time.Time {
	if value, ok := payload[key].(string); ok {
		if ts, err := time.Parse(time.RFC3339, value); err == nil {
			return ts.UTC()
		}
	}
	return time.Now().UTC()
}

func digestFromPayloadField(payload map[string]any, key string) string {
	raw, ok := payload[key].(map[string]any)
	if !ok {
		return "sha256:" + strings.Repeat("0", 64)
	}
	hashAlg, _ := raw["hash_alg"].(string)
	hash, _ := raw["hash"].(string)
	if hashAlg == "" || hash == "" {
		return "sha256:" + strings.Repeat("0", 64)
	}
	return hashAlg + ":" + hash
}

func stringFieldFromPayload(payload map[string]any, key string, fallback string) string {
	v, ok := payload[key].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func approvalIDForBrokerTest(t *testing.T, requestEnv *trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	approvalID, idErr := approvalIDFromRequest(*requestEnv)
	if idErr != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", idErr)
	}
	return approvalID
}
