package trustpolicy

import (
	"encoding/json"
	"strings"
)

func evaluateRequiredEvidenceInvariants(input AuditVerificationInput, report *AuditVerificationReportPayload, events []AuditEventPayload) {
	applyVerifierIdentityInvariant(input, report)
	applyRuntimeAttestationInvariant(input.Segment.Header.SegmentID, report, events)
}

func applyVerifierIdentityInvariant(input AuditVerificationInput, report *AuditVerificationReportPayload) {
	if len(input.VerifierRecords) == 0 {
		addDegraded(report, AuditVerificationReasonVerifierIdentityMissingOrUnknown, AuditVerificationDimensionIntegrity, "verification input did not include verifier records", input.Segment.Header.SegmentID, nil)
		if strings.TrimSpace(report.VerifierIdentity) == "" {
			report.VerifierIdentity = "unknown"
		}
		if len(report.TrustRootIdentities) == 0 {
			report.TrustRootIdentities = []string{"unknown"}
		}
		return
	}
	if strings.TrimSpace(report.VerifierIdentity) == "unknown" || trustRootIdentityUnknown(report.TrustRootIdentities) {
		addDegraded(report, AuditVerificationReasonVerifierIdentityMissingOrUnknown, AuditVerificationDimensionIntegrity, "verifier identity or trust-root identity could not be derived from verification inputs", input.Segment.Header.SegmentID, nil)
	}
}

func applyRuntimeAttestationInvariant(segmentID string, report *AuditVerificationReportPayload, events []AuditEventPayload) {
	for i := range events {
		payload := isolateSessionEventPayload(events[i])
		if payload == nil || strings.TrimSpace(payload.provisioningPosture) != "attested" {
			continue
		}
		if strings.TrimSpace(payload.attestationEvidenceDigest) != "" {
			continue
		}
		addDegraded(report, AuditVerificationReasonMissingRuntimeAttestationEvidence, AuditVerificationDimensionIntegrity, "attested runtime event requires attestation evidence digest but none was provided", segmentID, nil)
		break
	}
}

type isolateSessionPayloadSummary struct {
	provisioningPosture       string
	attestationEvidenceDigest string
}

func isolateSessionEventPayload(event AuditEventPayload) *isolateSessionPayloadSummary {
	if event.EventPayloadSchemaID != IsolateSessionStartedPayloadSchemaID && event.EventPayloadSchemaID != IsolateSessionBoundPayloadSchemaID {
		return nil
	}
	parsed := struct {
		ProvisioningPosture       string `json:"provisioning_posture"`
		AttestationEvidenceDigest string `json:"attestation_evidence_digest"`
	}{}
	if err := json.Unmarshal(event.EventPayload, &parsed); err != nil {
		return nil
	}
	return &isolateSessionPayloadSummary{provisioningPosture: strings.TrimSpace(parsed.ProvisioningPosture), attestationEvidenceDigest: strings.TrimSpace(parsed.AttestationEvidenceDigest)}
}

func trustRootIdentityUnknown(identities []string) bool {
	if len(identities) == 0 {
		return true
	}
	for i := range identities {
		if strings.TrimSpace(identities[i]) == "unknown" {
			return true
		}
	}
	return false
}
