//go:build linux

package main

import (
	"errors"
	"testing"
)

func TestParseArgsReturnsUsageErrorWhenRequiredFlagsMissing(t *testing.T) {
	_, err := parseArgs([]string{})
	if err == nil {
		t.Fatal("parseArgs error = nil, want usage error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("parseArgs error = %T, want usageError", err)
	}
}

func TestParseArgsReturnsUsageErrorForInvalidFlag(t *testing.T) {
	_, err := parseArgs([]string{"--bad-flag"})
	if err == nil {
		t.Fatal("parseArgs error = nil, want usage error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("parseArgs error = %T, want usageError", err)
	}
}
