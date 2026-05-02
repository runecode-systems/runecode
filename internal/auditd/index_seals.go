package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type sealMetadata struct {
	SegmentID          string
	SealDigestIdentity string
	SealChainIndex     int64
}

func (l *Ledger) listSealMetadataLocked() ([]sealMetadata, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, sealsDirName))
	if err != nil {
		return nil, err
	}
	metadata := make([]sealMetadata, 0, len(entries))
	for _, entry := range entries {
		entryMetadata, ok, err := l.loadSealMetadataFromEntryLocked(entry)
		if err != nil {
			return nil, err
		}
		if ok {
			metadata = append(metadata, entryMetadata)
		}
	}
	sortSealMetadata(metadata)
	return metadata, nil
}

func (l *Ledger) loadSealMetadataFromEntryLocked(entry os.DirEntry) (sealMetadata, bool, error) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return sealMetadata{}, false, nil
	}
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil {
		return sealMetadata{}, false, err
	}
	if !ok {
		return sealMetadata{}, false, nil
	}
	metadata, err := l.loadSealMetadataByDigestIdentityLocked(identity)
	if err != nil {
		return sealMetadata{}, false, err
	}
	return metadata, true, nil
}

func sortSealMetadata(metadata []sealMetadata) {
	sort.Slice(metadata, func(i, j int) bool {
		if metadata[i].SealChainIndex != metadata[j].SealChainIndex {
			return metadata[i].SealChainIndex < metadata[j].SealChainIndex
		}
		return metadata[i].SealDigestIdentity < metadata[j].SealDigestIdentity
	})
}

func (l *Ledger) loadSealMetadataByDigestIdentityLocked(identity string) (sealMetadata, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return sealMetadata{}, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, sealsDirName, digest.Hash+".json")
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(path, &envelope); err != nil {
		return sealMetadata{}, err
	}
	computedDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return sealMetadata{}, err
	}
	computedIdentity, _ := computedDigest.Identity()
	if computedIdentity != identity {
		return sealMetadata{}, fmt.Errorf("seal sidecar digest mismatch: expected %q computed %q", identity, computedIdentity)
	}
	sealp := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(envelope.Payload, &sealp); err != nil {
		return sealMetadata{}, fmt.Errorf("decode seal payload %q: %w", digest.Hash+".json", err)
	}
	if err := trustpolicy.ValidateAuditSegmentSealPayload(sealp); err != nil {
		return sealMetadata{}, fmt.Errorf("validate seal payload %q: %w", digest.Hash+".json", err)
	}
	return sealMetadata{SegmentID: sealp.SegmentID, SealDigestIdentity: identity, SealChainIndex: sealp.SealChainIndex}, nil
}
