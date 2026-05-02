package auditd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) PersistAuditProofBinding(payload trustpolicy.AuditProofBindingPayload) (trustpolicy.Digest, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
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
	return digest, true, nil
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
	return digest, nil
}

func (l *Ledger) findAuditProofBindingByIdempotencyKeyLocked(want string) (trustpolicy.Digest, bool, error) {
	entries, err := l.readOptionalSidecarDirEntries(proofBindingsDirName)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	for _, entry := range entries {
		digest, ok, err := l.auditProofBindingDigestForIdempotencyKeyLocked(entry, want)
		if err != nil {
			return trustpolicy.Digest{}, false, err
		}
		if ok {
			return digest, true, nil
		}
	}
	return trustpolicy.Digest{}, false, nil
}

func (l *Ledger) auditProofBindingDigestForIdempotencyKeyLocked(entry os.DirEntry, want string) (trustpolicy.Digest, bool, error) {
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return trustpolicy.Digest{}, ok, err
	}
	payload := trustpolicy.AuditProofBindingPayload{}
	path := filepath.Join(l.rootDir, sidecarDirName, proofBindingsDirName, entry.Name())
	if err := readJSONFile(path, &payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if err := trustpolicy.ValidateAuditProofBindingPayload(payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	key, err := proofBindingIdempotencyKey(payload)
	if err != nil || key != want {
		return trustpolicy.Digest{}, false, err
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.Digest{}, false, err
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
