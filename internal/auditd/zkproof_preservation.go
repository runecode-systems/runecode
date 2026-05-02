package auditd

import (
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
	sidecars, err := l.snapshotProofSidecarDigestsLocked()
	if err != nil {
		return ProofBackfillEvidenceSnapshot{}, err
	}
	return ProofBackfillEvidenceSnapshot{SegmentIDs: segmentIDs, SegmentSealDigests: sidecars.segmentSeals, AuditReceiptDigests: sidecars.receipts, AuditVerificationReportDigests: sidecars.reports, ExternalAnchorEvidenceDigests: sidecars.externalEvidence, ExternalAnchorSidecarDigests: sidecars.externalSidecars, AuditProofBindingDigests: sidecars.proofBindings, ZKProofArtifactDigests: sidecars.proofArtifacts, ZKProofVerificationDigests: sidecars.proofVerifications}, nil
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
