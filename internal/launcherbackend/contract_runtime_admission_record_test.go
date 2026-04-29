package launcherbackend

import "testing"

func TestNewRuntimeAdmissionRecordFromDescriptor(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	record, err := NewRuntimeAdmissionRecord(descriptor)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if record.DescriptorDigest != descriptor.DescriptorDigest {
		t.Fatalf("descriptor digest mismatch: got %q want %q", record.DescriptorDigest, descriptor.DescriptorDigest)
	}
}

func TestRuntimeAdmissionRecordRejectsHostPathSignerRef(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	record, err := NewRuntimeAdmissionRecord(descriptor)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	record.RuntimeImageSignerRef = "/var/lib/keys/runtime-image"
	if err := record.Validate(); err == nil {
		t.Fatal("Validate expected host-path signer ref rejection")
	}
}

func TestRuntimeAdmissionRecordIncludesToolchainIdentityWhenPresent(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	record, err := NewRuntimeAdmissionRecord(descriptor)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	if record.RuntimeToolchainDescriptorDigest == "" || record.RuntimeToolchainSignerRef == "" || record.RuntimeToolchainVerifierSetRef == "" || record.RuntimeToolchainSignatureDigest == "" {
		t.Fatal("expected toolchain identity fields in admission record")
	}
}

func TestRuntimeAdmissionRecordRejectsPartialToolchainIdentity(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	record, err := NewRuntimeAdmissionRecord(descriptor)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	record.RuntimeToolchainDescriptorDigest = ""
	if err := record.Validate(); err == nil {
		t.Fatal("Validate expected partial toolchain identity rejection")
	}
}

func TestRuntimeAdmissionRecordRejectsPartialAuthorityStateIdentity(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	record, err := NewRuntimeAdmissionRecord(descriptor)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	record.AuthorityStateDigest = "sha256:" + testDigestHex('a')
	if err := record.Validate(); err == nil {
		t.Fatal("Validate expected partial authority identity rejection")
	}
}

func testDigestHex(ch byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}
