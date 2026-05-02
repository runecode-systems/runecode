package auditd

import (
	"fmt"
	"sort"
	"strings"
)

func (l *Ledger) selectEvidenceBundleSegmentIDsLocked(scope AuditEvidenceBundleScope) ([]string, error) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return nil, err
	}
	allSealedSegments := sealedSegmentIDsFromIndex(index)
	if len(allSealedSegments) == 0 {
		return nil, fmt.Errorf("bundle export requires at least one sealed segment")
	}
	return l.resolveEvidenceBundleSegmentIDsForScope(index, scope, allSealedSegments)
}

func sealedSegmentIDsFromIndex(index derivedIndex) []string {
	allSealedSegments := make([]string, 0, len(index.SegmentSealLookup))
	for segmentID := range index.SegmentSealLookup {
		trimmed := strings.TrimSpace(segmentID)
		if trimmed != "" {
			allSealedSegments = append(allSealedSegments, trimmed)
		}
	}
	sort.Strings(allSealedSegments)
	return allSealedSegments
}

func (l *Ledger) resolveEvidenceBundleSegmentIDsForScope(index derivedIndex, scope AuditEvidenceBundleScope, allSealedSegments []string) ([]string, error) {
	switch scopeKind := strings.TrimSpace(scope.ScopeKind); scopeKind {
	case "run":
		return l.selectEvidenceBundleSegmentsForRunLocked(index, scope.RunID)
	case "artifact", "incident":
		return nil, fmt.Errorf("bundle scope_kind %q is not yet supported for deterministic segment resolution in this lane", scopeKind)
	case "auditor_minimal", "operator_private", "external_relying_party":
		return allSealedSegments, nil
	default:
		return nil, fmt.Errorf("unsupported bundle scope_kind %q", scopeKind)
	}
}

func (l *Ledger) selectEvidenceBundleSegmentsForRunLocked(index derivedIndex, runID string) ([]string, error) {
	runID = strings.TrimSpace(runID)
	matchedSet := map[string]struct{}{}
	for i := range index.RunTimeline {
		if strings.TrimSpace(index.RunTimeline[i].RunID) != runID {
			continue
		}
		segmentID := strings.TrimSpace(index.RunTimeline[i].SegmentID)
		if segmentID == "" {
			continue
		}
		if _, ok := index.SegmentSealLookup[segmentID]; ok {
			matchedSet[segmentID] = struct{}{}
		}
	}
	if len(matchedSet) == 0 {
		return nil, fmt.Errorf("bundle scope run_id %q has no sealed evidence", runID)
	}
	matched := make([]string, 0, len(matchedSet))
	for segmentID := range matchedSet {
		matched = append(matched, segmentID)
	}
	sort.Strings(matched)
	return matched, nil
}
