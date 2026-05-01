package auditd

import (
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
	_ = sealPayload
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
		SkipFrameAndSealReplay:   true,
		Now:                      l.nowFn(),
	}, nil
}
