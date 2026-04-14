package brokerapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func setupServiceWithApprovalFixture(t *testing.T) (*Service, artifacts.ArtifactReference, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	return setupServiceWithApprovalFixtureAndOutcome(t, "approve")
}

func setupServiceWithApprovalFixtureAndOutcome(t *testing.T, outcome string) (*Service, artifacts.ArtifactReference, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	root := t.TempDir()
	ledgerRoot := root + "/audit-ledger"
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	unapproved, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace", RunID: "run-approval", StepID: "step-1"})
	if err != nil {
		t.Fatalf("Put unapproved returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTestsWithOutcome(t, "human", unapproved.Digest, outcome)
	seedPendingApprovalForSignedRequest(t, s, "run-approval", "step-1", unapproved.Digest, *requestEnv)
	for _, verifier := range verifiers {
		if putErr := putTrustedVerifierRecordForService(s, verifier); putErr != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", putErr)
		}
	}
	return s, unapproved, requestEnv, decisionEnv
}

func seedPendingApprovalForSignedRequest(t *testing.T, s *Service, runID, stepID, sourceDigest string, requestEnv trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	approvalID, err := approvalIDFromRequest(requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	payload := decodeApprovalPayloadMap(requestEnv.Payload)
	seedPolicyDecisionForPendingApproval(t, s, runID, payload)
	record := pendingPromotionApprovalRecordForRequest(runID, stepID, sourceDigest, approvalID, payload, requestEnv)
	if err := s.RecordApproval(record); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
}

func seedPolicyDecisionForPendingApproval(t *testing.T, s *Service, runID string, payload map[string]any) {
	t.Helper()
	manifestHash := digestFromPayloadField(payload, "manifest_hash")
	actionHash := digestFromPayloadField(payload, "action_request_hash")
	if err := s.RecordPolicyDecision(runID, "", policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           manifestHash,
		ActionRequestHash:      actionHash,
		PolicyInputHashes:      []string{manifestHash},
		RelevantArtifactHashes: relevantArtifactHashesFromPayload(payload),
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "test_seed"},
	}); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
}

func decodeApprovalPayloadMap(payload []byte) map[string]any {
	out := map[string]any{}
	if err := json.Unmarshal(payload, &out); err != nil {
		panic(err)
	}
	return out
}

func pendingPromotionApprovalRecordForRequest(runID, stepID, sourceDigest, approvalID string, payload map[string]any, requestEnv trustpolicy.SignedObjectEnvelope) artifacts.ApprovalRecord {
	expiresAt := parseRFC3339OrNow(payload, "expires_at")
	return artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "pending",
		WorkspaceID:            workspaceIDForRun(runID),
		RunID:                  runID,
		StageID:                "artifact_flow",
		StepID:                 stepID,
		ActionKind:             policyengine.ActionKindPromotion,
		RequestedAt:            parseRFC3339OrNow(payload, "requested_at"),
		ExpiresAt:              &expiresAt,
		ApprovalTriggerCode:    stringFieldFromPayload(payload, "approval_trigger_code", "excerpt_promotion"),
		ChangesIfApproved:      stringFieldFromPayload(payload, "changes_if_approved", approvalChangesIfApprovedDefault),
		ApprovalAssuranceLevel: stringFieldFromPayload(payload, "approval_assurance_level", "session_authenticated"),
		PresenceMode:           stringFieldFromPayload(payload, "presence_mode", "os_confirmation"),
		ManifestHash:           digestFromPayloadField(payload, "manifest_hash"),
		ActionRequestHash:      digestFromPayloadField(payload, "action_request_hash"),
		RelevantArtifactHashes: relevantArtifactHashesFromPayload(payload),
		RequestDigest:          approvalID,
		SourceDigest:           sourceDigest,
		RequestEnvelope:        &requestEnv,
	}
}

func relevantArtifactHashesFromPayload(payload map[string]any) []string {
	items, ok := payload["relevant_artifact_hashes"].([]any)
	if !ok {
		return nil
	}
	relevant := make([]string, 0, len(items))
	for _, item := range items {
		digestObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hashAlg, _ := digestObj["hash_alg"].(string)
		hash, _ := digestObj["hash"].(string)
		if hashAlg != "" && hash != "" {
			relevant = append(relevant, fmt.Sprintf("%s:%s", hashAlg, hash))
		}
	}
	return relevant
}

func signedApprovalArtifactsForBrokerTests(t *testing.T, approver, digest string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	return signedApprovalArtifactsForBrokerTestsWithOutcome(t, approver, digest, "approve")
}

