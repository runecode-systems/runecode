package auditd

import (
	"path/filepath"
	"sort"
	"strings"
)

// ProofBackfillEvidenceSnapshot captures durable local evidence identities that
// proof backfill workflows can export later without ambient process state.
type ProofBackfillEvidenceSnapshot struct {
	SegmentIDs                           []string `json:"segment_ids"`
	SegmentSealDigests                   []string `json:"segment_seal_digests"`
	AuditReceiptDigests                  []string `json:"audit_receipt_digests"`
	AuditVerificationReportDigests       []string `json:"audit_verification_report_digests"`
	ProtocolBundleManifestHashes         []string `json:"protocol_bundle_manifest_hashes,omitempty"`
	RuntimeImageDescriptorDigests        []string `json:"runtime_image_descriptor_digests,omitempty"`
	AttestationEvidenceDigests           []string `json:"attestation_evidence_digests,omitempty"`
	AppliedHardeningPostureDigests       []string `json:"applied_hardening_posture_digests,omitempty"`
	SessionBindingDigests                []string `json:"session_binding_digests,omitempty"`
	ProjectSubstrateSnapshotDigests      []string `json:"project_substrate_snapshot_digests,omitempty"`
	AttestationVerificationRecordDigests []string `json:"attestation_verification_record_digests,omitempty"`
	VerifierRecordDigests                []string `json:"verifier_record_digests"`
	EventContractCatalogDigests          []string `json:"event_contract_catalog_digests"`
	SignerEvidenceDigests                []string `json:"signer_evidence_digests"`
	StoragePostureDigests                []string `json:"storage_posture_digests"`
	TypedRequestHashes                   []string `json:"typed_request_hashes,omitempty"`
	ActionRequestHashes                  []string `json:"action_request_hashes,omitempty"`
	PolicyDecisionHashes                 []string `json:"policy_decision_hashes,omitempty"`
	RequiredApprovalIDs                  []string `json:"required_approval_ids,omitempty"`
	ApprovalRequestHashes                []string `json:"approval_request_hashes,omitempty"`
	ApprovalDecisionHashes               []string `json:"approval_decision_hashes,omitempty"`
	ExternalAnchorEvidenceDigests        []string `json:"external_anchor_evidence_digests"`
	ExternalAnchorSidecarDigests         []string `json:"external_anchor_sidecar_digests"`
	AuditProofBindingDigests             []string `json:"audit_proof_binding_digests"`
	ZKProofArtifactDigests               []string `json:"zk_proof_artifact_digests"`
	ZKProofVerificationDigests           []string `json:"zk_proof_verification_digests"`
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
	bindingEvidence, err := l.snapshotAuditProofBindingEvidenceLocked(sidecars.proofBindings)
	if err != nil {
		return ProofBackfillEvidenceSnapshot{}, err
	}
	externalAnchorBindings, err := l.snapshotExternalAnchorIdentityBindingsLocked(sidecars.externalEvidence)
	if err != nil {
		return ProofBackfillEvidenceSnapshot{}, err
	}
	return ProofBackfillEvidenceSnapshot{SegmentIDs: segmentIDs, SegmentSealDigests: sidecars.segmentSeals, AuditReceiptDigests: sidecars.receipts, AuditVerificationReportDigests: sidecars.reports, ProtocolBundleManifestHashes: bindingEvidence.protocolBundleManifestHashes, RuntimeImageDescriptorDigests: bindingEvidence.runtimeImageDescriptorDigests, AttestationEvidenceDigests: bindingEvidence.attestationEvidenceDigests, AppliedHardeningPostureDigests: bindingEvidence.appliedHardeningPostureDigests, SessionBindingDigests: bindingEvidence.sessionBindingDigests, ProjectSubstrateSnapshotDigests: bindingEvidence.projectSubstrateSnapshotDigests, AttestationVerificationRecordDigests: bindingEvidence.attestationVerificationRecordDigests, VerifierRecordDigests: contractDigests.verifierRecords, EventContractCatalogDigests: contractDigests.eventCatalogs, SignerEvidenceDigests: contractDigests.signerEvidence, StoragePostureDigests: contractDigests.storagePosture, TypedRequestHashes: externalAnchorBindings.typedRequestHashes, ActionRequestHashes: externalAnchorBindings.actionRequestHashes, PolicyDecisionHashes: externalAnchorBindings.policyDecisionHashes, RequiredApprovalIDs: externalAnchorBindings.requiredApprovalIDs, ApprovalRequestHashes: externalAnchorBindings.approvalRequestHashes, ApprovalDecisionHashes: externalAnchorBindings.approvalDecisionHashes, ExternalAnchorEvidenceDigests: sidecars.externalEvidence, ExternalAnchorSidecarDigests: sidecars.externalSidecars, AuditProofBindingDigests: sidecars.proofBindings, ZKProofArtifactDigests: sidecars.proofArtifacts, ZKProofVerificationDigests: sidecars.proofVerifications}, nil
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

func (l *Ledger) snapshotAuditProofBindingEvidenceLocked(bindingDigests []string) (auditProofBindingEvidenceSnapshot, error) {
	snapshot := auditProofBindingEvidenceSnapshot{}
	for i := range bindingDigests {
		payload, found, err := l.loadAuditProofBindingByIdentityLocked(bindingDigests[i])
		if err != nil {
			return auditProofBindingEvidenceSnapshot{}, err
		}
		if !found {
			continue
		}
		snapshot.protocolBundleManifestHashes = appendIdentityUnique(snapshot.protocolBundleManifestHashes, mustDigestIdentityString(payload.ProtocolBundleManifest))
		snapshot.runtimeImageDescriptorDigests = appendIdentityUnique(snapshot.runtimeImageDescriptorDigests, payload.ProjectedPublicBindings.RuntimeImageDescriptorDigest)
		snapshot.attestationEvidenceDigests = appendIdentityUnique(snapshot.attestationEvidenceDigests, payload.ProjectedPublicBindings.AttestationEvidenceDigest)
		snapshot.appliedHardeningPostureDigests = appendIdentityUnique(snapshot.appliedHardeningPostureDigests, payload.ProjectedPublicBindings.AppliedHardeningPostureDigest)
		snapshot.sessionBindingDigests = appendIdentityUnique(snapshot.sessionBindingDigests, payload.ProjectedPublicBindings.SessionBindingDigest)
		snapshot.projectSubstrateSnapshotDigests = appendIdentityUnique(snapshot.projectSubstrateSnapshotDigests, payload.ProjectedPublicBindings.ProjectSubstrateSnapshotDigest)
		if payload.ProjectedPublicBindings.AttestationVerificationRecord != nil {
			snapshot.attestationVerificationRecordDigests = appendIdentityUnique(snapshot.attestationVerificationRecordDigests, mustDigestIdentityString(*payload.ProjectedPublicBindings.AttestationVerificationRecord))
		}
	}
	sort.Strings(snapshot.protocolBundleManifestHashes)
	sort.Strings(snapshot.runtimeImageDescriptorDigests)
	sort.Strings(snapshot.attestationEvidenceDigests)
	sort.Strings(snapshot.appliedHardeningPostureDigests)
	sort.Strings(snapshot.sessionBindingDigests)
	sort.Strings(snapshot.projectSubstrateSnapshotDigests)
	sort.Strings(snapshot.attestationVerificationRecordDigests)
	return snapshot, nil
}

func (l *Ledger) snapshotExternalAnchorIdentityBindingsLocked(evidenceDigests []string) (externalAnchorIdentityBindingSnapshot, error) {
	snapshot := externalAnchorIdentityBindingSnapshot{}
	for i := range evidenceDigests {
		rec, digest, ok, err := l.loadExternalAnchorEvidenceEntry(strings.TrimPrefix(evidenceDigests[i], "sha256:") + ".json")
		if err != nil {
			return externalAnchorIdentityBindingSnapshot{}, err
		}
		if !ok || digest == nil || mustDigestIdentity(*digest) != evidenceDigests[i] {
			continue
		}
		snapshot = appendExternalAnchorIdentityBindingSnapshot(snapshot, rec)
	}
	sortExternalAnchorIdentityBindingSnapshot(&snapshot)
	return snapshot, nil
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
