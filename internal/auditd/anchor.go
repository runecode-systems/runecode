package auditd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (l *Ledger) AnchorCurrentSegment(req AnchorSegmentRequest) (AnchorSegmentResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	segment, sealEnvelope, _, _, rawBytes, err := l.currentSegmentEvidenceLocked()
	if err != nil {
		return AnchorSegmentResult{}, err
	}
	if err := l.requireMatchingCurrentSealDigestLocked(segment.Header.SegmentID, req.SealDigest); err != nil {
		return AnchorSegmentResult{}, err
	}
	receiptEnvelope, err := buildAnchorReceiptEnvelopeForSegment(req)
	if err != nil {
		return AnchorSegmentResult{}, fmt.Errorf("%w: %v", ErrAnchorReceiptInvalid, err)
	}

	verificationInput, err := l.verificationInputWithExtraReceipt(segment, sealEnvelope, rawBytes, receiptEnvelope, req)
	if err != nil {
		return AnchorSegmentResult{}, err
	}
	report, err := trustpolicy.VerifyAuditEvidence(verificationInput)
	if err != nil {
		return AnchorSegmentResult{}, err
	}
	receiptDigest, err := l.persistEnvelopeSidecar(receiptsDirName, receiptEnvelope)
	if err != nil {
		return AnchorSegmentResult{}, err
	}
	verificationDigest, err := l.persistVerificationReportLocked(report)
	if err != nil {
		return AnchorSegmentResult{}, err
	}

	return AnchorSegmentResult{
		SealDigest:         req.SealDigest,
		ReceiptDigest:      receiptDigest,
		VerificationDigest: verificationDigest,
		AnchorStatus:       strings.TrimSpace(report.AnchoringStatus),
	}, nil
}

func (l *Ledger) requireMatchingCurrentSealDigestLocked(segmentID string, requested trustpolicy.Digest) error {
	if mustDigestIdentity(requested) == "" {
		return fmt.Errorf("%w: seal_digest is required", ErrAnchorReceiptInvalid)
	}
	_, currentSealDigest, _, err := l.loadSealEnvelopeForSegmentLocked(segmentID)
	if err != nil {
		return err
	}
	if mustDigestIdentity(currentSealDigest) != mustDigestIdentity(requested) {
		return fmt.Errorf("%w: seal_digest does not match current segment seal", ErrAnchorReceiptInvalid)
	}
	return nil
}

func (l *Ledger) verificationInputWithExtraReceipt(segment trustpolicy.AuditSegmentFilePayload, sealEnvelope trustpolicy.SignedObjectEnvelope, rawBytes []byte, receiptEnvelope trustpolicy.SignedObjectEnvelope, req AnchorSegmentRequest) (trustpolicy.AuditVerificationInput, error) {
	_, _, sealPayload, err := l.loadSealEnvelopeForSegmentLocked(segment.Header.SegmentID)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	previousDigest, err := l.previousSealDigestByIndexLocked(sealPayload.SealChainIndex - 1)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	runtimeInputs, err := l.loadVerificationInputsLocked()
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, err
	}
	anchorVerifier, err := anchorVerifierRecordFromRequest(req)
	if err != nil {
		return trustpolicy.AuditVerificationInput{}, fmt.Errorf("%w: %v", ErrAnchorReceiptInvalid, err)
	}
	runtimeInputs.verifierRecords = append(runtimeInputs.verifierRecords, anchorVerifier)
	receipts := append([]trustpolicy.SignedObjectEnvelope{}, runtimeInputs.receipts...)
	receipts = append(receipts, receiptEnvelope)
	return trustpolicy.AuditVerificationInput{
		Scope:                    trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: segment.Header.SegmentID},
		Segment:                  segment,
		RawFramedSegmentBytes:    rawBytes,
		SegmentSealEnvelope:      sealEnvelope,
		PreviousSealEnvelopeHash: previousDigest,
		KnownSealDigests:         runtimeInputs.knownSealDigests,
		ReceiptEnvelopes:         receipts,
		VerifierRecords:          runtimeInputs.verifierRecords,
		EventContractCatalog:     runtimeInputs.catalog,
		SignerEvidence:           runtimeInputs.signerEvidence,
		StoragePostureEvidence:   runtimeInputs.storagePosture,
		Now:                      l.nowFn(),
	}, nil
}

