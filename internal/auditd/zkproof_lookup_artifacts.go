package auditd

import (
	"errors"
	"fmt"
	"os"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) ZKProofArtifactByDigest(digest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	identity, err := digest.Identity()
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, false, err
	}
	path := sidecarPath(l.rootDir, proofArtifactsDirName, identity)
	payload := trustpolicy.ZKProofArtifactPayload{}
	if err := readJSONFile(path, &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trustpolicy.ZKProofArtifactPayload{}, false, nil
		}
		return trustpolicy.ZKProofArtifactPayload{}, false, err
	}
	if err := trustpolicy.ValidateZKProofArtifactPayload(payload); err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, false, err
	}
	computed, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return trustpolicy.ZKProofArtifactPayload{}, false, fmt.Errorf("zk proof artifact content digest mismatch for %s", identity)
	}
	return payload, true, nil
}

func (l *Ledger) AuditProofBindingByDigest(digest trustpolicy.Digest) (trustpolicy.AuditProofBindingPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	identity, err := digest.Identity()
	if err != nil {
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	path := sidecarPath(l.rootDir, proofBindingsDirName, identity)
	payload := trustpolicy.AuditProofBindingPayload{}
	if err := readJSONFile(path, &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trustpolicy.AuditProofBindingPayload{}, false, nil
		}
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if err := trustpolicy.ValidateAuditProofBindingPayload(payload); err != nil {
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	computed, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return trustpolicy.AuditProofBindingPayload{}, false, fmt.Errorf("audit proof binding content digest mismatch for %s", identity)
	}
	return payload, true, nil
}

func mustDigestIdentityString(d trustpolicy.Digest) string {
	identity, err := d.Identity()
	if err != nil {
		return ""
	}
	return identity
}

func (l *Ledger) loadAuditProofBindingByIdentityLocked(identity string) (trustpolicy.AuditProofBindingPayload, bool, error) {
	payload := trustpolicy.AuditProofBindingPayload{}
	if err := readJSONFile(sidecarPath(l.rootDir, proofBindingsDirName, identity), &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trustpolicy.AuditProofBindingPayload{}, false, nil
		}
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if err := trustpolicy.ValidateAuditProofBindingPayload(payload); err != nil {
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	computed, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.AuditProofBindingPayload{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return trustpolicy.AuditProofBindingPayload{}, false, fmt.Errorf("audit proof binding content digest mismatch for %s", identity)
	}
	return payload, true, nil
}

func (l *Ledger) loadZKProofVerificationRecordByIdentityLocked(identity string) (trustpolicy.ZKProofVerificationRecordPayload, bool, error) {
	payload := trustpolicy.ZKProofVerificationRecordPayload{}
	if err := readJSONFile(sidecarPath(l.rootDir, proofVerificationsDirName, identity), &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trustpolicy.ZKProofVerificationRecordPayload{}, false, nil
		}
		return trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if err := trustpolicy.ValidateZKProofVerificationRecordPayload(payload); err != nil {
		return trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	computed, err := canonicalDigest(payload)
	if err != nil {
		return trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return trustpolicy.ZKProofVerificationRecordPayload{}, false, fmt.Errorf("zk proof verification record content digest mismatch for %s", identity)
	}
	return payload, true, nil
}
