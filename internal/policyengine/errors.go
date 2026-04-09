package policyengine

import "fmt"

type ErrorCode string

const (
	ErrCodeUnknownSchemaID           ErrorCode = "unknown_schema_id"
	ErrCodeUnsupportedSchemaVersion  ErrorCode = "unsupported_schema_version"
	ErrCodeBrokerValidationSchema    ErrorCode = "broker_validation_schema_invalid"
	ErrCodeBrokerValidationOperation ErrorCode = "broker_validation_operation_invalid"
	ErrCodeBrokerLimitPolicyReject   ErrorCode = "broker_limit_policy_rejected"
)

type EvaluationError struct {
	Code            ErrorCode
	Category        string
	Retryable       bool
	Message         string
	DetailsSchemaID string
	Details         map[string]any
}

func (e *EvaluationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func schemaVersionError(schemaID, got, want string) *EvaluationError {
	return &EvaluationError{
		Code:            ErrCodeUnsupportedSchemaVersion,
		Category:        "validation",
		Retryable:       false,
		Message:         fmt.Sprintf("%s schema_version %q does not match expected %q", schemaID, got, want),
		DetailsSchemaID: "runecode.protocol.details.error.policy.schema-version.v0",
		Details:         map[string]any{"schema_id": schemaID, "got": got, "want": want},
	}
}

func schemaIDError(got, want string) *EvaluationError {
	return &EvaluationError{
		Code:            ErrCodeUnknownSchemaID,
		Category:        "validation",
		Retryable:       false,
		Message:         fmt.Sprintf("schema_id %q does not match expected %q", got, want),
		DetailsSchemaID: "runecode.protocol.details.error.policy.schema-id.v0",
		Details:         map[string]any{"got": got, "want": want},
	}
}
