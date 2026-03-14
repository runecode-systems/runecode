package protocolschema

import "testing"

func TestLLMRequestRequiresArtifactInputsAndExplicitLimits(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/LLMRequest.schema.json")

	for _, testCase := range llmRequestCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestLLMResponseKeepsOutputsTypedAndProposalOnly(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/LLMResponse.schema.json")

	for _, testCase := range llmResponseCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestLLMStreamEventRequiresTerminalAndPayloadFields(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/LLMStreamEvent.schema.json")

	for _, testCase := range llmStreamEventCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func llmRequestCases() []validationCase {
	return []validationCase{
		{name: "text response request", value: validLLMRequest()},
		{name: "structured output request", value: validStructuredOutputLLMRequest()},
		{name: "structured output requires schema", value: invalidStructuredOutputLLMRequestWithoutSchema(), wantErr: true},
		{name: "structured output requires schema version", value: invalidStructuredOutputLLMRequestWithoutSchemaVersion(), wantErr: true},
		{name: "text response rejects output schema refs", value: invalidTextLLMRequestWithOutputSchema(), wantErr: true},
		{name: "input artifacts reject duplicates", value: invalidLLMRequestWithDuplicateInputArtifacts(), wantErr: true},
		{name: "tool allowlist rejects duplicates", value: invalidLLMRequestWithDuplicateToolAllowlist(), wantErr: true},
		{name: "limits must stay explicit", value: invalidLLMRequestWithoutLimits(), wantErr: true},
	}
}

func llmResponseCases() []validationCase {
	return []validationCase{
		{name: "artifact backed response", value: validLLMResponse()},
		{name: "structured output response", value: validStructuredOutputLLMResponse()},
		{name: "tool call proposal requires schema version", value: invalidLLMResponseWithoutToolCallSchemaVersion(), wantErr: true},
		{name: "structured output requires schema version", value: invalidLLMResponseWithoutStructuredSchemaVersion(), wantErr: true},
		{name: "output artifacts reject duplicates", value: invalidLLMResponseWithDuplicateOutputArtifacts(), wantErr: true},
		{name: "tool calls reject duplicates", value: invalidLLMResponseWithDuplicateToolCalls(), wantErr: true},
	}
}

func llmStreamEventCases() []validationCase {
	return []validationCase{
		{name: "start event", value: validLLMStreamStartEvent()},
		{name: "content delta event", value: validLLMStreamDeltaEvent()},
		{name: "tool call event", value: validLLMStreamToolCallEvent()},
		{name: "structured candidate event", value: validLLMStructuredCandidateEvent()},
		{name: "successful terminal event", value: validLLMSuccessTerminalEvent()},
		{name: "failed terminal event", value: validLLMFailureTerminalEvent()},
		{name: "delta event requires content", value: invalidLLMDeltaEventWithoutContent(), wantErr: true},
		{name: "structured candidate requires schema metadata", value: invalidLLMStructuredCandidateWithoutSchemaID(), wantErr: true},
		{name: "structured candidate requires schema version", value: invalidLLMStructuredCandidateWithoutSchemaVersion(), wantErr: true},
		{name: "terminal success requires response hash", value: invalidLLMSuccessTerminalEventWithoutResponseHash(), wantErr: true},
		{name: "non terminal rejects terminal fields", value: invalidLLMStartEventWithTerminalStatus(), wantErr: true},
		{name: "terminal rejects content payloads", value: invalidLLMTerminalEventWithContentDelta(), wantErr: true},
		{name: "cancelled terminal event", value: validLLMNonSuccessTerminalEvent("cancelled")},
		{name: "interrupted terminal event", value: validLLMNonSuccessTerminalEvent("interrupted")},
		{name: "truncated terminal event", value: validLLMNonSuccessTerminalEvent("truncated")},
		{name: "failure terminal event", value: validLLMNonSuccessTerminalEvent("failure")},
		{name: "tool call event requires payload", value: invalidLLMToolCallEventWithoutToolCall(), wantErr: true},
	}
}

func validLLMRequest() map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.LLMRequest",
		"schema_version":   "0.3.0",
		"selection_source": "signed_allowlist",
		"provider":         "openai",
		"model":            "gpt-4.1-mini",
		"input_artifacts":  []any{validArtifactReference()},
		"tool_allowlist":   []any{validToolAllowlistEntry()},
		"response_mode":    "text",
		"streaming_mode":   "stream",
		"request_limits":   validRequestLimits(),
	}
}

func validStructuredOutputLLMRequest() map[string]any {
	request := validLLMRequest()
	request["response_mode"] = "structured_output"
	request["output_schema_id"] = "runecode.protocol.output.plan.v0"
	request["output_schema_version"] = "0.1.0"
	return request
}

func invalidStructuredOutputLLMRequestWithoutSchema() map[string]any {
	request := validStructuredOutputLLMRequest()
	delete(request, "output_schema_id")
	return request
}

func invalidStructuredOutputLLMRequestWithoutSchemaVersion() map[string]any {
	request := validStructuredOutputLLMRequest()
	delete(request, "output_schema_version")
	return request
}

func invalidTextLLMRequestWithOutputSchema() map[string]any {
	request := validLLMRequest()
	request["output_schema_id"] = "runecode.protocol.output.plan.v0"
	request["output_schema_version"] = "0.1.0"
	return request
}

func invalidLLMRequestWithDuplicateInputArtifacts() map[string]any {
	request := validLLMRequest()
	request["input_artifacts"] = []any{validArtifactReference(), validArtifactReference()}
	return request
}

func invalidLLMRequestWithDuplicateToolAllowlist() map[string]any {
	request := validLLMRequest()
	request["tool_allowlist"] = []any{validToolAllowlistEntry(), validToolAllowlistEntry()}
	return request
}

func invalidLLMRequestWithoutLimits() map[string]any {
	request := validLLMRequest()
	delete(request, "request_limits")
	return request
}

func validRequestLimits() map[string]any {
	return map[string]any{
		"max_request_bytes":                  262144,
		"max_tool_calls":                     8,
		"max_total_tool_call_argument_bytes": 65536,
		"max_structured_output_bytes":        262144,
		"max_streamed_bytes":                 16777216,
		"max_stream_chunk_bytes":             65536,
		"stream_idle_timeout_ms":             15000,
	}
}

func validLLMResponse() map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.LLMResponse",
		"schema_version":       "0.3.0",
		"request_hash":         testDigestValue("5"),
		"output_trust_posture": "untrusted_proposal",
		"output_artifacts":     []any{validArtifactReference()},
		"proposed_tool_calls":  []any{validToolCallProposal()},
	}
}

