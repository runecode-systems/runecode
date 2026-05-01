package auditd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) loadVerificationContractInputsOnlyLocked() (verificationInputs, error) {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	if err := requireVerificationContractFiles(contractsDir); err != nil {
		return verificationInputs{}, err
	}
	return loadVerificationContractInputs(contractsDir)
}

func (l *Ledger) incrementalVerificationDependencies(segmentID string, sealDigest trustpolicy.Digest) (verificationInputs, []trustpolicy.Digest, sealScopedVerificationInputs, error) {
	contractInputs, err := l.loadVerificationContractInputsOnlyLocked()
	if err != nil {
		return verificationInputs{}, nil, sealScopedVerificationInputs{}, err
	}
	sealDigests, err := l.loadAllSealDigestsLocked()
	if err != nil {
		return verificationInputs{}, nil, sealScopedVerificationInputs{}, err
	}
	sealScoped, err := l.loadSealScopedVerificationDurableInputsLocked(segmentID, sealDigest)
	if err != nil {
		return verificationInputs{}, nil, sealScopedVerificationInputs{}, err
	}
	return contractInputs, sealDigests, sealScoped, nil
}

func (l *Ledger) currentVerificationContextLocked() (trustpolicy.AuditSegmentFilePayload, trustpolicy.AuditVerificationInput, error) {
	snapshot, err := l.currentSealSnapshotLocked()
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditVerificationInput{}, err
	}
	return l.currentVerificationContextForSnapshot(snapshot)
}

func (l *Ledger) currentVerificationContextForSnapshot(snapshot currentSealSnapshot) (trustpolicy.AuditSegmentFilePayload, trustpolicy.AuditVerificationInput, error) {
	segment, sealEnvelope, sealPayload, previousDigest, rawBytes, err := l.currentSegmentEvidenceForSnapshot(snapshot)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditVerificationInput{}, err
	}
	runtimeInputs, err := l.loadVerificationInputsLocked()
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditVerificationInput{}, err
	}
	targetSet, err := l.loadExternalAnchorVerificationTargetSetForSealLocked(segment.Header.SegmentID, snapshot.sealDigest)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditVerificationInput{}, err
	}
	input := trustpolicy.AuditVerificationInput{
		Scope:                    trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: segment.Header.SegmentID},
		Segment:                  segment,
		RawFramedSegmentBytes:    rawBytes,
		SegmentSealEnvelope:      sealEnvelope,
		PreviousSealEnvelopeHash: previousDigest,
		KnownSealDigests:         runtimeInputs.knownSealDigests,
		ExternalAnchorTargetSet:  targetSet,
		ReceiptEnvelopes:         runtimeInputs.receipts,
		VerifierRecords:          runtimeInputs.verifierRecords,
		EventContractCatalog:     runtimeInputs.catalog,
		SignerEvidence:           runtimeInputs.signerEvidence,
		StoragePostureEvidence:   runtimeInputs.storagePosture,
		ExternalAnchorEvidence:   runtimeInputs.externalAnchorEvidence,
		ExternalAnchorSidecars:   runtimeInputs.externalAnchorSidecars,
		Now:                      l.nowFn(),
	}
	_ = sealPayload
	return segment, input, nil
}

type currentSealSnapshot struct {
	segmentID  string
	sealDigest trustpolicy.Digest
}

func (l *Ledger) currentSealSnapshotLocked() (currentSealSnapshot, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return currentSealSnapshot{}, err
	}
	if strings.TrimSpace(state.LastSealedSegmentID) == "" {
		return currentSealSnapshot{}, fmt.Errorf("no sealed segment available for verification")
	}
	sealdigest, err := l.currentSealDigestForStateLocked(state.LastSealedSegmentID, strings.TrimSpace(state.LastSealEnvelopeDigest))
	if err != nil {
		return currentSealSnapshot{}, err
	}
	return currentSealSnapshot{segmentID: state.LastSealedSegmentID, sealDigest: sealdigest}, nil
}

func (l *Ledger) currentSealDigestForStateLocked(segmentID, sealIdentity string) (trustpolicy.Digest, error) {
	if strings.TrimSpace(sealIdentity) == "" {
		_, digest, _, err := l.loadSealEnvelopeForSegmentLocked(segmentID)
		if err != nil {
			return trustpolicy.Digest{}, err
		}
		sealIdentity, _ = digest.Identity()
	}
	sealdigest, err := digestFromIdentity(sealIdentity)
	if err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("state last seal envelope digest invalid: %w", err)
	}
	return sealdigest, nil
}

func (l *Ledger) requireSealSnapshotCurrentLocked(snapshot currentSealSnapshot) error {
	current, err := l.currentSealSnapshotLocked()
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.segmentID) != strings.TrimSpace(snapshot.segmentID) || mustDigestIdentity(current.sealDigest) != mustDigestIdentity(snapshot.sealDigest) {
		return fmt.Errorf("current sealed segment changed during verification; retry")
	}
	return nil
}

func (l *Ledger) currentSegmentEvidenceForSnapshot(snapshot currentSealSnapshot) (trustpolicy.AuditSegmentFilePayload, trustpolicy.SignedObjectEnvelope, trustpolicy.AuditSegmentSealPayload, *trustpolicy.Digest, []byte, error) {
	if strings.TrimSpace(snapshot.segmentID) == "" {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, fmt.Errorf("current segment snapshot missing segment id")
	}
	segment, err := l.loadSegment(snapshot.segmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	rawBytes, err := l.rawSegmentFramedBytes(segment)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	sealenvelope, _, sealPayload, err := l.loadSealEnvelopeForSegmentLocked(snapshot.segmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	if err := requireSealEnvelopeMatchesSnapshot(sealenvelope, snapshot.sealDigest); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	previousDigest, err := l.previousSealDigestByIndexLocked(sealPayload.SealChainIndex - 1)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	return segment, sealenvelope, sealPayload, previousDigest, rawBytes, nil
}

func requireSealEnvelopeMatchesSnapshot(envelope trustpolicy.SignedObjectEnvelope, want trustpolicy.Digest) error {
	computedSealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return err
	}
	if mustDigestIdentity(computedSealDigest) != mustDigestIdentity(want) {
		return fmt.Errorf("current segment seal changed during verification input load")
	}
	return nil
}
