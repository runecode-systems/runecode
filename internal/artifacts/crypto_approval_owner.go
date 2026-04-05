package artifacts

import (
	"errors"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func validateDecisionApproverBinding(req PromotionRequest, decision trustpolicy.ApprovalDecision, verifier trustpolicy.VerifierRecord) error {
	if err := trustpolicy.ValidateApprovalDecisionEvidence(decision); err != nil {
		return err
	}
	if decision.DecisionOutcome != "approve" {
		return errors.New("approval decision outcome is not approve")
	}
	if decision.Approver.PrincipalID == "" {
		return errors.New("approval decision approver principal is required")
	}
	if req.Approver != "" && req.Approver != decision.Approver.PrincipalID {
		return errors.New("promotion approver does not match approval decision approver")
	}
	if !samePrincipalIdentity(verifier.OwnerPrincipal, decision.Approver) {
		return errors.New("approval decision approver does not match verifier owner identity")
	}
	return nil
}

func samePrincipalIdentity(left trustpolicy.PrincipalIdentity, right trustpolicy.PrincipalIdentity) bool {
	if left.SchemaID != right.SchemaID {
		return false
	}
	if left.SchemaVersion != right.SchemaVersion {
		return false
	}
	if left.ActorKind != right.ActorKind {
		return false
	}
	if left.PrincipalID != right.PrincipalID {
		return false
	}
	if left.InstanceID != right.InstanceID {
		return false
	}
	return true
}