func anchorVerifierRecordFromRequest(req AnchorSegmentRequest) (trustpolicy.VerifierRecord, error) {
	if strings.TrimSpace(req.SignerPublicKeyBase64) == "" {
		return trustpolicy.VerifierRecord{}, fmt.Errorf("signer public key is required")
	}
	record := trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             strings.TrimSpace(req.SignerKeyIDValue),
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: strings.TrimSpace(req.SignerPublicKeyBase64)},
		LogicalPurpose:         "audit_anchor",
		LogicalScope:           nonEmpty(strings.TrimSpace(req.SignerLogicalScope), "node"),
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "secretsd", InstanceID: nonEmpty(strings.TrimSpace(req.SignerInstanceID), "secretsd-1")},
		KeyProtectionPosture:   nonEmpty(strings.TrimSpace(req.KeyProtectionPosture), "os_keystore"),
		IdentityBindingPosture: "attested",
		PresenceMode:           nonEmpty(strings.TrimSpace(req.PresenceMode), "os_confirmation"),
		CreatedAt:              nonEmpty(strings.TrimSpace(req.RecordedAtRFC3339), time.Now().UTC().Format(time.RFC3339)),
		Status:                 "active",
	}
	if _, err := trustpolicy.NewVerifierRegistry([]trustpolicy.VerifierRecord{record}); err != nil {
		return trustpolicy.VerifierRecord{}, err
	}
	return record, nil
}

func buildAnchorReceiptEnvelopeForSegment(req AnchorSegmentRequest) (trustpolicy.SignedObjectEnvelope, error) {
	payloadJSON, err := marshalAnchorReceiptPayload(req)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.AuditReceiptSchemaID,
		PayloadSchemaVersion: trustpolicy.AuditReceiptSchemaVersion,
		Payload:              payloadJSON,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            req.Signature,
	}, nil
}

func marshalAnchorReceiptPayload(req AnchorSegmentRequest) ([]byte, error) {
	receiptPayload := map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            map[string]any{"hash_alg": req.SealDigest.HashAlg, "hash": req.SealDigest.Hash},
		"audit_receipt_kind":        "anchor",
		"subject_family":            "audit_segment_seal",
		"recorder":                  req.Recorder,
		"recorded_at":               receiptRecordedAt(req.RecordedAtRFC3339),
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.anchor.v0",
		"receipt_payload":           anchorReceiptPayloadMap(req),
	}
	payloadJSON, err := json.Marshal(receiptPayload)
	if err != nil {
		return nil, err
	}
	if _, err := jsoncanonicalizer.Transform(payloadJSON); err != nil {
		return nil, err
	}
	return payloadJSON, nil
}

func receiptRecordedAt(recordedAtRFC3339 string) string {
	recordedAt := strings.TrimSpace(recordedAtRFC3339)
	if recordedAt == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return recordedAt
}

func anchorReceiptPayloadMap(req AnchorSegmentRequest) map[string]any {
	anchorPayload := map[string]any{
		"anchor_kind":            nonEmpty(req.AnchorKind, "local_user_presence_signature"),
		"key_protection_posture": strings.TrimSpace(req.KeyProtectionPosture),
		"presence_mode":          strings.TrimSpace(req.PresenceMode),
		"anchor_witness": map[string]any{
			"witness_kind":   strings.TrimSpace(req.AnchorWitnessKind),
			"witness_digest": map[string]any{"hash_alg": req.AnchorWitnessDigest.HashAlg, "hash": req.AnchorWitnessDigest.Hash},
		},
	}
	approvalAssurance := strings.TrimSpace(req.ApprovalAssuranceLevel)
	if approvalAssurance != "" {
		anchorPayload["approval_assurance_level"] = approvalAssurance
	}
	if req.ApprovalDecisionDigest != nil {
		anchorPayload["approval_decision_digest"] = map[string]any{"hash_alg": req.ApprovalDecisionDigest.HashAlg, "hash": req.ApprovalDecisionDigest.Hash}
	}
	return anchorPayload
}

func nonEmpty(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func mustDigestIdentity(d trustpolicy.Digest) string {
	id, _ := d.Identity()
	return id
}
