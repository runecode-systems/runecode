package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Service) RecordRuntimeFacts(runID string, facts launcherbackend.RuntimeFactsSnapshot) {
	s.runtimeFactsMu.Lock()
	defer s.runtimeFactsMu.Unlock()
	facts.LaunchReceipt = facts.LaunchReceipt.Normalized()
	facts.HardeningPosture = normalizeRuntimeHardeningPosture(facts.HardeningPosture)
	facts.TerminalReport = normalizeRuntimeTerminalReport(facts.TerminalReport)
	if facts.LaunchReceipt.RunID == "" {
		facts.LaunchReceipt.RunID = runID
	}
	s.runtimeFacts[runID] = facts
}

func (s *Service) RuntimeFacts(runID string) launcherbackend.RuntimeFactsSnapshot {
	s.runtimeFactsMu.RLock()
	facts, ok := s.runtimeFacts[runID]
	s.runtimeFactsMu.RUnlock()
	if ok {
		facts.LaunchReceipt = facts.LaunchReceipt.Normalized()
		facts.HardeningPosture = normalizeRuntimeHardeningPosture(facts.HardeningPosture)
		facts.TerminalReport = normalizeRuntimeTerminalReport(facts.TerminalReport)
		return facts
	}
	return launcherbackend.DefaultRuntimeFacts(runID)
}

func normalizeRuntimeHardeningPosture(input launcherbackend.AppliedHardeningPosture) launcherbackend.AppliedHardeningPosture {
	hardening := input.Normalized()
	if err := hardening.Validate(); err != nil {
		hardening = launcherbackend.DefaultAppliedHardeningPosture()
		hardening.DegradedReasons = append(hardening.DegradedReasons, "hardening_posture_invalid")
		hardening = hardening.Normalized()
	}
	return hardening
}

func normalizeRuntimeTerminalReport(input *launcherbackend.BackendTerminalReport) *launcherbackend.BackendTerminalReport {
	if input == nil {
		return nil
	}
	normalized := input.Normalized()
	if err := normalized.Validate(); err != nil {
		normalized.TerminationKind = launcherbackend.BackendTerminationKindFailed
		normalized.FailClosed = true
		normalized.FallbackPosture = launcherbackend.BackendFallbackPostureNoAutomaticFallback
		normalized.FailureReasonCode = launcherbackend.BackendErrorCodeTerminalReportInvalid
		normalized = normalized.Normalized()
	}
	return &normalized
}

func ParseDataClass(value string) (artifacts.DataClass, error) {
	class := artifacts.DataClass(value)
	switch class {
	case artifacts.DataClassSpecText,
		artifacts.DataClassUnapprovedFileExcerpts,
		artifacts.DataClassApprovedFileExcerpts,
		artifacts.DataClassDiffs,
		artifacts.DataClassBuildLogs,
		artifacts.DataClassAuditEvents,
		artifacts.DataClassAuditVerificationReport,
		artifacts.DataClassWebQuery,
		artifacts.DataClassWebCitations:
		return class, nil
	default:
		return "", fmt.Errorf("unsupported data class %q", value)
	}
}
