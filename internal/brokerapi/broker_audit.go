package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	brokerAuditEventTypeRejection          = "broker_api_rejection"
	brokerAuditEventTypeApprovalResolution = "broker_approval_resolution"
	brokerAuditEventTypeLauncherRuntime    = "broker_launcher_runtime_event"
)

type brokerAuditEmitter struct {
}

func newBrokerAuditEmitter() (*brokerAuditEmitter, error) {
	return &brokerAuditEmitter{}, nil
}

func (e *brokerAuditEmitter) emitRejection(store *artifacts.Store, resp ErrorResponse) error {
	if store == nil {
		return fmt.Errorf("broker audit store unavailable")
	}
	details := map[string]interface{}{
		"request_id":     resp.RequestID,
		"reason_code":    resp.Error.Code,
		"error_category": resp.Error.Category,
		"retryable":      resp.Error.Retryable,
	}
	if resp.Error.Message != "" {
		details["message"] = resp.Error.Message
	}
	if err := store.AppendTrustedAuditEvent(brokerAuditEventTypeRejection, "brokerapi", details); err != nil {
		return err
	}
	return nil
}

func (e *brokerAuditEmitter) emitApprovalResolution(store *artifacts.Store, requestID string, approvalID string, status string, reasonCode string) error {
	if store == nil {
		return fmt.Errorf("broker audit store unavailable")
	}
	details := map[string]interface{}{
		"request_id":             requestID,
		"approval_id":            approvalID,
		"approval_status":        status,
		"resolution_reason_code": reasonCode,
	}
	if err := store.AppendTrustedAuditEvent(brokerAuditEventTypeApprovalResolution, "brokerapi", details); err != nil {
		return err
	}
	return nil
}

func (e *brokerAuditEmitter) emitLauncherRuntimeEvent(store *artifacts.Store, runtimeEventType string, details map[string]interface{}) error {
	if store == nil {
		return fmt.Errorf("broker audit store unavailable")
	}
	if details == nil {
		details = map[string]interface{}{}
	}
	details["runtime_event_type"] = runtimeEventType
	return store.AppendTrustedAuditEvent(brokerAuditEventTypeLauncherRuntime, "brokerapi", details)
}

func shouldAuditErrorCode(code string) bool {
	switch code {
	case "broker_api_auth_admission_denied",
		"broker_validation_schema_invalid",
		"broker_validation_operation_invalid",
		"broker_limit_message_size_exceeded",
		"broker_limit_structural_complexity_exceeded",
		"broker_limit_in_flight_exceeded",
		"broker_limit_rate_exceeded",
		"broker_timeout_request_deadline_exceeded",
		"request_cancelled",
		"broker_approval_state_invalid":
		return true
	default:
		return false
	}
}
