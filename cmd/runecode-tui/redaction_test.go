package main

import "testing"

func TestRedactSecretsMasksCommonCredentialPatterns(t *testing.T) {
	input := "token=abc123 api_key:xyz Authorization: Bearer qqq password = secret"
	got := redactSecrets(input)
	mustContainAll(t, got,
		"token=[REDACTED]",
		"api_key:[REDACTED]",
		"Authorization: Bearer [REDACTED]",
		"password = [REDACTED]",
	)
}
