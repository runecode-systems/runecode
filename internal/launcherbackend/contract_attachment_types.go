package launcherbackend

type AttachmentPlan struct {
	ByRole              map[string]AttachmentBinding     `json:"by_role"`
	Constraints         AttachmentRealizationConstraints `json:"constraints"`
	WorkspaceEncryption *WorkspaceEncryptionPosture      `json:"workspace_encryption,omitempty"`
}

type AttachmentBinding struct {
	ReadOnly        bool     `json:"read_only"`
	ChannelKind     string   `json:"channel_kind"`
	RequiredDigests []string `json:"required_digests,omitempty"`
}

type AttachmentRealizationConstraints struct {
	NoHostFilesystemMounts       bool `json:"no_host_filesystem_mounts"`
	HostLocalPathsVisible        bool `json:"host_local_paths_visible,omitempty"`
	DeviceNumberingVisible       bool `json:"device_numbering_visible,omitempty"`
	GuestMountAsContractIdentity bool `json:"guest_mount_as_contract_identity,omitempty"`
}

type WorkspaceEncryptionPosture struct {
	Required             bool     `json:"required"`
	AtRestProtection     string   `json:"at_rest_protection"`
	KeyProtectionPosture string   `json:"key_protection_posture"`
	Effective            bool     `json:"effective"`
	DegradedReasons      []string `json:"degraded_reasons,omitempty"`
	EvidenceRefs         []string `json:"evidence_refs,omitempty"`
}

type AttachmentPlanSummary struct {
	Roles       []AttachmentRoleSummary          `json:"roles"`
	Constraints AttachmentRealizationConstraints `json:"constraints"`
}

type AttachmentRoleSummary struct {
	Role        string `json:"role"`
	ReadOnly    bool   `json:"read_only"`
	ChannelKind string `json:"channel_kind"`
	DigestCount int    `json:"digest_count,omitempty"`
}
