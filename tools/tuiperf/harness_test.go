//go:build linux

package main

import (
	"strings"
	"testing"
)

func TestSummarizeStartupOutputSanitizesSensitiveData(t *testing.T) {
	t.Parallel()

	raw := "listen failed for /tmp/private/runtime.sock token=abcdefghijklmnopqrstuvwxyz123456"
	summary := summarizeStartupOutput(raw)

	if strings.Contains(summary, "/tmp/private/runtime.sock") {
		t.Fatalf("summary leaked absolute path: %q", summary)
	}
	if strings.Contains(summary, "abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("summary leaked long token: %q", summary)
	}
	if !strings.Contains(summary, "<path>") {
		t.Fatalf("summary missing redacted path marker: %q", summary)
	}
	if !strings.Contains(summary, "<redacted>") {
		t.Fatalf("summary missing redacted token marker: %q", summary)
	}
}

func TestSummarizeStartupOutputTruncatesLongOutput(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("segment ", 40)
	summary := summarizeStartupOutput(long)
	if !strings.HasPrefix(summary, "…") {
		t.Fatalf("summary = %q, want ellipsis prefix", summary)
	}
}

func TestSummarizeBrokerStartupOutputIncludesBothStreams(t *testing.T) {
	t.Parallel()

	summary := summarizeBrokerStartupOutput("stdout ok", "stderr boom")
	if !strings.Contains(summary, "stdout=stdout ok") {
		t.Fatalf("summary missing stdout segment: %q", summary)
	}
	if !strings.Contains(summary, "stderr=stderr boom") {
		t.Fatalf("summary missing stderr segment: %q", summary)
	}
}
