package brokerapi

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Service) currentRuntimeProvenanceSealDigest() (trustpolicy.Digest, bool) {
	if s == nil || s.auditLedger == nil {
		return trustpolicy.Digest{}, false
	}
	_, digest, err := s.auditLedger.LatestAnchorableSeal()
	if err != nil {
		return trustpolicy.Digest{}, false
	}
	if _, err := digest.Identity(); err != nil {
		return trustpolicy.Digest{}, false
	}
	return digest, true
}

func (s *Service) persistRuntimeProvenanceReceipt(subjectDigest trustpolicy.Digest, receiptKind string, subjectFamily string, payloadSchemaID string, receiptPayload map[string]any) error {
	if s == nil || s.auditLedger == nil {
		return fmt.Errorf("audit ledger unavailable")
	}
	if s.secretsSvc == nil {
		return fmt.Errorf("secrets service unavailable for trusted receipt signing")
	}
	envelope, verifier, err := s.signRuntimeProvenanceReceiptEnvelope(subjectDigest, receiptKind, subjectFamily, payloadSchemaID, receiptPayload)
	if err != nil {
		return err
	}
	if err := s.auditLedger.EnsureVerifierRecord(verifier); err != nil {
		return err
	}
	_, err = s.auditLedger.PersistReceiptEnvelope(envelope)
	return err
}

func (s *Service) persistMetaAuditReceipt(receiptKind string, scopeKind string, scopeRefDigest *trustpolicy.Digest, objectDigest *trustpolicy.Digest, manifestDigest *trustpolicy.Digest, sensitiveViewClass string) {
	if s == nil || s.auditLedger == nil {
		return
	}
	sealDigest, ok := s.currentRuntimeProvenanceSealDigest()
	if !ok {
		return
	}
	payload := runtimeMetaAuditReceiptPayload(receiptKind, scopeKind, scopeRefDigest, objectDigest, manifestDigest, sensitiveViewClass)
	if err := s.persistRuntimeProvenanceReceipt(sealDigest, strings.TrimSpace(receiptKind), trustpolicy.AuditSegmentAnchoringSubjectSeal, auditReceiptPayloadSchemaMetaAuditActionV0, payload); err != nil {
		log.Printf("brokerapi: meta-audit receipt persistence failed for %s: %v", strings.TrimSpace(receiptKind), err)
	}
}

func runtimeMetaAuditReceiptPayload(receiptKind string, scopeKind string, scopeRefDigest *trustpolicy.Digest, objectDigest *trustpolicy.Digest, manifestDigest *trustpolicy.Digest, sensitiveViewClass string) map[string]any {
	payload := map[string]any{
		"action_code":   strings.TrimSpace(receiptKind),
		"action_family": "meta_audit",
		"scope_kind":    strings.TrimSpace(scopeKind),
		"result":        "completed",
		"operator": map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "daemon",
			"principal_id":   "brokerapi",
			"instance_id":    "brokerapi-1",
		},
	}
	if scopeRefDigest != nil {
		payload["scope_ref_digest"] = *scopeRefDigest
	}
	if objectDigest != nil {
		payload["object_digest"] = *objectDigest
	}
	if manifestDigest != nil {
		payload["manifest_digest"] = *manifestDigest
	}
	if strings.TrimSpace(sensitiveViewClass) != "" {
		payload["sensitive_view_class"] = strings.TrimSpace(sensitiveViewClass)
	}
	return payload
}

func (s *Service) signRuntimeProvenanceReceiptEnvelope(subjectDigest trustpolicy.Digest, receiptKind string, subjectFamily string, payloadSchemaID string, receiptPayload map[string]any) (trustpolicy.SignedObjectEnvelope, trustpolicy.VerifierRecord, error) {
	payloadBytes, canonical, err := runtimeProvenanceReceiptEnvelopeBytes(subjectDigest, receiptKind, subjectFamily, payloadSchemaID, receiptPayload, s.now())
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	if s.secretsSvc == nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, fmt.Errorf("secrets service unavailable for trusted receipt signing")
	}
	presence, err := s.externalAnchorPresenceAttestation(subjectDigest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	signed, err := s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: canonical,
		TargetSealDigest:      subjectDigest,
		LogicalScope:          runtimeProvenanceVerifierScope,
		PresenceAttestation:   presence,
	})
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditReceiptSchemaID, PayloadSchemaVersion: trustpolicy.AuditReceiptSchemaVersion, Payload: payloadBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: signed.Signature}, externalAnchorVerifierRecord(signed, s.now()), nil
}

func runtimeProvenanceReceiptEnvelopeBytes(subjectDigest trustpolicy.Digest, receiptKind string, subjectFamily string, payloadSchemaID string, receiptPayload map[string]any, now time.Time) ([]byte, []byte, error) {
	payload := map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        strings.TrimSpace(receiptKind),
		"subject_family":            strings.TrimSpace(subjectFamily),
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"recorded_at":               now.UTC().Format(time.RFC3339),
		"receipt_payload_schema_id": strings.TrimSpace(payloadSchemaID),
		"receipt_payload":           receiptPayload,
	}
	if strings.TrimSpace(subjectFamily) == "" {
		delete(payload, "subject_family")
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
