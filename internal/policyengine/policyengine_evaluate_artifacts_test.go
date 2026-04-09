package policyengine

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestEvaluatePolicyDecisionCarriesRelevantArtifactHashes(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.RelevantArtifactHashes = []trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, {HashAlg: "sha256", Hash: strings.Repeat("2", 64)}}
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(decision.RelevantArtifactHashes) != 2 {
		t.Fatalf("relevant_artifact_hashes len = %d, want 2", len(decision.RelevantArtifactHashes))
	}
}
