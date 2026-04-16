package policyengine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
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
	if err := validateActionBindingInvariant(action); err != nil {
		return err
	}
	return nil
}

func validateActionBindingInvariant(action ActionRequest) error {
	if action.ActionKind != ActionKindStageSummarySign {
		return nil
	}
	return validateStageSummarySignOffPayload(action.ActionPayload)
}

func validateStageSummarySignOffPayload(payload map[string]any) error {
	stageSummary, ok := payload["stage_summary"].(map[string]any)
	if !ok || len(stageSummary) == 0 {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "stage_summary_sign_off requires canonical stage_summary object"}
	}
	runID, _ := payload["run_id"].(string)
	stageID, _ := payload["stage_id"].(string)
	summaryRunID, _ := stageSummary["run_id"].(string)
	summaryStageID, _ := stageSummary["stage_id"].(string)
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(stageID) == "" {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "stage_summary_sign_off requires run_id and stage_id"}
	}
	if strings.TrimSpace(summaryRunID) != strings.TrimSpace(runID) {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "stage_summary.run_id must match payload run_id"}
	}
	if strings.TrimSpace(summaryStageID) != strings.TrimSpace(stageID) {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "stage_summary.stage_id must match payload stage_id"}
	}

	summaryHash, err := canonicalHashValue(stageSummary)
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("canonical stage_summary hash failed: %v", err)}
	}
	payloadHash, err := digestIdentityFromPayloadValue(payload["stage_summary_hash"])
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("stage_summary_hash invalid: %v", err)}
	}
	if payloadHash != summaryHash {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "stage_summary_hash must match canonical stage_summary digest"}
	}
	return nil
}

func digestIdentityFromPayloadValue(value any) (string, error) {
	switch typed := value.(type) {
	case trustpolicy.Digest:
		return typed.Identity()
	case map[string]any:
		hashAlg, _ := typed["hash_alg"].(string)
		hash, _ := typed["hash"].(string)
		digest := trustpolicy.Digest{HashAlg: hashAlg, Hash: hash}
		return digest.Identity()
	default:
		return "", fmt.Errorf("must be digest object")
	}
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
