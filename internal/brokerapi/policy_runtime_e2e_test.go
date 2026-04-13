package brokerapi

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/secretsd"
)

func TestPolicyRuntimeE2EAlpha4SecureSliceFlow(t *testing.T) {
	_, _, s := newPersistentBrokerServiceForE2ETest(t)
	runAlpha4ArtifactHandoff(t, s)
	gatewayRunID := runAlpha4GatewaySetup(t, s)
	runAlpha4GatewayAllowAndDeny(t, s, gatewayRunID)
	runAlpha4SecretsLeaseLifecycle(t, gatewayRunID)
	assertAlpha4ModelEgressAudit(t, s)
	assertAlpha4AuditVerification(t, s)
}

func runAlpha4ArtifactHandoff(t *testing.T, s *Service) {
	t.Helper()

	workspaceRunID := "run-alpha4-artifact"
	putTrustedPolicyContextForRun(t, s, workspaceRunID, false)
	digest := putPayloadArtifactForLocalOpsTest(t, s, "handoff: policy-approved artifact payload", workspaceRunID, "step-1")
	readHandle, readErr := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-alpha4-artifact-read",
		Digest:        digest,
		ProducerRole:  "workspace",
		ConsumerRole:  "model_gateway",
		DataClass:     string(artifacts.DataClassSpecText),
	}, RequestContext{})
	if readErr != nil {
		t.Fatalf("HandleArtifactRead returned error response: %+v", readErr)
	}
	streamEvents, streamErr := s.StreamArtifactReadEvents(readHandle)
	if streamErr != nil {
		t.Fatalf("StreamArtifactReadEvents returned error: %v", streamErr)
	}
	assertArtifactStreamDecodedPayload(t, streamEvents, "handoff: policy-approved artifact payload")
	workspaceDecisionDigest := requireSinglePolicyDecisionDigestForRun(t, s, workspaceRunID)
	if !strings.HasPrefix(workspaceDecisionDigest, "sha256:") {
		t.Fatalf("workspace policy decision digest = %q, want sha256 identity", workspaceDecisionDigest)
	}
}

func runAlpha4GatewaySetup(t *testing.T, s *Service) string {
	t.Helper()

	gatewayRunID := "run-alpha4-gateway"
	putTrustedModelGatewayContextForRun(t, s, gatewayRunID, []any{trustedModelGatewayAllowlistEntry()})
	putRunScopedArtifactForLocalOpsTest(t, s, gatewayRunID, "step-1")
	if err := s.RecordRuntimeFacts(gatewayRunID, launcherRuntimeFactsFixture()); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	assertAlpha4IsolatedBackend(t, s, gatewayRunID)
	return gatewayRunID
}

func assertAlpha4IsolatedBackend(t *testing.T, s *Service, gatewayRunID string) {
	t.Helper()
	runGet := mustRunGetForRuntimeFactsRestartTest(t, s, gatewayRunID)
	if runGet.Run.Summary.BackendKind != launcherbackend.BackendKindMicroVM {
		t.Fatalf("backend_kind = %q, want %q", runGet.Run.Summary.BackendKind, launcherbackend.BackendKindMicroVM)
	}
	if runGet.Run.Summary.IsolationAssuranceLevel != launcherbackend.IsolationAssuranceIsolated {
		t.Fatalf("isolation_assurance_level = %q, want %q", runGet.Run.Summary.IsolationAssuranceLevel, launcherbackend.IsolationAssuranceIsolated)
	}
}

func runAlpha4GatewayAllowAndDeny(t *testing.T, s *Service, gatewayRunID string) {
	t.Helper()
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}
	allowDecision, allowErr := s.EvaluateAction(gatewayRunID, trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission"))
	if allowErr != nil {
		t.Fatalf("EvaluateAction(gateway allow) returned error: %v", allowErr)
	}
	if allowDecision.DecisionOutcome != policyengine.DecisionAllow {
		t.Fatalf("gateway allow decision_outcome = %q, want allow", allowDecision.DecisionOutcome)
	}
	if decisionDigest := requireSinglePolicyDecisionDigestForRun(t, s, gatewayRunID); !strings.HasPrefix(decisionDigest, "sha256:") {
		t.Fatalf("gateway policy decision digest = %q, want sha256 identity", decisionDigest)
	}

	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"10.0.0.24"}}}
	denyDecision, denyErr := s.EvaluateAction(gatewayRunID, trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission"))
	if denyErr != nil {
		t.Fatalf("EvaluateAction(gateway deny) returned error: %v", denyErr)
	}
	if denyDecision.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("gateway deny decision_outcome = %q, want deny", denyDecision.DecisionOutcome)
	}
	if got, _ := denyDecision.Details["reason"].(string); got != "runtime_gateway_dns_rebinding_or_private_ip_blocked" {
		t.Fatalf("gateway deny reason = %q, want runtime_gateway_dns_rebinding_or_private_ip_blocked", got)
	}
}

