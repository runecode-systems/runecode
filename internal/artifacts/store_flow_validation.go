package artifacts

import "errors"

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
	if err := enforceEgressRestrictions(s.state.Policy, req, s.appendAuditLocked); err != nil {
		return err
	}
	if flowAllowed(s.state.Policy.FlowMatrix, req) {
		return nil
	}
	return s.auditFlowBlockedLocked(
		req.ProducerRole,
		ErrFlowDenied,
		"artifact_flow_denied",
		map[string]interface{}{"digest": req.Digest, "data_class": req.DataClass},
	)
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
