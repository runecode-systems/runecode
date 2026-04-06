package trustpolicy

import (
	"fmt"
	"time"
)

type AuditSegmentCutWindowPolicy struct {
	OwnershipScope            string `json:"ownership_scope"`
	MaxSegmentBytes           int64  `json:"max_segment_bytes,omitempty"`
	MaxSegmentDurationSeconds int64  `json:"max_segment_duration_seconds,omitempty"`
	CutTrigger                string `json:"cut_trigger"`
}

type AuditSegmentSealPayload struct {
	SchemaID                   string                      `json:"schema_id"`
	SchemaVersion              string                      `json:"schema_version"`
	SegmentID                  string                      `json:"segment_id"`
	SealedAfterState           string                      `json:"sealed_after_state"`
	SegmentState               string                      `json:"segment_state"`
	SegmentCut                 AuditSegmentCutWindowPolicy `json:"segment_cut"`
	EventCount                 int64                       `json:"event_count"`
	FirstRecordDigest          Digest                      `json:"first_record_digest"`
	LastRecordDigest           Digest                      `json:"last_record_digest"`
	MerkleProfile              string                      `json:"merkle_profile"`
	MerkleRoot                 Digest                      `json:"merkle_root"`
	SegmentFileHashScope       string                      `json:"segment_file_hash_scope"`
	SegmentFileHash            Digest                      `json:"segment_file_hash"`
	SealChainIndex             int64                       `json:"seal_chain_index"`
	PreviousSealDigest         *Digest                     `json:"previous_seal_digest,omitempty"`
	AnchoringSubject           string                      `json:"anchoring_subject"`
	SealedAt                   string                      `json:"sealed_at"`
	ProtocolBundleManifestHash Digest                      `json:"protocol_bundle_manifest_hash"`
	SealReason                 string                      `json:"seal_reason,omitempty"`
}

func ValidateAuditSegmentSealPayload(seal AuditSegmentSealPayload) error {
	if err := validateAuditSegmentSealCoreFields(seal); err != nil {
		return err
	}
	if err := validateAuditSegmentSealDigestFields(seal); err != nil {
		return err
	}
	if err := validateAuditSegmentSealChainFields(seal); err != nil {
		return err
	}
	return validateAuditSegmentSealMetadataFields(seal)
}

func validateAuditSegmentSealCoreFields(seal AuditSegmentSealPayload) error {
	if seal.SchemaID != AuditSegmentSealSchemaID {
		return fmt.Errorf("unexpected segment seal schema_id %q", seal.SchemaID)
	}
	if seal.SchemaVersion != AuditSegmentSealSchemaVersion {
		return fmt.Errorf("unexpected segment seal schema_version %q", seal.SchemaVersion)
	}
	if seal.SegmentID == "" {
		return fmt.Errorf("segment_id is required")
	}
	if seal.SealedAfterState != AuditSegmentStateOpen {
		return fmt.Errorf("sealed_after_state must be %q", AuditSegmentStateOpen)
	}
	if !isAllowedSegmentSealState(seal.SegmentState) {
		return fmt.Errorf("unsupported segment_state %q", seal.SegmentState)
	}
	if err := ValidateAuditSegmentCutWindowPolicy(seal.SegmentCut); err != nil {
		return fmt.Errorf("segment_cut: %w", err)
	}
	if seal.EventCount < 1 {
		return fmt.Errorf("event_count must be >= 1")
	}
	return nil
}

func validateAuditSegmentSealDigestFields(seal AuditSegmentSealPayload) error {
	if _, err := seal.FirstRecordDigest.Identity(); err != nil {
		return fmt.Errorf("first_record_digest: %w", err)
	}
	if _, err := seal.LastRecordDigest.Identity(); err != nil {
		return fmt.Errorf("last_record_digest: %w", err)
	}
	if seal.MerkleProfile != AuditSegmentMerkleProfileOrderedDSEv1 {
		return fmt.Errorf("unsupported merkle_profile %q", seal.MerkleProfile)
	}
	if _, err := seal.MerkleRoot.Identity(); err != nil {
		return fmt.Errorf("merkle_root: %w", err)
	}
	if seal.SegmentFileHashScope != AuditSegmentFileHashScopeRawFramedV1 {
		return fmt.Errorf("unsupported segment_file_hash_scope %q", seal.SegmentFileHashScope)
	}
	if _, err := seal.SegmentFileHash.Identity(); err != nil {
		return fmt.Errorf("segment_file_hash: %w", err)
	}
	return nil
}

