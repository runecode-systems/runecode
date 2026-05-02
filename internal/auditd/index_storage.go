package auditd

import (
	"fmt"
	"os"
	"path/filepath"
)

func (l *Ledger) saveDerivedIndexLocked(index derivedIndex) error {
	index = normalizeDerivedIndex(index)
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, auditEvidenceIndexFileName), index)
}

func (l *Ledger) loadDerivedIndexLocked() (derivedIndex, bool, error) {
	path := filepath.Join(l.rootDir, indexDirName, auditEvidenceIndexFileName)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return derivedIndex{}, false, nil
		}
		return derivedIndex{}, false, err
	}
	index := derivedIndex{}
	if err := readJSONFile(path, &index); err != nil {
		return derivedIndex{}, false, err
	}
	if index.SchemaVersion != auditEvidenceIndexSchemaVersion {
		return derivedIndex{}, false, fmt.Errorf("audit evidence index schema_version %d unsupported", index.SchemaVersion)
	}
	return normalizeDerivedIndex(index), true, nil
}

func (l *Ledger) ensureDerivedIndexLocked() (derivedIndex, error) {
	index, exists, err := l.loadDerivedIndexLocked()
	if err == nil && exists {
		if validateErr := validateDerivedIndexStructure(index); validateErr == nil {
			return index, nil
		}
	}
	return l.rebuildAndPersistDerivedIndexLocked()
}

func (l *Ledger) refreshDerivedIndexLocked(_ string) (derivedIndex, error) {
	return l.rebuildAndPersistDerivedIndexLocked()
}

func normalizeDerivedIndex(index derivedIndex) derivedIndex {
	index.SchemaVersion = auditEvidenceIndexSchemaVersion
	if index.RecordDigestLookup == nil {
		index.RecordDigestLookup = map[string]RecordLookup{}
	}
	if index.SegmentSealLookup == nil {
		index.SegmentSealLookup = map[string]SegmentSealLookup{}
	}
	if index.SealChainIndexLookup == nil {
		index.SealChainIndexLookup = map[string]string{}
	}
	if len(index.RunTimeline) == 0 {
		index.RunTimeline = nil
	}
	if len(index.RecordDigestLookup) == 0 {
		index.RecordDigestLookup = nil
	}
	if len(index.SegmentSealLookup) == 0 {
		index.SegmentSealLookup = nil
	}
	if len(index.SealChainIndexLookup) == 0 {
		index.SealChainIndexLookup = nil
	}
	return index
}