func signedApprovalArtifactsForBrokerTestsWithOutcome(t *testing.T, approver, digest, outcome string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	t.Helper()
	requestedAt, expiresAt, decidedAt := approvalTimestamps(time.Now().UTC())
	publicKey, privateKey, keyIDValue := approvalSigningIdentity(t)
	actionHash := promotionActionHashForBrokerTests(digest, "repo/file.txt", "abc123", "tool-v1", approver)
	requestEnv := signedApprovalRequestEnvelopeForBrokerTests(t, privateKey, keyIDValue, promotionApprovalRequestPayload(actionHash, digest, requestedAt, expiresAt))
	requestDigest, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	decisionEnv := signedApprovalDecisionEnvelopeForBrokerTests(t, privateKey, keyIDValue, approver, outcome, decidedAt, requestDigest)
	verifier := approvalVerifierRecordForBrokerTests(publicKey, keyIDValue, approver)
	return requestEnv, decisionEnv, []trustpolicy.VerifierRecord{verifier}
}

func promotionApprovalRequestPayload(actionHash, digest, requestedAt, expiresAt string) map[string]any {
	return map[string]any{
		"schema_id":        trustpolicy.ApprovalRequestSchemaID,
		"schema_version":   trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile": "moderate",
		"requester": map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "daemon",
			"principal_id":   "broker",
			"instance_id":    "broker-1",
		},
		"approval_trigger_code": "excerpt_promotion",
		"manifest_hash":         map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
		"action_request_hash":   map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(actionHash, "sha256:")},
		"relevant_artifact_hashes": []any{
			map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(digest, "sha256:")},
		},
		"details_schema_id":        "runecode.protocol.details.approval.excerpt-promotion.v0",
		"details":                  map[string]any{"repo_path": "repo/file.txt", "commit": "abc123"},
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             requestedAt,
		"expires_at":               expiresAt,
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Promote reviewed file excerpts for downstream use.",
		"signatures": []any{map[string]any{
			"alg":          "ed25519",
			"key_id":       trustpolicy.KeyIDProfile,
			"key_id_value": "",
			"signature":    "c2ln",
		}},
	}
}

func promotionActionHashForBrokerTests(digest, repoPath, commit, extractorVersion, approver string) string {
	actionHash, err := artifacts.CanonicalPromotionActionRequestHash(artifacts.PromotionRequest{UnapprovedDigest: digest, Approver: approver, RepoPath: repoPath, Commit: commit, ExtractorToolVersion: extractorVersion})
	if err != nil {
		panic(err)
	}
	return actionHash
}

func createPendingApprovalFromPolicyDecision(t *testing.T, s *Service, runID, stepID, sourceDigest string) string {
	t.Helper()
	manifestHash := "sha256:" + strings.Repeat("1", 64)
	actionHash := promotionActionHashForBrokerTests(sourceDigest, "repo/file.txt", "abc123", "tool-v1", "human")
	if err := s.RecordPolicyDecision(runID, "", policyengine.PolicyDecision{SchemaID: "runecode.protocol.v0.PolicyDecision", SchemaVersion: "0.3.0", DecisionOutcome: policyengine.DecisionRequireHumanApproval, PolicyReasonCode: "approval_required", ManifestHash: manifestHash, PolicyInputHashes: []string{"sha256:" + strings.Repeat("2", 64)}, ActionRequestHash: actionHash, RelevantArtifactHashes: []string{sourceDigest}, DetailsSchemaID: "runecode.protocol.details.policy.evaluation.v0", Details: map[string]any{"precedence": "approval_profile_moderate"}, RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0", RequiredApproval: map[string]any{"approval_trigger_code": "excerpt_promotion", "approval_assurance_level": "session_authenticated", "presence_mode": "os_confirmation", "scope": map[string]any{"schema_id": "runecode.protocol.v0.ApprovalBoundScope", "schema_version": "0.1.0", "workspace_id": workspaceIDForRun(runID), "run_id": runID, "stage_id": "artifact_flow", "step_id": stepID, "action_kind": "promotion"}, "changes_if_approved": "Promote reviewed file excerpts for downstream use.", "approval_ttl_seconds": 1800}}); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	for _, rec := range s.ApprovalList() {
		if rec.RunID == runID && rec.Status == "pending" {
			return rec.ApprovalID
		}
	}
	t.Fatalf("missing pending approval for run %q", runID)
	return ""
}

func putTrustedVerifierRecordForService(service *Service, record trustpolicy.VerifierRecord) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	ref, err := service.Put(artifacts.PutRequest{Payload: b, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "auditd", TrustedSource: true})
	if err != nil {
		return err
	}
	return service.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", map[string]interface{}{artifacts.TrustedContractImportKindDetailKey: artifacts.TrustedContractImportKindVerifierRecord, artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest, artifacts.TrustedContractImportProvenanceDetailKey: "sha256:" + strings.Repeat("1", 64)})
}

