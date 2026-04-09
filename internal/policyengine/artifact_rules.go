package policyengine

import "strings"

type ArtifactFlowRule struct {
	ProducerRole       string
	ConsumerRole       string
	AllowedDataClasses []string
}

type ArtifactFlowPolicy struct {
	UnapprovedExcerptEgressDenied  bool
	ApprovedExcerptEgressOptInOnly bool
	// RevokedApprovedExcerptDigests is exclusively checked against approved_file_excerpts
	// data class requests. Digests of other data classes are not evaluated against this set.
	RevokedApprovedExcerptDigests map[string]bool
	FlowMatrix                    []ArtifactFlowRule
}

type ArtifactFlowRequest struct {
	ProducerRole  string
	ConsumerRole  string
	DataClass     string
	Digest        string
	IsEgress      bool
	ManifestOptIn bool
}

func EvaluateArtifactFlowRules(policy ArtifactFlowPolicy, req ArtifactFlowRequest) (DecisionOutcome, string, map[string]any) {
	if outcome, reason, details, decided := evaluateArtifactFlowRestrictions(policy, req); decided {
		return outcome, reason, details
	}
	return evaluateArtifactFlowMatrix(policy, req)
}

func evaluateArtifactFlowRestrictions(policy ArtifactFlowPolicy, req ArtifactFlowRequest) (DecisionOutcome, string, map[string]any, bool) {
	if req.IsEgress && req.DataClass == "unapproved_file_excerpts" && policy.UnapprovedExcerptEgressDenied {
		return DecisionDeny, "unapproved_excerpt_egress_denied", map[string]any{"digest": req.Digest}, true
	}
	if req.IsEgress && req.DataClass == "approved_file_excerpts" && policy.ApprovedExcerptEgressOptInOnly && !req.ManifestOptIn {
		return DecisionDeny, "approved_excerpt_requires_manifest_opt_in", map[string]any{"digest": req.Digest}, true
	}
	if req.DataClass == "approved_file_excerpts" && policy.RevokedApprovedExcerptDigests != nil && policy.RevokedApprovedExcerptDigests[req.Digest] {
		return DecisionDeny, "approved_excerpt_revoked", map[string]any{"digest": req.Digest}, true
	}
	return "", "", nil, false
}

func evaluateArtifactFlowMatrix(policy ArtifactFlowPolicy, req ArtifactFlowRequest) (DecisionOutcome, string, map[string]any) {
	matchedRolePair := false
	for _, rule := range policy.FlowMatrix {
		if !strings.EqualFold(rule.ProducerRole, req.ProducerRole) || !strings.EqualFold(rule.ConsumerRole, req.ConsumerRole) {
			continue
		}
		matchedRolePair = true
		if roleRuleAllowsDataClass(rule, req.DataClass) {
			return DecisionAllow, "allow_manifest_opt_in", map[string]any{}
		}
	}
	if matchedRolePair {
		return DecisionDeny, "artifact_flow_denied", map[string]any{"digest": req.Digest, "data_class": req.DataClass}
	}
	return DecisionDeny, "artifact_flow_denied", map[string]any{"digest": req.Digest, "data_class": req.DataClass}
}

func roleRuleAllowsDataClass(rule ArtifactFlowRule, dataClass string) bool {
	for _, allowedClass := range rule.AllowedDataClasses {
		if allowedClass == dataClass {
			return true
		}
	}
	return false
}
