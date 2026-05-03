package auditd

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) artifactDigestSegmentIDsLocked(index derivedIndex, artifactDigest string) ([]string, error) {
	artifactDigest = strings.TrimSpace(artifactDigest)
	if artifactDigest == "" {
		return nil, nil
	}
	segmentSet := map[string]struct{}{}
	l.addRecordArtifactSegments(index, artifactDigest, segmentSet)
	l.addSealArtifactSegments(index, artifactDigest, segmentSet)
	if err := l.addReceiptArtifactSegmentsLocked(index, artifactDigest, segmentSet); err != nil {
		return nil, err
	}
	if err := l.addReportArtifactSegmentsLocked(index, artifactDigest, segmentSet); err != nil {
		return nil, err
	}
	if err := l.addExternalAnchorArtifactSegmentsLocked(index, artifactDigest, segmentSet); err != nil {
		return nil, err
	}
	if len(segmentSet) == 0 {
		return nil, nil
	}
	return sortedSegmentSet(segmentSet), nil
}

func (l *Ledger) addRecordArtifactSegments(index derivedIndex, artifactDigest string, segmentSet map[string]struct{}) {
	if lookup, ok := index.RecordDigestLookup[artifactDigest]; ok && l.indexHasSealedSegment(index, lookup.SegmentID) {
		segmentSet[lookup.SegmentID] = struct{}{}
	}
}

func (l *Ledger) addSealArtifactSegments(index derivedIndex, artifactDigest string, segmentSet map[string]struct{}) {
	for segmentID, lookup := range index.SegmentSealLookup {
		if strings.TrimSpace(lookup.SealDigest) == artifactDigest {
			segmentSet[segmentID] = struct{}{}
		}
	}
}

func (l *Ledger) addReceiptArtifactSegmentsLocked(index derivedIndex, artifactDigest string, segmentSet map[string]struct{}) error {
	receipt, err := l.loadReceiptEnvelopeByDigestIdentityLocked(artifactDigest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	payload := bundleManifestReceiptPayload{}
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		return err
	}
	subjectIdentity, err := payload.SubjectDigest.Identity()
	if err != nil {
		return err
	}
	for segmentID, lookup := range index.SegmentSealLookup {
		if strings.TrimSpace(lookup.SealDigest) == strings.TrimSpace(subjectIdentity) {
			segmentSet[segmentID] = struct{}{}
		}
	}
	return nil
}

func (l *Ledger) addReportArtifactSegmentsLocked(index derivedIndex, artifactDigest string, segmentSet map[string]struct{}) error {
	report, err := l.loadVerificationReportByDigestIdentityLocked(artifactDigest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	segmentID := strings.TrimSpace(report.VerificationScope.LastSegmentID)
	if l.indexHasSealedSegment(index, segmentID) {
		segmentSet[segmentID] = struct{}{}
	}
	return nil
}

func (l *Ledger) addExternalAnchorArtifactSegmentsLocked(index derivedIndex, artifactDigest string, segmentSet map[string]struct{}) error {
	evidence, digest, ok, err := l.loadExternalAnchorEvidenceByDigestIdentityLocked(artifactDigest)
	if err != nil || !ok || digest == nil {
		return err
	}
	if err := l.addExternalAnchorSubjectSegment(index, evidence.AnchoringSubjectDigest, segmentSet); err != nil {
		return err
	}
	return nil
}

func (l *Ledger) addExternalAnchorSubjectSegment(index derivedIndex, subject trustpolicy.Digest, segmentSet map[string]struct{}) error {
	subjectIdentity, err := subject.Identity()
	if err != nil {
		return err
	}
	for segmentID, lookup := range index.SegmentSealLookup {
		if strings.TrimSpace(lookup.SealDigest) == strings.TrimSpace(subjectIdentity) {
			segmentSet[segmentID] = struct{}{}
		}
	}
	return nil
}

func (l *Ledger) segmentContainsIncidentIDLocked(segmentID string, incidentID string) (bool, error) {
	segment, err := l.loadSegment(segmentID)
	if err != nil {
		return false, err
	}
	for i := range segment.Frames {
		contains, err := frameContainsIncidentID(segment.Frames[i], incidentID)
		if err != nil {
			return false, err
		}
		if contains {
			return true, nil
		}
	}
	return false, nil
}

func frameContainsIncidentID(frame trustpolicy.AuditSegmentRecordFrame, incidentID string) (bool, error) {
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return false, err
	}
	if envelope.PayloadSchemaID != trustpolicy.AuditEventSchemaID {
		return false, nil
	}
	event := trustpolicy.AuditEventPayload{}
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		return false, err
	}
	contains := eventScopeContainsIncidentID(event.Scope, incidentID) || eventScopeContainsIncidentID(event.Correlation, incidentID)
	return contains, nil
}

func eventScopeContainsIncidentID(scope map[string]string, incidentID string) bool {
	if len(scope) == 0 {
		return false
	}
	want := strings.TrimSpace(incidentID)
	for _, key := range []string{"incident_id", "incident", "incident_ref"} {
		if strings.TrimSpace(scope[key]) == want {
			return true
		}
	}
	return false
}

func (l *Ledger) indexHasSealedSegment(index derivedIndex, segmentID string) bool {
	_, ok := index.SegmentSealLookup[strings.TrimSpace(segmentID)]
	return ok
}

func sortedSegmentSet(segmentSet map[string]struct{}) []string {
	out := make([]string, 0, len(segmentSet))
	for segmentID := range segmentSet {
		trimmed := strings.TrimSpace(segmentID)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return out
}
