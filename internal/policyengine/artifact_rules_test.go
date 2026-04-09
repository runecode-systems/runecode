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
