package brokerapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestPolicyRuntimeE2EPromotionApprovalLifecycleUsesDurableStoreAndLinkedPolicyDigest(t *testing.T) {
	storeRoot, ledgerRoot, s, runID, unapproved, decisionDigest, pending := setupPromotionApprovalE2EFixture(t)
	s, pending = reopenAndRequirePendingPromotionApproval(t, storeRoot, ledgerRoot, runID, decisionDigest)
	resolveReq := signedPromotionResolveRequest(t, s, pending, unapproved.Digest)
	assertPromotionApprovalResolutionLinkedHash(t, s, resolveReq, pending.ApprovalID, decisionDigest)
}

func TestPolicyRuntimeE2EStageSignOffStaleThenCurrentLifecycle(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	oldRequest, oldDecision, oldVerifiers := signedStageSummaryApprovalArtifactsForBrokerTests(t, "human", "run-stage-e2e", "stage-1", "sha256:"+strings.Repeat("6", 64), 1, "approve")
	newRequest, newDecision, newVerifiers := signedStageSummaryApprovalArtifactsForBrokerTests(t, "human", "run-stage-e2e", "stage-1", "sha256:"+strings.Repeat("7", 64), 2, "approve")
	for _, verifier := range append(oldVerifiers, newVerifiers...) {
		if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	oldID := seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage-e2e", "stage-1", *oldRequest)
	newID := seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage-e2e", "stage-1", *newRequest)

	oldResolveResp, oldErrResp := s.HandleApprovalResolve(context.Background(), stageResolveRequestForE2E(oldID, "req-stage-e2e-old", *oldRequest, *oldDecision), RequestContext{})
	if oldErrResp != nil {
		t.Fatalf("HandleApprovalResolve(old) error response: %+v", oldErrResp)
	}
	if oldResolveResp.ResolutionStatus != "no_change" || oldResolveResp.Approval.Status != "superseded" {
		t.Fatalf("old resolve status = (%q, %q), want (no_change, superseded)", oldResolveResp.ResolutionStatus, oldResolveResp.Approval.Status)
	}
	if oldResolveResp.Approval.SupersededByApprovalID != newID {
		t.Fatalf("superseded_by_approval_id = %q, want %q", oldResolveResp.Approval.SupersededByApprovalID, newID)
	}

	newResolveResp, newErrResp := s.HandleApprovalResolve(context.Background(), stageResolveRequestForE2E(newID, "req-stage-e2e-new", *newRequest, *newDecision), RequestContext{})
	if newErrResp != nil {
		t.Fatalf("HandleApprovalResolve(new) error response: %+v", newErrResp)
	}
	if newResolveResp.Approval.Status != "consumed" {
		t.Fatalf("new approval status = %q, want consumed", newResolveResp.Approval.Status)
	}
}

func TestPolicyRuntimeE2EMissingAllowlistsFailsClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-missing-allowlist"
	putTrustedPolicyContextForE2ERun(t, s, e2eContextInput{
		runID:         runID,
		roleFamily:    "workspace",
		roleKind:      "workspace-edit",
		capabilities:  []string{artifactReadCapabilityID},
		allowlistRefs: []string{"sha256:" + strings.Repeat("a", 64)},
	})
	action := policyengine.NewArtifactReadAction(policyengine.ArtifactReadActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: artifactReadCapabilityID,
			Actor: policyengine.ActionActor{
				ActorKind:  "role_instance",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		ArtifactHash: mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("8", 64)),
		ReadMode:     "full",
	})

	_, err := s.EvaluateAction(runID, action)
	if err == nil {
		t.Fatal("EvaluateAction returned nil error, want fail-closed missing allowlist")
	}
	if !strings.Contains(err.Error(), "missing trusted allowlist") {
		t.Fatalf("error = %v, want missing trusted allowlist", err)
	}
}

