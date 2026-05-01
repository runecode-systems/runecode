package auditd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	externalAnchorIncrementalFoundationFileName      = "external-anchor-incremental-foundation.json"
	externalAnchorIncrementalFoundationSchemaVersion = 1
)

type externalAnchorIncrementalFoundation struct {
	SchemaVersion int                                              `json:"schema_version"`
	Seals         map[string]externalAnchorIncrementalSealSnapshot `json:"seals,omitempty"`
}

type externalAnchorIncrementalSealSnapshot struct {
	SegmentID                      string                                     `json:"segment_id,omitempty"`
	ReceiptDigests                 []string                                   `json:"receipt_digests,omitempty"`
	ExternalAnchorEvidenceDigests  []string                                   `json:"external_anchor_evidence_digests,omitempty"`
	ExternalAnchorSidecarDigests   []string                                   `json:"external_anchor_sidecar_digests,omitempty"`
	ExternalAnchorTargets          []externalAnchorVerificationTargetSnapshot `json:"external_anchor_targets,omitempty"`
	BaselineVerificationReport     string                                     `json:"baseline_verification_report_digest,omitempty"`
	BaselineVerificationReportedAt string                                     `json:"baseline_verification_reported_at,omitempty"`
}

type externalAnchorVerificationTargetSnapshot struct {
	TargetKind             string `json:"target_kind"`
	TargetDescriptorDigest string `json:"target_descriptor_digest"`
	TargetRequirement      string `json:"target_requirement,omitempty"`
}

type receiptSubjectSealDigestPayload struct {
	SchemaID         string             `json:"schema_id"`
	SubjectDigest    trustpolicy.Digest `json:"subject_digest"`
	SubjectFamily    string             `json:"subject_family,omitempty"`
	AuditReceiptKind string             `json:"audit_receipt_kind"`
}

func receiptSubjectSealDigestIdentity(envelope trustpolicy.SignedObjectEnvelope) (string, bool, error) {
	if envelope.PayloadSchemaID != trustpolicy.AuditReceiptSchemaID {
		return "", false, nil
	}
	payload := receiptSubjectSealDigestPayload{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return "", false, fmt.Errorf("decode receipt payload for incremental foundation: %w", err)
	}
	if strings.TrimSpace(payload.SchemaID) != trustpolicy.AuditReceiptSchemaID {
		return "", false, nil
	}
	family := strings.TrimSpace(payload.SubjectFamily)
	if family != "" && family != trustpolicy.AuditSegmentAnchoringSubjectSeal {
		return "", false, nil
	}
	id, err := payload.SubjectDigest.Identity()
	if err != nil {
		return "", false, fmt.Errorf("receipt subject_digest: %w", err)
	}
	return id, true, nil
}

func digestIdentityFromSidecarName(name string) (string, bool, error) {
	if !strings.HasSuffix(name, ".json") {
		return "", false, nil
	}
	hash := strings.TrimSuffix(name, ".json")
	d := trustpolicy.Digest{HashAlg: "sha256", Hash: hash}
	id, err := d.Identity()
	if err != nil {
		return "", false, fmt.Errorf("sidecar filename digest invalid %q: %w", name, err)
	}
	return id, true, nil
}

func digestFromIdentity(identity string) (trustpolicy.Digest, error) {
	trimmed := strings.TrimSpace(identity)
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return trustpolicy.Digest{}, fmt.Errorf("digest identity must be hash_alg:hash")
	}
	d := trustpolicy.Digest{HashAlg: strings.TrimSpace(parts[0]), Hash: strings.TrimSpace(parts[1])}
	if _, err := d.Identity(); err != nil {
		return trustpolicy.Digest{}, err
	}
	return d, nil
}

func appendUniqueIdentity(existing []string, identity string) []string {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return existing
	}
	for i := range existing {
		if existing[i] == identity {
			return existing
		}
	}
	return append(existing, identity)
}
