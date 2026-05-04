package auditd

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

// ReceiptsForSealDigest returns all persisted audit receipts whose subject_digest
// targets the provided segment seal digest.
func (l *Ledger) ReceiptsForSealDigest(sealDigest trustpolicy.Digest) ([]trustpolicy.SignedObjectEnvelope, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	all, err := l.loadAllReceiptsLocked()
	if err != nil {
		return nil, err
	}
	want := mustDigestIdentity(sealDigest)
	out := make([]trustpolicy.SignedObjectEnvelope, 0, len(all))
	for i := range all {
		env := all[i]
		if strings.TrimSpace(env.PayloadSchemaID) != trustpolicy.AuditReceiptSchemaID {
			continue
		}
		payload := struct {
			SubjectDigest trustpolicy.Digest `json:"subject_digest"`
		}{}
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			continue
		}
		if mustDigestIdentity(payload.SubjectDigest) == want {
			out = append(out, env)
		}
	}
	return out, nil
}
