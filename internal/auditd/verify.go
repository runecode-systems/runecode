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
		ReceiptEnvelopes:         runtimeInputs.receipts,
		VerifierRecords:          runtimeInputs.verifierRecords,
		EventContractCatalog:     runtimeInputs.catalog,
		SignerEvidence:           runtimeInputs.signerEvidence,
		StoragePostureEvidence:   runtimeInputs.storagePosture,
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

type verificationInputs struct {
	verifierRecords []trustpolicy.VerifierRecord
	catalog         trustpolicy.AuditEventContractCatalog
	signerEvidence  []trustpolicy.AuditSignerEvidenceReference
	storagePosture  *trustpolicy.AuditStoragePostureEvidence
	receipts        []trustpolicy.SignedObjectEnvelope
}

func (l *Ledger) loadVerificationInputsLocked() (verificationInputs, error) {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	if err := requireVerificationContractFiles(contractsDir); err != nil {
		return verificationInputs{}, err
	}

	inputs := verificationInputs{}
	if err := readJSONFile(filepath.Join(contractsDir, "event-contract-catalog.json"), &inputs.catalog); err != nil {
		return verificationInputs{}, err
	}
	if err := readJSONFile(filepath.Join(contractsDir, "verifier-records.json"), &inputs.verifierRecords); err != nil {
		return verificationInputs{}, err
	}
	if err := loadOptionalContractFiles(contractsDir, &inputs); err != nil {
		return verificationInputs{}, err
	}
	receipts, err := l.loadAllReceiptsLocked()
	if err != nil {
		return verificationInputs{}, err
	}
	inputs.receipts = receipts
	return inputs, nil
}

func requireVerificationContractFiles(contractsDir string) error {
	if !fileExists(filepath.Join(contractsDir, "event-contract-catalog.json")) {
		return fmt.Errorf("missing event contract catalog")
	}
	if !fileExists(filepath.Join(contractsDir, "verifier-records.json")) {
		return fmt.Errorf("missing verifier records")
	}
	return nil
}

func loadOptionalContractFiles(contractsDir string, inputs *verificationInputs) error {
	if fileExists(filepath.Join(contractsDir, "signer-evidence.json")) {
		if err := readJSONFile(filepath.Join(contractsDir, "signer-evidence.json"), &inputs.signerEvidence); err != nil {
			return err
		}
	}
	if !fileExists(filepath.Join(contractsDir, "storage-posture.json")) {
		return nil
	}
	var posture trustpolicy.AuditStoragePostureEvidence
	if err := readJSONFile(filepath.Join(contractsDir, "storage-posture.json"), &posture); err != nil {
		return err
	}
	inputs.storagePosture = &posture
	return nil
}

func (l *Ledger) loadAllReceiptsLocked() ([]trustpolicy.SignedObjectEnvelope, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, receiptsDirName))
	if err != nil {
		return nil, err
	}
	receipts := make([]trustpolicy.SignedObjectEnvelope, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		envelope := trustpolicy.SignedObjectEnvelope{}
		if err := readJSONFile(filepath.Join(l.rootDir, sidecarDirName, receiptsDirName, entry.Name()), &envelope); err != nil {
			return nil, err
		}
		receipts = append(receipts, envelope)
	}
	return receipts, nil
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
