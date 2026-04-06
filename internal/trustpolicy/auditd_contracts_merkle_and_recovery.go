package trustpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func ComputeOrderedAuditSegmentMerkleRoot(recordDigests []Digest) (Digest, error) {
	if len(recordDigests) == 0 {
		return Digest{}, fmt.Errorf("record digests are required for merkle construction")
	}
	level := make([][32]byte, 0, len(recordDigests))
	for index := range recordDigests {
		recordDigestBytes, err := digestHexBytes(recordDigests[index])
		if err != nil {
			return Digest{}, fmt.Errorf("record_digests[%d]: %w", index, err)
		}
		leafMaterial := append(append([]byte{}, []byte("runecode.audit.merkle.leaf.v1:")...), recordDigestBytes...)
		level = append(level, sha256.Sum256(leafMaterial))
	}
	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for index := 0; index < len(level); index += 2 {
			left := level[index]
			right := left
			if index+1 < len(level) {
				right = level[index+1]
			}
			nodeMaterial := append(append(append([]byte{}, []byte("runecode.audit.merkle.node.v1:")...), left[:]...), right[:]...)
			next = append(next, sha256.Sum256(nodeMaterial))
		}
		level = next
	}
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(level[0][:])}, nil
}

func VerifyOrderedAuditSegmentMerkleRoot(recordDigests []Digest, expected Digest) error {
	expectedIdentity, err := expected.Identity()
	if err != nil {
		return fmt.Errorf("expected merkle root: %w", err)
	}
	computed, err := ComputeOrderedAuditSegmentMerkleRoot(recordDigests)
	if err != nil {
		return err
	}
	computedIdentity, _ := computed.Identity()
	if computedIdentity != expectedIdentity {
		return fmt.Errorf("merkle root mismatch: got %q want %q", computedIdentity, expectedIdentity)
	}
	return nil
}

func ComputeSegmentFileHash(rawFramedSegmentBytes []byte) (Digest, error) {
	if len(rawFramedSegmentBytes) == 0 {
		return Digest{}, fmt.Errorf("raw framed segment bytes are required")
	}
	sum := sha256.Sum256(rawFramedSegmentBytes)
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func VerifySegmentFileHash(rawFramedSegmentBytes []byte, expected Digest) error {
	expectedIdentity, err := expected.Identity()
	if err != nil {
		return fmt.Errorf("expected segment file hash: %w", err)
	}
	computed, err := ComputeSegmentFileHash(rawFramedSegmentBytes)
	if err != nil {
		return err
	}
	computedIdentity, _ := computed.Identity()
	if computedIdentity != expectedIdentity {
		return fmt.Errorf("segment_file_hash mismatch: got %q want %q", computedIdentity, expectedIdentity)
	}
	return nil
}

func digestHexBytes(digest Digest) ([]byte, error) {
	if _, err := digest.Identity(); err != nil {
		return nil, err
	}
	decoded, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

type AuditSegmentRecoveryState struct {
	SegmentID            string `json:"segment_id"`
	HeaderState          string `json:"header_state"`
	LifecycleMarkerState string `json:"lifecycle_marker_state"`
	HasTornTrailingFrame bool   `json:"has_torn_trailing_frame"`
	FrameIntegrityOK     bool   `json:"frame_integrity_ok"`
	SealIntegrityOK      bool   `json:"seal_integrity_ok"`
}

type AuditSegmentRecoveryDecision struct {
	Action                string `json:"action"`
	TruncateTrailingFrame bool   `json:"truncate_trailing_frame"`
	Quarantine            bool   `json:"quarantine"`
	FailClosed            bool   `json:"fail_closed"`
	Message               string `json:"message"`
}

func EvaluateAuditSegmentRecovery(state AuditSegmentRecoveryState) (AuditSegmentRecoveryDecision, error) {
	if state.SegmentID == "" {
		return AuditSegmentRecoveryDecision{}, fmt.Errorf("segment_id is required")
	}
	if decision, handled := evaluateQuarantinedOrInconsistentRecovery(state); handled {
		return decision, nil
	}
	if decision, handled := evaluateImmutableRecovery(state); handled {
		return decision, nil
	}
	return evaluateOpenSegmentRecovery(state)
}

func evaluateQuarantinedOrInconsistentRecovery(state AuditSegmentRecoveryState) (AuditSegmentRecoveryDecision, bool) {
	if state.HeaderState == "quarantined" || state.LifecycleMarkerState == "quarantined" {
		return AuditSegmentRecoveryDecision{
			Action:     "quarantine_inconsistent_segment",
			Quarantine: true,
			FailClosed: true,
			Message:    "segment is already quarantined and cannot be repaired silently",
		}, true
	}
	if state.HeaderState != state.LifecycleMarkerState {
		return AuditSegmentRecoveryDecision{
			Action:     "quarantine_inconsistent_segment",
			Quarantine: true,
			FailClosed: true,
			Message:    "header and lifecycle marker states disagree",
		}, true
	}
	return AuditSegmentRecoveryDecision{}, false
}

func evaluateImmutableRecovery(state AuditSegmentRecoveryState) (AuditSegmentRecoveryDecision, bool) {
	if state.HeaderState != "sealed" && state.HeaderState != "anchored" && state.HeaderState != "imported" {
		return AuditSegmentRecoveryDecision{}, false
	}
	if !state.FrameIntegrityOK || !state.SealIntegrityOK || state.HasTornTrailingFrame {
		return AuditSegmentRecoveryDecision{
			Action:     "quarantine_inconsistent_sealed_segment",
			Quarantine: true,
			FailClosed: true,
			Message:    "immutable segment mismatch requires quarantine; silent repair is forbidden",
		}, true
	}
	return AuditSegmentRecoveryDecision{
		Action:     "accept_immutable_segment",
		FailClosed: false,
		Message:    "segment verified and immutable",
	}, true
}

func evaluateOpenSegmentRecovery(state AuditSegmentRecoveryState) (AuditSegmentRecoveryDecision, error) {
	if state.HeaderState != "open" {
		return AuditSegmentRecoveryDecision{}, fmt.Errorf("unsupported segment state %q", state.HeaderState)
	}
	if state.HasTornTrailingFrame {
		return AuditSegmentRecoveryDecision{
			Action:                "truncate_open_torn_trailing_frame",
			TruncateTrailingFrame: true,
			FailClosed:            false,
			Message:               "open segment may truncate torn trailing frame before sealing",
		}, nil
	}
	if !state.FrameIntegrityOK {
		return AuditSegmentRecoveryDecision{
			Action:     "quarantine_inconsistent_segment",
			Quarantine: true,
			FailClosed: true,
			Message:    "open segment corruption is not a torn-trailing-frame case",
		}, nil
	}
	return AuditSegmentRecoveryDecision{
		Action:     "resume_open_append",
		FailClosed: false,
		Message:    "open segment integrity is stable for continued append",
	}, nil
}
