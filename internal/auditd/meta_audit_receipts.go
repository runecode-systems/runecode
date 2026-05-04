package auditd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	auditReceiptKindVerifierConfigurationChanged = "verifier_configuration_changed"
	auditReceiptKindTrustRootUpdated             = "trust_root_updated"
	auditReceiptKindEvidenceImport               = "evidence_import"
	auditReceiptPayloadSchemaMetaAuditActionV0   = "runecode.protocol.audit.receipt.meta_audit_action.v0"
	metaAuditActionFamily                        = "meta_audit"
	metaAuditResultApplied                       = "applied"
)

func (l *Ledger) persistMetaAuditReceiptsForVerificationContractsLocked(config VerificationConfiguration) error {
	sealDigest, ok, err := l.latestSealDigestForMetaReceiptLocked()
	if err != nil || !ok {
		return err
	}
	payloads, err := verificationContractMetaAuditPayloads(config)
	if err != nil {
		return err
	}
	for _, item := range payloads {
		if err := l.persistMetaAuditReceiptLocked(sealDigest, item.kind, item.payload); err != nil {
			return err
		}
	}
	return nil
}

type metaAuditPayload struct {
	kind    string
	payload map[string]any
}

func verificationContractMetaAuditPayloads(config VerificationConfiguration) ([]metaAuditPayload, error) {
	verifierDigest, err := canonicalDigest(config.VerifierRecords)
	if err != nil {
		return nil, err
	}
	catalogDigest, err := canonicalDigest(config.EventContractCatalog)
	if err != nil {
		return nil, err
	}
	return []metaAuditPayload{
		{kind: auditReceiptKindVerifierConfigurationChanged, payload: newVerificationContractMetaAuditPayload(auditReceiptKindVerifierConfigurationChanged, "audit_verification_configuration", nil, verifierDigest)},
		{kind: auditReceiptKindEvidenceImport, payload: newVerificationContractMetaAuditPayload(auditReceiptKindEvidenceImport, "audit_verification_configuration", &catalogDigest, verifierDigest)},
		{kind: auditReceiptKindTrustRootUpdated, payload: newVerificationContractMetaAuditPayload(auditReceiptKindTrustRootUpdated, "audit_trust_root_configuration", &catalogDigest, verifierDigest)},
	}, nil
}

func newVerificationContractMetaAuditPayload(actionCode, scopeKind string, scopeRefDigest *trustpolicy.Digest, objectDigest trustpolicy.Digest) map[string]any {
	payload := map[string]any{
		"action_code":   actionCode,
		"action_family": metaAuditActionFamily,
		"scope_kind":    scopeKind,
		"result":        metaAuditResultApplied,
		"object_digest": objectDigest,
		"operator":      metaAuditAuditdOperatorPrincipal("auditd-meta-audit"),
	}
	if scopeRefDigest != nil {
		payload["scope_ref_digest"] = *scopeRefDigest
	}
	return payload
}

func (l *Ledger) persistMetaAuditReceiptLocked(subjectDigest trustpolicy.Digest, receiptKind string, payload map[string]any) error {
	envelope, verifier, err := l.signMetaAuditReceiptEnvelope(subjectDigest, receiptKind, payload, l.nowFn().UTC())
	if err != nil {
		return err
	}
	if err := l.ensureVerifierRecordDurableLocked(verifier); err != nil {
		return err
	}
	_, err = l.persistReceiptEnvelopeLocked(envelope)
	return err
}

