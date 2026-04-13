package brokerapi

func isTerminalRunStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "closed", "retained":
		return true
	default:
		return false
	}
}
