package trustpolicy

import "strings"

func evaluateExternalAnchorEvidence(input AuditVerificationInput, report *AuditVerificationReportPayload, sealDigest *Digest) {
	if len(input.ExternalAnchorEvidence) == 0 {
		return
	}
	sidecars := externalAnchorAvailableSidecars(input)
	requiredTargets, hasExplicitTargetSet := requiredExternalAnchorTargetsForVerification(input)
	requiredState := map[string]externalAnchorRequiredTargetState{}
	for targetID := range requiredTargets {
		requiredState[targetID] = externalAnchorRequiredTargetState{}
	}
	for i := range input.ExternalAnchorEvidence {
		evaluateSingleExternalAnchorEvidence(input, report, sealDigest, sidecars, requiredTargets, hasExplicitTargetSet, requiredState, input.ExternalAnchorEvidence[i])
	}
	if report.AnchoringStatus == AuditVerificationStatusFailed {
		return
	}
	if externalAnchorRequiredTargetsPending(requiredState) {
		addDegraded(report, AuditVerificationReasonExternalAnchorDeferredOrUnavailable, AuditVerificationDimensionAnchoring, "one or more required external anchor targets remain deferred, unavailable, or unsatisfied", input.Segment.Header.SegmentID, nil)
	}
}

func externalAnchorAvailableSidecars(input AuditVerificationInput) map[string]struct{} {
	sidecars := map[string]struct{}{}
	for i := range input.ExternalAnchorSidecars {
		id, err := input.ExternalAnchorSidecars[i].Identity()
		if err != nil {
			continue
		}
		sidecars[id] = struct{}{}
	}
	return sidecars
}

func evaluateSingleExternalAnchorEvidence(input AuditVerificationInput, report *AuditVerificationReportPayload, sealDigest *Digest, sidecars map[string]struct{}, requiredTargets map[string]struct{}, hasExplicitTargetSet bool, requiredState map[string]externalAnchorRequiredTargetState, evidence ExternalAnchorEvidencePayload) {
	subject := evidence.AnchoringSubjectDigest
	if !externalAnchorEvidenceApplies(input, report, sealDigest, evidence, subject) {
		return
	}
	targetIdentity, requirement, ok := resolveExternalAnchorEvidenceTarget(input, report, evidence, subject, requiredTargets, hasExplicitTargetSet, requiredState)
	if !ok {
		return
	}
	if !externalAnchorEvidenceSidecarsAvailable(evidence, sidecars) {
		recordExternalAnchorRequiredState(requiredState, targetIdentity, requirement, externalAnchorOutcomeStateInvalid)
		handleExternalAnchorOutcomeInvalid(report, input.Segment.Header.SegmentID, subject, requirement, "external anchor evidence references missing sidecar digest")
		return
	}
	recordExternalAnchorOutcome(report, input.Segment.Header.SegmentID, evidence, subject, targetIdentity, requirement, requiredState)
}

func externalAnchorEvidenceApplies(input AuditVerificationInput, report *AuditVerificationReportPayload, sealDigest *Digest, evidence ExternalAnchorEvidencePayload, subject Digest) bool {
	if err := ValidateExternalAnchorEvidencePayload(evidence); err != nil {
		addHardFailure(report, AuditVerificationReasonExternalAnchorInvalid, AuditVerificationDimensionAnchoring, "external anchor evidence payload invalid: "+err.Error(), input.Segment.Header.SegmentID, &subject)
		return false
	}
	if sealDigest == nil {
		addHardFailure(report, AuditVerificationReasonExternalAnchorInvalid, AuditVerificationDimensionAnchoring, "external anchor evidence cannot be evaluated because segment seal verification failed", input.Segment.Header.SegmentID, &subject)
		return false
	}
	return mustDigestIdentity(evidence.AnchoringSubjectDigest) == mustDigestIdentity(*sealDigest)
}

func resolveExternalAnchorEvidenceTarget(input AuditVerificationInput, report *AuditVerificationReportPayload, evidence ExternalAnchorEvidencePayload, subject Digest, requiredTargets map[string]struct{}, hasExplicitTargetSet bool, requiredState map[string]externalAnchorRequiredTargetState) (string, string, bool) {
	targetIdentity, err := evidence.CanonicalTargetDigest.Identity()
	if err != nil {
		addHardFailure(report, AuditVerificationReasonExternalAnchorInvalid, AuditVerificationDimensionAnchoring, "external anchor evidence target identity invalid: "+err.Error(), input.Segment.Header.SegmentID, &subject)
		return "", "", false
	}
	requirement := resolveExternalAnchorEvidenceRequirement(evidence, requiredTargets, hasExplicitTargetSet)
	recordExternalAnchorRequiredState(requiredState, targetIdentity, requirement, externalAnchorOutcomeStateSeen)
	return targetIdentity, requirement, true
}

type externalAnchorOutcomeState string

const (
	externalAnchorOutcomeStateSeen      externalAnchorOutcomeState = "seen"
	externalAnchorOutcomeStateCompleted externalAnchorOutcomeState = "completed"
	externalAnchorOutcomeStateDeferred  externalAnchorOutcomeState = "deferred"
	externalAnchorOutcomeStateInvalid   externalAnchorOutcomeState = "invalid"
)

func recordExternalAnchorRequiredState(requiredState map[string]externalAnchorRequiredTargetState, targetIdentity, requirement string, outcome externalAnchorOutcomeState) {
	if requirement != ExternalAnchorTargetRequirementRequired {
		return
	}
	state := requiredState[targetIdentity]
	switch outcome {
	case externalAnchorOutcomeStateCompleted:
		state.seenCompleted = true
	case externalAnchorOutcomeStateDeferred:
		state.seenDeferredOrUnavailable = true
	case externalAnchorOutcomeStateInvalid:
		state.seenInvalid = true
	}
	requiredState[targetIdentity] = state
}

