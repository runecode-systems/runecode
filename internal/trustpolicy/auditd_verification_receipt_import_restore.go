package trustpolicy

import "fmt"

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
