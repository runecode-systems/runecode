package auditd

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type snapshotAuditReceiptPayload struct {
	SchemaID         string `json:"schema_id"`
	AuditReceiptKind string `json:"audit_receipt_kind"`
	ReceiptPayload   struct {
		ApprovalDecisionDigest *trustpolicy.Digest `json:"approval_decision_digest"`
	} `json:"receipt_payload"`
}

func approvalDecisionDigestFromReceipt(envelope trustpolicy.SignedObjectEnvelope) (string, bool, error) {
	if envelope.PayloadSchemaID != trustpolicy.AuditReceiptSchemaID {
		return "", false, nil
	}
	payload := snapshotAuditReceiptPayload{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return "", false, err
	}
	if strings.TrimSpace(payload.SchemaID) != trustpolicy.AuditReceiptSchemaID || strings.TrimSpace(payload.AuditReceiptKind) != "anchor" || payload.ReceiptPayload.ApprovalDecisionDigest == nil {
		return "", false, nil
	}
	identity, err := payload.ReceiptPayload.ApprovalDecisionDigest.Identity()
	if err != nil {
		return "", false, err
	}
	return identity, true, nil
}

func (l *Ledger) externalAnchorDerivedEvidenceIdentitiesLocked() (policyDigests []string, approvalDigests []string, requiredApprovalIDs []string, attestationDigests []string, instanceIdentityDigests []string, err error) {
	evidence, err := l.loadExternalAnchorEvidenceLocked()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for i := range evidence {
		rec := evidence[i]
		policyDigests, approvalDigests, requiredApprovalIDs, err = appendExternalAnchorCoreEvidence(rec, policyDigests, approvalDigests, requiredApprovalIDs)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		attestationDigests, instanceIdentityDigests, err = appendExternalAnchorSidecarEvidence(rec, attestationDigests, instanceIdentityDigests)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	return policyDigests, approvalDigests, requiredApprovalIDs, attestationDigests, instanceIdentityDigests, nil
}

func appendExternalAnchorCoreEvidence(rec trustpolicy.ExternalAnchorEvidencePayload, policyDigests []string, approvalDigests []string, requiredApprovalIDs []string) ([]string, []string, []string, error) {
	var err error
	policyDigests, err = appendOptionalDigestIdentity(policyDigests, rec.PolicyDecisionHash)
	if err != nil {
		return nil, nil, nil, err
	}
	approvalDigests, err = appendOptionalDigestIdentity(approvalDigests, rec.ApprovalRequestHash)
	if err != nil {
		return nil, nil, nil, err
	}
	approvalDigests, err = appendOptionalDigestIdentity(approvalDigests, rec.ApprovalDecisionHash)
	if err != nil {
		return nil, nil, nil, err
	}
	if approvalID := strings.TrimSpace(rec.RequiredApprovalID); approvalID != "" {
		requiredApprovalIDs = append(requiredApprovalIDs, approvalID)
	}
	return policyDigests, approvalDigests, requiredApprovalIDs, nil
}

func appendExternalAnchorSidecarEvidence(rec trustpolicy.ExternalAnchorEvidencePayload, attestationDigests []string, instanceIdentityDigests []string) ([]string, []string, error) {
	for j := range rec.SidecarRefs {
		identity, err := rec.SidecarRefs[j].Digest.Identity()
		if err != nil {
			return nil, nil, err
		}
		switch rec.SidecarRefs[j].EvidenceKind {
		case trustpolicy.ExternalAnchorSidecarKindAttestationRef:
			attestationDigests = append(attestationDigests, identity)
		case trustpolicy.ExternalAnchorSidecarKindProjectContextRef:
			instanceIdentityDigests = append(instanceIdentityDigests, identity)
		}
	}
	return attestationDigests, instanceIdentityDigests, nil
}

func appendOptionalDigestIdentity(target []string, digest *trustpolicy.Digest) ([]string, error) {
	if digest == nil {
		return target, nil
	}
	identity, err := digest.Identity()
	if err != nil {
		return nil, err
	}
	return append(target, identity), nil
}
