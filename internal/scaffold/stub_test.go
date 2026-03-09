package scaffold

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "short help", args: []string{"-h"}, want: false},
		{name: "long help", args: []string{"--help"}, want: false},
		{name: "help word", args: []string{"help"}, want: false},
		{name: "unknown arg", args: []string{"--token"}, want: true},
		{name: "multiple args", args: []string{"--help", "extra"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArgs(tt.args)
			if tt.want && err == nil {
				t.Fatalf("ValidateArgs(%v) expected error", tt.args)
			}

			if !tt.want && err != nil {
				t.Fatalf("ValidateArgs(%v) unexpected error: %v", tt.args, err)
			}
		})
	}
}

func TestWriteStubMessage(t *testing.T) {
	var out bytes.Buffer

	if err := WriteStubMessage(&out, "runecode-launcher"); err != nil {
		t.Fatalf("WriteStubMessage returned error: %v", err)
	}

	written := out.String()
	if !strings.Contains(written, "runecode-launcher is scaffolded") {
		t.Fatalf("WriteStubMessage output missing binary name: %q", written)
	}
}

func TestWriteStubMessageWriteError(t *testing.T) {
	err := WriteStubMessage(failingWriter{}, "runecode-launcher")
	if err == nil {
		t.Fatal("WriteStubMessage expected write error")
	}
}

func TestWriteHelp(t *testing.T) {
	var out bytes.Buffer

	if err := WriteHelp(&out, "runecode-launcher"); err != nil {
		t.Fatalf("WriteHelp returned error: %v", err)
	}

	written := out.String()
	if !strings.Contains(written, "Usage: runecode-launcher [--help]") {
		t.Fatalf("WriteHelp output missing usage line: %q", written)
	}
}
