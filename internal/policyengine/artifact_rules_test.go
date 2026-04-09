package policyengine

import "testing"

func TestEvaluateArtifactFlowRulesScansAllMatchingRoleRules(t *testing.T) {
	policy := ArtifactFlowPolicy{
		FlowMatrix: []ArtifactFlowRule{
			{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []string{"spec_text"}},
			{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []string{"approved_file_excerpts"}},
		},
	}
	outcome, reason, _ := EvaluateArtifactFlowRules(policy, ArtifactFlowRequest{
		ProducerRole: "workspace",
		ConsumerRole: "model_gateway",
		DataClass:    "approved_file_excerpts",
		Digest:       "sha256:abc",
	})
	if outcome != DecisionAllow {
		t.Fatalf("outcome = %q, want %q", outcome, DecisionAllow)
	}
	if reason != "allow_manifest_opt_in" {
		t.Fatalf("reason = %q, want allow_manifest_opt_in", reason)
	}
}

func TestEvaluateArtifactFlowRulesRevokedDigestDeniesApprovedExcerpts(t *testing.T) {
	policy := ArtifactFlowPolicy{
		RevokedApprovedExcerptDigests: map[string]bool{"sha256:deadbeef": true},
		FlowMatrix: []ArtifactFlowRule{
			{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []string{"approved_file_excerpts"}},
		},
	}
	outcome, reason, _ := EvaluateArtifactFlowRules(policy, ArtifactFlowRequest{
		ProducerRole: "workspace",
		ConsumerRole: "model_gateway",
		DataClass:    "approved_file_excerpts",
		Digest:       "sha256:deadbeef",
	})
	if outcome != DecisionDeny {
		t.Fatalf("outcome = %q, want %q", outcome, DecisionDeny)
	}
	if reason != "approved_excerpt_revoked" {
		t.Fatalf("reason = %q, want approved_excerpt_revoked", reason)
	}
}

func TestEvaluateArtifactFlowRulesRevokedDigestDoesNotDenyOtherDataClasses(t *testing.T) {
	policy := ArtifactFlowPolicy{
		RevokedApprovedExcerptDigests: map[string]bool{"sha256:deadbeef": true},
		FlowMatrix: []ArtifactFlowRule{
			{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []string{"spec_text"}},
		},
	}
	outcome, reason, _ := EvaluateArtifactFlowRules(policy, ArtifactFlowRequest{
		ProducerRole: "workspace",
		ConsumerRole: "model_gateway",
		DataClass:    "spec_text",
		Digest:       "sha256:deadbeef",
	})
	if outcome != DecisionAllow {
		t.Fatalf("outcome = %q, want %q", outcome, DecisionAllow)
	}
	if reason != "allow_manifest_opt_in" {
		t.Fatalf("reason = %q, want allow_manifest_opt_in", reason)
	}
}
