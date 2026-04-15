package brokerapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type anchorReceiptPayloadForTest struct {
	ApprovalAssurance string              `json:"approval_assurance_level,omitempty"`
	ApprovalDecision  *trustpolicy.Digest `json:"approval_decision_digest,omitempty"`
}

func mustReadAnchorReceiptSidecar(t *testing.T, ledgerRoot string, digest trustpolicy.Digest) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	id, _ := digest.Identity()
	path := filepath.Join(ledgerRoot, "sidecar", "receipts", strings.TrimPrefix(id, "sha256:")+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}
	env := trustpolicy.SignedObjectEnvelope{}
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatalf("Unmarshal anchor receipt envelope returned error: %v", err)
	}
	return env
}

func mustAnchorReceiptPayload(t *testing.T, envelope trustpolicy.SignedObjectEnvelope) anchorReceiptPayloadForTest {
	t.Helper()
	receipt := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &receipt); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	rawPayload, ok := receipt["receipt_payload"]
	if !ok {
		t.Fatal("receipt_payload missing")
	}
	payloadBytes, err := json.Marshal(rawPayload)
	if err != nil {
		t.Fatalf("Marshal receipt_payload returned error: %v", err)
	}
	payload := anchorReceiptPayloadForTest{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Unmarshal typed receipt_payload returned error: %v", err)
	}
	return payload
}

func mustDigestIdentityForAnchorTest(d trustpolicy.Digest) string {
	id, _ := d.Identity()
	return id
}

func stringValueForAnchorTest(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
