package auditd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) LatestAuditProofBindingForRecord(recordDigest trustpolicy.Digest, statementFamily, schemeAdapterID string) (trustpolicy.Digest, trustpolicy.AuditProofBindingPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	wantRecord, err := recordDigest.Identity()
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	entries, err := l.readOptionalSidecarDirEntries(proofBindingsDirName)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	best, found, err := l.latestAuditProofBindingCandidateLocked(entries, wantRecord, statementFamily, schemeAdapterID)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if !found {
		return trustpolicy.Digest{}, trustpolicy.AuditProofBindingPayload{}, false, nil
	}
	return best.digest, best.payload, true, nil
}

type auditProofBindingCandidate struct {
	digest   trustpolicy.Digest
	payload  trustpolicy.AuditProofBindingPayload
	modTime  int64
	identity string
}

func (l *Ledger) latestAuditProofBindingCandidateLocked(entries []os.DirEntry, wantRecord, statementFamily, schemeAdapterID string) (auditProofBindingCandidate, bool, error) {
	best := auditProofBindingCandidate{}
	bestFound := false
	for _, entry := range entries {
		candidate, matches, ok, err := l.matchingAuditProofBindingCandidate(entry, wantRecord, statementFamily, schemeAdapterID)
		if err != nil {
			return auditProofBindingCandidate{}, false, err
		}
		if !ok || !matches {
			continue
		}
		if !bestFound || candidate.isNewerThan(best) {
			best = candidate
			bestFound = true
		}
	}
	return best, bestFound, nil
}

func (l *Ledger) matchingAuditProofBindingCandidate(entry os.DirEntry, wantRecord, statementFamily, schemeAdapterID string) (auditProofBindingCandidate, bool, bool, error) {
	candidate, ok, err := l.loadVerifiedAuditProofBindingCandidate(entry)
	if err != nil || !ok {
		return auditProofBindingCandidate{}, false, ok, err
	}
	matches, err := auditProofBindingMatchesLookup(candidate.payload, wantRecord, statementFamily, schemeAdapterID)
	if err != nil {
		return auditProofBindingCandidate{}, false, false, err
	}
	return candidate, matches, true, nil
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
		return auditProofBindingCandidate{}, false, err
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return auditProofBindingCandidate{}, false, err
	}
	info, err := entry.Info()
	if err != nil {
		return auditProofBindingCandidate{}, false, err
	}
	return auditProofBindingCandidate{digest: digest, payload: payload, modTime: info.ModTime().UnixNano(), identity: identity}, true, nil
}

func auditProofBindingMatchesLookup(payload trustpolicy.AuditProofBindingPayload, wantRecord, statementFamily, schemeAdapterID string) (bool, error) {
	recordID, err := payload.AuditRecordDigest.Identity()
	if err != nil {
		return false, err
	}
	return recordID == wantRecord &&
		strings.TrimSpace(payload.StatementFamily) == strings.TrimSpace(statementFamily) &&
		strings.TrimSpace(payload.SchemeAdapterID) == strings.TrimSpace(schemeAdapterID), nil
}

func (candidate auditProofBindingCandidate) isNewerThan(other auditProofBindingCandidate) bool {
	return candidate.modTime > other.modTime || (candidate.modTime == other.modTime && candidate.identity > other.identity)
}