func validStructuredOutputLLMResponse() map[string]any {
	response := validLLMResponse()
	response["structured_output_schema_id"] = "runecode.protocol.output.plan.v0"
	response["structured_output_schema_version"] = "0.1.0"
	response["structured_output"] = map[string]any{"next_step": "run tests"}
	return response
}

func invalidLLMResponseWithoutStructuredSchemaVersion() map[string]any {
	response := validStructuredOutputLLMResponse()
	delete(response, "structured_output_schema_version")
	return response
}

func invalidLLMResponseWithoutToolCallSchemaVersion() map[string]any {
	response := validLLMResponse()
	toolCalls := response["proposed_tool_calls"].([]any)
	toolCall := toolCalls[0].(map[string]any)
	delete(toolCall, "arguments_schema_version")
	return response
}

func invalidLLMResponseWithDuplicateOutputArtifacts() map[string]any {
	response := validLLMResponse()
	response["output_artifacts"] = []any{validArtifactReference(), validArtifactReference()}
	return response
}

func invalidLLMResponseWithDuplicateToolCalls() map[string]any {
	response := validLLMResponse()
	response["proposed_tool_calls"] = []any{validToolCallProposal(), validToolCallProposal()}
	return response
}

func validToolAllowlistEntry() map[string]any {
	return map[string]any{
		"tool_name":                "write_patch",
		"arguments_schema_id":      "runecode.protocol.toolargs.write-patch.v0",
		"arguments_schema_version": "0.1.0",
		"tool_description":         "Apply a patch to workspace files.",
	}
}

func validToolCallProposal() map[string]any {
	return map[string]any{
		"tool_call_id":             "tool-call-1",
		"tool_name":                "write_patch",
		"arguments_schema_id":      "runecode.protocol.toolargs.write-patch.v0",
		"arguments_schema_version": "0.1.0",
		"arguments":                map[string]any{"target": "protocol/schemas/manifest.json"},
	}
}

func validLLMStreamStartEvent() map[string]any {
	return baseLLMStreamEvent("response_start")
}

func validLLMStreamDeltaEvent() map[string]any {
	event := baseLLMStreamEvent("output_delta")
	event["content_delta"] = "partial output"
	return event
}

func validLLMStreamToolCallEvent() map[string]any {
	event := baseLLMStreamEvent("tool_call_proposal")
	event["tool_call"] = validToolCallProposal()
	return event
}

func validLLMStructuredCandidateEvent() map[string]any {
	event := baseLLMStreamEvent("structured_output_candidate")
	event["structured_output_candidate_schema_id"] = "runecode.protocol.output.plan.v0"
	event["structured_output_candidate_schema_version"] = "0.1.0"
	event["structured_output_candidate"] = map[string]any{"next_step": "run tests"}
	return event
}

func validLLMSuccessTerminalEvent() map[string]any {
	event := baseLLMStreamEvent("response_terminal")
	event["terminal_status"] = "success"
	event["final_response_hash"] = testDigestValue("6")
	return event
}

func validLLMFailureTerminalEvent() map[string]any {
	return validLLMNonSuccessTerminalEvent("timeout")
}

func validLLMNonSuccessTerminalEvent(status string) map[string]any {
	event := baseLLMStreamEvent("response_terminal")
	event["terminal_status"] = status
	event["final_error"] = validErrorEnvelope()
	return event
}

func invalidLLMSuccessTerminalEventWithoutResponseHash() map[string]any {
	event := validLLMSuccessTerminalEvent()
	delete(event, "final_response_hash")
	return event
}

func invalidLLMDeltaEventWithoutContent() map[string]any {
	return baseLLMStreamEvent("output_delta")
}

func invalidLLMStructuredCandidateWithoutSchemaID() map[string]any {
	event := validLLMStructuredCandidateEvent()
	delete(event, "structured_output_candidate_schema_id")
	return event
}

func invalidLLMStructuredCandidateWithoutSchemaVersion() map[string]any {
	event := validLLMStructuredCandidateEvent()
	delete(event, "structured_output_candidate_schema_version")
	return event
}

func invalidLLMStartEventWithTerminalStatus() map[string]any {
	event := validLLMStreamStartEvent()
	event["terminal_status"] = "success"
	event["final_response_hash"] = testDigestValue("6")
	return event
}

func invalidLLMTerminalEventWithContentDelta() map[string]any {
	event := validLLMSuccessTerminalEvent()
	event["content_delta"] = "should not be present"
	return event
}

func invalidLLMToolCallEventWithoutToolCall() map[string]any {
	return baseLLMStreamEvent("tool_call_proposal")
}

func baseLLMStreamEvent(eventType string) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.LLMStreamEvent",
		"schema_version": "0.3.0",
		"stream_id":      "stream-1",
		"request_hash":   testDigestValue("5"),
		"seq":            1,
		"emitter":        manifestPrincipal(),
		"event_type":     eventType,
	}
}
