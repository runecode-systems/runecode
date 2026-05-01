package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) VerifyCurrentSegmentAndPersist() (VerificationResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	segment, input, err := l.currentVerificationContextLocked()
	if err != nil {
		return VerificationResult{}, err
	}
	report, err := trustpolicy.VerifyAuditEvidence(input)
	if err != nil {
		return VerificationResult{}, err
	}
	reportDigest, err := l.persistVerificationReportLocked(report)
	if err != nil {
		return VerificationResult{}, err
	}
	return VerificationResult{SegmentID: segment.Header.SegmentID, ReportDigest: reportDigest, Report: report}, nil
}

func (l *Ledger) VerifyCurrentSegmentIncrementalWithPreverifiedSeal(preverifiedSealDigest trustpolicy.Digest, verifier trustpolicy.VerifierRecord) (trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := preverifiedSealDigest.Identity(); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("preverified_seal_digest: %w", err)
	}
	if err := l.ensureVerifierRecordDurableLocked(verifier); err != nil {
		return trustpolicy.Digest{}, err
	}

	segment, input, err := l.currentVerificationContextLocked()
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	computedSealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(input.SegmentSealEnvelope)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	if mustDigestIdentity(computedSealDigest) != mustDigestIdentity(preverifiedSealDigest) {
		return trustpolicy.Digest{}, fmt.Errorf("preverified seal digest does not match current segment seal")
	}
	verifiers, _, err := addVerifierRecordIfMissing(input.VerifierRecords, verifier)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	input.VerifierRecords = verifiers
	input.Scope = trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: segment.Header.SegmentID}
	input.PreverifiedSealDigest = &preverifiedSealDigest
	input.SkipFrameAndSealReplay = true

	report, err := trustpolicy.VerifyAuditEvidence(input)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	return l.persistVerificationReportLocked(report)
}

func (l *Ledger) currentVerificationContextLocked() (trustpolicy.AuditSegmentFilePayload, trustpolicy.AuditVerificationInput, error) {
	segment, sealEnvelope, sealPayload, previousDigest, rawBytes, err := l.currentSegmentEvidenceLocked()
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditVerificationInput{}, err
	}
	runtimeInputs, err := l.loadVerificationInputsLocked()
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

func (l *Ledger) currentSegmentEvidenceLocked() (trustpolicy.AuditSegmentFilePayload, trustpolicy.SignedObjectEnvelope, trustpolicy.AuditSegmentSealPayload, *trustpolicy.Digest, []byte, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	if state.LastSealedSegmentID == "" {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, fmt.Errorf("no sealed segment available for verification")
	}
	segment, err := l.loadSegment(state.LastSealedSegmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	rawBytes, err := l.rawSegmentFramedBytes(segment)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	sealEnvelope, _, sealPayload, err := l.loadSealEnvelopeForSegmentLocked(state.LastSealedSegmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	previousDigest, err := l.previousSealDigestByIndexLocked(sealPayload.SealChainIndex - 1)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	return segment, sealEnvelope, sealPayload, previousDigest, rawBytes, nil
}

func (l *Ledger) loadSealEnvelopeForSegmentLocked(segmentID string) (trustpolicy.SignedObjectEnvelope, trustpolicy.Digest, trustpolicy.AuditSegmentSealPayload, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, sealsDirName))
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	bestIndex := int64(-1)
	bestEnvelope := trustpolicy.SignedObjectEnvelope{}
	bestDigest := trustpolicy.Digest{}
	bestPayload := trustpolicy.AuditSegmentSealPayload{}
	for _, entry := range entries {
		envelope, payload, digest, ok, err := l.sealEntryForSegment(entry.Name(), segmentID)
		if err != nil {
			return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
		}
		if !ok || payload.SealChainIndex < bestIndex {
			continue
		}
		bestIndex = payload.SealChainIndex
		bestEnvelope = envelope
		bestPayload = payload
		bestDigest = digest
	}
	if bestIndex < 0 {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("no segment seal found for %s", segmentID)
	}
	return bestEnvelope, bestDigest, bestPayload, nil
}

func (l *Ledger) sealEntryForSegment(name string, segmentID string) (trustpolicy.SignedObjectEnvelope, trustpolicy.AuditSegmentSealPayload, trustpolicy.Digest, bool, error) {
	if strings.HasSuffix(name, ".json") == false {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, trustpolicy.Digest{}, false, nil
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(filepath.Join(l.rootDir, sidecarDirName, sealsDirName, name), &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, trustpolicy.Digest{}, false, err
	}
	payload := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, trustpolicy.Digest{}, false, err
	}
	if payload.SegmentID != segmentID {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, trustpolicy.Digest{}, false, nil
	}
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(name, ".json")}
	return envelope, payload, digest, true, nil
}

func (l *Ledger) previousSealDigestByIndexLocked(index int64) (*trustpolicy.Digest, error) {
	if index < 0 {
		return nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, sealsDirName))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		digest, ok, err := l.sealDigestByIndexEntry(entry.Name(), index)
		if err != nil {
			return nil, err
		}
		if ok {
			return &digest, nil
		}
	}
	return nil, fmt.Errorf("missing previous seal digest at chain index %d", index)
}

func (l *Ledger) sealDigestByIndexEntry(name string, index int64) (trustpolicy.Digest, bool, error) {
	if !strings.HasSuffix(name, ".json") {
		return trustpolicy.Digest{}, false, nil
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(filepath.Join(l.rootDir, sidecarDirName, sealsDirName, name), &envelope); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	payload := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if payload.SealChainIndex != index {
		return trustpolicy.Digest{}, false, nil
	}
	return trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(name, ".json")}, true, nil
}
