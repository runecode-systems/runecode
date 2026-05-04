package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (l *Ledger) writeDerivedIndexRecordLookupLocked(lookup map[string]RecordLookup) error {
	root := filepath.Join(l.rootDir, indexDirName, indexRecordLookupDirName)
	if err := os.RemoveAll(root); err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(lookup) == 0 {
		return nil
	}
	entries := make([]string, 0, len(lookup))
	for digest := range lookup {
		entries = append(entries, digest)
	}
	sort.Strings(entries)
	for _, digest := range entries {
		rec := lookup[digest]
		entry := derivedRecordLookupEntry{RecordDigest: digest, Lookup: rec}
		if err := writeCanonicalJSONFile(filepath.Join(root, digestLookupFilename(digest)), entry); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) upsertDerivedIndexRecordLookupLocked(recordDigest string, lookup RecordLookup) error {
	entry := derivedRecordLookupEntry{RecordDigest: recordDigest, Lookup: lookup}
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, indexRecordLookupDirName, digestLookupFilename(recordDigest)), entry)
}

func (l *Ledger) writeDerivedIndexSegmentSealLookupLocked(lookup map[string]SegmentSealLookup) error {
	root := filepath.Join(l.rootDir, indexDirName, indexSegmentSealDirName)
	if err := os.RemoveAll(root); err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(lookup) == 0 {
		return nil
	}
	segmentIDs := make([]string, 0, len(lookup))
	for segmentID := range lookup {
		segmentIDs = append(segmentIDs, segmentID)
	}
	sort.Strings(segmentIDs)
	for _, segmentID := range segmentIDs {
		if err := writeCanonicalJSONFile(filepath.Join(root, segmentLookupFilename(segmentID)), lookup[segmentID]); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) upsertDerivedIndexSegmentSealLookupLocked(segmentID string, lookup SegmentSealLookup) error {
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, indexSegmentSealDirName, segmentLookupFilename(segmentID)), lookup)
}

func (l *Ledger) writeDerivedIndexSealChainLookupLocked(lookup map[string]string) error {
	root := filepath.Join(l.rootDir, indexDirName, indexSealChainDirName)
	if err := os.RemoveAll(root); err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(lookup) == 0 {
		return nil
	}
	chainIndices := make([]string, 0, len(lookup))
	for chainIndex := range lookup {
		chainIndices = append(chainIndices, chainIndex)
	}
	sort.Strings(chainIndices)
	for _, chainIndex := range chainIndices {
		if err := writeCanonicalJSONFile(filepath.Join(root, sealChainLookupFilename(chainIndex)), derivedSealChainLookupEntry{SealDigest: lookup[chainIndex]}); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) upsertDerivedIndexSealChainLookupLocked(chainIndex int64, sealDigest string) error {
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, indexSealChainDirName, sealChainLookupFilename(strconv.FormatInt(chainIndex, 10))), derivedSealChainLookupEntry{SealDigest: sealDigest})
}

func (l *Ledger) writeDerivedIndexRunTimelineLocked(timeline []TimelinePointer) error {
	root := filepath.Join(l.rootDir, indexDirName, indexRunTimelineDirName)
	if err := os.RemoveAll(root); err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(timeline) == 0 {
		return nil
	}
	for index := range timeline {
		if err := writeCanonicalJSONFile(filepath.Join(root, runTimelineFilename(index)), timeline[index]); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) appendDerivedIndexRunTimelinePointerLocked(pointer TimelinePointer) (int, error) {
	meta, exists, err := l.loadDerivedIndexMetaLocked()
	if err != nil {
		return 0, err
	}
	if !exists {
		meta = normalizeDerivedIndexMeta(derivedIndexMeta{})
	}
	index := meta.RunTimelineCount
	if err := writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, indexRunTimelineDirName, runTimelineFilename(index)), pointer); err != nil {
		return 0, err
	}
	return index + 1, nil
}

func (l *Ledger) hasDerivedIndexRecordLookupLocked(recordDigest string) (bool, error) {
	path := filepath.Join(l.rootDir, indexDirName, indexRecordLookupDirName, digestLookupFilename(recordDigest))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func digestLookupFilename(identity string) string {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return strings.TrimSpace(identity) + ".json"
	}
	return digest.Hash + ".json"
}

func segmentLookupFilename(segmentID string) string {
	return strings.NewReplacer("/", "%2F", string(filepath.Separator), "%2F").Replace(strings.TrimSpace(segmentID)) + ".json"
}

func segmentIDFromLookupFilename(name string) (string, bool) {
	if !strings.HasSuffix(name, ".json") {
		return "", false
	}
	trimmed := strings.TrimSuffix(name, ".json")
	trimmed = strings.ReplaceAll(trimmed, "%2F", "/")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func sealChainLookupFilename(chainIndex string) string {
	return strings.TrimSpace(chainIndex) + ".json"
}

func runTimelineFilename(index int) string {
	return fmt.Sprintf("%09d.json", index)
}

type derivedSealChainLookupEntry struct {
	SealDigest string `json:"seal_digest"`
}

type derivedRecordLookupEntry struct {
	RecordDigest string       `json:"record_digest"`
	Lookup       RecordLookup `json:"lookup"`
}
