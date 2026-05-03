//go:build runecode_devseed

package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type devManualAuditMaterial struct {
	profile             string
	verifier            trustpolicy.VerifierRecord
	signerEvidence      []trustpolicy.AuditSignerEvidenceReference
	recordDigest        trustpolicy.Digest
	canonicalEnvelope   []byte
	segmentRawBytes     []byte
	segmentFileHash     trustpolicy.Digest
	segmentMerkleRoot   trustpolicy.Digest
	segmentSealEnvelope trustpolicy.SignedObjectEnvelope
}

func seedDevManualAuditLedger(root string, profile string) (string, error) {
	material, err := buildDevManualAuditMaterial(profile)
	if err != nil {
		return "", err
	}
	validatedRoot, err := ensureDevManualLedgerDirs(root, profile)
	if err != nil {
		return "", err
	}
	if err := writeDevManualSegments(validatedRoot, material); err != nil {
		return "", err
	}
	if err := writeDevManualSeal(validatedRoot, material); err != nil {
		return "", err
	}
	ledger, err := auditd.Open(validatedRoot)
	if err != nil {
		return "", err
	}
	if err := ledger.ConfigureVerificationInputs(auditd.VerificationConfiguration{
		VerifierRecords:      []trustpolicy.VerifierRecord{material.verifier},
		EventContractCatalog: devManualEventContractCatalog(),
		SignerEvidence:       material.signerEvidence,
	}); err != nil {
		return "", err
	}
	if _, err := ledger.BuildIndex(); err != nil {
		return "", err
	}
	result, err := ledger.VerifyCurrentSegmentAndPersist()
	if err != nil {
		return "", err
	}
	if profile == devManualSeedDegradedProfile {
		degraded := degradeDevManualVerificationReport(result.Report)
		if _, err := ledger.PersistVerificationReport(degraded); err != nil {
			return "", err
		}
	}
	if err := writeDevManualSeedMarker(validatedRoot, profile); err != nil {
		return "", err
	}
	return recordDigestIdentity(material.recordDigest)
}

func buildDevManualAuditMaterial(profile string) (devManualAuditMaterial, error) {
	material, signerMaterial := newDevManualAuditMaterial(profile)
	if err := attachDevManualEventMaterial(&material, signerMaterial); err != nil {
		return devManualAuditMaterial{}, err
	}
	if err := attachDevManualSegmentMaterial(&material, signerMaterial); err != nil {
		return devManualAuditMaterial{}, err
	}
	return material, nil
}

func ensureDevManualLedgerDirs(root string, profile string) (string, error) {
	validatedRoot, err := ensureDevManualSeedLedgerAllowed(root, profile)
	if err != nil {
		return "", err
	}
	paths := []string{
		filepath.Join(validatedRoot, "segments"),
		filepath.Join(validatedRoot, "sidecar", "segment-seals"),
		filepath.Join(validatedRoot, "sidecar", "verification-reports"),
		filepath.Join(validatedRoot, "contracts"),
	}
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", err
		}
	}
	return validatedRoot, nil
}

func writeDevManualSeedMarker(root string, profile string) error {
	return os.WriteFile(devManualLedgerSeedMarkerPath(root), []byte(profile+"\n"), 0o600)
}

func writeDevManualSegments(root string, material devManualAuditMaterial) error {
	if err := writeDevManualCanonicalJSON(filepath.Join(root, "segments", "segment-000001.json"), sealedSegmentPayload(material)); err != nil {
		return err
	}
	open := trustpolicy.AuditSegmentFilePayload{
		SchemaID:      "runecode.protocol.v0.AuditSegmentFile",
		SchemaVersion: "0.1.0",
		Header: trustpolicy.AuditSegmentHeader{
			Format:       "audit_segment_framed_v1",
			SegmentID:    "segment-000002",
			SegmentState: trustpolicy.AuditSegmentStateOpen,
			CreatedAt:    "2026-03-13T12:21:00Z",
			Writer:       "auditd",
		},
		Frames:          []trustpolicy.AuditSegmentRecordFrame{},
		LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateOpen, MarkedAt: "2026-03-13T12:21:00Z"},
	}
	return writeDevManualCanonicalJSON(filepath.Join(root, "segments", "segment-000002.json"), open)
}

