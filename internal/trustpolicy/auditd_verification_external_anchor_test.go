package trustpolicy

import (
	"strings"
	"testing"
	"time"
)

func TestVerifyAuditEvidenceExternalAnchorCompletedAddsValidFinding(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	provider := Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}
	transcript := Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}, ExternalAnchorOutcomeCompleted, "", []ExternalAnchorEvidenceSidecarRef{
		{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof},
		{EvidenceKind: ExternalAnchorSidecarKindProviderReceipt, Digest: provider},
		{EvidenceKind: ExternalAnchorSidecarKindVerifyTranscript, Digest: transcript},
	})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                  AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                fixture.segment,
		RawFramedSegmentBytes:  fixture.rawSegmentBytes,
		SegmentSealEnvelope:    fixture.sealEnvelope,
		ReceiptEnvelopes:       []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:        fixture.verifierRecords,
		EventContractCatalog:   fixture.eventContractCatalog,
		SignerEvidence:         fixture.signerEvidence,
		ExternalAnchorEvidence: []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars: []Digest{proof, provider, transcript},
		Now:                    time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status=%q, want ok", report.AnchoringStatus)
	}
	if !hasFindingWithCode(report.Findings, AuditVerificationReasonExternalAnchorValid) {
		t.Fatalf("findings missing %q: %+v", AuditVerificationReasonExternalAnchorValid, report.Findings)
	}
	if report.AnchoringPosture != AuditVerificationAnchoringPostureExternalAnchorValidated {
		t.Fatalf("anchoring_posture=%q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureExternalAnchorValidated)
	}
}

func TestVerifyAuditEvidenceExternalAnchorDeferredDegrades(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}, ExternalAnchorOutcomeDeferred, "", []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                  AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                fixture.segment,
		RawFramedSegmentBytes:  fixture.rawSegmentBytes,
		SegmentSealEnvelope:    fixture.sealEnvelope,
		ReceiptEnvelopes:       []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:        fixture.verifierRecords,
		EventContractCatalog:   fixture.eventContractCatalog,
		SignerEvidence:         fixture.signerEvidence,
		ExternalAnchorEvidence: []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars: []Digest{proof},
		Now:                    time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusDegraded {
		t.Fatalf("anchoring_status=%q, want degraded", report.AnchoringStatus)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable) {
		t.Fatalf("degraded_reasons=%v, want %q", report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable)
	}
	if report.AnchoringPosture != AuditVerificationAnchoringPostureExternalAnchorDeferredOrUnknown {
		t.Fatalf("anchoring_posture=%q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureExternalAnchorDeferredOrUnknown)
	}
}

func TestVerifyAuditEvidenceExternalAnchorInvalidFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}, ExternalAnchorOutcomeInvalid, "", []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                  AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                fixture.segment,
		RawFramedSegmentBytes:  fixture.rawSegmentBytes,
		SegmentSealEnvelope:    fixture.sealEnvelope,
		ReceiptEnvelopes:       []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:        fixture.verifierRecords,
		EventContractCatalog:   fixture.eventContractCatalog,
		SignerEvidence:         fixture.signerEvidence,
		ExternalAnchorEvidence: []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars: []Digest{proof},
		Now:                    time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status=%q, want failed", report.AnchoringStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonExternalAnchorInvalid) {
		t.Fatalf("hard_failures=%v, want %q", report.HardFailures, AuditVerificationReasonExternalAnchorInvalid)
	}
	if report.AnchoringPosture != AuditVerificationAnchoringPostureExternalAnchorInvalid {
		t.Fatalf("anchoring_posture=%q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureExternalAnchorInvalid)
	}
}

func TestVerifyAuditEvidenceRequiredTargetSetSatisfiedIsOK(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	requiredDigest := Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, requiredDigest, ExternalAnchorOutcomeCompleted, ExternalAnchorTargetRequirementRequired, []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                   AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                 fixture.segment,
		RawFramedSegmentBytes:   fixture.rawSegmentBytes,
		SegmentSealEnvelope:     fixture.sealEnvelope,
		ReceiptEnvelopes:        []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:         fixture.verifierRecords,
		EventContractCatalog:    fixture.eventContractCatalog,
		SignerEvidence:          fixture.signerEvidence,
		ExternalAnchorTargetSet: []ExternalAnchorVerificationTarget{{TargetKind: "transparency_log", TargetDescriptorDigest: requiredDigest, TargetRequirement: ExternalAnchorTargetRequirementRequired}},
		ExternalAnchorEvidence:  []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars:  []Digest{proof},
		Now:                     time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status=%q, want ok", report.AnchoringStatus)
	}
}