func (l *Ledger) signMetaAuditReceiptEnvelope(subjectDigest trustpolicy.Digest, receiptKind string, payload map[string]any, now time.Time) (trustpolicy.SignedObjectEnvelope, trustpolicy.VerifierRecord, error) {
	if l == nil || l.metaAuditSigner == nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, fmt.Errorf("meta-audit signer unavailable")
	}
	rawPayload, canonicalPayload, err := metaAuditReceiptPayloadBytes(subjectDigest, receiptKind, payload, now)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	presence, err := metaAuditPresenceAttestation(l.metaAuditSigner, subjectDigest, now)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	signed, err := l.metaAuditSigner.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: canonicalPayload,
		TargetSealDigest:      subjectDigest,
		LogicalScope:          "node",
		PresenceAttestation:   presence,
	})
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.VerifierRecord{}, err
	}
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.AuditReceiptSchemaID,
		PayloadSchemaVersion: trustpolicy.AuditReceiptSchemaVersion,
		Payload:              rawPayload,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            signed.Signature,
	}, metaAuditVerifierRecord(signed, now), nil
}

func metaAuditPresenceAttestation(signer *secretsd.Service, sealDigest trustpolicy.Digest, now time.Time) (*secretsd.AuditAnchorPresenceAttestation, error) {
	if signer == nil {
		return nil, fmt.Errorf("meta-audit signer unavailable")
	}
	_ = sealDigest
	_ = now
	mode := strings.TrimSpace(signer.AuditAnchorPresenceMode())
	if mode != "os_confirmation" && mode != "hardware_touch" {
		return nil, nil
	}
	return nil, fmt.Errorf("meta-audit signer must not self-attest operator presence for mode %q", mode)
}

func metaAuditVerifierRecord(signed secretsd.AuditAnchorSignResult, now time.Time) trustpolicy.VerifierRecord {
	instanceID := "secretsd-" + strings.TrimSpace(signed.SignerKeyIDValue)
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             signed.SignerKeyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(signed.SignerPublicKey)},
		LogicalPurpose:         "audit_anchor",
		LogicalScope:           "node",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "secretsd", InstanceID: instanceID},
		KeyProtectionPosture:   signed.Preconditions.KeyProtectionPosture,
		IdentityBindingPosture: signed.Preconditions.IdentityBindingPosture,
		PresenceMode:           signed.Preconditions.PresenceMode,
		CreatedAt:              now.UTC().Format(time.RFC3339),
		Status:                 "active",
	}
}

func metaAuditReceiptPayloadBytes(subjectDigest trustpolicy.Digest, receiptKind string, payload map[string]any, now time.Time) ([]byte, []byte, error) {
	receiptPayload := map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        strings.TrimSpace(receiptKind),
		"subject_family":            trustpolicy.AuditSegmentAnchoringSubjectSeal,
		"recorder":                  metaAuditAuditdOperatorPrincipal("auditd-meta-audit"),
		"recorded_at":               now.Format(time.RFC3339),
		"receipt_payload_schema_id": auditReceiptPayloadSchemaMetaAuditActionV0,
		"receipt_payload":           payload,
	}
	rawPayload, err := json.Marshal(receiptPayload)
	if err != nil {
		return nil, nil, err
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(rawPayload)
	if err != nil {
		return nil, nil, err
	}
	return rawPayload, canonicalPayload, nil
}

func (l *Ledger) latestSealDigestForMetaReceiptLocked() (trustpolicy.Digest, bool, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if strings.TrimSpace(state.LastSealEnvelopeDigest) == "" {
		return trustpolicy.Digest{}, false, nil
	}
	digest, err := digestFromIdentity(state.LastSealEnvelopeDigest)
	if err != nil {
		return trustpolicy.Digest{}, false, fmt.Errorf("last seal digest identity invalid: %w", err)
	}
	return digest, true, nil
}

func metaAuditAuditdOperatorPrincipal(instanceID string) trustpolicy.PrincipalIdentity {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "auditd-unknown"
	}
	return trustpolicy.PrincipalIdentity{
		SchemaID:      "runecode.protocol.v0.PrincipalIdentity",
		SchemaVersion: "0.2.0",
		ActorKind:     "daemon",
		PrincipalID:   "auditd",
		InstanceID:    instanceID,
	}
}
