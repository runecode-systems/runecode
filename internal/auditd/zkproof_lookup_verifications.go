package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) FindMatchingZKProofVerificationRecord(match trustpolicy.ZKProofVerificationRecordPayload) (trustpolicy.Digest, trustpolicy.ZKProofVerificationRecordPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key, err := verificationIdentityKey(match)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	identity, found, err := l.matchingVerificationIdentityLocked(key)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if !found {
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, nil
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	payload, found, err := l.loadZKProofVerificationRecordByIdentityLocked(identity)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if !found {
		if err := l.refreshProofLookupIndexLocked(); err != nil {
			return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
		}
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, nil
	}
	return digest, payload, true, nil
}

func (l *Ledger) matchingVerificationIdentityLocked(key string) (string, bool, error) {
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return "", false, err
	}
	lookup, found := l.lookupIndex.VerificationByKey[key]
	if found {
		return lookup.DigestIdentity, true, nil
	}
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return "", false, err
	}
	lookup, found = l.lookupIndex.VerificationByKey[key]
	if !found {
		return "", false, nil
	}
	return lookup.DigestIdentity, true, nil
}

func (l *Ledger) loadVerifiedZKProofVerificationRecordCandidate(entry os.DirEntry) (string, trustpolicy.ZKProofVerificationRecordPayload, bool, error) {
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, ok, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, proofVerificationsDirName, entry.Name())
	candidate := trustpolicy.ZKProofVerificationRecordPayload{}
	if err := readJSONFile(path, &candidate); err != nil {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if err := trustpolicy.ValidateZKProofVerificationRecordPayload(candidate); err != nil {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	computed, err := canonicalDigest(candidate)
	if err != nil {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, fmt.Errorf("zk proof verification record content digest mismatch for %s", identity)
	}
	return identity, candidate, true, nil
}

func sameStringSet(left, right []string) bool {
	seen, ok := buildStringSet(left)
	if !ok || len(seen) != len(right) {
		return false
	}
	for _, value := range right {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return false
		}
		if _, ok := seen[trimmed]; !ok {
			return false
		}
	}
	return true
}

func buildStringSet(values []string) (map[string]struct{}, bool) {
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, false
		}
		if _, ok := seen[trimmed]; ok {
			return nil, false
		}
		seen[trimmed] = struct{}{}
	}
	return seen, true
}
