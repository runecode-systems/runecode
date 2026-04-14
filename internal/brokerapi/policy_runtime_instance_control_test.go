package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestEvaluateInstanceControlActionUsesStableSelectorAndIgnoresUnrelatedNewestManifests(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ctx := putTrustedPolicyContextForRun(t, s, "run-instance-stable", false)
	seedInstanceControlContextForBackendCapability(t, s, ctx.controlRunID)
	seedInstanceControlContextForBackendCapability(t, s, "run-other")

	action := policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{
		ActionEnvelope:               policyengine.ActionEnvelope{CapabilityID: "cap_backend", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}},
		RunID:                        ctx.controlRunID,
		TargetInstanceID:             "launcher-instance-1",
		TargetBackendKind:            "container",
		SelectionMode:                "explicit_selection",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
		Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
	})

	decision, err := s.EvaluateInstanceControlAction(action)
	if err != nil {
		t.Fatalf("EvaluateInstanceControlAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionRequireHumanApproval {
		t.Fatalf("decision_outcome = %q, want %q", decision.DecisionOutcome, policyengine.DecisionRequireHumanApproval)
	}
	if decision.PolicyReasonCode != "approval_required" {
		t.Fatalf("policy_reason_code = %q, want approval_required", decision.PolicyReasonCode)
	}
	assertBackendPostureDecisionScope(t, decision)
	assertPolicyDecisionStoredForControlRun(t, s, ctx.controlRunID)
}

func assertBackendPostureDecisionScope(t *testing.T, decision policyengine.PolicyDecision) {
	t.Helper()
	if decision.ManifestHash == "" {
		t.Fatal("manifest_hash empty")
	}
	scope, ok := decision.RequiredApproval["scope"].(map[string]any)
	if !ok {
		t.Fatalf("required_approval.scope = %T, want map", decision.RequiredApproval["scope"])
	}
	if scope["instance_id"] != "launcher-instance-1" {
		t.Fatalf("required_approval.scope.instance_id = %v, want launcher-instance-1", scope["instance_id"])
	}
}

func assertPolicyDecisionStoredForControlRun(t *testing.T, s *Service, controlRunID string) {
	t.Helper()
	refs := s.PolicyDecisionRefsForRun(controlRunID)
	if len(refs) != 1 {
		t.Fatalf("PolicyDecisionRefsForRun(%q) len = %d, want 1", controlRunID, len(refs))
	}
	if rec, ok := s.PolicyDecisionGet(refs[0]); !ok {
		t.Fatalf("PolicyDecisionGet(%q) missing", refs[0])
	} else if rec.RunID != controlRunID {
		t.Fatalf("policy decision run_id = %q, want %q", rec.RunID, controlRunID)
	}
}

func TestEvaluateInstanceControlActionFailsClosedWhenSelectorMissing(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_ = putTrustedPolicyContextForRun(t, s, "run-instance-missing-selector", false)

	seedSecondInstanceControlContextForAmbiguousInference(t, s)

	action := policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{
		ActionEnvelope:               policyengine.ActionEnvelope{CapabilityID: "cap_backend", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}},
		TargetInstanceID:             "launcher-instance-1",
		TargetBackendKind:            "container",
		SelectionMode:                "explicit_selection",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
	})

	_, err := s.EvaluateInstanceControlAction(action)
	if err == nil {
		t.Fatal("EvaluateInstanceControlAction expected error when selector is missing")
	}
	if !strings.Contains(err.Error(), "instance-control") && !strings.Contains(err.Error(), "selector") {
		t.Fatalf("EvaluateInstanceControlAction error = %v, want instance-control selector message", err)
	}
}

func seedSecondInstanceControlContextForAmbiguousInference(t *testing.T, s *Service) {
	t.Helper()
	seedInstanceControlContextForBackendCapability(t, s, instanceControlRunIDForTests("launcher-instance-2"))
}

func seedInstanceControlContextForBackendCapability(t *testing.T, s *Service, controlRunID string) {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistDigest := putTrustedPolicyArtifact(t, s, controlRunID, artifacts.TrustedContractImportKindPolicyAllowlist, trustedPolicyAllowlistPayload(t))
	controlRolePayload := signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal("workspace", "workspace-edit", controlRunID, ""),
		"role_family":        "workspace",
		"role_kind":          "workspace-edit",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_backend"},
		"allowlist_refs":     []any{digestObject(allowlistDigest)},
	}, verifier, privateKey)
	_ = putTrustedPolicyArtifact(t, s, controlRunID, artifacts.TrustedContractImportKindRoleManifest, controlRolePayload)
	controlRunPayload := signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal("workspace", "workspace-edit", controlRunID, ""),
		"manifest_scope":     "run",
		"run_id":             controlRunID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_backend"},
		"allowlist_refs":     []any{digestObject(allowlistDigest)},
	}, verifier, privateKey)
	_ = putTrustedPolicyArtifact(t, s, controlRunID, artifacts.TrustedContractImportKindRunCapability, controlRunPayload)
}
