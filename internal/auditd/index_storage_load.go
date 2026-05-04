package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (l *Ledger) loadDerivedIndexRecordLookupLocked() (map[string]RecordLookup, error) {
	root := filepath.Join(l.rootDir, indexDirName, indexRecordLookupDirName)
	entries, err := readDerivedIndexDirEntries(root)
	if err != nil {
		return nil, err
	}
	out := map[string]RecordLookup{}
	for _, entry := range entries {
		if !derivedIndexJSONFile(entry) {
			continue
		}
		digest, lookup, err := l.loadDerivedIndexRecordLookupEntryLocked(root, entry.Name())
		if err != nil {
			return nil, err
		}
		out[digest] = lookup
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func readDerivedIndexDirEntries(root string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return entries, nil
}

func derivedIndexJSONFile(entry os.DirEntry) bool {
	return entry != nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json")
}

func (l *Ledger) loadDerivedIndexRecordLookupEntryLocked(root string, filename string) (string, RecordLookup, error) {
	filenameHash := strings.TrimSuffix(filename, ".json")
	row := derivedRecordLookupEntry{}
	if err := readJSONFile(filepath.Join(root, filename), &row); err != nil {
		return "", RecordLookup{}, err
	}
	digest := strings.TrimSpace(row.RecordDigest)
	if err := validateRecordDigestIdentity(digest); err != nil {
		return "", RecordLookup{}, fmt.Errorf("audit evidence index record_digest_lookup key %q invalid: %w", digest, err)
	}
	decoded, err := digestFromIdentity(digest)
	if err != nil {
		return "", RecordLookup{}, err
	}
	if decoded.Hash != filenameHash {
		return "", RecordLookup{}, fmt.Errorf("audit evidence index record_digest_lookup filename mismatch for %q", digest)
	}
	return digest, row.Lookup, nil
}

func (l *Ledger) loadDerivedIndexSegmentSealLookupLocked() (map[string]SegmentSealLookup, error) {
	root := filepath.Join(l.rootDir, indexDirName, indexSegmentSealDirName)
	entries, err := readDerivedIndexDirEntries(root)
	if err != nil {
		return nil, err
	}
	out := map[string]SegmentSealLookup{}
	for _, entry := range entries {
		segmentID, rec, ok, err := l.loadDerivedIndexSegmentSealLookupEntryLocked(root, entry)
		if err != nil {
			return nil, err
		}
		if ok {
			out[segmentID] = rec
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (l *Ledger) loadDerivedIndexSegmentSealLookupEntryLocked(root string, entry os.DirEntry) (string, SegmentSealLookup, bool, error) {
	if !derivedIndexJSONFile(entry) {
		return "", SegmentSealLookup{}, false, nil
	}
	segmentID, ok := segmentIDFromLookupFilename(entry.Name())
	if !ok {
		return "", SegmentSealLookup{}, false, nil
	}
	rec := SegmentSealLookup{}
	if err := readJSONFile(filepath.Join(root, entry.Name()), &rec); err != nil {
		return "", SegmentSealLookup{}, false, err
	}
	return segmentID, rec, true, nil
}

func (l *Ledger) loadDerivedIndexSealChainLookupLocked() (map[string]string, error) {
	root := filepath.Join(l.rootDir, indexDirName, indexSealChainDirName)
	entries, err := readDerivedIndexDirEntries(root)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, entry := range entries {
		chainIndex, sealDigest, ok, err := l.loadDerivedIndexSealChainLookupEntryLocked(root, entry)
		if err != nil {
			return nil, err
		}
		if ok {
			out[chainIndex] = sealDigest
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (l *Ledger) loadDerivedIndexSealChainLookupEntryLocked(root string, entry os.DirEntry) (string, string, bool, error) {
	if !derivedIndexJSONFile(entry) {
		return "", "", false, nil
	}
	chainIndex := strings.TrimSuffix(entry.Name(), ".json")
	if _, err := strconv.ParseInt(chainIndex, 10, 64); err != nil {
		return "", "", false, fmt.Errorf("audit evidence index seal_chain_index_lookup key %q invalid: %w", chainIndex, err)
	}
	rec := derivedSealChainLookupEntry{}
	if err := readJSONFile(filepath.Join(root, entry.Name()), &rec); err != nil {
		return "", "", false, err
	}
	return chainIndex, strings.TrimSpace(rec.SealDigest), true, nil
}

func (l *Ledger) loadDerivedIndexRunTimelineLocked(expectedCount int) ([]TimelinePointer, error) {
	root := filepath.Join(l.rootDir, indexDirName, indexRunTimelineDirName)
	entries, err := readDerivedIndexDirEntries(root)
	if err != nil {
		return nil, err
	}
	files := sortedRunTimelineFilenames(entries)
	out, err := l.loadRunTimelinePointersLocked(root, files)
	if err != nil {
		return nil, err
	}
	if expectedCount > 0 && len(out) != expectedCount {
		return nil, fmt.Errorf("audit evidence index run_timeline count mismatch: meta=%d loaded=%d", expectedCount, len(out))
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func sortedRunTimelineFilenames(entries []os.DirEntry) []string {
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if derivedIndexJSONFile(entry) {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files
}

func (l *Ledger) loadRunTimelinePointersLocked(root string, files []string) ([]TimelinePointer, error) {
	out := make([]TimelinePointer, 0, len(files))
	for _, name := range files {
		pointer := TimelinePointer{}
		if err := readJSONFile(filepath.Join(root, name), &pointer); err != nil {
			return nil, err
		}
		out = append(out, pointer)
	}
	return out, nil
}