func TestVerifyAuditEvidenceRequiredTargetSetDeferredIsDegraded(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	requiredDigest := Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, requiredDigest, ExternalAnchorOutcomeDeferred, ExternalAnchorTargetRequirementRequired, []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                   AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                 fixture.segment,
		RawFramedSegmentBytes:   fixture.rawSegmentBytes,
		SegmentSealEnvelope:     fixture.sealEnvelope,
		ReceiptEnvelopes:        []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:         fixture.verifierRecords,
		EventContractCatalog:    fixture.eventContractCatalog,
		SignerEvidence:          fixture.signerEvidence,
		ExternalAnchorTargetSet: []ExternalAnchorVerificationTarget{{TargetKind: "transparency_log", TargetDescriptorDigest: requiredDigest, TargetRequirement: ExternalAnchorTargetRequirementRequired}},
		ExternalAnchorEvidence:  []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars:  []Digest{proof},
		Now:                     time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusDegraded {
		t.Fatalf("anchoring_status=%q, want degraded", report.AnchoringStatus)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable) {
		t.Fatalf("degraded_reasons=%v, want %q", report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable)
	}
}

func TestVerifyAuditEvidenceOptionalTargetInvalidDoesNotBlockRequiredOK(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	requiredDigest := Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}
	optionalDigest := Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)}
	requiredProof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	optionalProof := Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}
	requiredEvidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, requiredDigest, ExternalAnchorOutcomeCompleted, ExternalAnchorTargetRequirementRequired, []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: requiredProof}})
	optionalEvidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, optionalDigest, ExternalAnchorOutcomeInvalid, ExternalAnchorTargetRequirementOptional, []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: optionalProof}})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                   AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                 fixture.segment,
		RawFramedSegmentBytes:   fixture.rawSegmentBytes,
		SegmentSealEnvelope:     fixture.sealEnvelope,
		ReceiptEnvelopes:        []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:         fixture.verifierRecords,
		EventContractCatalog:    fixture.eventContractCatalog,
		SignerEvidence:          fixture.signerEvidence,
		ExternalAnchorTargetSet: []ExternalAnchorVerificationTarget{{TargetKind: "transparency_log", TargetDescriptorDigest: requiredDigest, TargetRequirement: ExternalAnchorTargetRequirementRequired}, {TargetKind: "transparency_log", TargetDescriptorDigest: optionalDigest, TargetRequirement: ExternalAnchorTargetRequirementOptional}},
		ExternalAnchorEvidence:  []ExternalAnchorEvidencePayload{requiredEvidence, optionalEvidence},
		ExternalAnchorSidecars:  []Digest{requiredProof, optionalProof},
		Now:                     time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status=%q, want ok", report.AnchoringStatus)
	}
	if containsReasonCode(report.HardFailures, AuditVerificationReasonExternalAnchorInvalid) {
		t.Fatalf("hard_failures=%v, want no %q for optional target invalid evidence", report.HardFailures, AuditVerificationReasonExternalAnchorInvalid)
	}
	if !hasFindingWithCode(report.Findings, AuditVerificationReasonExternalAnchorInvalid) {
		t.Fatalf("findings missing %q for optional target invalid evidence", AuditVerificationReasonExternalAnchorInvalid)
	}
}

func TestVerifyAuditEvidenceRequiredTargetInvalidFails(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	anchorReceipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	requiredDigest := Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	requiredEvidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, requiredDigest, ExternalAnchorOutcomeInvalid, ExternalAnchorTargetRequirementRequired, []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                   AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:                 fixture.segment,
		RawFramedSegmentBytes:   fixture.rawSegmentBytes,
		SegmentSealEnvelope:     fixture.sealEnvelope,
		ReceiptEnvelopes:        []SignedObjectEnvelope{anchorReceipt},
		VerifierRecords:         fixture.verifierRecords,
		EventContractCatalog:    fixture.eventContractCatalog,
		SignerEvidence:          fixture.signerEvidence,
		ExternalAnchorTargetSet: []ExternalAnchorVerificationTarget{{TargetKind: "transparency_log", TargetDescriptorDigest: requiredDigest, TargetRequirement: ExternalAnchorTargetRequirementRequired}},
		ExternalAnchorEvidence:  []ExternalAnchorEvidencePayload{requiredEvidence},
		ExternalAnchorSidecars:  []Digest{proof},
		Now:                     time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status=%q, want failed", report.AnchoringStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonExternalAnchorInvalid) {
		t.Fatalf("hard_failures=%v, want %q", report.HardFailures, AuditVerificationReasonExternalAnchorInvalid)
	}
}