func setupServiceWithStageSignOffApprovalFixture(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	root := t.TempDir()
	ledgerRoot := root + "/audit-ledger"
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedStageSummaryApprovalArtifactsForBrokerTests(t, "human", "run-stage", "stage-1", "sha256:"+strings.Repeat("6", 64), 1, "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage", "stage-1", *requestEnv)
	return s, requestEnv, decisionEnv
}

func setupServiceWithSupersededStageSignOffApprovals(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, string) {
	s, oldReq, oldDec := setupServiceWithStageSignOffApprovalFixture(t)
	newReq, _, _ := signedStageSummaryApprovalArtifactsForBrokerTests(t, "human", "run-stage", "stage-1", "sha256:"+strings.Repeat("7", 64), 2, "approve")
	newApprovalID := seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage", "stage-1", *newReq)
	return s, oldReq, oldDec, newApprovalID
}

func setupServiceWithBackendPostureApprovalFixture(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	root := t.TempDir()
	ledgerRoot := root + "/audit-ledger"
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedBackendPostureApprovalArtifactsForBrokerTests(t, "human", "container", "explicit_selection", "select_backend", "reduce_assurance", "exact_action_approval", "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	seedPendingBackendPostureApprovalForSignedRequest(t, s, *requestEnv)
	return s, requestEnv, decisionEnv
}

func seedPendingStageSignOffApprovalForSignedRequest(t *testing.T, s *Service, runID, stageID string, requestEnv trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	approvalID, err := approvalIDFromRequest(requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(requestEnv.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal approval request payload returned error: %v", err)
	}
	seedPolicyDecisionForPendingApproval(t, s, runID, payload)
	requestedAt := parseRFC3339OrNow(payload, "requested_at")
	expiresAt := parseRFC3339OrNow(payload, "expires_at")
	if err := s.RecordApproval(artifacts.ApprovalRecord{ApprovalID: approvalID, Status: "pending", WorkspaceID: workspaceIDForRun(runID), RunID: runID, StageID: stageID, ActionKind: policyengine.ActionKindStageSummarySign, RequestedAt: requestedAt, ExpiresAt: &expiresAt, ApprovalTriggerCode: stringFieldFromPayload(payload, "approval_trigger_code", "stage_sign_off"), ChangesIfApproved: stringFieldFromPayload(payload, "changes_if_approved", "Stage summary is signed off for this exact summary hash."), ApprovalAssuranceLevel: stringFieldFromPayload(payload, "approval_assurance_level", "session_authenticated"), PresenceMode: stringFieldFromPayload(payload, "presence_mode", "os_confirmation"), ManifestHash: digestFromPayloadField(payload, "manifest_hash"), ActionRequestHash: digestFromPayloadField(payload, "action_request_hash"), RequestDigest: approvalID, SourceDigest: "", RequestEnvelope: &requestEnv}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approvalID
}

func signedStageSummaryApprovalArtifactsForBrokerTests(t *testing.T, approver, runID, stageID, stageSummaryDigest string, summaryRevision int64, outcome string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	t.Helper()
	now := time.Now().UTC()
	requestedAt, expiresAt, decidedAt := approvalTimestamps(now)
	publicKey, privateKey, keyIDValue := approvalSigningIdentity(t)
	actionHash := stageSignOffActionHashForBrokerTests(runID, stageID, stageSummaryDigest, summaryRevision)
	requestEnv := signedApprovalRequestEnvelopeForBrokerTests(t, privateKey, keyIDValue, stageSummaryApprovalRequestPayload(actionHash, runID, stageID, stageSummaryDigest, summaryRevision, requestedAt, expiresAt))
	requestDigest, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	decisionEnv := signedApprovalDecisionEnvelopeForBrokerTests(t, privateKey, keyIDValue, approver, outcome, decidedAt, requestDigest)
	verifier := approvalVerifierRecordForBrokerTests(publicKey, keyIDValue, approver)
	return requestEnv, decisionEnv, []trustpolicy.VerifierRecord{verifier}
}

func approvalTimestamps(now time.Time) (string, string, string) {
	return now.Add(-1 * time.Minute).Format(time.RFC3339), now.Add(30 * time.Minute).Format(time.RFC3339), now.Format(time.RFC3339)
}

func approvalSigningIdentity(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyID := sha256.Sum256(publicKey)
	return publicKey, privateKey, hex.EncodeToString(keyID[:])
}

func stageSummaryApprovalRequestPayload(actionHash, runID, stageID, stageSummaryDigest string, summaryRevision int64, requestedAt, expiresAt string) map[string]any {
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1"},
		"approval_trigger_code":    "stage_sign_off",
		"manifest_hash":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
		"action_request_hash":      map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(actionHash, "sha256:")},
		"relevant_artifact_hashes": []any{map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}},
		"details_schema_id":        "runecode.protocol.details.policy.required_approval.stage_sign_off.v0",
		"details":                  map[string]any{"run_id": runID, "stage_id": stageID, "stage_summary_hash": map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(stageSummaryDigest, "sha256:")}, "summary_revision": summaryRevision},
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             requestedAt,
		"expires_at":               expiresAt,
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Stage summary is signed off for this exact summary hash.",
		"signatures":               []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": "", "signature": "c2ln"}},
	}
}

