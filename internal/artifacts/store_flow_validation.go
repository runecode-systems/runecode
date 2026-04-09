package artifacts

import (
	"errors"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Store) enforceFlowRecordConsistencyLocked(record ArtifactRecord, req FlowCheckRequest) error {
	if record.CreatedByRole != req.ProducerRole {
		return s.auditFlowBlockedLocked(
			req.ProducerRole,
			ErrFlowProducerRoleMismatch,
			"artifact_producer_role_mismatch",
			map[string]interface{}{
				"digest":                  req.Digest,
				"requested_producer_role": req.ProducerRole,
				"actual_producer_role":    record.CreatedByRole,
			},
		)
	}
	if record.Reference.DataClass != req.DataClass {
		return s.auditFlowBlockedLocked(
			req.ProducerRole,
			ErrFlowDenied,
			"artifact_data_class_mismatch",
			map[string]interface{}{
				"digest":               req.Digest,
				"requested_data_class": req.DataClass,
				"actual_data_class":    record.Reference.DataClass,
			},
		)
	}
	return nil
}

func (s *Store) enforceFlowPolicyLocked(req FlowCheckRequest) error {
	flowRules := artifactFlowRulesFromPolicy(s.state.Policy.FlowMatrix)
	outcome, reason, details := policyengine.EvaluateArtifactFlowRules(policyengine.ArtifactFlowPolicy{
		UnapprovedExcerptEgressDenied:  s.state.Policy.UnapprovedExcerptEgressDenied,
		ApprovedExcerptEgressOptInOnly: s.state.Policy.ApprovedExcerptEgressOptInOnly,
		RevokedApprovedExcerptDigests:  s.state.Policy.RevokedApprovedExcerptHashes,
		FlowMatrix:                     flowRules,
	}, policyengine.ArtifactFlowRequest{
		ProducerRole:  req.ProducerRole,
		ConsumerRole:  req.ConsumerRole,
		DataClass:     string(req.DataClass),
		Digest:        req.Digest,
		IsEgress:      req.IsEgress,
		ManifestOptIn: req.ManifestOptIn,
	})
	if outcome == policyengine.DecisionAllow {
		return nil
	}
	return s.auditFlowBlockedLocked(req.ProducerRole, mapFlowReasonToError(reason), reason, flowAuditDetails(req, details))
}

func artifactFlowRulesFromPolicy(flowMatrix []FlowRule) []policyengine.ArtifactFlowRule {
	flowRules := make([]policyengine.ArtifactFlowRule, 0, len(flowMatrix))
	for _, rule := range flowMatrix {
		flowRules = append(flowRules, policyengine.ArtifactFlowRule{
			ProducerRole:       rule.ProducerRole,
			ConsumerRole:       rule.ConsumerRole,
			AllowedDataClasses: stringifyDataClasses(rule.AllowedDataClasses),
		})
	}
	return flowRules
}

func stringifyDataClasses(dataClasses []DataClass) []string {
	out := make([]string, 0, len(dataClasses))
	for _, dc := range dataClasses {
		out = append(out, string(dc))
	}
	return out
}

func mapFlowReasonToError(reason string) error {
	switch reason {
	case "unapproved_excerpt_egress_denied":
		return ErrUnapprovedEgressDenied
	case "approved_excerpt_requires_manifest_opt_in":
		return ErrApprovedEgressRequiresManifest
	case "approved_excerpt_revoked":
		return ErrApprovedExcerptRevoked
	default:
		return ErrFlowDenied
	}
}

func flowAuditDetails(req FlowCheckRequest, details map[string]any) map[string]interface{} {
	converted := map[string]interface{}{"digest": req.Digest, "data_class": req.DataClass}
	for key, value := range details {
		converted[key] = value
	}
	return converted
}

func (s *Store) auditFlowBlockedLocked(actor string, denyErr error, reason string, details map[string]interface{}) error {
	out := map[string]interface{}{"reason": reason}
	for key, value := range details {
		out[key] = value
	}
	if auditErr := s.appendAuditLocked("artifact_flow_blocked", actor, out); auditErr != nil {
		return errors.Join(denyErr, auditErr)
	}
	return denyErr
}