func TestPolicyRuntimeE2EUnknownActionKindsAndSchemaIDsFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-unknown-action"
	putTrustedPolicyContextForRun(t, s, runID, false)

	known := policyengine.NewArtifactReadAction(policyengine.ArtifactReadActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: artifactReadCapabilityID,
			Actor: policyengine.ActionActor{
				ActorKind:  "role_instance",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		ArtifactHash: mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("8", 64)),
		ReadMode:     "full",
	})

	unknownKind := known
	unknownKind.ActionKind = "unknown_action_kind"
	_, err := s.EvaluateAction(runID, unknownKind)
	if err == nil {
		t.Fatal("EvaluateAction returned nil error for unknown action kind")
	}
	if !strings.Contains(err.Error(), "broker_validation_schema_invalid") {
		t.Fatalf("unknown action_kind error = %v, want fail-closed validation error", err)
	}

	unknownSchema := known
	unknownSchema.ActionPayloadSchemaID = "runecode.protocol.v0.ActionPayloadUnknown"
	_, err = s.EvaluateAction(runID, unknownSchema)
	if err == nil {
		t.Fatal("EvaluateAction returned nil error for unknown action payload schema id")
	}
	if !strings.Contains(err.Error(), "broker_validation_schema_invalid") {
		t.Fatalf("unknown action_payload_schema_id error = %v, want fail-closed validation error", err)
	}
}

func TestPolicyRuntimeE2ENonGatewayEgressDenied(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-non-gateway-egress"
	putTrustedPolicyContextForE2ERun(t, s, e2eContextInput{
		runID:        runID,
		roleFamily:   "workspace",
		roleKind:     "workspace-edit",
		capabilities: []string{"cap_exec"},
	})

	action := policyengine.NewExecutorRunAction(policyengine.ExecutorRunActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: "cap_exec",
			Actor: policyengine.ActionActor{
				ActorKind:  "role_instance",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		ExecutorClass: "workspace_ordinary",
		ExecutorID:    "python",
		Argv:          []string{"python", "script.py"},
		NetworkAccess: "gateway_only",
	})

	decision, err := s.EvaluateAction(runID, action)
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("decision_outcome = %q, want deny", decision.DecisionOutcome)
	}
}

func TestPolicyRuntimeE2EContainerFallbackDenied(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-container-fallback"
	putTrustedPolicyContextForE2ERun(t, s, e2eContextInput{
		runID:        runID,
		roleFamily:   "workspace",
		roleKind:     "workspace-edit",
		capabilities: []string{"cap_backend"},
	})

	requiresOptIn := true
	action := policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: "cap_backend",
			Actor: policyengine.ActionActor{
				ActorKind:  "role_instance",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		BackendClass:     "container",
		ChangeKind:       "select_backend",
		RequestedPosture: "automatic_fallback",
		RequiresOptIn:    &requiresOptIn,
	})

	decision, err := s.EvaluateAction(runID, action)
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("decision_outcome = %q, want deny", decision.DecisionOutcome)
	}
	if decision.PolicyReasonCode != "deny_container_automatic_fallback" {
		t.Fatalf("policy_reason_code = %q, want deny_container_automatic_fallback", decision.PolicyReasonCode)
	}
}

type e2eContextInput struct {
	runID         string
	roleFamily    string
	roleKind      string
	capabilities  []string
	allowlistRefs []string
	ruleSetRules  []map[string]any
}

func signedResolveEnvelopesForStoredPendingRequest(t *testing.T, request trustpolicy.SignedObjectEnvelope, approver, outcome string) (trustpolicy.SignedObjectEnvelope, trustpolicy.SignedObjectEnvelope, trustpolicy.VerifierRecord) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyID := sha256.Sum256(publicKey)
	keyIDValue := hex.EncodeToString(keyID[:])

	requestSig := signCanonicalPayloadForE2E(t, privateKey, request.Payload)
	signedRequest := request
	signedRequest.Signature = trustpolicy.SignatureBlock{
		Alg:        "ed25519",
		KeyID:      trustpolicy.KeyIDProfile,
		KeyIDValue: keyIDValue,
		Signature:  requestSig,
	}

	decision := signedDecisionEnvelopeForE2E(t, privateKey, signedRequest, approver, outcome, keyIDValue)
	verifier := approvalAuthorityVerifierForE2E(publicKey, keyIDValue, approver)
	return signedRequest, decision, verifier
}

