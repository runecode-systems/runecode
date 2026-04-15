package main

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
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
	if got := err.Error(); !strings.Contains(got, "approval resolve does not support this action kind") {
		t.Fatalf("unexpected error = %q", got)
	}
}
