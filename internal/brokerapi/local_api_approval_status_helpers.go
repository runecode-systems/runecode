package brokerapi

func approvalStatusForDecisionOutcome(outcome string) (string, bool) {
	switch outcome {
	case "approve":
		return "approved", true
	case "deny":
		return "denied", true
	case "expired":
		return "expired", true
	case "cancelled":
		return "cancelled", true
	default:
		return "", false
	}
}

func resolutionReasonCodeForApprovalStatus(status string) string {
	switch status {
	case "denied":
		return "approval_denied"
	case "expired":
		return "approval_expired"
	case "cancelled":
		return "approval_cancelled"
	case "superseded":
		return "approval_superseded"
	case "consumed":
		return "approval_consumed"
	default:
		return "approval_resolved"
	}
}
