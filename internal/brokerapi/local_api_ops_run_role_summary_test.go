package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestRunDetailPreservesConcreteWorkspaceRoleKinds(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if _, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-read"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("c", 64), CreatedByRole: "workspace-read", TrustedSource: true, RunID: "run-role-kinds", StepID: "step-1"}); putErr != nil {
		t.Fatalf("Put workspace-read artifact returned error: %v", putErr)
	}
	if _, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-test"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("d", 64), CreatedByRole: "workspace-test", TrustedSource: true, RunID: "run-role-kinds", StepID: "step-2"}); putErr != nil {
		t.Fatalf("Put workspace-test artifact returned error: %v", putErr)
	}

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-role-kinds", RunID: "run-role-kinds"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	kinds := map[string]bool{}
	for _, role := range runGet.Run.RoleSummaries {
		kinds[role.RoleKind] = true
	}
	if !kinds["workspace-read"] {
		t.Fatal("run detail role summaries should preserve workspace-read")
	}
	if !kinds["workspace-test"] {
		t.Fatal("run detail role summaries should preserve workspace-test")
	}
}
