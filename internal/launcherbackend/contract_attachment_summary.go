package launcherbackend

import (
	"fmt"
	"sort"
	"strings"
)

func (s AttachmentPlanSummary) Normalized() AttachmentPlanSummary {
	out := s
	if len(out.Roles) > 1 {
		sort.Slice(out.Roles, func(i, j int) bool {
			return strings.TrimSpace(out.Roles[i].Role) < strings.TrimSpace(out.Roles[j].Role)
		})
	}
	for idx := range out.Roles {
		out.Roles[idx].Role = strings.TrimSpace(out.Roles[idx].Role)
		out.Roles[idx].ChannelKind = normalizeAttachmentChannelKind(out.Roles[idx].ChannelKind)
		if out.Roles[idx].DigestCount < 0 {
			out.Roles[idx].DigestCount = 0
		}
	}
	return out
}

func (s AttachmentPlanSummary) Validate() error {
	normalized := s.Normalized()
	if err := normalized.Constraints.Validate(); err != nil {
		return fmt.Errorf("constraints: %w", err)
	}
	if len(normalized.Roles) == 0 {
		return fmt.Errorf("roles is required")
	}
	return validateAttachmentPlanSummaryRoles(normalized.Roles)
}

func validateAttachmentPlanSummaryRoles(roles []AttachmentRoleSummary) error {
	seen := map[string]struct{}{}
	for _, role := range roles {
		switch role.Role {
		case AttachmentRoleLaunchContext, AttachmentRoleWorkspace, AttachmentRoleInputArtifacts, AttachmentRoleScratch:
		default:
			return fmt.Errorf("unsupported role %q", role.Role)
		}
		if _, ok := seen[role.Role]; ok {
			return fmt.Errorf("role %q appears more than once", role.Role)
		}
		seen[role.Role] = struct{}{}
		if role.ChannelKind == "" {
			return fmt.Errorf("role %q channel_kind is required", role.Role)
		}
		if looksLikeDeviceNumberingMaterial(role.ChannelKind) {
			return fmt.Errorf("role %q channel_kind must not include device numbering", role.Role)
		}
	}
	return nil
}
