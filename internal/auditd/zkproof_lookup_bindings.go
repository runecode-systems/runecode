package auditd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) LatestAuditProofBindingForRecord(recordDigest trustpolicy.Digest, statementFamily, schemeAdapterID string) (trustpolicy.Digest, trustpolicy.AuditProofBindingPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	wantRecord, err := recordDigest.Identity()
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	identity, found, err := l.latestAuditProofBindingIdentityLocked(wantRecord, statementFamily, schemeAdapterID)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if !found {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, nil
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	payload, found, err := l.loadAuditProofBindingByIdentityLocked(identity)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if !found {
		if err := l.refreshProofLookupIndexLocked(); err != nil {
			return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
		}
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, nil
	}
	return digest, payload, true, nil
}

func (l *Ledger) latestAuditProofBindingIdentityLocked(recordIdentity, statementFamily, schemeAdapterID string) (string, bool, error) {
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return "", false, err
	}
	lookupKey := proofBindingLookupKey(statementFamily, schemeAdapterID)
	identity, found := latestAuditProofBindingIdentity(l.lookupIndex, recordIdentity, lookupKey)
	if found {
		return identity, true, nil
	}
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return "", false, err
	}
	identity, found = latestAuditProofBindingIdentity(l.lookupIndex, recordIdentity, lookupKey)
	return identity, found, nil
}

func latestAuditProofBindingIdentity(idx *proofLookupIndex, recordIdentity, lookupKey string) (string, bool) {
	lookup, ok := idx.LatestBindingByRecord[recordIdentity][lookupKey]
	if !ok {
		return "", false
	}
	return lookup.DigestIdentity, true
}

type auditProofBindingCandidate struct {
	digest   trustpolicy.Digest
	payload  trustpolicy.AuditProofBindingPayload
	identity string
}

func (l *Ledger) loadVerifiedAuditProofBindingCandidate(entry os.DirEntry) (auditProofBindingCandidate, bool, error) {
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return auditProofBindingCandidate{}, ok, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, proofBindingsDirName, entry.Name())
	payload := trustpolicy.AuditProofBindingPayload{}
	if err := readJSONFile(path, &payload); err != nil {
		return auditProofBindingCandidate{}, false, err
	}
	if err := trustpolicy.ValidateAuditProofBindingPayload(payload); err != nil {
		return auditProofBindingCandidate{}, false, err
	}
	computed, err := canonicalDigest(payload)
	if err != nil {
		return auditProofBindingCandidate{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return auditProofBindingCandidate{}, false, fmt.Errorf("audit proof binding content digest mismatch for %s", identity)
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return auditProofBindingCandidate{}, false, err
	}
	return auditProofBindingCandidate{digest: digest, payload: payload, identity: identity}, true, nil
}
