package brokerapi

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func defaultWorkflowRoutingForTriggerTests() *SessionWorkflowPackRouting {
	return &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"}
}

func seedSessionExecutionTriggerProjectionLinks(t *testing.T, s *Service) {
	t.Helper()
	_ = mustSessionSendMessage(t, s, SessionSendMessageRequest{SchemaID: "runecode.protocol.v0.SessionSendMessageRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-links-msg", SessionID: "sess-trigger-links", Role: "user", ContentText: "link", RelatedLinks: &SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{"run-session-trigger-links"}, ApprovalIDs: []string{digestForBrokerTest("a")}, ArtifactDigests: []string{digestForBrokerTest("b")}, AuditRecordDigests: []string{digestForBrokerTest("c")}}})
}

func requireCurrentSessionExecution(t *testing.T, detail SessionDetail) *SessionTurnExecution {
	t.Helper()
	if detail.CurrentTurnExecution == nil {
		t.Fatal("current_turn_execution missing")
	}
	return detail.CurrentTurnExecution
}

func assertSessionExecutionBindings(t *testing.T, exec *SessionTurnExecution) {
	t.Helper()
	if exec.PrimaryRunID != "run-session-trigger-links" {
		t.Fatalf("primary_run_id = %q, want run-session-trigger-links", exec.PrimaryRunID)
	}
	if exec.BoundValidatedProjectSubstrateDigest == "" {
		t.Fatal("bound_validated_project_substrate_digest is empty")
	}
	if len(exec.LinkedRunIDs) == 0 || len(exec.LinkedApprovalIDs) == 0 || len(exec.LinkedArtifactDigests) == 0 || len(exec.LinkedAuditRecordDigests) == 0 {
		t.Fatalf("execution links missing: %+v", exec)
	}
}

func requireBoundExecutionDigest(t *testing.T, detail SessionDetail) string {
	t.Helper()
	exec := requireCurrentSessionExecution(t, detail)
	if exec.BoundValidatedProjectSubstrateDigest == "" {
		t.Fatal("bound_validated_project_substrate_digest missing after start")
	}
	return exec.BoundValidatedProjectSubstrateDigest
}

func blockSessionPosturePreserved(t *testing.T, blocked SessionDetail) {
	t.Helper()
	if blocked.Summary.WorkPosture != "blocked" {
		t.Fatalf("summary.work_posture = %q, want blocked", blocked.Summary.WorkPosture)
	}
	if blocked.Summary.WorkPostureReasonCode != "project_substrate_digest_drift" {
		t.Fatalf("summary.work_posture_reason_code = %q, want project_substrate_digest_drift", blocked.Summary.WorkPostureReasonCode)
	}
	if blocked.CurrentTurnExecution == nil || blocked.CurrentTurnExecution.ExecutionState != "blocked" {
		t.Fatal("current blocked execution missing after runtime facts refresh")
	}
}

func markSessionExecutionWaiting(t *testing.T, s *Service, turnID, sessionID string) {
	t.Helper()
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: sessionID, TurnID: turnID, ExecutionState: "waiting", WaitKind: "operator_input", WaitState: "waiting_operator_input", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
}

func markSessionExecutionBlocked(t *testing.T, s *Service, turnID, sessionID string) {
	t.Helper()
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: sessionID, TurnID: turnID, ExecutionState: "blocked", WaitKind: "project_blocked", WaitState: "waiting_project_blocked", BlockedReasonCode: "project_substrate_digest_drift", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
}

func assertSessionExecutionContinueBlocked(t *testing.T, errResp *ErrorResponse, wantCode string) {
	t.Helper()
	if errResp == nil {
		t.Fatal("expected session execution error")
	}
	if errResp.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", errResp.Error.Code, wantCode)
	}
}

func requireSessionExecutionLinkedArtifactByStepAndSchema(t *testing.T, s *Service, runID, stepID, schemaID, expectedText string) map[string]any {
	t.Helper()
	record := requireRunArtifactRecordByStep(t, s, runID, stepID)
	payload := mustArtifactPayload(t, s, record.Reference.Digest)
	if schemaID == "" {
		assertArtifactTextPayload(t, stepID, payload, expectedText)
		return nil
	}
	decoded := mustDecodeArtifactJSON(t, stepID, payload)
	assertTypedArtifactSchemaAndBindings(t, stepID, decoded, schemaID)
	return decoded
}

func requireRunArtifactRecordByStep(t *testing.T, s *Service, runID, stepID string) artifacts.ArtifactRecord {
	t.Helper()
	for _, record := range s.List() {
		if strings.TrimSpace(record.RunID) == strings.TrimSpace(runID) && strings.TrimSpace(record.StepID) == strings.TrimSpace(stepID) {
			return record
		}
	}
	t.Fatalf("artifact for run=%s step=%s not found", runID, stepID)
	return artifacts.ArtifactRecord{}
}

func mustArtifactPayload(t *testing.T, s *Service, digest string) []byte {
	t.Helper()
	reader, err := s.Get(digest)
	if err != nil {
		t.Fatalf("Get(%q) returned error: %v", digest, err)
	}
	payload, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("ReadAll(%q) returned error: %v", digest, err)
	}
	return payload
}

