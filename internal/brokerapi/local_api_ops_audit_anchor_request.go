package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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
	approvalDigest, approvalDecision, approvalAssurance, err := s.resolveAnchorApprovalContext(req)
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
	return s.signAndFinalizeAnchorRequest(base, approvalDecision)
}

func (s *Service) signAndFinalizeAnchorRequest(base auditd.AnchorSegmentRequest, approvalDecision *trustpolicy.ApprovalDecision) (auditd.AnchorSegmentRequest, error) {
	previewCanonical, err := canonicalAnchorReceiptPayloadBytes(base)
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	preview, err := s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: previewCanonical,
		TargetSealDigest:      base.SealDigest,
		LogicalScope:          base.SignerLogicalScope,
		ApprovalDecision:      approvalDecision,
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

	finalCanonical, err := canonicalAnchorReceiptPayloadBytes(finalReq)
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	finalSig, err := s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: finalCanonical,
		TargetSealDigest:      finalReq.SealDigest,
		LogicalScope:          finalReq.SignerLogicalScope,
		ApprovalDecision:      approvalDecision,
	})
	if err != nil {
		return auditd.AnchorSegmentRequest{}, err
	}
	finalReq.Signature = finalSig.Signature
	finalReq.SignerPublicKeyBase64 = base64.StdEncoding.EncodeToString(finalSig.SignerPublicKey)
	finalReq.SignerKeyIDValue = strings.TrimSpace(finalSig.SignerKeyIDValue)
	return finalReq, nil
}

func (s *Service) resolveAnchorApprovalContext(req AuditAnchorSegmentRequest) (*trustpolicy.Digest, *trustpolicy.ApprovalDecision, string, error) {
	requestedAssurance := strings.TrimSpace(req.ApprovalAssuranceLevel)
	if req.ApprovalDecisionDigest == nil {
		return nil, nil, requestedAssurance, nil
	}
	decisionDigestIdentity, err := req.ApprovalDecisionDigest.Identity()
	if err != nil {
		return nil, nil, "", err
	}
	approval, found := s.findApprovalByDecisionDigest(decisionDigestIdentity)
	if !found {
		return nil, nil, "", errors.New("approval decision digest is not available")
	}
	if strings.TrimSpace(approval.Summary.Status) != "consumed" {
		return nil, nil, "", errors.New("approval decision is not consumed")
	}
	if approval.DecisionEnvelope == nil {
		return nil, nil, "", errors.New("approval decision envelope is missing")
	}
	decision, err := decodeApprovalDecision(*approval.DecisionEnvelope)
	if err != nil {
		return nil, nil, "", err
	}
	derivedAssurance := strings.TrimSpace(decision.ApprovalAssuranceLevel)
	if requestedAssurance != "" && requestedAssurance != derivedAssurance {
		return nil, nil, "", errors.New("approval_assurance_level does not match approval decision")
	}
	if requestedAssurance != "" {
		derivedAssurance = requestedAssurance
	}
	resolvedDigest := *req.ApprovalDecisionDigest
	return &resolvedDigest, &decision, derivedAssurance, nil
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
		PrincipalID:   "secretsd",
		InstanceID:    "secretsd-1",
	}
}

func nowRFC3339(nowFn func() time.Time) string {
	if nowFn == nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return nowFn().UTC().Format(time.RFC3339)
}