func signedApprovalRequestEnvelopeForBrokerTests(t *testing.T, privateKey ed25519.PrivateKey, keyIDValue string, payload map[string]any) *trustpolicy.SignedObjectEnvelope {
	t.Helper()
	payload["signatures"] = []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": keyIDValue, "signature": "c2ln"}}
	reqBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal stage sign-off approval request returned error: %v", err)
	}
	reqCanonical, err := jsoncanonicalizer.Transform(reqBytes)
	if err != nil {
		t.Fatalf("canonicalize stage sign-off approval request returned error: %v", err)
	}
	reqSig := ed25519.Sign(privateKey, reqCanonical)
	return &trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalRequestSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion, Payload: reqBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(reqSig)}}
}

func signedApprovalDecisionEnvelopeForBrokerTests(t *testing.T, privateKey ed25519.PrivateKey, keyIDValue, approver, outcome, decidedAt, requestDigest string) *trustpolicy.SignedObjectEnvelope {
	t.Helper()
	decisionPayload := map[string]any{"schema_id": trustpolicy.ApprovalDecisionSchemaID, "schema_version": trustpolicy.ApprovalDecisionSchemaVersion, "approval_request_hash": map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(requestDigest, "sha256:")}, "approver": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"}, "decision_outcome": outcome, "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "key_protection_posture": "hardware_backed", "identity_binding_posture": "attested", "approval_assertion_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "decided_at": decidedAt, "consumption_posture": "single_use", "signatures": []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": keyIDValue, "signature": "c2ln"}}}
	decisionBytes, err := json.Marshal(decisionPayload)
	if err != nil {
		t.Fatalf("Marshal stage sign-off approval decision returned error: %v", err)
	}
	decisionCanonical, err := jsoncanonicalizer.Transform(decisionBytes)
	if err != nil {
		t.Fatalf("canonicalize stage sign-off approval decision returned error: %v", err)
	}
	decisionSig := ed25519.Sign(privateKey, decisionCanonical)
	return &trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalDecisionSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion, Payload: decisionBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(decisionSig)}}
}

func approvalVerifierRecordForBrokerTests(publicKey ed25519.PublicKey, keyIDValue, approver string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "approval_authority", LogicalScope: "user", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: approver, InstanceID: "approval-session"}, KeyProtectionPosture: "hardware_backed", IdentityBindingPosture: "attested", PresenceMode: "hardware_touch", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
}

func stageSignOffActionHashForBrokerTests(runID, stageID, stageSummaryDigest string, summaryRevision int64) string {
	stageSummaryHash, err := digestFromIdentity(stageSummaryDigest)
	if err != nil {
		panic(err)
	}
	action := policyengine.NewStageSummarySignOffAction(policyengine.StageSummarySignOffActionInput{ActionEnvelope: policyengine.ActionEnvelope{CapabilityID: "cap_stage", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}}, RunID: runID, StageID: stageID, StageSummaryHash: stageSummaryHash, ApprovalProfile: "moderate", SummaryRevision: &summaryRevision})
	hash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		panic(err)
	}
	return hash
}

