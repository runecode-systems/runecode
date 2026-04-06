package auditd

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) AppendAdmittedEvent(req trustpolicy.AuditAdmissionRequest) (AppendResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := trustpolicy.ValidateAuditAdmissionRequest(req); err != nil {
		return AppendResult{}, err
	}
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return AppendResult{}, err
	}
	openSegment, err := l.loadSegment(state.CurrentOpenSegmentID)
	if err != nil {
		return AppendResult{}, err
	}
	frame, err := frameForEnvelope(req.Envelope)
	if err != nil {
		return AppendResult{}, err
	}
	openSegment.Frames = append(openSegment.Frames, frame)
	openSegment.LifecycleMarker.MarkedAt = l.nowFn().UTC().Format(time.RFC3339)
	if err := l.saveSegment(openSegment); err != nil {
		return AppendResult{}, err
	}
	state.OpenFrameCount = len(openSegment.Frames)
	if err := l.saveState(state); err != nil {
		return AppendResult{}, err
	}
	return AppendResult{SegmentID: openSegment.Header.SegmentID, RecordDigest: frame.RecordDigest, ByteLength: frame.ByteLength, FrameCount: len(openSegment.Frames)}, nil
}

func frameForEnvelope(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.AuditSegmentRecordFrame, error) {
	canonicalEnvelopeBytes, digest, err := canonicalEnvelopeAndDigest(envelope)
	if err != nil {
		return trustpolicy.AuditSegmentRecordFrame{}, err
	}
	return trustpolicy.AuditSegmentRecordFrame{RecordDigest: digest, ByteLength: int64(len(canonicalEnvelopeBytes)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelopeBytes)}, nil
}

func (l *Ledger) SealCurrentSegment(sealEnvelope trustpolicy.SignedObjectEnvelope) (SealResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state, segment, err := l.currentOpenSegmentForSealLocked()
	if err != nil {
		return SealResult{}, err
	}
	if err := l.validateSealForSegment(segment, state, sealEnvelope); err != nil {
		return SealResult{}, err
	}
	sealDigest, err := l.persistEnvelopeSidecar(sealsDirName, sealEnvelope)
	if err != nil {
		return SealResult{}, err
	}
	nextOpen, err := l.sealAndRotateSegmentsLocked(state, segment, sealDigest)
	if err != nil {
		return SealResult{}, err
	}
	return SealResult{SegmentID: segment.Header.SegmentID, SealEnvelopeDigest: sealDigest, NextOpenSegmentID: nextOpen.Header.SegmentID}, nil
}

func (l *Ledger) currentOpenSegmentForSealLocked() (ledgerState, trustpolicy.AuditSegmentFilePayload, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return ledgerState{}, trustpolicy.AuditSegmentFilePayload{}, err
	}
	segment, err := l.loadSegment(state.CurrentOpenSegmentID)
	if err != nil {
		return ledgerState{}, trustpolicy.AuditSegmentFilePayload{}, err
	}
	if len(segment.Frames) == 0 {
		return ledgerState{}, trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("cannot seal empty segment")
	}
	return state, segment, nil
}

func (l *Ledger) sealAndRotateSegmentsLocked(state ledgerState, segment trustpolicy.AuditSegmentFilePayload, sealDigest trustpolicy.Digest) (trustpolicy.AuditSegmentFilePayload, error) {
	segment.Header.SegmentState = trustpolicy.AuditSegmentStateSealed
	segment.LifecycleMarker.State = trustpolicy.AuditSegmentStateSealed
	segment.LifecycleMarker.MarkedAt = l.nowFn().UTC().Format(time.RFC3339)
	if err := l.saveSegment(segment); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, err
	}
	nextOpen := newOpenSegment(nextSegmentID(state.NextSegmentNumber), l.nowFn())
	if err := l.saveSegment(nextOpen); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, err
	}
	identity, _ := sealDigest.Identity()
	state.LastSealedSegmentID = segment.Header.SegmentID
	state.LastSealEnvelopeDigest = identity
	state.CurrentOpenSegmentID = nextOpen.Header.SegmentID
	state.NextSegmentNumber++
	state.OpenFrameCount = 0
	if err := l.saveState(state); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, err
	}
	return nextOpen, nil
}
