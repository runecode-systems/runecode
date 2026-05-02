package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	proofLookupIndexFileName      = "proof-lookup-index.json"
	proofLookupIndexSchemaVersion = 1
)

type proofLookupIndex struct {
	SchemaVersion         int                                      `json:"schema_version"`
	NextBindingSequence   uint64                                   `json:"next_binding_sequence,omitempty"`
	LatestBindingByRecord map[string]map[string]proofBindingLookup `json:"latest_binding_by_record,omitempty"`
	VerificationByKey     map[string]verificationLookup            `json:"verification_by_key,omitempty"`
	RecordInclusions      map[string]recordInclusionLookup         `json:"record_inclusions,omitempty"`
	SegmentSeals          map[string]segmentSealLookup             `json:"segment_seals,omitempty"`
	SealByChainIndex      map[string]string                        `json:"seal_by_chain_index,omitempty"`
}

type proofBindingLookup struct {
	DigestIdentity string `json:"digest_identity"`
	Sequence       uint64 `json:"sequence"`
}

type verificationLookup struct {
	DigestIdentity string `json:"digest_identity"`
	VerifiedAt     string `json:"verified_at,omitempty"`
}

type recordInclusionLookup struct {
	SegmentID  string `json:"segment_id"`
	FrameIndex int    `json:"frame_index"`
}

type segmentSealLookup struct {
	DigestIdentity string `json:"digest_identity"`
	SealChainIndex int64  `json:"seal_chain_index"`
}

func (l *Ledger) ensureProofLookupIndexLocked() error {
	if l.lookupIndex != nil {
		return nil
	}
	idx, exists, err := l.loadProofLookupIndexLocked()
	if err != nil {
		return err
	}
	if exists {
		l.lookupIndex = idx
		return nil
	}
	return l.refreshProofLookupIndexLocked()
}

func (l *Ledger) refreshProofLookupIndexLocked() error {
	idx, err := l.rebuildProofLookupIndexLocked()
	if err != nil {
		return err
	}
	if err := l.saveProofLookupIndexLocked(idx); err != nil {
		return err
	}
	l.lookupIndex = idx
	return nil
}

func (l *Ledger) loadProofLookupIndexLocked() (*proofLookupIndex, bool, error) {
	path := filepath.Join(l.rootDir, indexDirName, proofLookupIndexFileName)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	idx := proofLookupIndex{}
	if err := readJSONFile(path, &idx); err != nil {
		return nil, false, err
	}
	if idx.SchemaVersion != proofLookupIndexSchemaVersion {
		return nil, false, nil
	}
	normalizeProofLookupIndex(&idx)
	return &idx, true, nil
}

func (l *Ledger) saveProofLookupIndexLocked(idx *proofLookupIndex) error {
	if idx == nil {
		return fmt.Errorf("proof lookup index is required")
	}
	idx.SchemaVersion = proofLookupIndexSchemaVersion
	normalizeProofLookupIndex(idx)
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, proofLookupIndexFileName), idx)
}

func normalizeProofLookupIndex(idx *proofLookupIndex) {
	if idx.LatestBindingByRecord == nil {
		idx.LatestBindingByRecord = map[string]map[string]proofBindingLookup{}
	}
	if idx.VerificationByKey == nil {
		idx.VerificationByKey = map[string]verificationLookup{}
	}
	if idx.RecordInclusions == nil {
		idx.RecordInclusions = map[string]recordInclusionLookup{}
	}
	if idx.SegmentSeals == nil {
		idx.SegmentSeals = map[string]segmentSealLookup{}
	}
	if idx.SealByChainIndex == nil {
		idx.SealByChainIndex = map[string]string{}
	}
}

func (l *Ledger) rebuildProofLookupIndexLocked() (*proofLookupIndex, error) {
	idx := &proofLookupIndex{SchemaVersion: proofLookupIndexSchemaVersion}
	normalizeProofLookupIndex(idx)
	for _, rebuild := range []func(*proofLookupIndex) error{l.rebuildRecordInclusionLookupsLocked, l.rebuildProofBindingLookupsLocked, l.rebuildVerificationLookupsLocked, l.rebuildSealLookupsLocked} {
		if err := rebuild(idx); err != nil {
			return nil, err
		}
	}
	return idx, nil
}

func (l *Ledger) rebuildRecordInclusionLookupsLocked(idx *proofLookupIndex) error {
	segments, err := l.listSegments()
	if err != nil {
		return err
	}
	for _, segment := range segments {
		segmentID := strings.TrimSpace(segment.Header.SegmentID)
		for frameIndex := range segment.Frames {
			if err := idx.notePersistedRecordFrame(segment.Frames[frameIndex].RecordDigest, segmentID, frameIndex); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Ledger) rebuildProofBindingLookupsLocked(idx *proofLookupIndex) error {
	entries, err := l.readOptionalSidecarDirEntries(proofBindingsDirName)
	if err != nil {
		return err
	}
	for i := range entries {
		candidate, ok, err := l.loadVerifiedAuditProofBindingCandidate(entries[i])
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := idx.notePersistedAuditProofBinding(candidate.digest, candidate.payload); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) rebuildVerificationLookupsLocked(idx *proofLookupIndex) error {
	entries, err := l.readOptionalSidecarDirEntries(proofVerificationsDirName)
	if err != nil {
		return err
	}
	for i := range entries {
		identity, payload, ok, err := l.loadVerifiedZKProofVerificationRecordCandidate(entries[i])
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		digest, err := digestFromIdentity(identity)
		if err != nil {
			return err
		}
		if err := idx.notePersistedZKProofVerificationRecord(digest, payload); err != nil {
			return err
		}
	}
	return nil
}

func (idx *proofLookupIndex) notePersistedRecordFrame(recordDigest trustpolicy.Digest, segmentID string, frameIndex int) error {
	if idx == nil {
		return fmt.Errorf("proof lookup index is required")
	}
	normalizeProofLookupIndex(idx)
	identity, err := recordDigest.Identity()
	if err != nil {
		return err
	}
	if _, exists := idx.RecordInclusions[identity]; exists {
		return nil
	}
	idx.RecordInclusions[identity] = recordInclusionLookup{SegmentID: strings.TrimSpace(segmentID), FrameIndex: frameIndex}
	return nil
}
