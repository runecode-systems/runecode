package main

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestValidateApprovalResolveInputRejectsUnsupportedActionKind(t *testing.T) {
	resp := brokerapi.ApprovalGetResponse{
		Approval: brokerapi.ApprovalSummary{
			ApprovalID: "ap-unsupported",
			BoundScope: brokerapi.ApprovalBoundScope{ActionKind: "__unsupported_test_kind__"},
		},
		SignedApprovalRequest:  testSignedApprovalRequest,
		SignedApprovalDecision: testSignedApprovalDecision,
	}
	err := validateApprovalResolveInput(resp)
	if err == nil {
		t.Fatal("expected unsupported action_kind error")
	}
	if got := err.Error(); !strings.Contains(got, "resolve unavailable: this approval action kind is not supported by the current TUI resolve flow") {
		t.Fatalf("unexpected error = %q", got)
	}
}

func TestApprovalDecisionMatchesRequestRequiresBoundHash(t *testing.T) {
	requestID, err := approvalRequestDigestIdentity(*testSignedApprovalRequest)
	if err != nil {
		t.Fatalf("approvalRequestDigestIdentity returned error: %v", err)
	}
	if err := approvalDecisionMatchesRequest(*testSignedApprovalDecision, requestID); err != nil {
		t.Fatalf("expected matching approval_request_hash to pass, got %v", err)
	}

	missing := *testSignedApprovalDecision
	missing.Payload = []byte(`{"schema_id":"runecode.protocol.v0.ApprovalDecision","schema_version":"0.3.0"}`)
	if err := approvalDecisionMatchesRequest(missing, requestID); err == nil || !strings.Contains(err.Error(), "approval_request_hash missing") {
		t.Fatalf("expected missing approval_request_hash to fail closed, got %v", err)
	}

	mismatch := *testSignedApprovalDecision
	mismatch.Payload = []byte(`{"schema_id":"runecode.protocol.v0.ApprovalDecision","schema_version":"0.3.0","approval_request_hash":{"hash_alg":"sha256","hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`)
	if err := approvalDecisionMatchesRequest(mismatch, requestID); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected mismatched approval_request_hash to fail closed, got %v", err)
	}
}

func TestApprovalDecisionMatchesRequestRejectsInvalidHash(t *testing.T) {
	requestID, err := approvalRequestDigestIdentity(*testSignedApprovalRequest)
	if err != nil {
		t.Fatalf("approvalRequestDigestIdentity returned error: %v", err)
	}
	invalid := *testSignedApprovalDecision
	invalid.Payload = []byte(`{"schema_id":"runecode.protocol.v0.ApprovalDecision","schema_version":"0.3.0","approval_request_hash":{"hash_alg":"sha256","hash":"not-hex"}}`)
	err = approvalDecisionMatchesRequest(invalid, requestID)
	if err == nil {
		t.Fatal("expected invalid approval_request_hash to fail closed")
	}
	if !strings.Contains(err.Error(), "invalid approval_request_hash") && !strings.Contains(err.Error(), "sha256 hash") {
		t.Fatalf("unexpected invalid hash error: %v", err)
	}

	wrongSchema := *testSignedApprovalDecision
	wrongSchema.PayloadSchemaID = trustpolicy.ApprovalRequestSchemaID
	if err := approvalDecisionMatchesRequest(wrongSchema, requestID); err == nil || !strings.Contains(err.Error(), "unexpected decision payload schema") {
		t.Fatalf("expected wrong payload schema to fail closed, got %v", err)
	}
}
