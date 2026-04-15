package trustpolicy

import (
	"fmt"
)

const (
	mvpAnchorKindLocalUserPresence      = "local_user_presence_signature"
	mvpAnchorWitnessKindLocalPresenceV0 = "local_user_presence_signature_v0"
)

func validateAuditReceiptPayloadPresence(receipt auditReceiptPayloadStrict) error {
	hasPayloadSchema := receipt.ReceiptPayloadSchema != ""
	hasPayload := len(receipt.ReceiptPayload) > 0
	if hasPayloadSchema != hasPayload {
		return fmt.Errorf("receipt payload schema and payload must be set together")
	}
	return nil
}

func validateImportRestoreReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != "runecode.protocol.audit.receipt.import_restore_provenance.v0" {
		return fmt.Errorf("%s receipts require import/restore provenance payload schema", receipt.AuditReceiptKind)
	}
	payload := importRestoreReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode import/restore payload: %w", err)
	}
	if err := validateImportRestoreReceiptPayloadHeader(receipt, payload); err != nil {
		return err
	}
	if err := validateImportRestoreReceiptSegments(payload.ImportedSegments); err != nil {
		return err
	}
	return validateImportRestoreManifestDigests(payload.SourceManifestDigests)
}

func validateImportRestoreReceiptPayloadHeader(receipt auditReceiptPayloadStrict, payload importRestoreReceiptPayload) error {
	if payload.ProvenanceAction != receipt.AuditReceiptKind {
		return fmt.Errorf("provenance_action=%q does not match audit_receipt_kind=%q", payload.ProvenanceAction, receipt.AuditReceiptKind)
	}
	if payload.SegmentFileHashScope != AuditSegmentFileHashScopeRawFramedV1 {
		return fmt.Errorf("unsupported segment_file_hash_scope %q", payload.SegmentFileHashScope)
	}
	if len(payload.ImportedSegments) == 0 {
		return fmt.Errorf("import/restore payload requires imported_segments")
	}
	if len(payload.SourceManifestDigests) == 0 {
		return fmt.Errorf("import/restore payload requires source_manifest_digests")
	}
	return nil
}

func validateImportRestoreReceiptSegments(segments []importRestoreSegmentLink) error {
	for index := range segments {
		if err := validateImportRestoreSegmentLink(segments[index], index); err != nil {
			return err
		}
	}
	return nil
}

func validateImportRestoreSegmentLink(segment importRestoreSegmentLink, index int) error {
	if _, err := segment.ImportedSegmentSealDigest.Identity(); err != nil {
		return fmt.Errorf("imported_segments[%d].imported_segment_seal_digest: %w", index, err)
	}
	if _, err := segment.ImportedSegmentRoot.Identity(); err != nil {
		return fmt.Errorf("imported_segments[%d].imported_segment_root: %w", index, err)
	}
	if _, err := segment.SourceSegmentFileHash.Identity(); err != nil {
		return fmt.Errorf("imported_segments[%d].source_segment_file_hash: %w", index, err)
	}
	if _, err := segment.LocalSegmentFileHash.Identity(); err != nil {
		return fmt.Errorf("imported_segments[%d].local_segment_file_hash: %w", index, err)
	}
	if !segment.ByteIdentityVerified {
		return fmt.Errorf("imported_segments[%d].byte_identity_verified must be true", index)
	}
	if mustDigestIdentity(segment.SourceSegmentFileHash) != mustDigestIdentity(segment.LocalSegmentFileHash) {
		return fmt.Errorf("imported_segments[%d] source/local file hashes must match", index)
	}
	return nil
}

func validateImportRestoreManifestDigests(digests []Digest) error {
	for index := range digests {
		if _, err := digests[index].Identity(); err != nil {
			return fmt.Errorf("source_manifest_digests[%d]: %w", index, err)
		}
	}
	return nil
}

func validateAnchorReceiptPayload(receipt auditReceiptPayloadStrict) error {
	payload, err := decodeAnchorReceiptPayload(receipt)
	if err != nil {
		return err
	}
	if err := validateAnchorPayloadCore(payload); err != nil {
		return err
	}
	if err := validateAnchorPayloadApprovalFields(payload); err != nil {
		return err
	}
	if err := validateAnchorPayloadWitness(payload); err != nil {
		return err
	}
	return validateAnchorApprovalLinkAndPosture(payload)
}

func decodeAnchorReceiptPayload(receipt auditReceiptPayloadStrict) (anchorReceiptPayload, error) {
	if receipt.ReceiptPayloadSchema != "runecode.protocol.audit.receipt.anchor.v0" {
		return anchorReceiptPayload{}, fmt.Errorf("anchor receipts require anchor payload schema")
	}
	payload := anchorReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return anchorReceiptPayload{}, fmt.Errorf("decode anchor payload: %w", err)
	}
	return payload, nil
}

func validateAnchorPayloadCore(payload anchorReceiptPayload) error {
	// Keep runtime verification fail-closed for MVP baseline kinds.
	// Schema remains additive so CHG-2026-025 can introduce external kinds without replacing the family.
	if payload.AnchorKind != mvpAnchorKindLocalUserPresence {
		return fmt.Errorf("unsupported anchor_kind %q", payload.AnchorKind)
	}
	if _, ok := allowedKeyProtectionPostures[payload.KeyProtectionPosture]; !ok {
		return fmt.Errorf("unsupported key_protection_posture %q", payload.KeyProtectionPosture)
	}
	if _, ok := allowedPresenceModes[payload.PresenceMode]; !ok {
		return fmt.Errorf("unsupported presence_mode %q", payload.PresenceMode)
	}
	if payload.PresenceMode == "none" {
		return fmt.Errorf("unsupported presence_mode %q", payload.PresenceMode)
	}
	return nil
}

