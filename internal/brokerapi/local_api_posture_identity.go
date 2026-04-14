package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func decisionDigestIdentity(decision policyengine.PolicyDecision) string {
	payload := map[string]any{
		"schema_id":                decision.SchemaID,
		"schema_version":           decision.SchemaVersion,
		"decision_outcome":         string(decision.DecisionOutcome),
		"policy_reason_code":       decision.PolicyReasonCode,
		"manifest_hash":            digestObjectForIdentity(decision.ManifestHash),
		"action_request_hash":      digestObjectForIdentity(decision.ActionRequestHash),
		"relevant_artifact_hashes": digestObjectSliceForIdentities(decision.RelevantArtifactHashes),
		"policy_input_hashes":      digestObjectSliceForIdentities(decision.PolicyInputHashes),
		"details_schema_id":        decision.DetailsSchemaID,
		"details":                  decision.Details,
	}
	if strings.TrimSpace(decision.RequiredApprovalSchemaID) != "" {
		payload["required_approval_schema_id"] = decision.RequiredApprovalSchemaID
	}
	if decision.RequiredApproval != nil {
		payload["required_approval"] = decision.RequiredApproval
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestObjectForIdentity(identity string) map[string]any {
	hash := strings.TrimSpace(strings.TrimPrefix(identity, "sha256:"))
	if len(hash) != 64 {
		return map[string]any{"hash_alg": "sha256", "hash": ""}
	}
	return map[string]any{"hash_alg": "sha256", "hash": hash}
}

func digestObjectSliceForIdentities(identities []string) []map[string]any {
	out := make([]map[string]any, 0, len(identities))
	for _, identity := range identities {
		out = append(out, digestObjectForIdentity(identity))
	}
	return out
}
