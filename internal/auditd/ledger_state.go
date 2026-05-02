package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) recoverAndPersistStateLocked() (ledgerState, error) {
	state, err := l.recoverStateLocked()
	if err != nil {
		return ledgerState{}, err
	}
	if err := l.saveState(state); err != nil {
		return ledgerState{}, err
	}
	return state, nil
}

func (l *Ledger) recoverStateLocked() (ledgerState, error) {
	segments, err := l.listSegments()
	if err != nil {
		return ledgerState{}, err
	}
	if len(segments) == 0 {
		return l.bootstrapInitialStateLocked()
	}
	openSegments, maxNumber, _ := summarizeSegmentSet(segments)
	if len(openSegments) > 1 {
		return ledgerState{}, fmt.Errorf("multiple open segments detected")
	}
	state, err := l.resolveOpenSegmentStateLocked(openSegments, maxNumber)
	if err != nil {
		return ledgerState{}, err
	}
	if err := l.attachSealAndIndexStateLocked(&state); err != nil {
		return ledgerState{}, err
	}
	state.LastVerificationReportDigest = l.recoverLatestVerificationReportDigestLocked()
	return state, nil
}

func (l *Ledger) attachSealAndIndexStateLocked(state *ledgerState) error {
	sealDigest, lastSealed, err := l.discoverLatestSealLocked()
	if err != nil {
		return err
	}
	state.LastSealEnvelopeDigest = sealDigest
	state.LastSealedSegmentID = lastSealed
	if index, _, err := l.indexStatusLocked(); err == nil {
		state.LastIndexedRecordCount = index
	}
	return nil
}

func (l *Ledger) preservedLastReportDigestLocked() string {
	prevState, err := l.loadState()
	if err != nil {
		return ""
	}
	return prevState.LastVerificationReportDigest
}

func (l *Ledger) recoverLatestVerificationReportDigestLocked() string {
	latestDigest := l.discoverLatestVerificationReportDigestLocked()
	if latestDigest != "" {
		return latestDigest
	}
	return l.preservedLastReportDigestLocked()
}

type verificationReportCandidate struct {
	digest     string
	verifiedAt time.Time
}

func (l *Ledger) discoverLatestVerificationReportDigestLocked() string {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName))
	if err != nil {
		return ""
	}
	best := verificationReportCandidate{}
	for _, entry := range entries {
		next, ok := l.verificationReportCandidateFromEntry(entry)
		if !ok {
			continue
		}
		if chooseVerificationReportCandidate(best, next) {
			best = next
		}
	}
	return best.digest
}

func chooseVerificationReportCandidate(current, next verificationReportCandidate) bool {
	if current.digest == "" {
		return true
	}
	if next.verifiedAt.After(current.verifiedAt) {
		return true
	}
	return next.verifiedAt.Equal(current.verifiedAt) && next.digest > current.digest
}

func (l *Ledger) verificationReportCandidateFromEntry(entry os.DirEntry) (verificationReportCandidate, bool) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return verificationReportCandidate{}, false
	}
	report := trustpolicy.AuditVerificationReportPayload{}
	path := filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName, entry.Name())
	if err := readJSONFile(path, &report); err != nil {
		return verificationReportCandidate{}, false
	}
	verifiedAt, err := time.Parse(time.RFC3339, report.VerifiedAt)
	if err != nil {
		return verificationReportCandidate{}, false
	}
	digest := "sha256:" + strings.TrimSuffix(entry.Name(), ".json")
	return verificationReportCandidate{digest: digest, verifiedAt: verifiedAt}, true
}

func (l *Ledger) bootstrapInitialStateLocked() (ledgerState, error) {
	initial := newOpenSegment(nextSegmentID(1), l.nowFn())
	if err := l.saveSegment(initial); err != nil {
		return ledgerState{}, err
	}
	return ledgerState{SchemaVersion: stateSchemaVersion, CurrentOpenSegmentID: initial.Header.SegmentID, NextSegmentNumber: 2, OpenFrameCount: 0, RecoveryComplete: true}, nil
}

func summarizeSegmentSet(segments []trustpolicy.AuditSegmentFilePayload) ([]trustpolicy.AuditSegmentFilePayload, int64, error) {
	openSegments := make([]trustpolicy.AuditSegmentFilePayload, 0, 1)
	var maxNumber int64
	for _, segment := range segments {
		number := parseSegmentNumber(segment.Header.SegmentID)
		if number > maxNumber {
			maxNumber = number
		}
		if segment.Header.SegmentState == trustpolicy.AuditSegmentStateOpen {
			openSegments = append(openSegments, segment)
		}
	}
	return openSegments, maxNumber, nil
}

func (l *Ledger) resolveOpenSegmentStateLocked(openSegments []trustpolicy.AuditSegmentFilePayload, maxNumber int64) (ledgerState, error) {
	state := ledgerState{SchemaVersion: stateSchemaVersion, NextSegmentNumber: maxNumber + 1, RecoveryComplete: true}
	if len(openSegments) == 1 {
		state.CurrentOpenSegmentID = openSegments[0].Header.SegmentID
		state.OpenFrameCount = len(openSegments[0].Frames)
		return state, nil
	}
	nextID := nextSegmentID(maxNumber + 1)
	open := newOpenSegment(nextID, l.nowFn())
	if err := l.saveSegment(open); err != nil {
		return ledgerState{}, err
	}
	state.CurrentOpenSegmentID = open.Header.SegmentID
	state.NextSegmentNumber = maxNumber + 2
	return state, nil
}

type discoveredSeal struct {
	digestIdentity string
	segmentID      string
	index          int64
}

func (l *Ledger) discoverLatestSealLocked() (digestIdentity string, segmentID string, err error) {
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return "", "", err
	}
	best := discoveredSeal{index: -1}
	for segmentID, lookup := range l.lookupIndex.SegmentSeals {
		if lookup.SealChainIndex > best.index || (lookup.SealChainIndex == best.index && lookup.DigestIdentity > best.digestIdentity) {
			best = discoveredSeal{digestIdentity: lookup.DigestIdentity, segmentID: segmentID, index: lookup.SealChainIndex}
		}
	}
	if best.index < 0 {
		return "", "", nil
	}
	return best.digestIdentity, best.segmentID, nil
}

func nextSegmentID(number int64) string {
	if number < 1 {
		number = 1
	}
	return fmt.Sprintf("segment-%06d", number)
}

func parseSegmentNumber(segmentID string) int64 {
	parts := strings.Split(segmentID, "-")
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func newOpenSegment(segmentID string, now time.Time) trustpolicy.AuditSegmentFilePayload {
	timestamp := now.UTC().Format(time.RFC3339)
	return trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: segmentID, SegmentState: trustpolicy.AuditSegmentStateOpen, CreatedAt: timestamp, Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateOpen, MarkedAt: timestamp}}
}

func (l *Ledger) listSegments() ([]trustpolicy.AuditSegmentFilePayload, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, segmentsDirName))
	if err != nil {
		return nil, err
	}
	segments := make([]trustpolicy.AuditSegmentFilePayload, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		segment := trustpolicy.AuditSegmentFilePayload{}
		if err := readJSONFile(filepath.Join(l.rootDir, segmentsDirName, entry.Name()), &segment); err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	sort.Slice(segments, func(i, j int) bool {
		return parseSegmentNumber(segments[i].Header.SegmentID) < parseSegmentNumber(segments[j].Header.SegmentID)
	})
	return segments, nil
}
