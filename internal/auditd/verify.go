package auditd

import (
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) VerifyCurrentSegmentAndPersist() (VerificationResult, error) {
	l.mu.Lock()
	snapshot, err := l.currentSealSnapshotLocked()
	l.mu.Unlock()
	if err != nil {
		return VerificationResult{}, err
	}
	segment, input, err := l.currentVerificationContextForSnapshot(snapshot)
	if err != nil {
		return VerificationResult{}, err
	}
	report, err := trustpolicy.VerifyAuditEvidence(input)
	if err != nil {
		return VerificationResult{}, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.requireSealSnapshotCurrentLocked(snapshot); err != nil {
		return VerificationResult{}, err
	}
	reportDigest, err := l.persistVerificationReportLocked(report)
	if err != nil {
		return VerificationResult{}, err
	}
	if err := l.noteIncrementalVerificationBaselineLocked(segment.Header.SegmentID, snapshot.sealDigest, report, reportDigest); err != nil {
		return VerificationResult{}, err
	}
	return VerificationResult{SegmentID: segment.Header.SegmentID, ReportDigest: reportDigest, Report: report}, nil
}

func (l *Ledger) VerifyCurrentSegmentIncrementalWithPreverifiedSeal(preverifiedSealDigest trustpolicy.Digest, verifier trustpolicy.VerifierRecord) (trustpolicy.Digest, error) {
	l.mu.Lock()
	snapshot, err := l.currentSealSnapshotLocked()
	l.mu.Unlock()
	if err != nil {
		return trustpolicy.Digest{}, err
	}

	if _, err := preverifiedSealDigest.Identity(); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("preverified_seal_digest: %w", err)
	}
	if mustDigestIdentity(snapshot.sealDigest) != mustDigestIdentity(preverifiedSealDigest) {
		return trustpolicy.Digest{}, fmt.Errorf("preverified seal digest does not match current segment seal")
	}
	input, err := l.incrementalVerificationInput(snapshot, preverifiedSealDigest, verifier)
	if err != nil {
		return trustpolicy.Digest{}, err
	}

	report, err := trustpolicy.VerifyAuditEvidence(input)
	if err != nil {
		return trustpolicy.Digest{}, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.requireSealSnapshotCurrentLocked(snapshot); err != nil {
		return trustpolicy.Digest{}, err
	}
	if err := l.ensureVerifierRecordDurableLocked(verifier); err != nil {
		return trustpolicy.Digest{}, err
	}
	return l.persistVerificationReportLocked(report)
}

func (l *Ledger) incrementalVerificationInput(snapshot currentSealSnapshot, preverifiedSealDigest trustpolicy.Digest, verifier trustpolicy.VerifierRecord) (trustpolicy.AuditVerificationInput, error) {
	segment, sealEnvelope, sealPayload, previousDigest, rawBytes, err := l.currentSegmentEvidenceForSnapshot(snapshot)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	verifiedEvents, err := validatedPreverifiedEvents(segment, sealPayload)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	contractInputs, sealDigests, sealScoped, err := l.incrementalVerificationDependencies(segment.Header.SegmentID, snapshot.sealDigest)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	verifiers, _, err := addVerifierRecordIfMissing(contractInputs.verifierRecords, verifier)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	return trustpolicy.AuditVerificationInput{
		Scope:                    trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: segment.Header.SegmentID},
		Segment:                  segment,
		RawFramedSegmentBytes:    rawBytes,
		SegmentSealEnvelope:      sealEnvelope,
		PreviousSealEnvelopeHash: previousDigest,
		KnownSealDigests:         sealDigests,
		ExternalAnchorTargetSet:  sealScoped.externalAnchorTargetSet,
		ReceiptEnvelopes:         sealScoped.receipts,
		VerifierRecords:          verifiers,
		EventContractCatalog:     contractInputs.catalog,
		SignerEvidence:           contractInputs.signerEvidence,
		StoragePostureEvidence:   contractInputs.storagePosture,
		ExternalAnchorEvidence:   sealScoped.externalAnchorEvidence,
		ExternalAnchorSidecars:   sealScoped.externalAnchorSidecars,
		PreverifiedSealDigest:    &preverifiedSealDigest,
		PreverifiedSealPayload:   &sealPayload,
		PreverifiedEvents:        verifiedEvents,
		SkipFrameAndSealReplay:   true,
		TrustedPreverifiedSeal:   true,
		Now:                      l.nowFn(),
	}, nil
}

func validatedPreverifiedEvents(segment trustpolicy.AuditSegmentFilePayload, sealPayload trustpolicy.AuditSegmentSealPayload) ([]trustpolicy.AuditEventPayload, error) {
	events, err := preverifiedEventsFromSegment(segment)
	if err != nil {
		return nil, err
	}
	if err := preverifiedFramesMatchSeal(segment, sealPayload); err != nil {
		return nil, err
	}
	return events, nil
}

func preverifiedEventsFromSegment(segment trustpolicy.AuditSegmentFilePayload) ([]trustpolicy.AuditEventPayload, error) {
	if len(segment.Frames) == 0 {
		return nil, nil
	}
	events := make([]trustpolicy.AuditEventPayload, 0, len(segment.Frames))
	for i := range segment.Frames {
		envelope, err := decodeFrameEnvelope(segment.Frames[i])
		if err != nil {
			return nil, err
		}
		if envelope.PayloadSchemaID != trustpolicy.AuditEventSchemaID {
			continue
		}
		event := trustpolicy.AuditEventPayload{}
		if err := json.Unmarshal(envelope.Payload, &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func preverifiedFramesMatchSeal(segment trustpolicy.AuditSegmentFilePayload, sealPayload trustpolicy.AuditSegmentSealPayload) error {
	if len(segment.Frames) == 0 {
		return nil
	}
	frameDigests := make([]trustpolicy.Digest, 0, len(segment.Frames))
	for i := range segment.Frames {
		frameDigests = append(frameDigests, segment.Frames[i].RecordDigest)
	}
	merkleRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(frameDigests)
	if err != nil {
		return err
	}
	if mustDigestIdentity(merkleRoot) != mustDigestIdentity(sealPayload.MerkleRoot) {
		return fmt.Errorf("preverified events do not match seal merkle root")
	}
	return nil
}
