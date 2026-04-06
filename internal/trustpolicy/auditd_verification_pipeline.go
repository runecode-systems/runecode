package trustpolicy

import (
	"fmt"
	"time"
)

func VerifyAuditEvidence(input AuditVerificationInput) (AuditVerificationReportPayload, error) {
	report := initializeAuditVerificationReport(input)

	registry, err := NewVerifierRegistry(input.VerifierRecords)
	if err != nil {
		addHardFailure(&report, AuditVerificationReasonDetachedSignatureInvalid, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
		return finalizeAuditVerificationReport(report), fmt.Errorf("verifier registry: %w", err)
	}
	if err := validateAuditEventContractCatalog(input.EventContractCatalog); err != nil {
		addHardFailure(&report, AuditVerificationReasonEventContractMismatch, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
		return finalizeAuditVerificationReport(report), fmt.Errorf("event contract catalog: %w", err)
	}

	if err := validateAuditSegmentLifecycle(input.Segment); err != nil {
		addHardFailure(&report, AuditVerificationReasonSegmentLifecycleInconsistent, AuditVerificationDimensionSegmentLifecycle, err.Error(), input.Segment.Header.SegmentID, nil)
	}

	if len(input.RawFramedSegmentBytes) == 0 {
		addHardFailure(&report, AuditVerificationReasonSegmentFileHashMismatch, AuditVerificationDimensionIntegrity, "raw framed segment bytes are required", input.Segment.Header.SegmentID, nil)
	}

	frameDigests, frameEnvelopes, eventTimes := verifySegmentFramesAndEvents(input, registry, &report)

	sealDigest, sealPayload, sealErr := verifySegmentSeal(input, registry, frameDigests, &report)
	if sealErr != nil {
		_ = sealDigest
	}

	verifyReceipts(input, registry, sealDigest, sealPayload, &report)

	evaluateStoragePostureEvidence(input, &report)
	setDerivedVerificationPosture(&report, frameEnvelopes, eventTimes)

	return finalizeAuditVerificationReport(report), nil
}

func initializeAuditVerificationReport(input AuditVerificationInput) AuditVerificationReportPayload {
	report := AuditVerificationReportPayload{
		SchemaID:               AuditVerificationReportSchemaID,
		SchemaVersion:          AuditVerificationReportSchemaVersion,
		VerificationScope:      input.Scope,
		IntegrityStatus:        AuditVerificationStatusOK,
		AnchoringStatus:        AuditVerificationStatusOK,
		StoragePostureStatus:   AuditVerificationStatusOK,
		SegmentLifecycleStatus: AuditVerificationStatusOK,
	}
	verifiedAt := input.Now.UTC()
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	report.VerifiedAt = verifiedAt.Format(time.RFC3339)
	if report.VerificationScope.ScopeKind == "" {
		report.VerificationScope = AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: input.Segment.Header.SegmentID}
	}
	return report
}

func evaluateStoragePostureEvidence(input AuditVerificationInput, report *AuditVerificationReportPayload) {
	if input.StoragePostureEvidence == nil {
		return
	}
	if err := ValidateAuditStoragePostureEvidence(*input.StoragePostureEvidence); err != nil {
		addHardFailure(report, AuditVerificationReasonStoragePostureInvalid, AuditVerificationDimensionStoragePosture, err.Error(), input.Segment.Header.SegmentID, nil)
		return
	}
	if input.StoragePostureEvidence.DevPlaintextOverrideActive {
		addDegraded(report, AuditVerificationReasonStoragePostureDegraded, AuditVerificationDimensionStoragePosture, "storage posture is degraded by explicit dev plaintext override", input.Segment.Header.SegmentID, nil)
	}
}

func setDerivedVerificationPosture(report *AuditVerificationReportPayload, frameEnvelopes []SignedObjectEnvelope, eventTimes []time.Time) {
	report.CryptographicallyValid = deriveCryptographicValidity(report.HardFailures)
	report.HistoricallyAdmissible = len(report.HardFailures) == 0 && !hasFindingWithCode(report.Findings, AuditVerificationReasonSignerHistoricallyInadmissible)
	report.CurrentlyDegraded = len(report.DegradedReasons) > 0
	if len(eventTimes) == 0 && len(frameEnvelopes) > 0 {
		report.HistoricallyAdmissible = false
	}
}

func validateAuditSegmentLifecycle(segment AuditSegmentFilePayload) error {
	if err := validateAuditSegmentLifecycleIdentity(segment); err != nil {
		return err
	}
	if err := validateAuditSegmentLifecycleStateConsistency(segment); err != nil {
		return err
	}
	if len(segment.Frames) == 0 {
		return fmt.Errorf("segment must contain at least one frame")
	}
	return nil
}

func validateAuditSegmentLifecycleIdentity(segment AuditSegmentFilePayload) error {
	if segment.SchemaID != "runecode.protocol.v0.AuditSegmentFile" {
		return fmt.Errorf("unexpected segment schema_id %q", segment.SchemaID)
	}
	if segment.SchemaVersion != "0.1.0" {
		return fmt.Errorf("unexpected segment schema_version %q", segment.SchemaVersion)
	}
	if segment.Header.Format != "audit_segment_framed_v1" {
		return fmt.Errorf("unsupported segment header format %q", segment.Header.Format)
	}
	if segment.Header.SegmentID == "" {
		return fmt.Errorf("segment header segment_id is required")
	}
	return nil
}

func validateAuditSegmentLifecycleStateConsistency(segment AuditSegmentFilePayload) error {
	if segment.Header.SegmentState != segment.LifecycleMarker.State {
		return fmt.Errorf("segment header state %q disagrees with lifecycle marker state %q", segment.Header.SegmentState, segment.LifecycleMarker.State)
	}
	if segment.TrailingPartialFrameBytes <= 0 {
		return nil
	}
	if segment.Header.SegmentState == "sealed" || segment.Header.SegmentState == "quarantined" {
		return fmt.Errorf("sealed or quarantined segments cannot have trailing partial frame bytes")
	}
	if segment.Header.SegmentState == "anchored" || segment.Header.SegmentState == "imported" {
		return fmt.Errorf("anchored or imported segments cannot have trailing partial frame bytes")
	}
	return nil
}
