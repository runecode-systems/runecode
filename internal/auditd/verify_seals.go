package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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
		if err := validateSealEnvelopeCandidate(envelope); err != nil {
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
	if err := validateSealFilenameDigest(envelope, digest); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, trustpolicy.Digest{}, false, err
	}
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
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(name, ".json")}
	if err := validateSealFilenameDigest(envelope, digest); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	return digest, true, nil
}

func validateSealFilenameDigest(envelope trustpolicy.SignedObjectEnvelope, want trustpolicy.Digest) error {
	computed, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return err
	}
	if mustDigestIdentity(computed) != mustDigestIdentity(want) {
		return fmt.Errorf("seal filename digest does not match envelope digest")
	}
	return nil
}

func validateSealEnvelopeCandidate(envelope trustpolicy.SignedObjectEnvelope) error {
	if envelope.PayloadSchemaID != trustpolicy.AuditSegmentSealSchemaID {
		return fmt.Errorf("candidate seal envelope uses unexpected payload schema")
	}
	return nil
}