func recordExternalAnchorOutcome(report *AuditVerificationReportPayload, segmentID string, evidence ExternalAnchorEvidencePayload, subject Digest, targetIdentity, requirement string, requiredState map[string]externalAnchorRequiredTargetState) {
	details := externalAnchorFindingDetails(evidence, requirement)
	switch evidence.Outcome {
	case ExternalAnchorOutcomeCompleted:
		addFinding(report, AuditVerificationFinding{Code: AuditVerificationReasonExternalAnchorValid, Dimension: AuditVerificationDimensionAnchoring, Severity: AuditVerificationSeverityInfo, Message: "external anchor evidence verified as completed", SegmentID: segmentID, Details: details}, &subject)
		recordExternalAnchorRequiredState(requiredState, targetIdentity, requirement, externalAnchorOutcomeStateCompleted)
	case ExternalAnchorOutcomeDeferred, ExternalAnchorOutcomeUnavailable:
		addFinding(report, AuditVerificationFinding{Code: AuditVerificationReasonExternalAnchorDeferredOrUnavailable, Dimension: AuditVerificationDimensionAnchoring, Severity: AuditVerificationSeverityWarning, Message: "external anchor evidence is deferred or unavailable", SegmentID: segmentID, Details: details}, &subject)
		recordExternalAnchorRequiredState(requiredState, targetIdentity, requirement, externalAnchorOutcomeStateDeferred)
	case ExternalAnchorOutcomeInvalid, ExternalAnchorOutcomeFailed:
		handleExternalAnchorOutcomeInvalid(report, segmentID, subject, requirement, "external anchor evidence is invalid")
		recordExternalAnchorRequiredState(requiredState, targetIdentity, requirement, externalAnchorOutcomeStateInvalid)
	default:
		handleExternalAnchorOutcomeInvalid(report, segmentID, subject, requirement, "external anchor evidence outcome is unsupported")
		recordExternalAnchorRequiredState(requiredState, targetIdentity, requirement, externalAnchorOutcomeStateInvalid)
	}
}

type externalAnchorRequiredTargetState struct {
	seenCompleted             bool
	seenDeferredOrUnavailable bool
	seenInvalid               bool
}

func requiredExternalAnchorTargetsForVerification(input AuditVerificationInput) (map[string]struct{}, bool) {
	targets := map[string]struct{}{}
	if len(input.ExternalAnchorTargetSet) == 0 {
		return targets, false
	}
	for i := range input.ExternalAnchorTargetSet {
		target := input.ExternalAnchorTargetSet[i]
		requirement := NormalizeExternalAnchorTargetRequirement(target.TargetRequirement)
		if requirement != ExternalAnchorTargetRequirementRequired {
			continue
		}
		identity, err := target.TargetDescriptorDigest.Identity()
		if err != nil {
			continue
		}
		targets[identity] = struct{}{}
	}
	return targets, true
}

func resolveExternalAnchorEvidenceRequirement(evidence ExternalAnchorEvidencePayload, requiredTargets map[string]struct{}, hasExplicitTargetSet bool) string {
	identity, err := evidence.CanonicalTargetDigest.Identity()
	if err != nil {
		return ExternalAnchorTargetRequirementRequired
	}
	if hasExplicitTargetSet {
		if _, ok := requiredTargets[identity]; ok {
			return ExternalAnchorTargetRequirementRequired
		}
		return ExternalAnchorTargetRequirementOptional
	}
	return NormalizeExternalAnchorTargetRequirement(evidence.TargetRequirement)
}

func externalAnchorFindingDetails(evidence ExternalAnchorEvidencePayload, requirement string) map[string]any {
	details := map[string]any{
		"target_requirement": NormalizeExternalAnchorTargetRequirement(requirement),
		"target_kind":        strings.TrimSpace(evidence.CanonicalTargetKind),
		"outcome":            strings.TrimSpace(evidence.Outcome),
	}
	if identity, err := evidence.CanonicalTargetDigest.Identity(); err == nil {
		details["target_descriptor_digest"] = identity
	}
	return details
}

func handleExternalAnchorOutcomeInvalid(report *AuditVerificationReportPayload, segmentID string, subject Digest, requirement string, message string) {
	if NormalizeExternalAnchorTargetRequirement(requirement) == ExternalAnchorTargetRequirementOptional {
		addFinding(report, AuditVerificationFinding{Code: AuditVerificationReasonExternalAnchorInvalid, Dimension: AuditVerificationDimensionAnchoring, Severity: AuditVerificationSeverityWarning, Message: message + " (optional supplemental target)", SegmentID: segmentID, Details: map[string]any{"target_requirement": ExternalAnchorTargetRequirementOptional}}, &subject)
		return
	}
	addHardFailure(report, AuditVerificationReasonExternalAnchorInvalid, AuditVerificationDimensionAnchoring, message, segmentID, &subject)
}

func externalAnchorRequiredTargetsPending(state map[string]externalAnchorRequiredTargetState) bool {
	for _, entry := range state {
		if entry.seenInvalid {
			return false
		}
		if !entry.seenCompleted {
			return true
		}
	}
	return false
}

func externalAnchorEvidenceSidecarsAvailable(evidence ExternalAnchorEvidencePayload, available map[string]struct{}) bool {
	for i := range evidence.SidecarRefs {
		id, err := evidence.SidecarRefs[i].Digest.Identity()
		if err != nil {
			return false
		}
		if _, ok := available[id]; !ok {
			return false
		}
	}
	return true
}
