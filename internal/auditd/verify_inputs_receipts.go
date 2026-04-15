package auditd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) loadAllReceiptsLocked() ([]trustpolicy.SignedObjectEnvelope, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, receiptsDirName))
	if err != nil {
		return nil, err
	}
	receipts := make([]trustpolicy.SignedObjectEnvelope, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		envelope := trustpolicy.SignedObjectEnvelope{}
		if err := readJSONFile(filepath.Join(l.rootDir, sidecarDirName, receiptsDirName, entry.Name()), &envelope); err != nil {
			return nil, err
		}
		receipts = append(receipts, envelope)
	}
	return receipts, nil
}

func (l *Ledger) loadAllSealDigestsLocked() ([]trustpolicy.Digest, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, sealsDirName))
	if err != nil {
		return nil, err
	}
	digests := make([]trustpolicy.Digest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		digests = append(digests, trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(entry.Name(), ".json")})
	}
	return digests, nil
}
