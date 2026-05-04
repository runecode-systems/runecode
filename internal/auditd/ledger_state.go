package auditd

import (
	"crypto/rand"
	"encoding/hex"
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
	if persisted, loadErr := l.loadState(); loadErr == nil && strings.TrimSpace(persisted.LedgerIdentity) != "" {
		state.LedgerIdentity = strings.TrimSpace(persisted.LedgerIdentity)
	}
	if err := ensureLedgerIdentity(&state); err != nil {
		return ledgerState{}, err
	}
	state.LastVerificationReportDigest = l.recoverLatestVerificationReportDigestLocked()
	return state, nil
}

func (l *Ledger) attachSealAndIndexStateLocked(state *ledgerState) error {
	sealDigest, lastSealed, err := l.discoverLatestSealFromIndexLocked()
	if err != nil {
		return err
	}
	if sealDigest == "" && lastSealed == "" {
		sealDigest, lastSealed, err = l.discoverLatestSealLocked()
		if err != nil {
			return err
		}
	}
	state.LastSealEnvelopeDigest = sealDigest
	state.LastSealedSegmentID = lastSealed
	if index, idxErr := l.ensureDerivedIndexLocked(); idxErr == nil {
		state.LastIndexedRecordCount = index.TotalRecords
		if index.LatestVerificationReportDigest != "" {
			state.LastVerificationReportDigest = index.LatestVerificationReportDigest
		}
	} else if indexCount, _, statusErr := l.indexStatusLocked(); statusErr == nil {
		state.LastIndexedRecordCount = indexCount
	}
	return nil
}

func (l *Ledger) bootstrapInitialStateLocked() (ledgerState, error) {
	initial := newOpenSegment(nextSegmentID(1), l.nowFn())
	if err := l.saveSegment(initial); err != nil {
		return ledgerState{}, err
	}
	state := ledgerState{SchemaVersion: stateSchemaVersion, CurrentOpenSegmentID: initial.Header.SegmentID, NextSegmentNumber: 2, OpenFrameCount: 0, RecoveryComplete: true}
	if err := ensureLedgerIdentity(&state); err != nil {
		return ledgerState{}, err
	}
	return state, nil
}

func ensureLedgerIdentity(state *ledgerState) error {
	if state == nil {
		return fmt.Errorf("ledger state is required")
	}
	if strings.TrimSpace(state.LedgerIdentity) != "" {
		return nil
	}
	identity, err := newLedgerIdentity()
	if err != nil {
		return err
	}
	state.LedgerIdentity = identity
	return nil
}

func newLedgerIdentity() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "ledger-" + hex.EncodeToString(raw), nil
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
