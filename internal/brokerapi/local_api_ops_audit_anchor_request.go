package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Service) buildAnchorSegmentRequest(req AuditAnchorSegmentRequest) (auditd.AnchorSegmentRequest, error) {
	if _, err := req.SealDigest.Identity(); err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	requiredApproval, err := s.anchorApprovalRequirement(req.SealDigest)
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	approvalDigest, approvalDecision, approvalAssurance, err := s.resolveAnchorApprovalContext(req, requiredApproval)
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	base := auditd.AnchorSegmentRequest{
		SealDigest:             req.SealDigest,
		ApprovalDecisionDigest: approvalDigest,
		ApprovalAssuranceLevel: approvalAssurance,
		RecordedAtRFC3339:      nowRFC3339(s.now),
		Recorder:               anchorReceiptRecorderPrincipal(),
		SignerLogicalScope:     strings.TrimSpace(req.SignerLogicalScope),
		SignerInstanceID:       strings.TrimSpace(req.SignerInstanceID),
	}
	return s.signAndFinalizeAnchorRequest(base, approvalDecision, req.PresenceAttestation)
}

func (s *Service) signAndFinalizeAnchorRequest(base auditd.AnchorSegmentRequest, approvalDecision *trustpolicy.ApprovalDecision, presenceAttestation *AuditAnchorPresenceAttestation) (auditd.AnchorSegmentRequest, error) {
	previewCanonical, err := canonicalAnchorReceiptPayloadBytes(base)
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	secretsPresence := toSecretsAuditAnchorPresenceAttestation(presenceAttestation)
	preview, err := s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: previewCanonical,
		TargetSealDigest:      base.SealDigest,
		LogicalScope:          base.SignerLogicalScope,
		ApprovalDecision:      approvalDecision,
		PresenceAttestation:   secretsPresence,
	})
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}

	finalReq := base
	finalReq.AnchorKind = "local_user_presence_signature"
	finalReq.KeyProtectionPosture = strings.TrimSpace(preview.Preconditions.KeyProtectionPosture)
	finalReq.PresenceMode = strings.TrimSpace(preview.Preconditions.PresenceMode)
	finalReq.AnchorWitnessKind = strings.TrimSpace(preview.AnchorWitnessKind)
	finalReq.AnchorWitnessDigest = preview.AnchorWitnessDigest

	finalSig, err := s.finalAnchorSignature(finalReq, approvalDecision, secretsPresence)
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	finalReq.Signature = finalSig.Signature
	finalReq.SignerPublicKeyBase64 = base64.StdEncoding.EncodeToString(finalSig.SignerPublicKey)
	finalReq.SignerKeyIDValue = strings.TrimSpace(finalSig.SignerKeyIDValue)
	return finalReq, nil
}

func (s *Service) finalAnchorSignature(finalReq auditd.AnchorSegmentRequest, approvalDecision *trustpolicy.ApprovalDecision, presenceAttestation *secretsd.AuditAnchorPresenceAttestation) (secretsd.AuditAnchorSignResult, error) {
	finalCanonical, err := canonicalAnchorReceiptPayloadBytes(finalReq)
	if err != nil {
		return secretsd.AuditAnchorSignResult{}, err
	}
	return s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: finalCanonical,
		TargetSealDigest:      finalReq.SealDigest,
		LogicalScope:          finalReq.SignerLogicalScope,
		ApprovalDecision:      approvalDecision,
		PresenceAttestation:   presenceAttestation,
	})
}

func toSecretsAuditAnchorPresenceAttestation(att *AuditAnchorPresenceAttestation) *secretsd.AuditAnchorPresenceAttestation {
	if att == nil {
		return nil
	}
	return &secretsd.AuditAnchorPresenceAttestation{
		Challenge:           strings.TrimSpace(att.Challenge),
		AcknowledgmentToken: strings.TrimSpace(att.AcknowledgmentToken),
	}
}

type anchorApprovalRequirement struct {
	Required          bool
	RequiredAssurance string
	PolicyDecisionRef string
}

func (s *Service) findApprovalByDecisionDigest(decisionDigestIdentity string) (approvalRecord, bool) {
	for _, approval := range s.approvalRecordsByID() {
		if strings.TrimSpace(approval.Summary.DecisionDigest) == decisionDigestIdentity {
			return approval, true
		}
	}
	return approvalRecord{}, false
}

func canonicalAnchorReceiptPayloadBytes(req auditd.AnchorSegmentRequest) ([]byte, error) {
	payload := map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            map[string]any{"hash_alg": req.SealDigest.HashAlg, "hash": req.SealDigest.Hash},
		"audit_receipt_kind":        "anchor",
		"subject_family":            "audit_segment_seal",
		"recorder":                  req.Recorder,
		"recorded_at":               req.RecordedAtRFC3339,
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.anchor.v0",
		"receipt_payload":           anchorReceiptPayloadMap(req),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

func anchorReceiptPayloadMap(req auditd.AnchorSegmentRequest) map[string]any {
	payload := map[string]any{
		"anchor_kind":            "local_user_presence_signature",
		"key_protection_posture": strings.TrimSpace(req.KeyProtectionPosture),
		"presence_mode":          strings.TrimSpace(req.PresenceMode),
		"anchor_witness": map[string]any{
			"witness_kind":   strings.TrimSpace(req.AnchorWitnessKind),
			"witness_digest": map[string]any{"hash_alg": req.AnchorWitnessDigest.HashAlg, "hash": req.AnchorWitnessDigest.Hash},
		},
	}
	if assurance := strings.TrimSpace(req.ApprovalAssuranceLevel); assurance != "" {
		payload["approval_assurance_level"] = assurance
	}
	if req.ApprovalDecisionDigest != nil {
		payload["approval_decision_digest"] = map[string]any{"hash_alg": req.ApprovalDecisionDigest.HashAlg, "hash": req.ApprovalDecisionDigest.Hash}
	}
	return payload
}

func anchorReceiptRecorderPrincipal() trustpolicy.PrincipalIdentity {
	return trustpolicy.PrincipalIdentity{
		SchemaID:      "runecode.protocol.v0.PrincipalIdentity",
		SchemaVersion: "0.2.0",
		ActorKind:     "daemon",
		PrincipalID:   "auditd",
		InstanceID:    "auditd-1",
	}
}

func nowRFC3339(nowFn func() time.Time) string {
	if nowFn == nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return nowFn().UTC().Format(time.RFC3339)
}
