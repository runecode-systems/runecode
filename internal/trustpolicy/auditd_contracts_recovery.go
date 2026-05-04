package trustpolicy

import "fmt"

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
