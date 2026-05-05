package brokerapi

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func runtimeAuditDetailsForPayload(eventType, payloadSchemaID string, payload any, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot, runtimeSupportState map[string]any) (map[string]interface{}, error) {
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	attestationPosture := runtimeAttestationPosture(evidence)
	attestationReasonCodes := runtimeAttestationReasonCodesForPosture(attestationPosture, evidence)
	details := map[string]interface{}{
		"audit_event_type":            eventType,
		"event_payload_schema_id":     payloadSchemaID,
		"event_payload":               json.RawMessage(payloadRaw),
		"operation_id":                buildRuntimeEventOperationID(eventType, evidence),
		"run_id":                      evidence.Launch.RunID,
		"evidence_digest_refs":        runtimeEvidenceDigestRefs(evidence),
		"stored_runtime_fact_digests": runtimeStoredDigestMap(evidence),
		"provisioning_posture":        evidence.Launch.ProvisioningPosture,
		"attestation_posture":         attestationPosture,
		"attestation_verifier_class":  runtimeAttestationVerifierClass(evidence),
	}
	if len(attestationReasonCodes) > 0 {
		details["attestation_reason_codes"] = attestationReasonCodes
	}
	mergeRuntimeSupportAuditDetails(details, runtimeSupportState)
	if sessionID := strings.TrimSpace(evidence.Launch.SessionID); sessionID != "" {
		details["session_id"] = sessionID
	}
	if stageID := strings.TrimSpace(facts.LaunchReceipt.StageID); stageID != "" {
		details["stage_id"] = stageID
	}
	return details, nil
}

func buildRuntimeEventOperationID(eventType string, evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	base := runtimeAuditOperationBase(eventType, evidence)
	if eventType == "isolate_session_started" {
		return "runtime-start:" + base
	}
	if eventType == "runtime_launch_admission" {
		return "runtime-launch-admission:" + base
	}
	if eventType == "runtime_launch_denied" {
		return "runtime-launch-denied:" + base
	}
	return "runtime-bind:" + base
}

func runtimeAuditOperationBase(eventType string, evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	if eventType == "runtime_launch_admission" || eventType == "runtime_launch_denied" {
		return evidence.Launch.EvidenceDigest
	}
	return runtimeSessionAuditIdentityKey(evidence)
}

func runtimeSessionAuditIdentityKey(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	parts := []string{
		evidence.Launch.EvidenceDigest,
		evidence.Hardening.EvidenceDigest,
		runtimeSessionEvidenceDigest(evidence),
		runtimeAttestationEvidenceDigest(evidence),
		runtimeAttestationVerificationDigest(evidence),
		runtimeAttestationPosture(evidence),
	}
	return strings.Join(parts, ":")
}

func runtimeAttestationPosture(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	posture, _ := launcherbackend.DeriveAttestationPostureFromEvidence(evidence)
	return posture
}

func runtimeAttestationReasonCodes(evidence launcherbackend.RuntimeEvidenceSnapshot) []string {
	_, reasons := launcherbackend.DeriveAttestationPostureFromEvidence(evidence)
	return append([]string{}, reasons...)
}

func runtimeAttestationReasonCodesForPosture(posture string, evidence launcherbackend.RuntimeEvidenceSnapshot) []string {
	if posture != launcherbackend.AttestationPostureInvalid {
		return nil
	}
	return runtimeAttestationReasonCodes(evidence)
}

func runtimeAttestationVerifierClass(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	return launcherbackend.DeriveAttestationVerifierClassFromEvidence(evidence)
}

func runtimeAuditSupportState(evidence launcherbackend.RuntimeEvidenceSnapshot, instanceID, runID string, policyRefs []string, approvals []ApprovalSummary) map[string]any {
	state := map[string]any{
		"backend_kind":               evidence.Launch.BackendKind,
		"runtime_posture_degraded":   runtimePostureDegraded(evidence.Launch.BackendKind, evidence.Launch.IsolationAssuranceLevel),
		"attestation_posture":        runtimeAttestationPosture(evidence),
		"attestation_verifier_class": runtimeAttestationVerifierClass(evidence),
	}
	projectBackendPostureSelectionEvidenceState(state, instanceID, runID, policyRefs, approvals)
	projectSupportedRuntimeRequirementsState(state)
	return state
}

func mergeRuntimeSupportAuditDetails(details map[string]interface{}, runtimeSupportState map[string]any) {
	if len(runtimeSupportState) == 0 {
		return
	}
	if supported, ok := runtimeSupportState["supported_runtime_requirements_satisfied"].(bool); ok {
		details["supported_runtime_requirements_satisfied"] = supported
	}
	if reasons, ok := runtimeSupportState["supported_runtime_requirement_reason_codes"].([]string); ok && len(reasons) > 0 {
		details["supported_runtime_requirement_reason_codes"] = append([]string{}, reasons...)
	}
}

func runtimeEvidenceDigestRefs(evidence launcherbackend.RuntimeEvidenceSnapshot) []map[string]string {
	refs := []map[string]string{
		{"kind": "launch_receipt", "digest": evidence.Launch.EvidenceDigest},
		{"kind": "applied_hardening_posture", "digest": evidence.Hardening.EvidenceDigest},
	}
	if session := runtimeSessionEvidenceDigest(evidence); session != "" {
		refs = append(refs, map[string]string{"kind": "session_binding", "digest": session})
	}
	if terminal := runtimeTerminalEvidenceDigest(evidence); terminal != "" {
		refs = append(refs, map[string]string{"kind": "terminal_report", "digest": terminal})
	}
	if attestation := runtimeAttestationEvidenceDigest(evidence); attestation != "" {
		refs = append(refs, map[string]string{"kind": "attestation_evidence", "digest": attestation})
	}
	if verification := runtimeAttestationVerificationDigest(evidence); verification != "" {
		refs = append(refs, map[string]string{"kind": "attestation_verification", "digest": verification})
	}
	return refs
}

func runtimeStoredDigestMap(evidence launcherbackend.RuntimeEvidenceSnapshot) map[string]string {
	return map[string]string{
		"launch_receipt":           evidence.Launch.EvidenceDigest,
		"hardening_posture":        evidence.Hardening.EvidenceDigest,
		"session_binding":          runtimeSessionEvidenceDigest(evidence),
		"terminal_report":          runtimeTerminalEvidenceDigest(evidence),
		"attestation_evidence":     runtimeAttestationEvidenceDigest(evidence),
		"attestation_verification": runtimeAttestationVerificationDigest(evidence),
	}
}

func runtimeSessionEvidenceDigest(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	if evidence.Session == nil {
		return ""
	}
	return evidence.Session.EvidenceDigest
}

func runtimeTerminalEvidenceDigest(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	if evidence.Terminal == nil {
		return ""
	}
	return evidence.Terminal.EvidenceDigest
}

func runtimeAttestationEvidenceDigest(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	if evidence.Attestation == nil {
		return ""
	}
	return evidence.Attestation.EvidenceDigest
}

func runtimeAttestationVerificationDigest(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	if evidence.AttestationVerification == nil {
		return ""
	}
	return evidence.AttestationVerification.VerificationDigest
}
