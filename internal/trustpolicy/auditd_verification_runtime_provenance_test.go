package trustpolicy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVerifyAuditEvidenceProviderInvocationReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_authorized")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceProviderInvocationDeniedReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_denied")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceSecretLeaseReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeSecretLeaseReceiptEnvelope(t, fixture.sealEnvelopeDigest, "secret_lease_issued")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceRuntimeSummaryReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest, 0, 0)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceDegradedPostureSummaryReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.degradedPostureSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceDegradedPostureSummaryMissingApprovalConsumptionLinkFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.degradedPostureSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		delete(receiptPayload, "approval_consumption_link")
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceNegativeCapabilitySummaryReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.negativeCapabilitySummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceNegativeCapabilitySummaryInvalidSupportFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.negativeCapabilitySummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		receiptPayload["approval_consumption_evidence_support"] = "unknown"
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFlagsMissingNegativeCapabilitySummaryWhenRuntimeSummaryPresent(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest, 0, 0)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonNegativeCapabilitySummaryMissing) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonNegativeCapabilitySummaryMissing)
	}
}

func TestVerifyAuditEvidenceFlagsMissingApprovalEvidenceForBoundaryAuthorization(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_authorized")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonMissingRequiredApprovalEvidence) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonMissingRequiredApprovalEvidence)
	}
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if report.CryptographicallyValid {
		t.Fatal("cryptographically_valid=true, want false when required approval evidence is missing as a hard failure")
	}
}

func TestVerifyAuditEvidenceApprovalResolutionAndConsumptionReceiptsPass(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	resolution := fixture.approvalEvidenceReceiptEnvelope(t, fixture.sealEnvelopeDigest, auditReceiptKindApprovalResolution)
	consumption := fixture.approvalEvidenceReceiptEnvelope(t, fixture.sealEnvelopeDigest, auditReceiptKindApprovalConsumption)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{resolution, consumption})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceArtifactPublishedAndOverrideReceiptsPass(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	publication := fixture.publicationEvidenceReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	override := fixture.overrideEvidenceReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{publication, override})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceNegativeCapabilityLimitedSupportDegrades(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.negativeCapabilitySummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonNegativeCapabilitySupportLimitedOrUnknown) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonNegativeCapabilitySupportLimitedOrUnknown)
	}
}

func TestVerifyAuditEvidenceFlagsEvidenceExportIncompleteWhenMetaAuditPresentWithoutExport(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.metaAuditReceiptEnvelope(t, fixture.sealEnvelopeDigest, "trust_root_updated")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonEvidenceExportIncomplete) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonEvidenceExportIncomplete)
	}
}

func TestVerifyAuditEvidenceMetaAuditActionKindMismatchFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.metaAuditReceiptEnvelope(t, fixture.sealEnvelopeDigest, auditReceiptKindTrustRootUpdated)
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		receiptPayload["action_code"] = auditReceiptKindVerifierConfigurationChanged
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFlagsMissingRuntimeAttestationForAttestedPayload(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	frameEnvelope := SignedObjectEnvelope{}
	if err := json.Unmarshal([]byte(base64DecodeFixture(t, fixture.segment.Frames[0].CanonicalSignedEnvelopeBytes)), &frameEnvelope); err != nil {
		t.Fatalf("Unmarshal frame envelope returned error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(frameEnvelope.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal event payload returned error: %v", err)
	}
	eventPayload, _ := payload["event_payload"].(map[string]any)
	eventPayload["provisioning_posture"] = "attested"
	delete(eventPayload, "attestation_evidence_digest")
	updatedEventPayloadHash := hashCanonicalJSONFixture(t, eventPayload)
	payload["event_payload"] = eventPayload
	payload["event_payload_hash"] = map[string]any{"hash_alg": "sha256", "hash": updatedEventPayloadHash}
	frameEnvelope.Payload = marshalJSONFixture(t, payload)
	frameEnvelope = resignEnvelopeFixture(t, fixture.privateKey, frameEnvelope)
	canonicalEnvelope := canonicalEnvelopeBytesFixture(t, frameEnvelope)
	updatedDigest := digestForBytesFixture(canonicalEnvelope)
	fixture.segment.Frames[0].CanonicalSignedEnvelopeBytes = base64EncodeFixture(canonicalEnvelope)
	fixture.segment.Frames[0].ByteLength = int64(len(canonicalEnvelope))
	fixture.segment.Frames[0].RecordDigest = updatedDigest
	fixture.sealEnvelope = fixture.resealForUpdatedFrame(t, updatedDigest)
	sealDigest, err := ComputeSignedEnvelopeAuditRecordDigest(fixture.sealEnvelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	fixture.sealEnvelopeDigest = sealDigest
	report := mustVerifyAuditEvidenceReport(t, fixture, nil)
	if !report.CurrentlyDegraded {
		t.Fatal("currently_degraded=false, want true when attestation evidence is missing")
	}
	if report.IntegrityStatus != AuditVerificationStatusDegraded {
		t.Fatalf("integrity_status=%q, want degraded", report.IntegrityStatus)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonMissingRuntimeAttestationEvidence) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonMissingRuntimeAttestationEvidence)
	}
}

func TestVerifyAuditEvidenceProviderInvocationNetworkDigestMismatchFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_authorized")
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		receiptPayload["network_target_digest"] = map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}
