package main

import "testing"

func TestReplaceVendorHashLine(t *testing.T) {
	t.Parallel()
	content := "vendorHash = \"sha256-old\";\n"
	updated, changed, err := replaceVendorHashLine(content, "sha256-new")
	if err != nil {
		t.Fatalf("replaceVendorHashLine returned error: %v", err)
	}
	if !changed {
		t.Fatal("replaceVendorHashLine changed = false, want true")
	}
	if updated != "vendorHash = \"sha256-new\";\n" {
		t.Fatalf("updated content = %q, want vendorHash replacement", updated)
	}
}

func TestReplaceVendorHashLineErrorsWhenMissing(t *testing.T) {
	t.Parallel()
	if _, _, err := replaceVendorHashLine("pname = \"runecode\";\n", "sha256-new"); err == nil {
		t.Fatal("replaceVendorHashLine error = nil, want missing vendorHash error")
	}
}