func setupPromotionApprovalE2EFixture(t *testing.T) (string, string, *Service, string, artifacts.ArtifactReference, string, artifacts.ApprovalRecord) {
	t.Helper()
	storeRoot, ledgerRoot, s := newPersistentBrokerServiceForE2ETest(t)
	runID := "run-e2e-promotion"
	putTrustedPolicyContextForE2ERun(t, s, e2eContextInput{
		runID:        runID,
		roleFamily:   "workspace",
		roleKind:     "workspace-edit",
		capabilities: []string{policyengine.ActionKindPromotion},
		ruleSetRules: []map[string]any{{
			"rule_id":           "require-promotion-approval",
			"effect":            "require_human_approval",
			"action_kind":       policyengine.ActionKindPromotion,
			"capability_id":     policyengine.ActionKindPromotion,
			"reason_code":       "approval_required",
			"details_schema_id": "runecode.protocol.details.policy.evaluation.v0",
		}},
	})
	unapproved := putUnapprovedForE2E(t, s, runID)
	action := promotionActionForE2E(t, unapproved.Digest)
	decision, err := s.EvaluateAction(runID, action)
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionRequireHumanApproval {
		t.Fatalf("decision_outcome = %q, want require_human_approval", decision.DecisionOutcome)
	}
	decisionDigest := requireSinglePolicyDecisionDigestForRun(t, s, runID)
	pending := requirePendingPromotionApprovalForDecision(t, s, runID, decisionDigest)
	return storeRoot, ledgerRoot, s, runID, unapproved, decisionDigest, pending
}

func requireSinglePolicyDecisionDigestForRun(t *testing.T, s *Service, runID string) string {
	t.Helper()
	decisionRefs := s.PolicyDecisionRefsForRun(runID)
	if len(decisionRefs) != 1 {
		t.Fatalf("policy decision refs len = %d, want 1", len(decisionRefs))
	}
	return decisionRefs[0]
}

func requirePendingPromotionApprovalForDecision(t *testing.T, s *Service, runID, decisionDigest string) artifacts.ApprovalRecord {
	t.Helper()
	pending := pendingApprovalForRunAndKind(t, s, runID, policyengine.ActionKindPromotion)
	if pending.PolicyDecisionHash != decisionDigest {
		t.Fatalf("pending policy_decision_hash = %q, want %q", pending.PolicyDecisionHash, decisionDigest)
	}
	if pending.RequestEnvelope == nil {
		t.Fatal("pending approval missing request_envelope")
	}
	return pending
}

func putUnapprovedForE2E(t *testing.T, s *Service, runID string) artifacts.ArtifactReference {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64),
		CreatedByRole:         "workspace",
		RunID:                 runID,
		StepID:                "step-1",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return ref
}

func promotionActionForE2E(t *testing.T, digest string) policyengine.ActionRequest {
	t.Helper()
	action, err := artifacts.BuildPromotionActionRequest(artifacts.PromotionRequest{
		UnapprovedDigest:     digest,
		Approver:             "human",
		RepoPath:             "repo/file.txt",
		Commit:               "abc123",
		ExtractorToolVersion: "tool-v1",
		FullContentVisible:   true,
	})
	if err != nil {
		t.Fatalf("BuildPromotionActionRequest returned error: %v", err)
	}
	return action
}

