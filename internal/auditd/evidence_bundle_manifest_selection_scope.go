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
	case "artifact":
		return l.selectEvidenceBundleSegmentsForArtifactLocked(index, scope.ArtifactDigests)
	case "incident":
		return l.selectEvidenceBundleSegmentsForIncidentLocked(index, scope.IncidentID)
	case "auditor_minimal", "operator_private", "external_relying_party":
		return allSealedSegments, nil
	default:
		return nil, fmt.Errorf("unsupported bundle scope_kind %q", scopeKind)
	}
}

func (l *Ledger) selectEvidenceBundleSegmentsForArtifactLocked(index derivedIndex, artifactDigests []string) ([]string, error) {
	segmentSet := map[string]struct{}{}
	normalizedDigests := normalizeIdentityList(artifactDigests)
	for i := range normalizedDigests {
		matched, err := l.artifactDigestSegmentIDsLocked(index, normalizedDigests[i])
		if err != nil {
			return nil, err
		}
		if len(matched) == 0 {
			return nil, fmt.Errorf("bundle scope artifact digest %q has no sealed evidence", normalizedDigests[i])
		}
		for j := range matched {
			segmentSet[matched[j]] = struct{}{}
		}
	}
	if len(segmentSet) == 0 {
		return nil, fmt.Errorf("bundle scope artifact has no sealed evidence")
	}
	out := make([]string, 0, len(segmentSet))
	for segmentID := range segmentSet {
		out = append(out, segmentID)
	}
	sort.Strings(out)
	return out, nil
}

func (l *Ledger) selectEvidenceBundleSegmentsForIncidentLocked(index derivedIndex, incidentID string) ([]string, error) {
	incidentID = strings.TrimSpace(incidentID)
	matchedSet := map[string]struct{}{}
	for segmentID := range index.SegmentSealLookup {
		hasIncident, err := l.segmentContainsIncidentIDLocked(segmentID, incidentID)
		if err != nil {
			return nil, err
		}
		if hasIncident {
			matchedSet[segmentID] = struct{}{}
		}
	}
	if len(matchedSet) == 0 {
		return nil, fmt.Errorf("bundle scope incident_id %q has no sealed evidence", incidentID)
	}
	matched := make([]string, 0, len(matchedSet))
	for segmentID := range matchedSet {
		matched = append(matched, segmentID)
	}
	sort.Strings(matched)
	return matched, nil
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