func TestEvaluateExternalAnchorEvidencePreservesMoreSevereExistingPosture(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}, ExternalAnchorOutcomeCompleted, "", []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report := &AuditVerificationReportPayload{
		AnchoringStatus:  AuditVerificationStatusDegraded,
		AnchoringPosture: AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound,
	}

	evaluateExternalAnchorEvidence(AuditVerificationInput{
		Segment:                fixture.segment,
		ExternalAnchorEvidence: []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars: []Digest{proof},
	}, report, &fixture.sealEnvelopeDigest)

	if report.AnchoringPosture != AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound {
		t.Fatalf("anchoring_posture=%q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound)
	}
	if !hasFindingWithCode(report.Findings, AuditVerificationReasonExternalAnchorValid) {
		t.Fatalf("findings missing %q: %+v", AuditVerificationReasonExternalAnchorValid, report.Findings)
	}
}

func TestEvaluateExternalAnchorEvidenceDefaultsLocalOnlyWhenUnsetAndNoEvidence(t *testing.T) {
	report := &AuditVerificationReportPayload{}
	evaluateExternalAnchorEvidence(AuditVerificationInput{}, report, nil)
	if report.AnchoringPosture != AuditVerificationAnchoringPostureLocalAnchorReceiptOnly {
		t.Fatalf("anchoring_posture=%q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureLocalAnchorReceiptOnly)
	}
}

func TestEvaluateExternalAnchorEvidenceDeferredPreservesMoreSevereExistingPosture(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{})
	proof := Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	evidence := externalAnchorEvidenceFixture(fixture.sealEnvelopeDigest, Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}, ExternalAnchorOutcomeDeferred, ExternalAnchorTargetRequirementRequired, []ExternalAnchorEvidenceSidecarRef{{EvidenceKind: ExternalAnchorSidecarKindProofBytes, Digest: proof}})
	report := &AuditVerificationReportPayload{
		AnchoringStatus:  AuditVerificationStatusDegraded,
		AnchoringPosture: AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound,
	}

	evaluateExternalAnchorEvidence(AuditVerificationInput{
		Segment:                fixture.segment,
		ExternalAnchorEvidence: []ExternalAnchorEvidencePayload{evidence},
		ExternalAnchorSidecars: []Digest{proof},
	}, report, &fixture.sealEnvelopeDigest)

	if report.AnchoringPosture != AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound {
		t.Fatalf("anchoring_posture=%q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable) {
		t.Fatalf("degraded_reasons=%v, want %q", report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable)
	}
}

func externalAnchorEvidenceFixture(subjectDigest Digest, targetDigest Digest, outcome string, requirement string, sidecars []ExternalAnchorEvidenceSidecarRef) ExternalAnchorEvidencePayload {
	return ExternalAnchorEvidencePayload{
		SchemaID:                ExternalAnchorEvidenceSchemaID,
		SchemaVersion:           ExternalAnchorEvidenceSchemaVersion,
		RecordedAt:              "2026-03-13T12:30:00Z",
		RunID:                   "run-1",
		PreparedMutationID:      "sha256:" + strings.Repeat("4", 64),
		ExecutionAttemptID:      "sha256:" + strings.Repeat("5", 64),
		CanonicalTargetKind:     "transparency_log",
		CanonicalTargetDigest:   targetDigest,
		CanonicalTargetIdentity: mustExternalAnchorTestDigestIdentity(targetDigest),
		TargetRequirement:       requirement,
		AnchoringSubjectFamily:  AuditSegmentAnchoringSubjectSeal,
		AnchoringSubjectDigest:  subjectDigest,
		OutboundPayloadDigest:   cloneDigestForExternalAnchorTest(Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)}),
		OutboundBytes:           128,
		StartedAt:               "2026-03-13T12:20:00Z",
		CompletedAt:             "2026-03-13T12:29:00Z",
		Outcome:                 outcome,
		OutcomeReasonCode:       "external_anchor_execution_completed",
		ProofSchemaID:           "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
		ProofKind:               "transparency_log_receipt_v0",
		SidecarRefs:             sidecars,
	}
}

func mustExternalAnchorTestDigestIdentity(d Digest) string {
	identity, err := d.Identity()
	if err != nil {
		panic(err)
	}
	return identity
}

func cloneDigestForExternalAnchorTest(d Digest) *Digest {
	v := d
	return &v
}
