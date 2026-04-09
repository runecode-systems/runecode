package policyengine

import (
	"encoding/json"
	"fmt"
)

func validateActionRequest(action ActionRequest) error {
	if err := validateActionEnvelope(action); err != nil {
		return err
	}
	registries, err := loadActionRegistries()
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("load action registries: %v", err)}
	}
	if err := validateActionRegistryMembership(action, registries); err != nil {
		return err
	}
	descriptor := actionPayloadByKind[action.ActionKind]
	if err := validateTypedActionPayload(action.ActionPayload, descriptor.schemaPath); err != nil {
		return err
	}
	return nil
}

func validateActionEnvelope(action ActionRequest) error {
	if action.SchemaID != actionRequestSchemaID {
		return schemaIDError(action.SchemaID, actionRequestSchemaID)
	}
	if action.SchemaVersion != actionRequestSchemaVersion {
		return schemaVersionError(action.SchemaID, action.SchemaVersion, actionRequestSchemaVersion)
	}
	actionPayload, err := json.Marshal(action)
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal action request: %v", err)}
	}
	if err := validateObjectPayloadAgainstSchema(actionPayload, actionRequestSchemaPath); err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	return nil
}

func validateActionRegistryMembership(action ActionRequest, registries actionRegistries) error {
	if _, ok := registries.actionKinds[action.ActionKind]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown action_kind %q (fail-closed)", action.ActionKind)}
	}
	descriptor, ok := actionPayloadByKind[action.ActionKind]
	if !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("action_kind %q missing payload descriptor (fail-closed)", action.ActionKind)}
	}
	if _, ok := registries.payloadSchemaIDs[action.ActionPayloadSchemaID]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown action_payload_schema_id %q (fail-closed)", action.ActionPayloadSchemaID)}
	}
	if action.ActionPayloadSchemaID != descriptor.schemaID {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("action_kind %q requires action_payload_schema_id %q, got %q", action.ActionKind, descriptor.schemaID, action.ActionPayloadSchemaID)}
	}
	return nil
}

func validateTypedActionPayload(payload map[string]any, schemaPath string) error {
	typedPayload, err := json.Marshal(payload)
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal action payload: %v", err)}
	}
	if err := validateObjectPayloadAgainstSchema(typedPayload, schemaPath); err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	return nil
}
