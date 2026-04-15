package auditd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) ReceiptEnvelopeByDigest(digest trustpolicy.Digest) (trustpolicy.SignedObjectEnvelope, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	identity, err := digest.Identity()
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("receipt digest: %w", err)
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	path := filepath.Join(l.rootDir, sidecarDirName, receiptsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := readJSONFile(path, &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	computedDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("receipt envelope digest: %w", err)
	}
	if computedIdentity, _ := computedDigest.Identity(); computedIdentity != identity {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("receipt envelope digest %q does not match requested digest %q", computedIdentity, identity)
	}
	return envelope, nil
}
