package brokerapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

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

func signedStageSummaryApprovalArtifactsForBrokerTests(t *testing.T, approver, runID, stageID, stageSummaryDigest string, summaryRevision int64, outcome string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	return signedStageSummaryApprovalArtifactsForBrokerTestsWithPlan(t, approver, runID, "", stageID, stageSummaryDigest, summaryRevision, outcome)
}

func signedStageSummaryApprovalArtifactsForBrokerTestsWithPlan(t *testing.T, approver, runID, planID, stageID, stageSummaryDigest string, summaryRevision int64, outcome string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	return signedStageSummaryApprovalArtifactsForBrokerTestsWithPlanAt(t, approver, runID, planID, stageID, stageSummaryDigest, summaryRevision, outcome, time.Now().UTC())
}

func signedStageSummaryApprovalArtifactsForBrokerTestsWithPlanAt(t *testing.T, approver, runID, planID, stageID, stageSummaryDigest string, summaryRevision int64, outcome string, now time.Time) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	t.Helper()
	requestedAt, expiresAt, decidedAt := approvalTimestamps(now)
	publicKey, privateKey, keyIDValue := approvalSigningIdentity(t)
	actionHash := stageSignOffActionHashForBrokerTests(runID, effectiveStageSignOffPlanID(planID), stageID, stageSummaryDigest, summaryRevision)
	requestEnv := signedApprovalRequestEnvelopeForBrokerTests(t, privateKey, keyIDValue, stageSummaryApprovalRequestPayload(actionHash, runID, planID, stageID, stageSummaryDigest, summaryRevision, requestedAt, expiresAt))
	requestDigest, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	decisionEnv := signedApprovalDecisionEnvelopeForBrokerTests(t, privateKey, keyIDValue, approver, outcome, decidedAt, requestDigest)
	verifier := approvalVerifierRecordForBrokerTests(publicKey, keyIDValue, approver)
	return requestEnv, decisionEnv, []trustpolicy.VerifierRecord{verifier}
}

func effectiveStageSignOffPlanID(planID string) string {
	if strings.TrimSpace(planID) == "" {
		return "plan-1"
	}
	return strings.TrimSpace(planID)
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

func stageSummaryApprovalRequestPayload(actionHash, runID, planID, stageID, stageSummaryDigest string, summaryRevision int64, requestedAt, expiresAt string) map[string]any {
	details := map[string]any{"run_id": runID, "stage_id": stageID, "stage_summary_hash": map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(stageSummaryDigest, "sha256:")}, "summary_revision": summaryRevision}
	if strings.TrimSpace(planID) != "" {
		details["plan_id"] = strings.TrimSpace(planID)
	}
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
		"details":                  details,
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
			"target_instance_id":             "launcher-instance-1",
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
