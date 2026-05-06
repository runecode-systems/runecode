//go:build linux

package main

import (
	"bytes"
	"io"
	"reflect"
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

func TestStableTTYEnvOverridesOrAddsTERM(t *testing.T) {
	t.Parallel()

	got := stableTTYEnv([]string{"FOO=bar", "TERM=dumb"})
	if !reflect.DeepEqual(got, []string{"FOO=bar", "TERM=" + stableTUITerm}) {
		t.Fatalf("stableTTYEnv override = %v", got)
	}

	got = stableTTYEnv([]string{"FOO=bar"})
	if !reflect.DeepEqual(got, []string{"FOO=bar", "TERM=" + stableTUITerm}) {
		t.Fatalf("stableTTYEnv append = %v", got)
	}
}

func TestTerminalQueryResponderAnswersSplitQueries(t *testing.T) {
	t.Parallel()

	reader := io.NopCloser(strings.NewReader("prefix \x1b]11;?\x1b\\ middle \x1b[6n suffix"))
	var responses bytes.Buffer
	responder := newTerminalQueryResponder(reader, &responses)
	buf := make([]byte, 4)
	for {
		_, err := responder.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error = %v", err)
		}
	}
	got := responses.String()
	if !strings.Contains(got, terminalBackgroundColorResponse) {
		t.Fatalf("responses missing background color response: %q", got)
	}
	if !strings.Contains(got, terminalCPRResponse) {
		t.Fatalf("responses missing CPR response: %q", got)
	}
}
