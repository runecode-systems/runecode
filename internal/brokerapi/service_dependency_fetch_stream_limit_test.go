package brokerapi

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestStreamingSizeLimitReaderUnderLimit(t *testing.T) {
	r := newStreamingSizeLimitReader(strings.NewReader("abcd"), 8)
	buf := make([]byte, 16)

	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("first read error = %v, want nil", err)
	}
	if got := string(buf[:n]); got != "abcd" {
		t.Fatalf("first read bytes = %q, want %q", got, "abcd")
	}

	n, err = r.Read(buf)
	if n != 0 {
		t.Fatalf("second read n = %d, want 0", n)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("second read err = %v, want EOF", err)
	}
}

func TestStreamingSizeLimitReaderExactLimitEOFWhenUpstreamExhausted(t *testing.T) {
	r := newStreamingSizeLimitReader(strings.NewReader("abcde"), 5)
	buf := make([]byte, 32) // intentionally larger than remaining budget

	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("first read error = %v, want nil", err)
	}
	if n != 5 {
		t.Fatalf("first read n = %d, want 5", n)
	}
	if got := string(buf[:n]); got != "abcde" {
		t.Fatalf("first read bytes = %q, want %q", got, "abcde")
	}

	n, err = r.Read(buf)
	if n != 0 {
		t.Fatalf("second read n = %d, want 0", n)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("second read err = %v, want EOF", err)
	}
}

func TestStreamingSizeLimitReaderReturnsLimitErrorWhenUpstreamHasExtraBytes(t *testing.T) {
	r := newStreamingSizeLimitReader(strings.NewReader("abcdef"), 5)
	buf := make([]byte, 32) // intentionally larger than remaining budget

	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("first read error = %v, want nil", err)
	}
	if n != 5 {
		t.Fatalf("first read n = %d, want 5", n)
	}
	if got := string(buf[:n]); got != "abcde" {
		t.Fatalf("first read bytes = %q, want %q", got, "abcde")
	}

	n, err = r.Read(buf)
	if n != 0 {
		t.Fatalf("second read n = %d, want 0", n)
	}
	if !isStreamingSizeLimitError(err) {
		t.Fatalf("second read err = %v, want dependencyResponseSizeLimitError", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "max_response_bytes") {
		t.Fatalf("second read error message = %q, want max_response_bytes", err.Error())
	}

	n, err = r.Read(buf)
	if n != 0 {
		t.Fatalf("third read n = %d, want 0", n)
	}
	if !isStreamingSizeLimitError(err) {
		t.Fatalf("third read err = %v, want dependencyResponseSizeLimitError", err)
	}
}