func assertArtifactTextPayload(t *testing.T, stepID string, payload []byte, expectedText string) {
	t.Helper()
	if got := strings.TrimSpace(string(payload)); got != strings.TrimSpace(expectedText) {
		t.Fatalf("artifact %s payload = %q, want %q", stepID, got, expectedText)
	}
}

func mustDecodeArtifactJSON(t *testing.T, stepID string, payload []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(%s) returned error: %v", stepID, err)
	}
	return decoded
}

func assertTypedArtifactSchemaAndBindings(t *testing.T, stepID string, decoded map[string]any, schemaID string) {
	t.Helper()
	if got := strings.TrimSpace(stringValueFromMap(decoded, "schema_id")); got != schemaID {
		t.Fatalf("artifact %s schema_id = %q, want %q", stepID, got, schemaID)
	}
	if digest := digestObjectValueFromMap(decoded, "artifact_digest"); digest == "" {
		t.Fatalf("artifact %s missing artifact_digest: %+v", stepID, decoded)
	}
	if digest := digestObjectValueFromMap(decoded, "source_prompt_identity_digest"); digest == "" {
		t.Fatalf("artifact %s missing source_prompt_identity_digest: %+v", stepID, decoded)
	}
}

func digestObjectValueFromMap(in map[string]any, key string) string {
	raw, ok := in[key]
	if !ok {
		return ""
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	hash, _ := value["hash"].(string)
	if strings.TrimSpace(hash) == "" {
		return ""
	}
	return "sha256:" + strings.TrimSpace(hash)
}

func stringValueFromMap(in map[string]any, key string) string {
	raw, ok := in[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return value
}

func requireFileContents(t *testing.T, root, relativePath, want string) {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", relativePath, err)
	}
	if got := string(payload); got != want {
		t.Fatalf("file %s contents mismatch\nwant:\n%s\n\ngot:\n%s", relativePath, want, got)
	}
}

func putApprovedImplementationMutationArtifactForTest(t *testing.T, s *Service, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(raw)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	ref, err := s.Put(artifacts.PutRequest{Payload: canonical, ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: artifacts.DigestBytes(canonical), CreatedByRole: "test", TrustedSource: true})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if _, _, _, err := s.approvedImplementationWriteIntent(canonical); err != nil {
		t.Fatalf("approvedImplementationWriteIntent returned error: %v", err)
	}
	return ref.Digest
}

func approvedImplementationInputSetFixture(t *testing.T, s *Service, approvedDigests, workspaceDigests, metadataDigests []string) map[string]any {
	t.Helper()
	entry := approvedImplementationCatalogEntry()
	if entry.WorkflowID == "" {
		t.Fatal("approved implementation catalog entry missing")
	}
	payload := map[string]any{
		"schema_id":                          "runecode.protocol.v0.RuneContextApprovedImplementationInputSet",
		"schema_version":                     "0.1.0",
		"approved_input_digests":             digestObjects(approvedDigests),
		"workflow_definition_hash":           digestObject(entry.WorkflowDefinitionHash),
		"process_definition_hash":            digestObject(entry.ProcessDefinitionHash),
		"approval_profile":                   "moderate",
		"autonomy_posture":                   "operator_guided",
		"validated_project_substrate_digest": digestObject(s.projectSubstrate.Snapshot.ValidatedSnapshotDigest),
		"project_substrate_snapshot_digest":  digestObject(s.projectSubstrate.Snapshot.SnapshotDigest),
		"control_input_digest":               digestObject(artifacts.DigestBytes([]byte("approved-implementation-control"))),
		"repo_identity_digest":               digestObject(artifacts.DigestBytes([]byte("approved-implementation-repo"))),
		"repo_state_identity_digest":         digestObject(artifacts.DigestBytes([]byte("approved-implementation-state"))),
	}
	if len(workspaceDigests) > 0 {
		payload["workspace_mutation_digests"] = digestObjects(workspaceDigests)
	}
	if len(metadataDigests) > 0 {
		payload["lifecycle_metadata_mutation_digests"] = digestObjects(metadataDigests)
	}
	setApprovedImplementationInputSetDigest(t, payload)
	return payload
}

func putApprovedImplementationInputSetForTest(t *testing.T, s *Service, payload map[string]any) string {
	t.Helper()
	setApprovedImplementationInputSetDigest(t, payload)
	return putApprovedImplementationInputSetArtifactForTest(t, s, payload)
}

func putApprovedImplementationInputSetArtifactForTest(t *testing.T, s *Service, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(raw)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	ref, err := s.Put(artifacts.PutRequest{Payload: canonical, ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: artifacts.DigestBytes(canonical), CreatedByRole: "test", TrustedSource: true})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return ref.Digest
}

func setApprovedImplementationInputSetDigest(t *testing.T, payload map[string]any) {
	t.Helper()
	delete(payload, "input_set_digest")
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(raw)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	payload["input_set_digest"] = digestObject(artifacts.DigestBytes(canonical))
}

func digestObjects(digests []string) []any {
	out := make([]any, 0, len(digests))
	for _, digest := range digests {
		out = append(out, digestObject(digest))
	}
	return out
}
