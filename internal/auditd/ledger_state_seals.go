package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) discoverLatestSealLocked() (digestIdentity string, segmentID string, err error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, sealsDirName))
	if err != nil {
		return "", "", err
	}
	best := discoveredSeal{index: -1}
	for _, entry := range entries {
		next, ok, err := l.discoverSealEntry(entry.Name())
		if err != nil {
			return "", "", err
		}
		if ok && next.index > best.index {
			best = next
		}
	}
	if best.index < 0 {
		return "", "", nil
	}
	return best.digestIdentity, best.segmentID, nil
}

func (l *Ledger) discoverLatestSealFromIndexLocked() (digestIdentity string, segmentID string, err error) {
	index, idxErr := l.ensureDerivedIndexLocked()
	if idxErr != nil {
		return "", "", idxErr
	}
	bestSegment, bestSeal, bestIndex := latestSealCandidate(index.SegmentSealLookup)
	if bestIndex < 0 {
		return "", "", nil
	}
	if validateErr := l.validateLatestSealCandidateLocked(bestSegment, bestSeal, bestIndex); validateErr == nil {
		return bestSeal, bestSegment, nil
	}
	refreshed, refreshErr := l.refreshDerivedIndexLocked("latest seal mismatch")
	if refreshErr != nil {
		return "", "", refreshErr
	}
	bestSegment, bestSeal, bestIndex = latestSealCandidate(refreshed.SegmentSealLookup)
	if bestIndex < 0 {
		return "", "", nil
	}
	if validateErr := l.validateLatestSealCandidateLocked(bestSegment, bestSeal, bestIndex); validateErr != nil {
		return "", "", fmt.Errorf("latest seal mismatch after index refresh: %w", validateErr)
	}
	return bestSeal, bestSegment, nil
}

func latestSealCandidate(lookups map[string]SegmentSealLookup) (segmentID string, sealDigest string, chainIndex int64) {
	bestIndex := int64(-1)
	bestSegment := ""
	bestSeal := ""
	for sid, entry := range lookups {
		if entry.SealChainIndex > bestIndex || (entry.SealChainIndex == bestIndex && sid > bestSegment) {
			bestIndex = entry.SealChainIndex
			bestSegment = sid
			bestSeal = entry.SealDigest
		}
	}
	return bestSegment, bestSeal, bestIndex
}

func (l *Ledger) validateLatestSealCandidateLocked(segmentID string, sealDigest string, chainIndex int64) error {
	return l.validateSegmentSealLookupAgainstCanonicalLocked(segmentID, SegmentSealLookup{SealDigest: sealDigest, SealChainIndex: chainIndex})
}

func (l *Ledger) discoverSealEntry(name string) (discoveredSeal, bool, error) {
	if !strings.HasSuffix(name, ".json") {
		return discoveredSeal{}, false, nil
	}
	path := filepath.Join(l.rootDir, sidecarDirName, sealsDirName, name)
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(path, &envelope); err != nil {
		return discoveredSeal{}, false, err
	}
	seal := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(envelope.Payload, &seal); err != nil {
		return discoveredSeal{}, false, fmt.Errorf("decode seal payload %q: %w", name, err)
	}
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(name, ".json")}
	identity, _ := digest.Identity()
	return discoveredSeal{digestIdentity: identity, segmentID: seal.SegmentID, index: seal.SealChainIndex}, true, nil
}
