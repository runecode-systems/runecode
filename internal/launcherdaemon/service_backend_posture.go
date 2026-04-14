package launcherdaemon

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func validateContainerRoleScope(spec launcherbackend.BackendLaunchSpec) error {
	if normalizeInstanceBackendKind(spec.RequestedBackend) != launcherbackend.BackendKindContainer {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(spec.RoleFamily), "workspace") {
		return nil
	}
	return fmt.Errorf("container backend v0 only supports role_family=workspace (offline workspace role launches)")
}

func (s *Service) specWithInstanceBackendPosture(spec launcherbackend.BackendLaunchSpec) launcherbackend.BackendLaunchSpec {
	s.mu.RLock()
	backendKind := s.instanceBackendKind
	s.mu.RUnlock()
	normalized := normalizeInstanceBackendKind(backendKind)
	if normalized == launcherbackend.BackendKindUnknown {
		normalized = launcherbackend.BackendKindMicroVM
	}
	spec.RequestedBackend = normalized
	return spec
}

func normalizeInstanceBackendKind(backendKind string) string {
	switch strings.ToLower(strings.TrimSpace(backendKind)) {
	case launcherbackend.BackendKindMicroVM:
		return launcherbackend.BackendKindMicroVM
	case launcherbackend.BackendKindContainer:
		return launcherbackend.BackendKindContainer
	default:
		return launcherbackend.BackendKindUnknown
	}
}
