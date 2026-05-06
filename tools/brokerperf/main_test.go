package main

import (
	"errors"
	"testing"
)

func TestRunReturnsUsageErrorWhenOutputMissing(t *testing.T) {
	err := run([]string{})
	if err == nil {
		t.Fatal("run error = nil, want usage error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("run error = %T, want usageError", err)
	}
}

func TestRunReturnsUsageErrorForInvalidFlag(t *testing.T) {
	err := run([]string{"--bad-flag"})
	if err == nil {
		t.Fatal("run error = nil, want usage error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("run error = %T, want usageError", err)
	}
}
