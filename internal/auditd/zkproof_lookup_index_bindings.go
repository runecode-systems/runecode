package auditd

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (idx *proofLookupIndex) notePersistedAuditProofBinding(digest trustpolicy.Digest, payload trustpolicy.AuditProofBindingPayload) error {
	if idx == nil {
		return fmt.Errorf("proof lookup index is required")
	}
	normalizeProofLookupIndex(idx)
	recordIdentity, err := payload.AuditRecordDigest.Identity()
	if err != nil {
		return err
	}
	digestIdentity, err := digest.Identity()
	if err != nil {
		return err
	}
	idx.NextBindingSequence++
	lookupKey := proofBindingLookupKey(payload.StatementFamily, payload.SchemeAdapterID)
	byKey := idx.LatestBindingByRecord[recordIdentity]
	if byKey == nil {
		byKey = map[string]proofBindingLookup{}
	}
	current, ok := byKey[lookupKey]
	candidate := proofBindingLookup{DigestIdentity: digestIdentity, Sequence: idx.NextBindingSequence}
	if !ok || candidate.Sequence > current.Sequence || (candidate.Sequence == current.Sequence && candidate.DigestIdentity > current.DigestIdentity) {
		byKey[lookupKey] = candidate
	}
	idx.LatestBindingByRecord[recordIdentity] = byKey
	return nil
}

func proofBindingLookupKey(statementFamily, schemeAdapterID string) string {
	return strings.TrimSpace(statementFamily) + "\x00" + strings.TrimSpace(schemeAdapterID)
}
