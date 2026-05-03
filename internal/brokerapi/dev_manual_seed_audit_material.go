//go:build runecode_devseed

package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func newDevManualAuditMaterial(profile string) (devManualAuditMaterial, devManualVerificationMaterial) {
	signerMaterial := devManualVerificationSignerMaterial()
	material := devManualAuditMaterial{
		profile:        profile,
		verifier:       devManualVerifierRecord(signerMaterial),
		signerEvidence: devManualSignerEvidenceRefs(signerMaterial),
	}
	return material, signerMaterial
}

func attachDevManualEventMaterial(material *devManualAuditMaterial, signerMaterial devManualVerificationMaterial) error {
	eventPayload := devManualAuditEventPayload(signerMaterial, material.profile)
	eventPayloadHash := sha256.Sum256(mustDevManualCanonicalJSON(eventPayload))
	eventEnvelope, err := devManualSignedEnvelope(
		trustpolicy.AuditEventSchemaID,
		trustpolicy.AuditEventSchemaVersion,
		devManualAuditEvent(eventPayload, eventPayloadHash, material.profile),
		signerMaterial,
	)
	if err != nil {
		return err
	}
	canonicalEnvelope := mustDevManualCanonicalJSON(eventEnvelope)
	recordSum := sha256.Sum256(canonicalEnvelope)
	material.canonicalEnvelope = canonicalEnvelope
	material.recordDigest = trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(recordSum[:])}
	return nil
}

func attachDevManualSegmentMaterial(material *devManualAuditMaterial, signerMaterial devManualVerificationMaterial) error {
	segmentRawBytes, segmentFileHash, segmentMerkleRoot, err := devManualSegmentEvidence(material.recordDigest, material.canonicalEnvelope)
	if err != nil {
		return err
	}
	material.segmentRawBytes = segmentRawBytes
	material.segmentFileHash = segmentFileHash
	material.segmentMerkleRoot = segmentMerkleRoot
	material.segmentSealEnvelope, err = devManualSignedEnvelope(
		trustpolicy.AuditSegmentSealSchemaID,
		trustpolicy.AuditSegmentSealSchemaVersion,
		devManualSealPayload(material.recordDigest, material.segmentFileHash, material.segmentMerkleRoot),
		signerMaterial,
	)
	return err
}

func devManualSegmentEvidence(recordDigest trustpolicy.Digest, canonicalEnvelope []byte) ([]byte, trustpolicy.Digest, trustpolicy.Digest, error) {
	segmentRawBytes := append(append([]byte{}, canonicalEnvelope...), '\n')
	segmentFileHash, err := trustpolicy.ComputeSegmentFileHash(segmentRawBytes)
	if err != nil {
		return nil, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	segmentMerkleRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot([]trustpolicy.Digest{recordDigest})
	if err != nil {
		return nil, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return segmentRawBytes, segmentFileHash, segmentMerkleRoot, nil
}

func attachDevManualSeedReport(material *devManualAuditMaterial) error {
	report, err := trustpolicy.VerifyAuditEvidence(trustpolicy.AuditVerificationInput{
		Scope:                 trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: "segment-000001"},
		Segment:               sealedSegmentPayload(*material),
		RawFramedSegmentBytes: material.segmentRawBytes,
		SegmentSealEnvelope:   material.segmentSealEnvelope,
		VerifierRecords:       []trustpolicy.VerifierRecord{material.verifier},
		EventContractCatalog:  devManualEventContractCatalog(),
		SignerEvidence:        material.signerEvidence,
	})
	if err != nil {
		return err
	}
	if material.profile == devManualSeedDegradedProfile {
		report = degradeDevManualVerificationReport(report)
	}
	material.seedReport = report
	return nil
}
