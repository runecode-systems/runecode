package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Service) persistExternalAnchorReceiptAndVerify(record artifacts.ExternalAnchorPreparedMutationRecord, proofDigest trustpolicy.Digest) (trustpolicy.Digest, trustpolicy.Digest, error) {
	if s.auditLedger == nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, fmt.Errorf("audit ledger unavailable")
	}
	inputs, err := externalAnchorReceiptVerificationInputs(record, proofDigest)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	envelope, verifier, err := s.signExternalAnchorReceiptEnvelope(inputs.sealDigest, inputs.receiptPayload)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	receiptDigest, err := s.auditLedger.PersistReceiptEnvelope(envelope)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return s.verifyPersistedExternalAnchorReceipt(receiptDigest, record, verifier)
}

type externalAnchorReceiptInputs struct {
	sealDigest     trustpolicy.Digest
	receiptPayload map[string]any
}

func externalAnchorReceiptVerificationInputs(record artifacts.ExternalAnchorPreparedMutationRecord, proofDigest trustpolicy.Digest) (externalAnchorReceiptInputs, error) {
	sealDigest, _, err := externalAnchorSealDigest(record.TypedRequest)
	if err != nil {
		return externalAnchorReceiptInputs{}, err
	}
	primaryTarget, err := externalAnchorResolvedPrimaryTargetFromPreparedRecord(record)
	if err != nil {
		return externalAnchorReceiptInputs{}, err
	}
	return externalAnchorReceiptInputs{
		sealDigest:     sealDigest,
		receiptPayload: externalAnchorReceiptPayload(primaryTarget, proofDigest),
	}, nil
}

func externalAnchorReceiptPayload(primaryTarget externalAnchorResolvedTarget, proofDigest trustpolicy.Digest) map[string]any {
	return map[string]any{
		"anchor_kind": primaryTarget.ReceiptKind,
		"external_anchor": map[string]any{
			"target_kind":              primaryTarget.TargetKind,
			"runtime_adapter":          primaryTarget.RuntimeAdapter,
			"target_descriptor":        cloneStringAnyMap(primaryTarget.TargetDescriptor),
			"target_descriptor_digest": primaryTarget.TargetDescriptorDigest,
			"proof": map[string]any{
				"proof_kind":      primaryTarget.ProofKind,
				"proof_schema_id": primaryTarget.ProofSchemaID,
				"proof_digest":    proofDigest,
			},
		},
	}
}

func (s *Service) verifyPersistedExternalAnchorReceipt(receiptDigest trustpolicy.Digest, record artifacts.ExternalAnchorPreparedMutationRecord, verifier trustpolicy.VerifierRecord) (trustpolicy.Digest, trustpolicy.Digest, error) {
	if strings.TrimSpace(record.LastExecuteSnapshotSegmentID) == "" || strings.TrimSpace(record.LastExecuteSnapshotSealID) == "" {
		return receiptDigest, trustpolicy.Digest{}, nil
	}
	preverifiedSealDigest, err := digestFromIdentity(record.LastExecuteSnapshotSealID)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, fmt.Errorf("last_execute_snapshot_seal_digest invalid: %w", err)
	}
	reportDigest, err := s.auditLedger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(preverifiedSealDigest, verifier)
	if err != nil {
		if shouldSkipExternalAnchorVerification(err) {
			return receiptDigest, trustpolicy.Digest{}, nil
		}
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return receiptDigest, reportDigest, nil
}

func (s *Service) signExternalAnchorReceiptEnvelope(sealDigest trustpolicy.Digest, receiptPayload map[string]any) (trustpolicy.SignedObjectEnvelope, trustpolicy.VerifierRecord, error) {
	if s == nil || s.secretsSvc == nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, fmt.Errorf("secrets service unavailable")
	}
	payloadBytes, canonical, err := externalAnchorReceiptEnvelopeBytes(sealDigest, receiptPayload, s.now())
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	presenceAttestation, err := s.externalAnchorPresenceAttestation(sealDigest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	signed, err := s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{PayloadCanonicalBytes: canonical, TargetSealDigest: sealDigest, LogicalScope: "node", PresenceAttestation: presenceAttestation})
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	return externalAnchorSignedEnvelope(payloadBytes, signed), externalAnchorVerifierRecord(signed, s.now()), nil
}

func externalAnchorReceiptEnvelopeBytes(sealDigest trustpolicy.Digest, receiptPayload map[string]any, now time.Time) ([]byte, []byte, error) {
	payload := map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            sealDigest,
		"audit_receipt_kind":        "anchor",
		"subject_family":            trustpolicy.AuditSegmentAnchoringSubjectSeal,
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"recorded_at":               now.UTC().Format(time.RFC3339),
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.anchor.v0",
		"receipt_payload":           receiptPayload,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		return nil, nil, err
	}
	return payloadBytes, canonical, nil
}

func (s *Service) externalAnchorPresenceAttestation(sealDigest trustpolicy.Digest) (*secretsd.AuditAnchorPresenceAttestation, error) {
	mode := strings.TrimSpace(s.secretsSvc.AuditAnchorPresenceMode())
	if mode != "os_confirmation" && mode != "hardware_touch" {
		return nil, nil
	}
	challenge := fmt.Sprintf("external-anchor-%d", s.now().UTC().UnixNano())
	token, err := s.secretsSvc.ComputeAuditAnchorPresenceAcknowledgmentToken(mode, sealDigest, challenge)
	if err != nil {
		return nil, err
	}
	return &secretsd.AuditAnchorPresenceAttestation{Challenge: challenge, AcknowledgmentToken: token}, nil
}

func externalAnchorSignedEnvelope(payload []byte, signed secretsd.AuditAnchorSignResult) trustpolicy.SignedObjectEnvelope {
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditReceiptSchemaID, PayloadSchemaVersion: trustpolicy.AuditReceiptSchemaVersion, Payload: payload, SignatureInput: trustpolicy.SignatureInputProfile, Signature: signed.Signature}
}

func externalAnchorVerifierRecord(signed secretsd.AuditAnchorSignResult, now time.Time) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             signed.SignerKeyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(signed.SignerPublicKey)},
		LogicalPurpose:         "audit_anchor",
		LogicalScope:           "node",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "secretsd", InstanceID: "secretsd-1"},
		KeyProtectionPosture:   signed.Preconditions.KeyProtectionPosture,
		IdentityBindingPosture: signed.Preconditions.IdentityBindingPosture,
		PresenceMode:           signed.Preconditions.PresenceMode,
		CreatedAt:              now.UTC().Format(time.RFC3339),
		Status:                 "active",
	}
}

func shouldSkipExternalAnchorVerification(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.TrimSpace(err.Error())
	return strings.Contains(msg, "no sealed segment available for verification") ||
		strings.Contains(msg, "missing event contract catalog") ||
		strings.Contains(msg, "missing verifier records")
}