func sealedSegmentPayload(material devManualAuditMaterial) trustpolicy.AuditSegmentFilePayload {
	return trustpolicy.AuditSegmentFilePayload{
		SchemaID:      "runecode.protocol.v0.AuditSegmentFile",
		SchemaVersion: "0.1.0",
		Header: trustpolicy.AuditSegmentHeader{
			Format:       "audit_segment_framed_v1",
			SegmentID:    "segment-000001",
			SegmentState: trustpolicy.AuditSegmentStateSealed,
			CreatedAt:    "2026-03-13T12:00:00Z",
			Writer:       "auditd",
		},
		Frames: []trustpolicy.AuditSegmentRecordFrame{{
			RecordDigest:                 material.recordDigest,
			ByteLength:                   int64(len(material.canonicalEnvelope)),
			CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(material.canonicalEnvelope),
		}},
		LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateSealed, MarkedAt: "2026-03-13T12:20:00Z"},
	}
}

func writeDevManualSeal(root string, material devManualAuditMaterial) error {
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(material.segmentSealEnvelope)
	if err != nil {
		return err
	}
	sealID, err := sealDigest.Identity()
	if err != nil {
		return err
	}
	return writeDevManualCanonicalJSON(filepath.Join(root, "sidecar", "segment-seals", trimSHA256Prefix(sealID)+".json"), material.segmentSealEnvelope)
}

func devManualSealPayload(recordDigest trustpolicy.Digest, segmentFileHash trustpolicy.Digest, merkleRoot trustpolicy.Digest) trustpolicy.AuditSegmentSealPayload {
	return trustpolicy.AuditSegmentSealPayload{
		SchemaID:                   trustpolicy.AuditSegmentSealSchemaID,
		SchemaVersion:              trustpolicy.AuditSegmentSealSchemaVersion,
		SegmentID:                  "segment-000001",
		SealedAfterState:           trustpolicy.AuditSegmentStateOpen,
		SegmentState:               trustpolicy.AuditSegmentStateSealed,
		SegmentCut:                 trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 2048, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow},
		EventCount:                 1,
		FirstRecordDigest:          recordDigest,
		LastRecordDigest:           recordDigest,
		MerkleProfile:              trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
		MerkleRoot:                 merkleRoot,
		SegmentFileHashScope:       trustpolicy.AuditSegmentFileHashScopeRawFramedV1,
		SegmentFileHash:            segmentFileHash,
		SealChainIndex:             0,
		AnchoringSubject:           trustpolicy.AuditSegmentAnchoringSubjectSeal,
		SealedAt:                   "2026-03-13T12:20:00Z",
		ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: stringsRepeat("b")},
		SealReason:                 "size_threshold",
	}
}

func degradeDevManualVerificationReport(report trustpolicy.AuditVerificationReportPayload) trustpolicy.AuditVerificationReportPayload {
	report.CurrentlyDegraded = true
	report.AnchoringStatus = trustpolicy.AuditVerificationStatusDegraded
	report.AnchoringPosture = trustpolicy.AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound
	report.DegradedReasons = uniqueSortedStrings(append(report.DegradedReasons, trustpolicy.AuditVerificationReasonAnchorReceiptMissing))
	report.Findings = append(report.Findings, trustpolicy.AuditVerificationFinding{
		Code:      trustpolicy.AuditVerificationReasonAnchorReceiptMissing,
		Dimension: trustpolicy.AuditVerificationDimensionAnchoring,
		Severity:  trustpolicy.AuditVerificationSeverityWarning,
		Message:   "dev manual degraded profile omits anchor receipts",
		SegmentID: "segment-000001",
	})
	report.Summary = "deterministic dev seed with intentionally degraded anchoring posture"
	return report
}

func trimSHA256Prefix(identity string) string {
	if len(identity) > len("sha256:") && identity[:len("sha256:")] == "sha256:" {
		return identity[len("sha256:"):]
	}
	return identity
}

func writeDevManualCanonicalJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, canonical, 0o600)
}

func mustDevManualCanonicalJSON(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		panic(err)
	}
	return canonical
}

func recordDigestIdentity(digest trustpolicy.Digest) (string, error) {
	identity, err := digest.Identity()
	if err != nil {
		return "", err
	}
	return identity, nil
}
