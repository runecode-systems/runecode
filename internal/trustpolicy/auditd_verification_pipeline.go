package trustpolicy

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func VerifyAuditEvidence(input AuditVerificationInput) (AuditVerificationReportPayload, error) {
	report := initializeAuditVerificationReport(input)
	registry, err := initializeAuditVerificationRegistry(input, &report)
	if err != nil {
		return finalizeAuditVerificationReport(report), err
	}

	if err := validateAuditSegmentLifecycle(input.Segment); err != nil {
		addHardFailure(&report, AuditVerificationReasonSegmentLifecycleInconsistent, AuditVerificationDimensionSegmentLifecycle, err.Error(), input.Segment.Header.SegmentID, nil)
	}

	if len(input.RawFramedSegmentBytes) == 0 {
		addHardFailure(&report, AuditVerificationReasonSegmentFileHashMismatch, AuditVerificationDimensionIntegrity, "raw framed segment bytes are required", input.Segment.Header.SegmentID, nil)
	}

	_, frameEnvelopes, eventTimes, verifiedEvents, sealDigest, sealPayload, _ := verificationFrameAndSealState(input, registry, &report)

	verifyReceipts(input, registry, sealDigest, sealPayload, &report)

	evaluateStoragePostureEvidence(input, &report)
	evaluateRequiredEvidenceInvariants(input, &report, verifiedEvents)
	setDerivedVerificationPosture(&report, frameEnvelopes, eventTimes)

	return finalizeAuditVerificationReport(report), nil
}

func verificationFrameAndSealState(input AuditVerificationInput, registry *VerifierRegistry, report *AuditVerificationReportPayload) ([]Digest, []SignedObjectEnvelope, []time.Time, []AuditEventPayload, *Digest, AuditSegmentSealPayload, error) {
	frameDigests, frameEnvelopes, eventTimes, verifiedEvents := []Digest{}, []SignedObjectEnvelope{}, []time.Time{}, []AuditEventPayload{}
	sealDigest, sealPayload, sealErr := input.PreverifiedSealDigest, AuditSegmentSealPayload{}, error(nil)
	if input.SkipFrameAndSealReplay && input.PreverifiedSealDigest != nil {
		sealPayload = preverifiedSealPayloadOrZero(input)
		verifiedEvents = append([]AuditEventPayload{}, input.PreverifiedEvents...)
		return frameDigests, frameEnvelopes, eventTimes, verifiedEvents, validateTrustedPreverifiedSeal(input, report), sealPayload, sealErr
	}
	frameDigests, frameEnvelopes, eventTimes, verifiedEvents = verifySegmentFramesAndEvents(input, registry, report)
	sealDigest, sealPayload, sealErr = verifySegmentSeal(input, registry, frameDigests, report)
	if sealErr != nil {
		_ = sealDigest
	}
	return frameDigests, frameEnvelopes, eventTimes, verifiedEvents, sealDigest, sealPayload, sealErr
}

func preverifiedSealPayloadOrZero(input AuditVerificationInput) AuditSegmentSealPayload {
	if input.PreverifiedSealPayload == nil {
		return AuditSegmentSealPayload{}
	}
	return *input.PreverifiedSealPayload
}

func validateTrustedPreverifiedSeal(input AuditVerificationInput, report *AuditVerificationReportPayload) *Digest {
	if !input.TrustedPreverifiedSeal {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, "preverified seal replay bypass requires trusted input", input.Segment.Header.SegmentID, nil)
		return nil
	}
	if _, err := input.PreverifiedSealDigest.Identity(); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("preverified seal digest invalid: %v", err), input.Segment.Header.SegmentID, nil)
		return nil
	}
	return input.PreverifiedSealDigest
}

func initializeAuditVerificationRegistry(input AuditVerificationInput, report *AuditVerificationReportPayload) (*VerifierRegistry, error) {
	registry, err := NewVerifierRegistry(input.VerifierRecords)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonDetachedSignatureInvalid, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
		return nil, fmt.Errorf("verifier registry: %w", err)
	}
	if err := validateAuditEventContractCatalog(input.EventContractCatalog); err != nil {
		addHardFailure(report, AuditVerificationReasonEventContractMismatch, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
		return nil, fmt.Errorf("event contract catalog: %w", err)
	}
	return registry, nil
}

func initializeAuditVerificationReport(input AuditVerificationInput) AuditVerificationReportPayload {
	report := AuditVerificationReportPayload{
		SchemaID:               AuditVerificationReportSchemaID,
		SchemaVersion:          AuditVerificationReportSchemaVersion,
		VerificationScope:      input.Scope,
		IntegrityStatus:        AuditVerificationStatusOK,
		AnchoringStatus:        AuditVerificationStatusOK,
		AnchoringPosture:       AuditVerificationAnchoringPostureLocalAnchorReceiptOnly,
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
	report.VerifierIdentity, report.TrustRootIdentities = verificationIdentityFootprint(input.VerifierRecords)
	return report
}

func verificationIdentityFootprint(records []VerifierRecord) (string, []string) {
	if len(records) == 0 {
		return "unknown", []string{"unknown"}
	}
	selected := canonicalVerificationIdentityRecord(records)
	verifierID, err := canonicalVerifierIdentityFromRecord(selected)
	if err != nil {
		verifierID = "unknown"
	}
	roots := map[string]struct{}{}
	for i := range records {
		identity, identityErr := trustRootDigestIdentityFromRecord(records[i])
		if identityErr == nil && identity != "" {
			roots[identity] = struct{}{}
		}
	}
	out := make([]string, 0, len(roots))
	for identity := range roots {
		out = append(out, identity)
	}
	sort.Strings(out)
	if len(out) == 0 {
		out = []string{"unknown"}
	}
	return verifierID, out
}

func canonicalVerificationIdentityRecord(records []VerifierRecord) VerifierRecord {
	selected := records[0]
	for i := 1; i < len(records); i++ {
		if strings.TrimSpace(records[i].KeyIDValue) < strings.TrimSpace(selected.KeyIDValue) {
			selected = records[i]
		}
	}
	return selected
}

func canonicalVerifierIdentityFromRecord(record VerifierRecord) (string, error) {
	decodedPublicKey, err := record.PublicKey.DecodedBytes()
	if err != nil {
		return "", err
	}
	return canonicalVerifierIdentity(record.KeyIDValue, decodedPublicKey)
}

func trustRootDigestIdentityFromRecord(record VerifierRecord) (string, error) {
	digest := Digest{HashAlg: "sha256", Hash: record.KeyIDValue}
	return digest.Identity()
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
