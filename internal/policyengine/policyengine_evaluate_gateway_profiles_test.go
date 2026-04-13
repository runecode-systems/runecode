package policyengine

import (
	"errors"
	"strings"
	"testing"
)

func TestEvaluateGatewayRequiresSignedAllowlistDestinationMatch(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("allowlisted destination should allow, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateGatewayAllowsCaseInsensitiveDestinationHostMatch(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "ALLOWLIST-MODEL.EXAMPLE.COM"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("case-insensitive host should allow, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateGatewayDeniesWhenCanonicalPortRequiredButMissing(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	destination := entry["destination"].(map[string]any)
	destination["canonical_port"] = float64(8443)

	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateGatewayFailsClosedWhenOperationMissing(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	delete(action.ActionPayload, "operation")
	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate error = nil, want fail-closed schema validation error")
	}
	var evalErr *EvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateGatewayDeniesEscapingPathPrefix(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	destination := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1/allowed/"

	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com/v1/allowed/../escape"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateGatewayDeniesPathPrefixCaseMismatch(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	destination := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1/allowed/"

	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com/V1/ALLOWED/request"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateGatewayDeniesModelInvokeWhenPayloadHashNotBoundToCanonicalRequestHash(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	action.ActionPayload["payload_hash"] = mustDigestObject("sha256:" + strings.Repeat("e", 64))
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got != "typed_model_request_binding" {
		t.Fatalf("invariant = %v, want typed_model_request_binding", decision.Details["invariant"])
	}
	if got, _ := decision.Details["reason"].(string); got != "payload_hash_not_bound_to_canonical_llm_request_hash" {
		t.Fatalf("reason = %v, want payload_hash_not_bound_to_canonical_llm_request_hash", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesModelInvokeWhenCanonicalRequestHashBindingMissing(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	action.RelevantArtifactHashes = nil
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got != "typed_model_request_binding" {
		t.Fatalf("invariant = %v, want typed_model_request_binding", decision.Details["invariant"])
	}
	if got, _ := decision.Details["reason"].(string); got != "missing_canonical_llm_request_hash_binding" {
		t.Fatalf("reason = %v, want missing_canonical_llm_request_hash_binding", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesAuthRefreshWhenCanonicalRequestHashBindingMissing(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-auth", "auth-gateway", "auth_provider", "refresh_auth_token", "spec_text")
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("auth-gateway", "cap_auth", allowlist))
	action := validGatewayEgressActionRequest("cap_auth", "gateway", "auth-gateway", "auth-gateway", "auth_provider", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-auth.example.com"
	action.ActionPayload["operation"] = "refresh_auth_token"
	action.RelevantArtifactHashes = nil
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "missing_canonical_gateway_request_hash_binding" {
		t.Fatalf("reason = %v, want missing_canonical_gateway_request_hash_binding", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesWhenRequestTimeoutMissing(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	delete(action.ActionPayload, "timeout_seconds")
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "request_timeout_missing" {
		t.Fatalf("reason = %v, want request_timeout_missing", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesWhenRequestTimeoutExceedsAllowlistLimit(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	action.ActionPayload["timeout_seconds"] = float64(121)
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "request_timeout_exceeds_allowlist_limit" {
		t.Fatalf("reason = %v, want request_timeout_exceeds_allowlist_limit", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesWhenAllowlistResponseSizeLimitMissing(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	delete(entry, "max_response_bytes")
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "allowlist_response_size_limit_missing" {
		t.Fatalf("reason = %v, want allowlist_response_size_limit_missing", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesWhenEgressDataClassNotAllowlisted(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	action.ActionPayload["egress_data_class"] = "diffs"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "egress_data_class_not_allowlisted" {
		t.Fatalf("reason = %v, want egress_data_class_not_allowlisted", decision.Details["reason"])
	}
	if got, _ := decision.Details["egress_data_class"].(string); got != "diffs" {
		t.Fatalf("egress_data_class = %v, want diffs", decision.Details["egress_data_class"])
	}
}

func TestEvaluateGatewayDeniesWhenAuditContextMissing(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	delete(action.ActionPayload, "audit_context")
	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate error = nil, want fail-closed schema validation error")
	}
	var evalErr *EvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateGatewayDeniesWhenQuotaContextMissing(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	delete(action.ActionPayload, "quota_context")
	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate error = nil, want fail-closed schema validation error")
	}
	var evalErr *EvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateGatewayDeniesWhenStreamQuotaExceedsLimit(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	quota := action.ActionPayload["quota_context"].(map[string]any)
	quota["phase"] = "stream"
	quota["enforce_during_stream"] = true
	quota["stream_limit_bytes"] = float64(256)
	meters := quota["meters"].(map[string]any)
	meters["streamed_bytes"] = float64(1024)
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "streamed_bytes_exceed_stream_limit" {
		t.Fatalf("reason = %v, want streamed_bytes_exceed_stream_limit", decision.Details["reason"])
	}
}

func TestEvaluateGatewayDeniesDisallowedModelGatewayEgressDataClassAtBoundary(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "unapproved_file_excerpts")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	action.ActionPayload["egress_data_class"] = "unapproved_file_excerpts"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["reason"].(string); got != "disallowed_egress_data_class" {
		t.Fatalf("reason = %v, want disallowed_egress_data_class", decision.Details["reason"])
	}
	if got, _ := decision.Details["precedence"].(string); got != "invariants_first" {
		t.Fatalf("precedence = %v, want invariants_first", decision.Details["precedence"])
	}
}

func TestCompileGatewayFailsClosedWhenDestinationDNSRebindingProtectionNotEnforced(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	destination := entry["destination"].(map[string]any)
	destination["dns_rebinding_protection"] = "advisory"
	_, err := Compile(compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	if err == nil {
		t.Fatal("Compile returned nil error, want fail-closed schema validation error")
	}
	var evalErr *EvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}
