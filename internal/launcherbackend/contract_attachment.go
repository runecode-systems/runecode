package launcherbackend

import (
	"fmt"
	"sort"
	"strings"
)

func (p AttachmentPlan) Validate() error {
	if len(p.ByRole) == 0 {
		return fmt.Errorf("at least one attachment role is required")
	}
	if err := p.Constraints.Validate(); err != nil {
		return fmt.Errorf("constraints: %w", err)
	}
	if err := validateAttachmentPlanRoles(p.ByRole); err != nil {
		return err
	}
	if p.WorkspaceEncryption == nil {
		return fmt.Errorf("workspace_encryption is required for workspace attachment role")
	}
	normalized := p.WorkspaceEncryption.Normalized()
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("workspace_encryption: %w", err)
	}
	if !normalized.Required {
		return fmt.Errorf("workspace_encryption.required must be true (fail-closed default)")
	}
	return nil
}

func validateAttachmentPlanRoles(byRole map[string]AttachmentBinding) error {
	requiredRoles := map[string]struct{}{
		AttachmentRoleLaunchContext:  {},
		AttachmentRoleWorkspace:      {},
		AttachmentRoleInputArtifacts: {},
		AttachmentRoleScratch:        {},
	}
	for role, binding := range byRole {
		normalizedRole, err := normalizeAttachmentRole(role)
		if err != nil {
			return err
		}
		if err := binding.ValidateForRole(normalizedRole); err != nil {
			return fmt.Errorf("attachment role %q: %w", role, err)
		}
		delete(requiredRoles, normalizedRole)
	}
	if len(requiredRoles) == 0 {
		return nil
	}
	missing := make([]string, 0, len(requiredRoles))
	for role := range requiredRoles {
		missing = append(missing, role)
	}
	sort.Strings(missing)
	return fmt.Errorf("missing required attachment roles: %s", strings.Join(missing, ", "))
}

func normalizeAttachmentRole(role string) (string, error) {
	r := strings.TrimSpace(role)
	if r == "" {
		return "", fmt.Errorf("attachment role must be non-empty")
	}
	switch r {
	case AttachmentRoleLaunchContext, AttachmentRoleWorkspace, AttachmentRoleInputArtifacts, AttachmentRoleScratch:
		return r, nil
	default:
		return "", fmt.Errorf("unsupported attachment role %q", role)
	}
}

func (c AttachmentRealizationConstraints) Validate() error {
	if !c.NoHostFilesystemMounts {
		return fmt.Errorf("no_host_filesystem_mounts must be true")
	}
	if c.HostLocalPathsVisible {
		return fmt.Errorf("host_local_paths_visible must be false")
	}
	if c.DeviceNumberingVisible {
		return fmt.Errorf("device_numbering_visible must be false")
	}
	if c.GuestMountAsContractIdentity {
		return fmt.Errorf("guest_mount_as_contract_identity must be false")
	}
	return nil
}

func (b AttachmentBinding) ValidateForRole(role string) error {
	channelKind, err := validateAttachmentBindingCommon(b)
	if err != nil {
		return err
	}
	switch role {
	case AttachmentRoleLaunchContext:
		return validateLaunchContextBinding(b, channelKind)
	case AttachmentRoleInputArtifacts:
		return validateInputArtifactsBinding(b, channelKind)
	case AttachmentRoleWorkspace:
		return validateWorkspaceBinding(b, channelKind)
	case AttachmentRoleScratch:
		return validateScratchBinding(b, channelKind)
	}
	return nil
}

func validateAttachmentBindingCommon(binding AttachmentBinding) (string, error) {
	channelKind := normalizeAttachmentChannelKind(binding.ChannelKind)
	if channelKind == "" {
		return "", fmt.Errorf("channel_kind must be one of %q, %q, %q, or %q", AttachmentChannelArtifactImage, AttachmentChannelReadOnlyVolume, AttachmentChannelWritableVolume, AttachmentChannelEphemeralVolume)
	}
	for _, digest := range binding.RequiredDigests {
		if strings.TrimSpace(digest) == "" {
			return "", fmt.Errorf("required_digests cannot contain empty values")
		}
		if !looksLikeDigest(digest) {
			return "", fmt.Errorf("required_digests values must be sha256:<64 lowercase hex>")
		}
	}
	return channelKind, nil
}

func validateLaunchContextBinding(binding AttachmentBinding, channelKind string) error {
	if !binding.ReadOnly {
		return fmt.Errorf("launch_context must be read_only")
	}
	if channelKind != AttachmentChannelArtifactImage && channelKind != AttachmentChannelReadOnlyVolume {
		return fmt.Errorf("launch_context channel_kind must be %q or %q", AttachmentChannelArtifactImage, AttachmentChannelReadOnlyVolume)
	}
	if len(binding.RequiredDigests) == 0 {
		return fmt.Errorf("launch_context requires at least one digest")
	}
	return nil
}

func validateInputArtifactsBinding(binding AttachmentBinding, channelKind string) error {
	if !binding.ReadOnly {
		return fmt.Errorf("input_artifacts must be read_only")
	}
	if channelKind != AttachmentChannelArtifactImage && channelKind != AttachmentChannelReadOnlyVolume {
		return fmt.Errorf("input_artifacts channel_kind must be %q or %q", AttachmentChannelArtifactImage, AttachmentChannelReadOnlyVolume)
	}
	if len(binding.RequiredDigests) == 0 {
		return fmt.Errorf("input_artifacts requires at least one digest")
	}
	return nil
}

func validateWorkspaceBinding(binding AttachmentBinding, channelKind string) error {
	if binding.ReadOnly {
		return fmt.Errorf("workspace must be read-write")
	}
	if channelKind != AttachmentChannelWritableVolume {
		return fmt.Errorf("workspace channel_kind must be %q", AttachmentChannelWritableVolume)
	}
	return nil
}

func validateScratchBinding(binding AttachmentBinding, channelKind string) error {
	if binding.ReadOnly {
		return fmt.Errorf("scratch must be read-write")
	}
	if channelKind != AttachmentChannelEphemeralVolume && channelKind != AttachmentChannelWritableVolume {
		return fmt.Errorf("scratch channel_kind must be %q or %q", AttachmentChannelEphemeralVolume, AttachmentChannelWritableVolume)
	}
	if len(binding.RequiredDigests) > 0 {
		return fmt.Errorf("scratch must not include required_digests")
	}
	return nil
}
