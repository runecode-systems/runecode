package brokerapi

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func runtimeAuditDetailsForPayload(eventType, payloadSchemaID string, payload any, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) (map[string]interface{}, error) {
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	details := map[string]interface{}{
		"audit_event_type":            eventType,
		"event_payload_schema_id":     payloadSchemaID,
		"event_payload":               json.RawMessage(payloadRaw),
		"operation_id":                buildRuntimeEventOperationID(eventType, evidence),
		"run_id":                      evidence.Launch.RunID,
		"evidence_digest_refs":        runtimeEvidenceDigestRefs(evidence),
		"stored_runtime_fact_digests": runtimeStoredDigestMap(evidence),
	}
	if sessionID := strings.TrimSpace(evidence.Launch.SessionID); sessionID != "" {
		details["session_id"] = sessionID
	}
	if stageID := strings.TrimSpace(facts.LaunchReceipt.StageID); stageID != "" {
		details["stage_id"] = stageID
	}
	return details, nil
}

func buildRuntimeEventOperationID(eventType string, evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	sessionDigest := runtimeSessionEvidenceDigest(evidence)
	base := evidence.Launch.RunID + ":" + evidence.Launch.SessionID + ":" + sessionDigest
	if strings.TrimSpace(base) == "::" {
		base = evidence.Launch.EvidenceDigest
	}
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
	return refs
}

func runtimeStoredDigestMap(evidence launcherbackend.RuntimeEvidenceSnapshot) map[string]string {
	return map[string]string{
		"launch_receipt":    evidence.Launch.EvidenceDigest,
		"hardening_posture": evidence.Hardening.EvidenceDigest,
		"session_binding":   runtimeSessionEvidenceDigest(evidence),
		"terminal_report":   runtimeTerminalEvidenceDigest(evidence),
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