func seedPendingBackendPostureApprovalForSignedRequest(t *testing.T, s *Service, requestEnv trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	approvalID, err := approvalIDFromRequest(requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(requestEnv.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal approval request payload returned error: %v", err)
	}
	runID := "run-backend"
	seedPolicyDecisionForPendingApproval(t, s, runID, payload)
	requestedAt := parseRFC3339OrNow(payload, "requested_at")
	expiresAt := parseRFC3339OrNow(payload, "expires_at")
	if err := s.RecordApproval(artifacts.ApprovalRecord{ApprovalID: approvalID, Status: "pending", WorkspaceID: workspaceIDForRun(runID), RunID: runID, ActionKind: policyengine.ActionKindBackendPosture, RequestedAt: requestedAt, ExpiresAt: &expiresAt, ApprovalTriggerCode: stringFieldFromPayload(payload, "approval_trigger_code", "reduced_assurance_backend"), ChangesIfApproved: stringFieldFromPayload(payload, "changes_if_approved", "Reduced-assurance backend posture change may be applied."), ApprovalAssuranceLevel: stringFieldFromPayload(payload, "approval_assurance_level", "reauthenticated"), PresenceMode: stringFieldFromPayload(payload, "presence_mode", "hardware_touch"), ManifestHash: digestFromPayloadField(payload, "manifest_hash"), ActionRequestHash: digestFromPayloadField(payload, "action_request_hash"), RequestDigest: approvalID, SourceDigest: "", RequestEnvelope: &requestEnv}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approvalID
}

func signedBackendPostureApprovalArtifactsForBrokerTests(t *testing.T, approver, backendKind, selectionMode, changeKind, assuranceChangeKind, optInKind, outcome string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	t.Helper()
	now := time.Now().UTC()
	requestedAt, expiresAt, decidedAt := approvalTimestamps(now)
	publicKey, privateKey, keyIDValue := approvalSigningIdentity(t)
	actionHash := backendPostureActionHashForBrokerTests(backendKind, selectionMode, changeKind, assuranceChangeKind, optInKind)
	requestEnv := signedApprovalRequestEnvelopeForBrokerTests(t, privateKey, keyIDValue, backendPostureApprovalRequestPayload(actionHash, backendKind, selectionMode, changeKind, assuranceChangeKind, optInKind, requestedAt, expiresAt))
	requestDigest, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	decisionEnv := signedApprovalDecisionEnvelopeForBrokerTests(t, privateKey, keyIDValue, approver, outcome, decidedAt, requestDigest)
	verifier := approvalVerifierRecordForBrokerTests(publicKey, keyIDValue, approver)
	return requestEnv, decisionEnv, []trustpolicy.VerifierRecord{verifier}
}

func backendPostureApprovalRequestPayload(actionHash, backendKind, selectionMode, changeKind, assuranceChangeKind, optInKind, requestedAt, expiresAt string) map[string]any {
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1"},
		"approval_trigger_code":    "reduced_assurance_backend",
		"manifest_hash":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
		"action_request_hash":      map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(actionHash, "sha256:")},
		"relevant_artifact_hashes": []any{},
		"details_schema_id":        "runecode.protocol.details.policy.required_approval.reduced_assurance_backend.v0",
		"details": map[string]any{
			"target_backend_kind":            backendKind,
			"selection_mode":                 selectionMode,
			"change_kind":                    changeKind,
			"requested_posture":              "container_mode_explicit_opt_in",
			"assurance_change_kind":          assuranceChangeKind,
			"opt_in_kind":                    optInKind,
			"reduced_assurance_acknowledged": true,
			"approval_binding_posture":       "exact_action",
		},
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             requestedAt,
		"expires_at":               expiresAt,
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Reduced-assurance backend posture change may be applied.",
		"signatures":               []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": "", "signature": "c2ln"}},
	}
}

func backendPostureActionHashForBrokerTests(backendKind, selectionMode, changeKind, assuranceChangeKind, optInKind string) string {
	action := policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{ActionEnvelope: policyengine.ActionEnvelope{CapabilityID: "cap_stage", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}}, TargetBackendKind: backendKind, SelectionMode: selectionMode, ChangeKind: changeKind, AssuranceChangeKind: assuranceChangeKind, OptInKind: optInKind, ReducedAssuranceAcknowledged: true, Reason: "operator_requested_reduced_assurance_backend_opt_in"})
	hash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		panic(err)
	}
	return hash
}

func assertApprovalAndAuditReadEndpoints(t *testing.T, s *Service, approvalID string) {
	t.Helper()
	_, _ = s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get", ApprovalID: approvalID}, RequestContext{})
}

func policyDecisionHashForStoredApproval(t *testing.T, s *Service, approvalID string) string {
	t.Helper()
	rec, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	if rec.PolicyDecisionHash == "" {
		t.Fatalf("ApprovalGet(%q) missing policy_decision_hash", approvalID)
	}
	return rec.PolicyDecisionHash
}

func assertVersionAndLogEndpoints(t *testing.T, s *Service) {
	t.Helper()
	assertReadinessAndVersionEndpoints(t, s)
	assertLogStreamEndpoints(t, s)
}

func assertReadinessAndVersionEndpoints(t *testing.T, s *Service) { t.Helper() }
func assertLogStreamEndpoints(t *testing.T, s *Service)           { t.Helper() }
