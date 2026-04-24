package brokerapi

import "fmt"

func validateSessionSendMessageRoleForTranscriptOnly(role string) error {
	if role != "user" && role != "assistant" && role != "system" && role != "tool" {
		return fmt.Errorf("role is invalid")
	}
	return nil
}
