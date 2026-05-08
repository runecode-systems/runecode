package brokerapi

import (
	"encoding/json"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func findLauncherRuntimeAuditEvent(t *testing.T, events []artifacts.AuditEvent, runtimeEventType string) artifacts.AuditEvent {
	t.Helper()
	for _, event := range events {
		if event.Type != brokerAuditEventTypeLauncherRuntime {
			continue
		}
		if event.Details["runtime_event_type"] != runtimeEventType {
			continue
		}
		return event
	}
	t.Fatalf("missing %s launcher runtime audit event", runtimeEventType)
	return artifacts.AuditEvent{}
}

func launcherRuntimeAuditEventsByRuntimeType(events []artifacts.AuditEvent, runtimeEventType string) []artifacts.AuditEvent {
	matched := make([]artifacts.AuditEvent, 0, 1)
	for _, event := range events {
		if event.Type != brokerAuditEventTypeLauncherRuntime {
			continue
		}
		if event.Details["runtime_event_type"] != runtimeEventType {
			continue
		}
		matched = append(matched, event)
	}
	return matched
}

func launcherRuntimeAuditEventPayload(t *testing.T, event artifacts.AuditEvent) map[string]any {
	t.Helper()
	rawPayload, ok := event.Details["event_payload"]
	if !ok {
		t.Fatal("event_payload missing from launcher runtime audit details")
	}
	switch payload := rawPayload.(type) {
	case map[string]any:
		return payload
	case json.RawMessage:
		return decodeLauncherRuntimeAuditPayload(t, payload, "RawMessage")
	case []byte:
		return decodeLauncherRuntimeAuditPayload(t, payload, "bytes")
	case string:
		return decodeLauncherRuntimeAuditPayload(t, []byte(payload), "string")
	default:
		t.Fatalf("event_payload = %T, want object or JSON payload", rawPayload)
		return nil
	}
}

func decodeLauncherRuntimeAuditPayload(t *testing.T, payload []byte, label string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(event_payload %s) returned error: %v", label, err)
	}
	return decoded
}

func assertLauncherRuntimeAuditDigests(t *testing.T, event artifacts.AuditEvent, launchDigest, hardeningDigest, sessionDigest string) {
	t.Helper()
	digests := launcherRuntimeAuditDigests(t, event)
	assertLauncherRuntimeAuditDigestValue(t, digests, "launch_receipt", launchDigest)
	assertLauncherRuntimeAuditDigestValue(t, digests, "hardening_posture", hardeningDigest)
	assertLauncherRuntimeAuditDigestValue(t, digests, "session_binding", sessionDigest)
}

func launcherRuntimeAuditDigests(t *testing.T, event artifacts.AuditEvent) map[string]any {
	t.Helper()
	digests, ok := event.Details["stored_runtime_fact_digests"].(map[string]any)
	if !ok {
		t.Fatalf("stored_runtime_fact_digests = %T, want map", event.Details["stored_runtime_fact_digests"])
	}
	return digests
}

func assertLauncherRuntimeAuditDigestValue(t *testing.T, digests map[string]any, name, want string) {
	t.Helper()
	if digests[name] != want {
		t.Fatalf("%s digest = %v, want %q", name, digests[name], want)
	}
}

func countLauncherRuntimeAuditEvents(events []artifacts.AuditEvent) int {
	count := 0
	for _, event := range events {
		if event.Type == brokerAuditEventTypeLauncherRuntime {
			count++
		}
	}
	return count
}

func countRuntimeLaunchDeniedEventsByReasonCode(t *testing.T, events []artifacts.AuditEvent, reasonCode string) int {
	t.Helper()
	count := 0
	for _, event := range events {
		if event.Type != brokerAuditEventTypeLauncherRuntime {
			continue
		}
		if event.Details["runtime_event_type"] != "runtime_launch_denied" {
			continue
		}
		payload := launcherRuntimeAuditEventPayload(t, event)
		if payload["launch_failure_reason_code"] == reasonCode {
			count++
		}
	}
	return count
}

func assertBrokerRejectionAuditEvent(t *testing.T, events []artifacts.AuditEvent, requestID, reasonCode string) {
	t.Helper()
	for _, event := range events {
		if event.Type != brokerAuditEventTypeRejection {
			continue
		}
		if event.Details["request_id"] != requestID {
			continue
		}
		if event.Details["reason_code"] != reasonCode {
			t.Fatalf("reason_code = %v, want %s", event.Details["reason_code"], reasonCode)
		}
		return
	}
	t.Fatalf("missing broker rejection audit event for request_id=%s reason_code=%s", requestID, reasonCode)
}
