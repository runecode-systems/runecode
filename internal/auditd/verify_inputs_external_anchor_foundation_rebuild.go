package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) rebuildExternalAnchorIncrementalFoundationLocked() (externalAnchorIncrementalFoundation, error) {
	foundation := externalAnchorIncrementalFoundation{
		SchemaVersion: externalAnchorIncrementalFoundationSchemaVersion,
		Seals:         map[string]externalAnchorIncrementalSealSnapshot{},
	}
	receipts, err := l.rebuildFoundationFromReceiptsLocked(foundation)
	if err != nil {
		return externalAnchorIncrementalFoundation{}, err
	}
	return l.rebuildFoundationFromExternalAnchorEvidenceLocked(receipts)
}

func (l *Ledger) rebuildFoundationFromReceiptsLocked(foundation externalAnchorIncrementalFoundation) (externalAnchorIncrementalFoundation, error) {
	entries, err := l.readOptionalSidecarDirEntries(receiptsDirName)
	if err != nil {
		return externalAnchorIncrementalFoundation{}, err
	}
	for i := range entries {
		sealIdentity, receiptDigest, ok, entryErr := l.receiptFoundationEntryLocked(entries[i])
		if entryErr != nil {
			return externalAnchorIncrementalFoundation{}, entryErr
		}
		if !ok {
			continue
		}
		foundation = appendReceiptFoundationEntry(foundation, sealIdentity, receiptDigest)
	}
	return foundation, nil
}

func (l *Ledger) rebuildFoundationFromExternalAnchorEvidenceLocked(foundation externalAnchorIncrementalFoundation) (externalAnchorIncrementalFoundation, error) {
	entries, err := l.readOptionalSidecarDirEntries(externalAnchorEvidenceDir)
	if err != nil {
		return externalAnchorIncrementalFoundation{}, err
	}
	for i := range entries {
		sealIdentity, evidenceIdentity, rec, ok, entryErr := l.externalAnchorEvidenceFoundationEntryLocked(entries[i])
		if entryErr != nil {
			return externalAnchorIncrementalFoundation{}, entryErr
		}
		if !ok {
			continue
		}
		foundation, err = appendExternalAnchorEvidenceFoundationEntry(foundation, sealIdentity, evidenceIdentity, rec)
		if err != nil {
			return externalAnchorIncrementalFoundation{}, err
		}
	}
	return foundation, nil
}

func (l *Ledger) readOptionalSidecarDirEntries(dirName string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, dirName))
	if err == nil {
		return entries, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

func (l *Ledger) receiptFoundationEntryLocked(entry os.DirEntry) (string, string, bool, error) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return "", "", false, nil
	}
	receiptDigest, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return "", "", ok, err
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	path := filepath.Join(l.rootDir, sidecarDirName, receiptsDirName, entry.Name())
	if err := readJSONFile(path, &envelope); err != nil {
		return "", "", false, err
	}
	sealIdentity, applies, err := receiptSubjectSealDigestIdentity(envelope)
	if err != nil || !applies {
		return "", "", applies, err
	}
	return sealIdentity, receiptDigest, true, nil
}

func appendReceiptFoundationEntry(foundation externalAnchorIncrementalFoundation, sealIdentity, receiptDigest string) externalAnchorIncrementalFoundation {
	entry := foundation.Seals[sealIdentity]
	entry.ReceiptDigests = appendUniqueIdentity(entry.ReceiptDigests, receiptDigest)
	foundation.Seals[sealIdentity] = entry
	return foundation
}

func (l *Ledger) externalAnchorEvidenceFoundationEntryLocked(entry os.DirEntry) (string, string, trustpolicy.ExternalAnchorEvidencePayload, bool, error) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return "", "", trustpolicy.ExternalAnchorEvidencePayload{}, false, nil
	}
	evidenceIdentity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return "", "", trustpolicy.ExternalAnchorEvidencePayload{}, ok, err
	}
	rec, digest, isEvidence, err := l.loadExternalAnchorEvidenceEntry(entry.Name())
	if err != nil {
		return "", "", trustpolicy.ExternalAnchorEvidencePayload{}, false, err
	}
	if !isEvidence {
		return "", "", trustpolicy.ExternalAnchorEvidencePayload{}, false, nil
	}
	if digest == nil || mustDigestIdentity(*digest) != evidenceIdentity {
		return "", "", trustpolicy.ExternalAnchorEvidencePayload{}, false, fmt.Errorf("external anchor evidence digest mismatch for %s", evidenceIdentity)
	}
	sealIdentity, err := rec.AnchoringSubjectDigest.Identity()
	if err != nil {
		return "", "", trustpolicy.ExternalAnchorEvidencePayload{}, false, err
	}
	return sealIdentity, evidenceIdentity, rec, true, nil
}

func appendExternalAnchorEvidenceFoundationEntry(foundation externalAnchorIncrementalFoundation, sealIdentity, evidenceIdentity string, rec trustpolicy.ExternalAnchorEvidencePayload) (externalAnchorIncrementalFoundation, error) {
	entry, err := appendExternalAnchorEvidenceEntryIdentity(foundation.Seals[sealIdentity], evidenceIdentity, rec)
	if err != nil {
		return externalAnchorIncrementalFoundation{}, err
	}
	foundation.Seals[sealIdentity] = entry
	return foundation, nil
}
