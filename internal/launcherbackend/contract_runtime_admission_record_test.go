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
	if record.AttestationMeasurementProfile != MeasurementProfileMicroVMBootV1 {
		t.Fatalf("attestation measurement profile = %q, want %q", record.AttestationMeasurementProfile, MeasurementProfileMicroVMBootV1)
	}
	expectedMeasurementDigests, err := DeriveExpectedMeasurementDigests(MeasurementProfileMicroVMBootV1, descriptor.BootContractVersion, descriptor.ComponentDigests)
	if err != nil {
		t.Fatalf("DeriveExpectedMeasurementDigests returned error: %v", err)
	}
	if len(record.AttestationExpectedMeasurementDigests) != 1 || record.AttestationExpectedMeasurementDigests[0] != expectedMeasurementDigests[0] {
		t.Fatalf("attestation expected measurement digests = %#v, want %#v", record.AttestationExpectedMeasurementDigests, expectedMeasurementDigests)
	}
	if record.AuthorityStateDigest != "" || record.AuthorityStateRevision != 0 {
		t.Fatalf("authority state identity = (%q, %d), want empty until launcher admission assigns effective authority state", record.AuthorityStateDigest, record.AuthorityStateRevision)
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
	record.AuthorityStateRevision = 0
	if err := record.Validate(); err == nil {
		t.Fatal("Validate expected partial authority identity rejection")
	}
}

func TestRuntimeAdmissionRecordRejectsUnknownMeasurementProfile(t *testing.T) {
	record, err := NewRuntimeAdmissionRecord(validRuntimeImageDescriptorForContractTests())
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	record.AttestationMeasurementProfile = "microvm-boot-v2"
	if err := record.Validate(); err == nil {
		t.Fatal("Validate expected unknown measurement profile rejection")
	}
}

func TestRuntimeAdmissionRecordRejectsMissingExpectedMeasurementDigests(t *testing.T) {
	record, err := NewRuntimeAdmissionRecord(validRuntimeImageDescriptorForContractTests())
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	record.AttestationExpectedMeasurementDigests = nil
	if err := record.Validate(); err == nil {
		t.Fatal("Validate expected missing expected measurement digests rejection")
	}
}

func TestNewRuntimeAdmissionRecordRejectsMismatchedDeclaredMeasurementDigests(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	descriptor.Attestation.ExpectedMeasurementDigests = []string{testDigest("f")}
	if _, err := NewRuntimeAdmissionRecord(descriptor); err == nil {
		t.Fatal("NewRuntimeAdmissionRecord expected mismatched attestation expected measurement digests rejection")
	}
}

func testDigestHex(ch byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}
