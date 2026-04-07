package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type approvalRecord struct {
	Summary          ApprovalSummary
	RequestEnvelope  *trustpolicy.SignedObjectEnvelope
	DecisionEnvelope *trustpolicy.SignedObjectEnvelope
}

type approvalState struct {
	mu      sync.Mutex
	seeded  bool
	records map[string]approvalRecord
}

func approvalIDFromRequest(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(envelope.Payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize approval request payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func signedEnvelopeDigest(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	b, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func decodeDecisionString(payload []byte, field string, fallback string) string {
	value := map[string]any{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return fallback
	}
	v, ok := value[field].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func ptrArtifactSummary(value ArtifactSummary) *ArtifactSummary {
	v := value
	return &v
}

func shaDigestIdentity(input string) string {
	sum := sha256.Sum256([]byte(input))
	return "sha256:" + hex.EncodeToString(sum[:])
}