func assertPromotionApprovalResolutionLinkedHash(t *testing.T, s *Service, resolveReq ApprovalResolveRequest, approvalID, decisionDigest string) {
	t.Helper()
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.Approval.Status != "consumed" {
		t.Fatalf("approval status = %q, want consumed", resolveResp.Approval.Status)
	}
	if resolveResp.Approval.PolicyDecisionHash != decisionDigest {
		t.Fatalf("resolved approval policy_decision_hash = %q, want %q", resolveResp.Approval.PolicyDecisionHash, decisionDigest)
	}
	if resolveResp.ApprovedArtifact == nil {
		t.Fatal("approved_artifact = nil, want promoted artifact")
	}
	getResp, getErr := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-e2e-promotion-get",
		ApprovalID:    approvalID,
	}, RequestContext{})
	if getErr != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", getErr)
	}
	if getResp.Approval.PolicyDecisionHash != decisionDigest {
		t.Fatalf("stored approval policy_decision_hash = %q, want %q", getResp.Approval.PolicyDecisionHash, decisionDigest)
	}
}

func stageResolveRequestForE2E(approvalID, requestID string, request, decision trustpolicy.SignedObjectEnvelope) ApprovalResolveRequest {
	return ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		ApprovalID:    approvalID,
		BoundScope: ApprovalBoundScope{
			SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion: "0.1.0",
			WorkspaceID:   workspaceIDForRun("run-stage-e2e"),
			RunID:         "run-stage-e2e",
			StageID:       "stage-1",
			ActionKind:    policyengine.ActionKindStageSummarySign,
		},
		UnapprovedDigest:       "sha256:" + strings.Repeat("d", 64),
		Approver:               "human",
		RepoPath:               "repo/file.txt",
		Commit:                 "abc123",
		ExtractorToolVersion:   "tool-v1",
		FullContentVisible:     true,
		SignedApprovalRequest:  request,
		SignedApprovalDecision: decision,
	}
}

func signedDecisionEnvelopeForE2E(t *testing.T, privateKey ed25519.PrivateKey, request trustpolicy.SignedObjectEnvelope, approver, outcome, keyIDValue string) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	approvalID, err := approvalIDFromRequest(request)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	decidedAt := time.Now().UTC().Format(time.RFC3339)
	decisionPayload := map[string]any{
		"schema_id":                trustpolicy.ApprovalDecisionSchemaID,
		"schema_version":           trustpolicy.ApprovalDecisionSchemaVersion,
		"approval_request_hash":    map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(approvalID, "sha256:")},
		"approver":                 map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"},
		"decision_outcome":         outcome,
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"key_protection_posture":   "hardware_backed",
		"identity_binding_posture": "attested",
		"approval_assertion_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
		"decided_at":               decidedAt,
		"consumption_posture":      "single_use",
		"signatures":               []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": keyIDValue, "signature": "c2ln"}},
	}
	decisionBytes := mustJSONBytes(t, decisionPayload)
	decisionSignature := signCanonicalPayloadForE2E(t, privateKey, decisionBytes)
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalDecisionSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion,
		Payload:              decisionBytes,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: decisionSignature},
	}
}

func approvalAuthorityVerifierForE2E(publicKey ed25519.PublicKey, keyIDValue, approver string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "approval_authority",
		LogicalScope:           "user",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: approver, InstanceID: "approval-session"},
		KeyProtectionPosture:   "hardware_backed",
		IdentityBindingPosture: "attested",
		PresenceMode:           "hardware_touch",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func signCanonicalPayloadForE2E(t *testing.T, privateKey ed25519.PrivateKey, payload []byte) string {
	t.Helper()
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		t.Fatalf("canonicalize payload returned error: %v", err)
	}
	sig := ed25519.Sign(privateKey, canonical)
	return base64.StdEncoding.EncodeToString(sig)
}

func toAnyStrings(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func mustDigestIdentityForE2E(t *testing.T, identity string) trustpolicy.Digest {
	t.Helper()
	digest, err := digestFromIdentity(identity)
	if err != nil {
		t.Fatalf("digestFromIdentity returned error: %v", err)
	}
	return digest
}