func runAlpha4SecretsLeaseLifecycle(t *testing.T, gatewayRunID string) {
	t.Helper()
	secretsSvc, err := secretsd.Open(filepath.Join(t.TempDir(), "secrets"))
	if err != nil {
		t.Fatalf("secretsd.Open returned error: %v", err)
	}
	if _, err := secretsSvc.ImportSecret("secrets/prod/model-token", strings.NewReader("token-material")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := secretsSvc.IssueLease(secretsd.IssueLeaseRequest{SecretRef: "secrets/prod/model-token", ConsumerID: "principal:gateway:1", RoleKind: "model-gateway", Scope: "run:" + gatewayRunID, TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	material, activeLease, err := secretsSvc.Retrieve(secretsd.RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:gateway:1", RoleKind: "model-gateway", Scope: "run:" + gatewayRunID})
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if string(material) != "token-material" {
		t.Fatalf("retrieved secret material = %q, want token-material", string(material))
	}
	if activeLease.Status != "active" {
		t.Fatalf("lease status after retrieve = %q, want active", activeLease.Status)
	}
	if _, err := secretsSvc.RevokeLease(secretsd.RevokeLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:gateway:1", RoleKind: "model-gateway", Scope: "run:" + gatewayRunID, Reason: "flow complete"}); err != nil {
		t.Fatalf("RevokeLease returned error: %v", err)
	}
	_, _, err = secretsSvc.Retrieve(secretsd.RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:gateway:1", RoleKind: "model-gateway", Scope: "run:" + gatewayRunID})
	if !errors.Is(err, secretsd.ErrAccessDenied) {
		t.Fatalf("Retrieve after revoke error = %v, want ErrAccessDenied", err)
	}
}

func assertAlpha4ModelEgressAudit(t *testing.T, s *Service) {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	var modelEgress map[string]interface{}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == "model_egress" {
			modelEgress = events[i].Details
			break
		}
	}
	if modelEgress == nil {
		t.Fatal("model_egress audit event not found")
	}
	if got, _ := modelEgress["lease_id"].(string); got != "lease-model-1" {
		t.Fatalf("model_egress lease_id = %q, want lease-model-1", got)
	}
	if bound, _ := modelEgress["request_payload_hash_bound"].(bool); !bound {
		t.Fatalf("request_payload_hash_bound = %v, want true", modelEgress["request_payload_hash_bound"])
	}
}

func assertAlpha4AuditVerification(t *testing.T, s *Service) {
	t.Helper()
	verificationResp, verificationErr := s.HandleAuditVerificationGet(context.Background(), AuditVerificationGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditVerificationGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-alpha4-audit-verify",
	}, RequestContext{})
	if verificationErr != nil {
		t.Fatalf("HandleAuditVerificationGet returned error response: %+v", verificationErr)
	}
	if !verificationResp.Summary.CryptographicallyValid {
		t.Fatal("audit verification summary cryptographically_valid = false, want true")
	}
}

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
	oldPolicyDecisionHash := policyDecisionHashForStoredApproval(t, s, oldID)
	newPolicyDecisionHash := policyDecisionHashForStoredApproval(t, s, newID)

	oldResolveResp, oldErrResp := s.HandleApprovalResolve(context.Background(), stageResolveRequestForE2E(oldID, oldPolicyDecisionHash, "req-stage-e2e-old", *oldRequest, *oldDecision), RequestContext{})
	if oldErrResp != nil {
		t.Fatalf("HandleApprovalResolve(old) error response: %+v", oldErrResp)
	}
	if oldResolveResp.ResolutionStatus != "no_change" || oldResolveResp.Approval.Status != "superseded" {
		t.Fatalf("old resolve status = (%q, %q), want (no_change, superseded)", oldResolveResp.ResolutionStatus, oldResolveResp.Approval.Status)
	}
	if oldResolveResp.Approval.SupersededByApprovalID != newID {
		t.Fatalf("superseded_by_approval_id = %q, want %q", oldResolveResp.Approval.SupersededByApprovalID, newID)
	}

	newResolveResp, newErrResp := s.HandleApprovalResolve(context.Background(), stageResolveRequestForE2E(newID, newPolicyDecisionHash, "req-stage-e2e-new", *newRequest, *newDecision), RequestContext{})
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
