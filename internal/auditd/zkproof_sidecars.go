package auditd

import (
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) PersistAuditProofBinding(payload trustpolicy.AuditProofBindingPayload) (trustpolicy.Digest, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if err := trustpolicy.ValidateAuditProofBindingPayload(payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}

	idempotencyKey, err := proofBindingIdempotencyKey(payload)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}

	existing, ok, err := l.findAuditProofBindingByIdempotencyKeyLocked(idempotencyKey)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if ok {
		return existing, false, nil
	}

	digest, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, proofBindingsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if err := l.persistAuditProofBindingLookupLocked(digest, payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	return digest, true, nil
}

func (l *Ledger) persistAuditProofBindingLookupLocked(digest trustpolicy.Digest, payload trustpolicy.AuditProofBindingPayload) error {
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return err
	}
	if err := l.lookupIndex.notePersistedAuditProofBinding(digest, payload); err != nil {
		return err
	}
	return l.saveProofLookupIndexLocked(l.lookupIndex)
}

func (l *Ledger) PersistZKProofArtifact(payload trustpolicy.ZKProofArtifactPayload) (trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := trustpolicy.ValidateZKProofArtifactPayload(payload); err != nil {
		return trustpolicy.Digest{}, err
	}
	digest, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, proofArtifactsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, payload); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

func (l *Ledger) PersistZKProofVerificationRecord(payload trustpolicy.ZKProofVerificationRecordPayload) (trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := trustpolicy.ValidateZKProofVerificationRecordPayload(payload); err != nil {
		return trustpolicy.Digest{}, err
	}
	digest, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, proofVerificationsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, payload); err != nil {
		return trustpolicy.Digest{}, err
	}
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return trustpolicy.Digest{}, err
	}
	if err := l.lookupIndex.notePersistedZKProofVerificationRecord(digest, payload); err != nil {
		return trustpolicy.Digest{}, err
	}
	if err := l.saveProofLookupIndexLocked(l.lookupIndex); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

func (l *Ledger) findAuditProofBindingByIdempotencyKeyLocked(want string) (trustpolicy.Digest, bool, error) {
	digest, err := digestFromIdentity(want)
	if err != nil {
		return trustpolicy.Digest{}, false, nil
	}
	payload, found, err := l.loadAuditProofBindingByIdentityLocked(want)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if !found {
		return trustpolicy.Digest{}, false, nil
	}
	key, err := proofBindingIdempotencyKey(payload)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if key != want {
		return trustpolicy.Digest{}, false, nil
	}
	return digest, true, nil
}

func proofBindingIdempotencyKey(payload trustpolicy.AuditProofBindingPayload) (string, error) {
	canonical, err := canonicalDigest(payload)
	if err != nil {
		return "", err
	}
	return canonical.Identity()
}
