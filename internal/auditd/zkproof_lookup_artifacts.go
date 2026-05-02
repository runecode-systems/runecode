package auditd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) ZKProofArtifactByDigest(digest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	identity, err := digest.Identity()
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, false, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, proofArtifactsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
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
	path := filepath.Join(l.rootDir, sidecarDirName, proofBindingsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
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