func validateAuditSegmentSealChainFields(seal AuditSegmentSealPayload) error {
	if seal.SealChainIndex < 0 {
		return fmt.Errorf("seal_chain_index must be >= 0")
	}
	if seal.SealChainIndex == 0 {
		if seal.PreviousSealDigest != nil {
			return fmt.Errorf("previous_seal_digest must be absent for seal_chain_index=0")
		}
		return nil
	}
	if seal.PreviousSealDigest == nil {
		return fmt.Errorf("previous_seal_digest is required for seal_chain_index>0")
	}
	if _, err := seal.PreviousSealDigest.Identity(); err != nil {
		return fmt.Errorf("previous_seal_digest: %w", err)
	}
	return nil
}

func validateAuditSegmentSealMetadataFields(seal AuditSegmentSealPayload) error {
	if seal.AnchoringSubject != AuditSegmentAnchoringSubjectSeal {
		return fmt.Errorf("anchoring_subject must be %q", AuditSegmentAnchoringSubjectSeal)
	}
	if seal.SealedAt == "" {
		return fmt.Errorf("sealed_at is required")
	}
	if _, err := time.Parse(time.RFC3339, seal.SealedAt); err != nil {
		return fmt.Errorf("invalid sealed_at: %w", err)
	}
	if _, err := seal.ProtocolBundleManifestHash.Identity(); err != nil {
		return fmt.Errorf("protocol_bundle_manifest_hash: %w", err)
	}
	if seal.SealReason != "" && !sealReasonPattern.MatchString(seal.SealReason) {
		return fmt.Errorf("seal_reason must match %s", sealReasonPattern.String())
	}
	return nil
}

func ValidateAuditSegmentCutWindowPolicy(policy AuditSegmentCutWindowPolicy) error {
	if policy.OwnershipScope != AuditSegmentOwnershipScopeInstanceGlobal {
		return fmt.Errorf("ownership_scope must be %q", AuditSegmentOwnershipScopeInstanceGlobal)
	}
	if policy.CutTrigger != AuditSegmentCutTriggerSizeWindow && policy.CutTrigger != AuditSegmentCutTriggerTimeWindow {
		return fmt.Errorf("unsupported cut_trigger %q", policy.CutTrigger)
	}
	hasSize := policy.MaxSegmentBytes > 0
	hasTime := policy.MaxSegmentDurationSeconds > 0
	if !hasSize && !hasTime {
		return fmt.Errorf("segment cut policy requires max_segment_bytes and/or max_segment_duration_seconds")
	}
	if policy.CutTrigger == AuditSegmentCutTriggerSizeWindow && !hasSize {
		return fmt.Errorf("size_window cut_trigger requires max_segment_bytes")
	}
	if policy.CutTrigger == AuditSegmentCutTriggerTimeWindow && !hasTime {
		return fmt.Errorf("time_window cut_trigger requires max_segment_duration_seconds")
	}
	return nil
}

func ValidateAuditSegmentSealChainLink(current AuditSegmentSealPayload, previousSealEnvelopeDigest *Digest) error {
	if err := ValidateAuditSegmentSealPayload(current); err != nil {
		return err
	}
	if current.SealChainIndex == 0 {
		if previousSealEnvelopeDigest != nil {
			return fmt.Errorf("unexpected previous seal digest for genesis segment seal")
		}
		return nil
	}
	if previousSealEnvelopeDigest == nil {
		return fmt.Errorf("previous seal envelope digest is required for non-genesis segment seal")
	}
	prevIdentity, err := previousSealEnvelopeDigest.Identity()
	if err != nil {
		return fmt.Errorf("previous seal envelope digest: %w", err)
	}
	linkedIdentity, err := current.PreviousSealDigest.Identity()
	if err != nil {
		return fmt.Errorf("previous_seal_digest: %w", err)
	}
	if prevIdentity != linkedIdentity {
		return fmt.Errorf("previous_seal_digest %q does not match expected prior seal digest %q", linkedIdentity, prevIdentity)
	}
	return nil
}

func isAllowedSegmentSealState(state string) bool {
	switch state {
	case AuditSegmentStateSealed, AuditSegmentStateAnchored, AuditSegmentStateImported, AuditSegmentStateQuarantined:
		return true
	default:
		return false
	}
}
