package auditd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProofBackfillEvidenceSnapshot captures durable local evidence identities that
// proof backfill workflows can export later without ambient process state.
type ProofBackfillEvidenceSnapshot struct {
	SegmentIDs                     []string `json:"segment_ids"`
	SegmentSealDigests             []string `json:"segment_seal_digests"`
	AuditReceiptDigests            []string `json:"audit_receipt_digests"`
	AuditVerificationReportDigests []string `json:"audit_verification_report_digests"`
	VerifierRecordDigests          []string `json:"verifier_record_digests"`
	EventContractCatalogDigests    []string `json:"event_contract_catalog_digests"`
	SignerEvidenceDigests          []string `json:"signer_evidence_digests"`
	StoragePostureDigests          []string `json:"storage_posture_digests"`
	ExternalAnchorEvidenceDigests  []string `json:"external_anchor_evidence_digests"`
	ExternalAnchorSidecarDigests   []string `json:"external_anchor_sidecar_digests"`
	AuditProofBindingDigests       []string `json:"audit_proof_binding_digests"`
	ZKProofArtifactDigests         []string `json:"zk_proof_artifact_digests"`
	ZKProofVerificationDigests     []string `json:"zk_proof_verification_digests"`
}

func (l *Ledger) ProofBackfillEvidenceSnapshot() (ProofBackfillEvidenceSnapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	segmentIDs, err := l.snapshotSegmentIDsLocked()
	if err != nil {
		return ProofBackfillEvidenceSnapshot{}, err
	}
	contractDigests, err := l.snapshotVerificationContractDigestsLocked()
	if err != nil {
		return ProofBackfillEvidenceSnapshot{}, err
	}
	sidecars, err := l.snapshotProofSidecarDigestsLocked()
	if err != nil {
		return ProofBackfillEvidenceSnapshot{}, err
	}
	return ProofBackfillEvidenceSnapshot{SegmentIDs: segmentIDs, SegmentSealDigests: sidecars.segmentSeals, AuditReceiptDigests: sidecars.receipts, AuditVerificationReportDigests: sidecars.reports, VerifierRecordDigests: contractDigests.verifierRecords, EventContractCatalogDigests: contractDigests.eventCatalogs, SignerEvidenceDigests: contractDigests.signerEvidence, StoragePostureDigests: contractDigests.storagePosture, ExternalAnchorEvidenceDigests: sidecars.externalEvidence, ExternalAnchorSidecarDigests: sidecars.externalSidecars, AuditProofBindingDigests: sidecars.proofBindings, ZKProofArtifactDigests: sidecars.proofArtifacts, ZKProofVerificationDigests: sidecars.proofVerifications}, nil
}

type proofSidecarDigestSnapshot struct {
	segmentSeals       []string
	receipts           []string
	reports            []string
	externalEvidence   []string
	externalSidecars   []string
	proofBindings      []string
	proofArtifacts     []string
	proofVerifications []string
}

type proofVerificationContractSnapshot struct {
	verifierRecords []string
	eventCatalogs   []string
	signerEvidence  []string
	storagePosture  []string
}

func (l *Ledger) snapshotSegmentIDsLocked() ([]string, error) {
	segments, err := l.listSegments()
	if err != nil {
		return nil, err
	}
	segmentIDs := make([]string, 0, len(segments))
	for i := range segments {
		segmentIDs = append(segmentIDs, strings.TrimSpace(segments[i].Header.SegmentID))
	}
	sort.Strings(segmentIDs)
	return segmentIDs, nil
}

func (l *Ledger) snapshotProofSidecarDigestsLocked() (proofSidecarDigestSnapshot, error) {
	segmentSeals, err := l.sidecarDigestIdentitiesLocked(sealsDirName)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	receipts, err := l.sidecarDigestIdentitiesLocked(receiptsDirName)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	reports, err := l.sidecarDigestIdentitiesLocked(verificationReportsDirName)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	externalEvidence, err := l.sidecarDigestIdentitiesLocked(externalAnchorEvidenceDir)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	externalSidecars, err := l.sidecarDigestIdentitiesLocked(externalAnchorSidecarsDir)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	proofBindings, err := l.sidecarDigestIdentitiesLocked(proofBindingsDirName)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	proofArtifacts, err := l.sidecarDigestIdentitiesLocked(proofArtifactsDirName)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	proofVerifications, err := l.sidecarDigestIdentitiesLocked(proofVerificationsDirName)
	if err != nil {
		return proofSidecarDigestSnapshot{}, err
	}
	return proofSidecarDigestSnapshot{segmentSeals: segmentSeals, receipts: receipts, reports: reports, externalEvidence: externalEvidence, externalSidecars: externalSidecars, proofBindings: proofBindings, proofArtifacts: proofArtifacts, proofVerifications: proofVerifications}, nil
}

func (l *Ledger) snapshotVerificationContractDigestsLocked() (proofVerificationContractSnapshot, error) {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	verifierRecords, err := canonicalDigestIdentityForJSONPath(filepath.Join(contractsDir, "verifier-records.json"))
	if err != nil {
		return proofVerificationContractSnapshot{}, err
	}
	eventCatalog, err := canonicalDigestIdentityForJSONPath(filepath.Join(contractsDir, "event-contract-catalog.json"))
	if err != nil {
		return proofVerificationContractSnapshot{}, err
	}
	signerEvidence, err := canonicalDigestIdentityForOptionalJSONPath(filepath.Join(contractsDir, "signer-evidence.json"))
	if err != nil {
		return proofVerificationContractSnapshot{}, err
	}
	storagePosture, err := canonicalDigestIdentityForOptionalJSONPath(filepath.Join(contractsDir, "storage-posture.json"))
	if err != nil {
		return proofVerificationContractSnapshot{}, err
	}
	return proofVerificationContractSnapshot{verifierRecords: wrapIdentityList(verifierRecords), eventCatalogs: wrapIdentityList(eventCatalog), signerEvidence: wrapIdentityList(signerEvidence), storagePosture: wrapIdentityList(storagePosture)}, nil
}

func (l *Ledger) sidecarDigestIdentitiesLocked(dirName string) ([]string, error) {
	entries, err := l.readOptionalSidecarDirEntries(dirName)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for i := range entries {
		if entries[i].IsDir() {
			continue
		}
		id, ok, err := digestIdentityFromSidecarName(entries[i].Name())
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func sidecarPath(root, dirName, identity string) string {
	return filepath.Join(root, sidecarDirName, dirName, strings.TrimPrefix(identity, "sha256:")+".json")
}

func canonicalDigestIdentityForJSONPath(path string) (string, error) {
	raw := json.RawMessage{}
	if err := readJSONFile(path, &raw); err != nil {
		return "", err
	}
	digest, err := canonicalDigest(raw)
	if err != nil {
		return "", err
	}
	return digest.Identity()
}

func canonicalDigestIdentityForOptionalJSONPath(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return canonicalDigestIdentityForJSONPath(path)
}

func wrapIdentityList(identity string) []string {
	if strings.TrimSpace(identity) == "" {
		return nil
	}
	return []string{identity}
}
