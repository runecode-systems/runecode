package launcherbackend

import (
	"testing"
)

func TestAttachmentPlanRejectsMissingRequiredRoles(t *testing.T) {
	plan := AttachmentPlan{
		ByRole: map[string]AttachmentBinding{
			AttachmentRoleLaunchContext: {ReadOnly: true, ChannelKind: AttachmentChannelVirtualDisk, RequiredDigests: []string{testDigest("a")}},
		},
		Constraints: AttachmentRealizationConstraints{NoHostFilesystemMounts: true},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate expected missing required attachment roles error")
	}
}

func TestAttachmentPlanRejectsBoundaryLeakageAndDeviceNumbering(t *testing.T) {
	plan := validAttachmentPlanForContractTests()
	plan.Constraints.HostLocalPathsVisible = true
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate expected host-local paths visibility rejection")
	}
	plan = validAttachmentPlanForContractTests()
	plan.Constraints.DeviceNumberingVisible = true
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate expected device numbering visibility rejection")
	}
	plan = validAttachmentPlanForContractTests()
	plan.ByRole[AttachmentRoleWorkspace] = AttachmentBinding{ReadOnly: false, ChannelKind: "virtio-disk-slot0"}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate expected channel_kind vocabulary rejection")
	}
}

func TestWorkspaceEncryptionPostureValidateFailClosedDefaults(t *testing.T) {
	posture := WorkspaceEncryptionPosture{Required: true, AtRestProtection: WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: WorkspaceKeyProtectionHardwareBacked, Effective: true}
	if err := posture.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	testWorkspaceEncryptionFailureCases(t, &posture)
	posture = WorkspaceEncryptionPosture{Required: false, AtRestProtection: WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: WorkspaceKeyProtectionExplicitDevOptIn, Effective: false, DegradedReasons: []string{"dev_opt_in_unavailable"}}
	if err := posture.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	testWorkspaceEncryptionLeakageCases(t, &posture)
	testWorkspaceEncryptionVocabularyCases(t)
}

func testWorkspaceEncryptionFailureCases(t *testing.T, posture *WorkspaceEncryptionPosture) {
	t.Helper()
	posture.Effective = false
	posture.DegradedReasons = []string{"missing_hardware_key_support"}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected fail-closed rejection when required encryption is ineffective")
	}
}

func testWorkspaceEncryptionLeakageCases(t *testing.T, posture *WorkspaceEncryptionPosture) {
	t.Helper()
	posture.EvidenceRefs = []string{"/var/lib/keys"}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected host path rejection in evidence_refs")
	}
	posture.EvidenceRefs = []string{"~/keys"}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected home-relative path rejection in evidence_refs")
	}
	posture.EvidenceRefs = []string{"$HOME/keys"}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected environment-variable path rejection in evidence_refs")
	}
	posture.EvidenceRefs = []string{"vda0-keyring"}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected device numbering rejection in evidence_refs")
	}
	posture.EvidenceRefs = nil
	posture.DegradedReasons = []string{"uses /dev/sda"}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected device path rejection in degraded_reasons")
	}
	posture.DegradedReasons = nil
	posture.Effective = false
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected degraded_reasons requirement when effective=false")
	}
}

func testWorkspaceEncryptionVocabularyCases(t *testing.T) {
	t.Helper()
	posture := WorkspaceEncryptionPosture{Required: false, AtRestProtection: "luks2", KeyProtectionPosture: WorkspaceKeyProtectionHardwareBacked, Effective: true}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected at_rest_protection vocabulary rejection")
	}
	posture = WorkspaceEncryptionPosture{Required: false, AtRestProtection: WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: "plaintext_disk", Effective: true}
	if err := posture.Validate(); err == nil {
		t.Fatal("Validate expected key_protection_posture vocabulary rejection")
	}
}

func TestAttachmentPlanSummaryValidateRejectsLeakage(t *testing.T) {
	summary := AttachmentPlanSummary{
		Roles: []AttachmentRoleSummary{
			{Role: AttachmentRoleLaunchContext, ReadOnly: true, ChannelKind: AttachmentChannelReadOnlyChannel, DigestCount: 1},
			{Role: AttachmentRoleWorkspace, ReadOnly: false, ChannelKind: AttachmentChannelVirtualDisk},
			{Role: AttachmentRoleInputArtifacts, ReadOnly: true, ChannelKind: AttachmentChannelVirtualDisk, DigestCount: 2},
			{Role: AttachmentRoleScratch, ReadOnly: false, ChannelKind: AttachmentChannelEphemeralDisk},
		},
		Constraints: AttachmentRealizationConstraints{NoHostFilesystemMounts: true},
	}
	if err := summary.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	testAttachmentSummaryInvalidCases(t, &summary)
}

func testAttachmentSummaryInvalidCases(t *testing.T, summary *AttachmentPlanSummary) {
	t.Helper()
	summary.Roles[1].ChannelKind = "vda"
	if err := summary.Validate(); err == nil {
		t.Fatal("Validate expected device numbering material rejection")
	}
	summary.Roles[1].ChannelKind = AttachmentChannelVirtualDisk
	summary.Constraints.GuestMountAsContractIdentity = true
	if err := summary.Validate(); err == nil {
		t.Fatal("Validate expected guest mount identity rejection")
	}
	summary.Constraints.GuestMountAsContractIdentity = false
	summary.Roles = append(summary.Roles, AttachmentRoleSummary{Role: AttachmentRoleWorkspace, ChannelKind: AttachmentChannelVirtualDisk})
	if err := summary.Validate(); err == nil {
		t.Fatal("Validate expected duplicate role rejection")
	}
	summary.Roles = []AttachmentRoleSummary{{Role: AttachmentRoleWorkspace, ChannelKind: AttachmentChannelVirtualDisk}}
	if err := summary.Validate(); err != nil {
		t.Fatalf("Validate returned error for single-role summary: %v", err)
	}
	summary.Roles[0].ChannelKind = "virtio-0"
	if err := summary.Validate(); err == nil {
		t.Fatal("Validate expected unknown channel_kind rejection")
	}
}
