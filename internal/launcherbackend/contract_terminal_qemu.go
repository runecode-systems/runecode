package launcherbackend

import (
	"fmt"
	"strings"
)

func (r BackendTerminalReport) Normalized() BackendTerminalReport {
	out := r
	out.RunID = strings.TrimSpace(out.RunID)
	out.StageID = strings.TrimSpace(out.StageID)
	out.RoleInstanceID = strings.TrimSpace(out.RoleInstanceID)
	out.IsolateID = strings.TrimSpace(out.IsolateID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.TerminationKind = normalizeBackendTerminationKind(out.TerminationKind)
	out.FailureReasonCode = normalizeBackendErrorCode(out.FailureReasonCode)
	out.FallbackPosture = normalizeBackendFallbackPosture(out.FallbackPosture)
	out.TerminatedAt = strings.TrimSpace(out.TerminatedAt)
	if !out.FailClosed {
		out.FailClosed = true
	}
	if out.FallbackPosture == "" {
		out.FallbackPosture = BackendFallbackPostureNoAutomaticFallback
	}
	if out.TerminationKind == BackendTerminationKindCompleted {
		out.FailureReasonCode = ""
	}
	if out.TerminationKind == BackendTerminationKindUnknown {
		out.TerminationKind = BackendTerminationKindFailed
		if out.FailureReasonCode == "" {
			out.FailureReasonCode = BackendErrorCodeTerminalReportInvalid
		}
	}
	if out.TerminationKind == BackendTerminationKindFailed && out.FailureReasonCode == "" {
		out.FailureReasonCode = BackendErrorCodeTerminalReportInvalid
	}
	return out
}

func (r BackendTerminalReport) Validate() error {
	normalized := r.Normalized()
	if err := validateTerminalKindAndFailClosed(normalized); err != nil {
		return err
	}
	if err := validateTerminalFallbackAndFailureReason(normalized); err != nil {
		return err
	}
	return validateTerminalFailureCode(normalized)
}

func validateTerminalKindAndFailClosed(normalized BackendTerminalReport) error {
	if normalized.TerminationKind != BackendTerminationKindCompleted && normalized.TerminationKind != BackendTerminationKindFailed {
		return fmt.Errorf("termination_kind must be %q or %q", BackendTerminationKindCompleted, BackendTerminationKindFailed)
	}
	if !normalized.FailClosed {
		return fmt.Errorf("fail_closed must be true")
	}
	return nil
}

func validateTerminalFallbackAndFailureReason(normalized BackendTerminalReport) error {
	if normalized.FallbackPosture != BackendFallbackPostureNoAutomaticFallback && normalized.FallbackPosture != BackendFallbackPostureContainerOptInOnly {
		return fmt.Errorf("fallback_posture must be %q or %q", BackendFallbackPostureNoAutomaticFallback, BackendFallbackPostureContainerOptInOnly)
	}
	if normalized.TerminationKind == BackendTerminationKindCompleted && normalized.FailureReasonCode != "" {
		return fmt.Errorf("failure_reason_code must be empty when termination_kind is %q", BackendTerminationKindCompleted)
	}
	if normalized.TerminationKind == BackendTerminationKindFailed && normalized.FailureReasonCode == "" {
		return fmt.Errorf("failure_reason_code is required when termination_kind is %q", BackendTerminationKindFailed)
	}
	return nil
}

func validateTerminalFailureCode(normalized BackendTerminalReport) error {
	if normalized.FailureReasonCode == "" {
		return nil
	}
	if err := ValidateBackendErrorCode(normalized.FailureReasonCode); err != nil {
		return fmt.Errorf("failure_reason_code: %w", err)
	}
	return nil
}

func ValidateBackendErrorCode(code string) error {
	if normalizeBackendErrorCode(code) == "" {
		return fmt.Errorf("must be a known backend error code")
	}
	return nil
}

func (q QEMUProvenance) Trimmed() QEMUProvenance {
	return QEMUProvenance{
		Version:       strings.TrimSpace(q.Version),
		BuildIdentity: strings.TrimSpace(q.BuildIdentity),
	}
}

func (q QEMUProvenance) IsZero() bool {
	trimmed := q.Trimmed()
	return trimmed.Version == "" && trimmed.BuildIdentity == ""
}

func (q QEMUProvenance) Validate() error {
	trimmed := q.Trimmed()
	if trimmed.Version == "" {
		return fmt.Errorf("version is required")
	}
	if looksLikeHostPath(trimmed.Version) {
		return fmt.Errorf("version must not contain host-local path material")
	}
	if looksLikeHostPath(trimmed.BuildIdentity) {
		return fmt.Errorf("build_identity must not contain host-local path material")
	}
	return nil
}
