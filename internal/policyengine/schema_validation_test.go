package policyengine

import (
	"encoding/json"
	"testing"
)

func TestValidateObjectPayloadAgainstSchemaAcceptsValidPayload(t *testing.T) {
	payload := validRoleManifestPayload()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	if err := validateObjectPayloadAgainstSchema(b, roleManifestSchemaPath); err != nil {
		t.Fatalf("validateObjectPayloadAgainstSchema returned error: %v", err)
	}
}

func TestValidateObjectPayloadAgainstSchemaRejectsTrailingJSON(t *testing.T) {
	payload := validRoleManifestPayload()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	b = append(b, []byte("{}")...)
	if err := validateObjectPayloadAgainstSchema(b, roleManifestSchemaPath); err == nil {
		t.Fatal("validateObjectPayloadAgainstSchema expected trailing JSON failure")
	}
}