func validateAnchorPayloadApprovalFields(payload anchorReceiptPayload) error {
	if err := validateAnchorPayloadApprovalAssurance(payload.ApprovalAssurance); err != nil {
		return err
	}
	if err := validateAnchorPayloadApprovalDecision(payload.ApprovalDecision); err != nil {
		return err
	}
	return validateAnchorPayloadApprovalCoupling(payload.ApprovalAssurance, payload.ApprovalDecision != nil)
}

func validateAnchorPayloadApprovalAssurance(assurance string) error {
	if assurance == "" {
		return nil
	}
	if _, ok := allowedAssuranceLevels[assurance]; !ok {
		return fmt.Errorf("unsupported approval_assurance_level %q", assurance)
	}
	return nil
}

func validateAnchorPayloadApprovalDecision(decision *Digest) error {
	if decision == nil {
		return nil
	}
	if _, err := decision.Identity(); err != nil {
		return fmt.Errorf("approval_decision_digest: %w", err)
	}
	return nil
}

func validateAnchorPayloadApprovalCoupling(assurance string, hasDecision bool) error {
	if assurance == "none" && hasDecision {
		return fmt.Errorf("approval_assurance_level=none cannot declare approval_decision_digest")
	}
	if assurance == "hardware_backed" && !hasDecision {
		return fmt.Errorf("approval_decision_digest is required when approval_assurance_level=hardware_backed")
	}
	if hasDecision && assurance == "" {
		return fmt.Errorf("approval_assurance_level is required when approval_decision_digest is present")
	}
	return nil
}

func validateAnchorPayloadWitness(payload anchorReceiptPayload) error {
	if payload.AnchorWitness.WitnessKind != mvpAnchorWitnessKindLocalPresenceV0 {
		return fmt.Errorf("unsupported anchor_witness.witness_kind %q", payload.AnchorWitness.WitnessKind)
	}
	if _, err := payload.AnchorWitness.WitnessDigest.Identity(); err != nil {
		return fmt.Errorf("anchor_witness.witness_digest: %w", err)
	}
	return nil
}

func validateAnchorApprovalLinkAndPosture(payload anchorReceiptPayload) error {
	if payload.ApprovalAssurance != "hardware_backed" {
		return nil
	}
	if payload.PresenceMode != "hardware_touch" {
		return fmt.Errorf("approval_assurance_level=hardware_backed requires presence_mode=hardware_touch")
	}
	if payload.KeyProtectionPosture != "hardware_backed" {
		return fmt.Errorf("approval_assurance_level=hardware_backed requires key_protection_posture=hardware_backed")
	}
	return nil
}

func validateReceiptSignerContract(receipt auditReceiptPayloadStrict, verifier VerifierRecord) error {
	if receipt.AuditReceiptKind != "anchor" {
		return nil
	}
	if verifier.LogicalPurpose != "audit_anchor" {
		return fmt.Errorf("anchor receipts require verifier logical_purpose audit_anchor")
	}
	if verifier.LogicalScope != "node" && verifier.LogicalScope != "deployment" {
		return fmt.Errorf("anchor receipts require verifier logical_scope node or deployment")
	}
	payload := anchorReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode anchor payload for signer posture check: %w", err)
	}
	if payload.KeyProtectionPosture != verifier.KeyProtectionPosture {
		return fmt.Errorf("anchor key_protection_posture %q does not match verifier key_protection_posture %q", payload.KeyProtectionPosture, verifier.KeyProtectionPosture)
	}
	if payload.PresenceMode != verifier.PresenceMode {
		return fmt.Errorf("anchor presence_mode %q does not match verifier presence_mode %q", payload.PresenceMode, verifier.PresenceMode)
	}
	return nil
}

func verifyImportRestoreConsistency(receipt auditReceiptPayloadStrict, sealDigest Digest, sealPayload AuditSegmentSealPayload) error {
	payload := importRestoreReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode import/restore payload: %w", err)
	}
	sealDigestID := mustDigestIdentity(sealDigest)
	if mustDigestIdentity(receipt.SubjectDigest) != sealDigestID {
		return fmt.Errorf("receipt subject_digest does not target verified segment seal digest")
	}
	matchIndex := -1
	for index := range payload.ImportedSegments {
		segment := payload.ImportedSegments[index]
		if mustDigestIdentity(segment.ImportedSegmentSealDigest) != sealDigestID {
			continue
		}
		if matchIndex >= 0 {
			return fmt.Errorf("multiple imported_segments entries match verified segment seal digest")
		}
		matchIndex = index
	}
	if matchIndex < 0 {
		return fmt.Errorf("no imported_segments entry matches verified segment seal digest")
	}
	match := payload.ImportedSegments[matchIndex]
	if mustDigestIdentity(match.ImportedSegmentRoot) != mustDigestIdentity(sealPayload.MerkleRoot) {
		return fmt.Errorf("imported_segments[%d] imported_segment_root does not match segment seal root", matchIndex)
	}
	if mustDigestIdentity(match.LocalSegmentFileHash) != mustDigestIdentity(sealPayload.SegmentFileHash) {
		return fmt.Errorf("imported_segments[%d] local_segment_file_hash does not match segment seal file hash", matchIndex)
	}
	return nil
}
