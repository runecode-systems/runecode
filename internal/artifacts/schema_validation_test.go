package artifacts

import (
	"encoding/json"
	"testing"
)

func TestValidateObjectPayloadAgainstSchemaRejectsUnknownFields(t *testing.T) {
	payload := validApprovalDecisionPayloadForSchemaTests()
	payload["unknown_field"] = "fail-closed"
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	if err := validateObjectPayloadAgainstSchema(b, "objects/ApprovalDecision.schema.json"); err == nil {
		t.Fatal("validateObjectPayloadAgainstSchema expected unknown field failure")
	}
}

func validApprovalDecisionPayloadForSchemaTests() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.ApprovalDecision",
		"schema_version":           "0.3.0",
		"approval_request_hash":    map[string]any{"hash_alg": "sha256", "hash": testDigestHash("d")},
		"approver":                 map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": "alice", "instance_id": "approval-session"},
		"decision_outcome":         "approve",
		"approval_assurance_level": "none",
		"presence_mode":            "none",
		"key_protection_posture":   "os_keystore",
		"identity_binding_posture": "tofu",
		"decided_at":               "2026-03-13T12:05:00Z",
		"consumption_posture":      "single_use",
		"signatures":               []any{signatureBlockForSchemaTests()},
	}
}

func signatureBlockForSchemaTests() map[string]any {
	return map[string]any{
		"alg":          "ed25519",
		"key_id":       "key_sha256",
		"key_id_value": testDigestHash("a"),
		"signature":    "c2ln",
	}
}

func testDigestHash(nibble string) string {
	out := ""
	for len(out) < 64 {
		out += nibble
	}
	return out
}
