package main

import (
	"strings"
	"testing"
)

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

func TestRedactSecretsMasksExportAssignmentPattern(t *testing.T) {
	input := "export GITHUB_TOKEN=ghp_secret_value"
	got := redactSecrets(input)
	if !strings.Contains(got, "export GITHUB_TOKEN=[REDACTED]") {
		t.Fatalf("expected export assignment to be redacted, got %q", got)
	}
}

func TestSanitizeUITextStripsEscapesAndCapsLength(t *testing.T) {
	input := "token=abc123\x1b[31m /tmp/secret\n" + strings.Repeat("a", 600)
	got := sanitizeUIText(input)
	if strings.Contains(got, "abc123") {
		t.Fatalf("expected secret redacted, got %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Fatalf("expected escape sequence removed, got %q", got)
	}
	if len(got) > 515 {
		t.Fatalf("expected capped output length, got %d", len(got))
	}
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("expected newlines removed from sanitized UI text, got %q", got)
	}
}
