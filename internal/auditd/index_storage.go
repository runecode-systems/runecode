package auditd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var errStaleLegacyDerivedIndexRepresentation = errors.New("audit evidence index legacy representation is newer than sharded representation")

func (l *Ledger) saveDerivedIndexLocked(index derivedIndex) error {
	index = normalizeDerivedIndex(index)
	meta := derivedIndexMetaFromIndex(index)
	if err := l.saveDerivedIndexMetaLocked(meta); err != nil {
		return err
	}
	if err := l.writeDerivedIndexShardsLocked(index); err != nil {
		return err
	}
	legacyPath := filepath.Join(l.rootDir, indexDirName, auditEvidenceIndexFileName)
	if err := os.Remove(legacyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *Ledger) writeDerivedIndexShardsLocked(index derivedIndex) error {
	if err := l.writeDerivedIndexRecordLookupLocked(index.RecordDigestLookup); err != nil {
		return err
	}
	if err := l.writeDerivedIndexSegmentSealLookupLocked(index.SegmentSealLookup); err != nil {
		return err
	}
	if err := l.writeDerivedIndexSealChainLookupLocked(index.SealChainIndexLookup); err != nil {
		return err
	}
	return l.writeDerivedIndexRunTimelineLocked(index.RunTimeline)
}

func (l *Ledger) loadDerivedIndexLocked() (derivedIndex, bool, error) {
	if staleErr := l.failClosedOnStaleLegacyDerivedIndexLocked(); staleErr != nil {
		return derivedIndex{}, false, staleErr
	}
	meta, exists, err := l.loadDerivedIndexMetaLocked()
	if err != nil {
		return derivedIndex{}, false, err
	}
	if !exists {
		if legacy, loaded, err := l.loadLegacyDerivedIndexLocked(); err != nil {
			return derivedIndex{}, false, err
		} else if loaded {
			return legacy, true, nil
		}
		return derivedIndex{}, false, nil
	}
	index, err := l.loadShardedDerivedIndexLocked(meta)
	if err != nil {
		return derivedIndex{}, false, err
	}
	if err := validateDerivedIndexStructure(index); err != nil {
		return derivedIndex{}, false, err
	}
	return normalizeDerivedIndex(index), true, nil
}

func (l *Ledger) loadShardedDerivedIndexLocked(meta derivedIndexMeta) (derivedIndex, error) {
	recordLookup, err := l.loadDerivedIndexRecordLookupLocked()
	if err != nil {
		return derivedIndex{}, err
	}
	segmentSealLookup, err := l.loadDerivedIndexSegmentSealLookupLocked()
	if err != nil {
		return derivedIndex{}, err
	}
	sealChainLookup, err := l.loadDerivedIndexSealChainLookupLocked()
	if err != nil {
		return derivedIndex{}, err
	}
	runTimeline, err := l.loadDerivedIndexRunTimelineLocked(meta.RunTimelineCount)
	if err != nil {
		return derivedIndex{}, err
	}
	return derivedIndex{
		SchemaVersion:                  meta.SchemaVersion,
		BuiltAt:                        meta.BuiltAt,
		TotalRecords:                   meta.TotalRecords,
		LastIndexedSegmentID:           meta.LastIndexedSegmentID,
		LatestVerificationReportDigest: meta.LatestVerificationReportDigest,
		RecordDigestLookup:             recordLookup,
		SegmentSealLookup:              segmentSealLookup,
		SealChainIndexLookup:           sealChainLookup,
		RunTimeline:                    runTimeline,
	}, nil
}

func (l *Ledger) failClosedOnStaleLegacyDerivedIndexLocked() error {
	legacyPath := filepath.Join(l.rootDir, indexDirName, auditEvidenceIndexFileName)
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	metaPath := filepath.Join(l.rootDir, indexDirName, indexMetaFileName)
	if _, err := os.Stat(metaPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	legacyInfo, err := os.Stat(legacyPath)
	if err != nil {
		return err
	}
	metaInfo, err := os.Stat(metaPath)
	if err != nil {
		return err
	}
	if !legacyInfo.ModTime().Before(metaInfo.ModTime()) {
		return errStaleLegacyDerivedIndexRepresentation
	}
	return nil
}

func (l *Ledger) loadLegacyDerivedIndexLocked() (derivedIndex, bool, error) {
	legacyPath := filepath.Join(l.rootDir, indexDirName, auditEvidenceIndexFileName)
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return derivedIndex{}, false, nil
		}
		return derivedIndex{}, false, err
	}
	legacy := derivedIndex{}
	if err := readJSONFile(legacyPath, &legacy); err != nil {
		return derivedIndex{}, false, err
	}
	if legacy.SchemaVersion != auditEvidenceIndexSchemaVersion {
		return derivedIndex{}, false, fmt.Errorf("audit evidence index schema_version %d unsupported", legacy.SchemaVersion)
	}
	if err := validateDerivedIndexStructure(legacy); err != nil {
		return derivedIndex{}, false, err
	}
	normalized := normalizeDerivedIndex(legacy)
	if err := l.saveDerivedIndexLocked(normalized); err != nil {
		return derivedIndex{}, false, err
	}
	return normalized, true, nil
}

func (l *Ledger) loadDerivedIndexMetaLocked() (derivedIndexMeta, bool, error) {
	metaPath := filepath.Join(l.rootDir, indexDirName, indexMetaFileName)
	if _, err := os.Stat(metaPath); err == nil {
		meta := derivedIndexMeta{}
		if err := readJSONFile(metaPath, &meta); err != nil {
			return derivedIndexMeta{}, false, err
		}
		if meta.SchemaVersion != auditEvidenceIndexSchemaVersion {
			return derivedIndexMeta{}, false, fmt.Errorf("audit evidence index schema_version %d unsupported", meta.SchemaVersion)
		}
		return normalizeDerivedIndexMeta(meta), true, nil
	} else if !os.IsNotExist(err) {
		return derivedIndexMeta{}, false, err
	}
	return derivedIndexMeta{}, false, nil
}

func (l *Ledger) saveDerivedIndexMetaLocked(meta derivedIndexMeta) error {
	meta = normalizeDerivedIndexMeta(meta)
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, indexMetaFileName), meta)
}

func (l *Ledger) ensureDerivedIndexLocked() (derivedIndex, error) {
	index, exists, err := l.loadDerivedIndexLocked()
	if err != nil {
		if errors.Is(err, errStaleLegacyDerivedIndexRepresentation) {
			return derivedIndex{}, err
		}
		return l.rebuildAndPersistDerivedIndexLocked()
	}
	if exists {
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

func normalizeDerivedIndexMeta(meta derivedIndexMeta) derivedIndexMeta {
	meta.SchemaVersion = auditEvidenceIndexSchemaVersion
	return meta
}

func derivedIndexMetaFromIndex(index derivedIndex) derivedIndexMeta {
	return normalizeDerivedIndexMeta(derivedIndexMeta{
		SchemaVersion:                  index.SchemaVersion,
		BuiltAt:                        index.BuiltAt,
		TotalRecords:                   index.TotalRecords,
		RunTimelineCount:               len(index.RunTimeline),
		LastIndexedSegmentID:           index.LastIndexedSegmentID,
		LatestVerificationReportDigest: index.LatestVerificationReportDigest,
	})
}
