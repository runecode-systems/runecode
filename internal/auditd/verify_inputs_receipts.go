package auditd

import (
	"encoding/json"
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

func (l *Ledger) loadExternalAnchorEvidenceLocked() ([]trustpolicy.ExternalAnchorEvidencePayload, []trustpolicy.Digest, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, externalAnchorEvidenceDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return l.collectExternalAnchorEvidenceEntries(entries)
}

func (l *Ledger) collectExternalAnchorEvidenceEntries(entries []os.DirEntry) ([]trustpolicy.ExternalAnchorEvidencePayload, []trustpolicy.Digest, error) {
	allSidecarDigests := make([]trustpolicy.Digest, 0, len(entries))
	evidence := []trustpolicy.ExternalAnchorEvidencePayload{}
	for _, entry := range entries {
		rec, digest, ok, err := l.readExternalAnchorEvidenceDirEntry(entry)
		if err != nil {
			return nil, nil, err
		}
		if digest != nil {
			allSidecarDigests = append(allSidecarDigests, *digest)
		}
		if ok {
			evidence = append(evidence, rec)
		}
	}
	return evidence, allSidecarDigests, nil
}

func (l *Ledger) readExternalAnchorEvidenceDirEntry(entry os.DirEntry) (trustpolicy.ExternalAnchorEvidencePayload, *trustpolicy.Digest, bool, error) {
	if !isJSONSidecarEntry(entry) {
		return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, nil
	}
	return l.loadExternalAnchorEvidenceEntry(entry.Name())
}

func isJSONSidecarEntry(entry os.DirEntry) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json")
}

func (l *Ledger) loadExternalAnchorEvidenceEntry(name string) (trustpolicy.ExternalAnchorEvidencePayload, *trustpolicy.Digest, bool, error) {
	raw := map[string]any{}
	if err := readJSONFile(filepath.Join(l.rootDir, sidecarDirName, externalAnchorEvidenceDir, name), &raw); err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, err
	}
	digest := externalAnchorEvidenceEntryDigest(name)
	rec, ok, err := decodeExternalAnchorEvidenceRecord(raw)
	if err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, digest, false, err
	}
	return rec, digest, ok, nil
}

func externalAnchorEvidenceEntryDigest(name string) *trustpolicy.Digest {
	d := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(name, ".json")}
	if _, err := d.Identity(); err != nil {
		return nil
	}
	return &d
}

func decodeExternalAnchorEvidenceRecord(raw map[string]any) (trustpolicy.ExternalAnchorEvidencePayload, bool, error) {
	if raw == nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, false, nil
	}
	schemaID, _ := raw["schema_id"].(string)
	if strings.TrimSpace(schemaID) != trustpolicy.ExternalAnchorEvidenceSchemaID {
		return trustpolicy.ExternalAnchorEvidencePayload{}, false, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, true, err
	}
	rec := trustpolicy.ExternalAnchorEvidencePayload{}
	if err := json.Unmarshal(b, &rec); err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, true, err
	}
	if err := trustpolicy.ValidateExternalAnchorEvidencePayload(rec); err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, true, err
	}
	return rec, true, nil
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
