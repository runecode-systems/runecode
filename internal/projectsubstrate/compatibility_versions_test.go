package projectsubstrate

import "testing"

func TestParseVersionRejectsInvalidSemverForms(t *testing.T) {
	tests := []string{
		"-1.0.0",
		"01.2.3",
		"1.02.3",
		"1.2.03",
		"1.2.3-01",
		"1.2.3-",
		"1.2.3-rc!",
	}
	for _, input := range tests {
		if _, err := parseVersion(input); err == nil {
			t.Fatalf("parseVersion(%q) error = nil, want invalid version", input)
		}
	}
}

func TestParseVersionAcceptsCanonicalSemverForms(t *testing.T) {
	tests := []string{
		"0.1.0-alpha.14",
		"1.2.3",
		"1.2.3-rc.1",
		"10.20.30-beta-1",
	}
	for _, input := range tests {
		if _, err := parseVersion(input); err != nil {
			t.Fatalf("parseVersion(%q) returned error: %v", input, err)
		}
	}
}
