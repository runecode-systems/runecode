package launcherdaemon

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Service) mintInstanceIDLocked() string {
	s.nextInstanceID++
	return fmt.Sprintf("launcher-instance-%d", s.nextInstanceID)
}

func (s *Service) ActiveInstanceID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.instanceID
}

func (s *Service) InstanceBackendKind() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.instanceBackendKind
}

func (s *Service) SetInstanceBackendKind(backendKind string) error {
	normalized := normalizeInstanceBackendKind(backendKind)
	if normalized == launcherbackend.BackendKindUnknown {
		return fmt.Errorf("instance backend kind must be %q or %q", launcherbackend.BackendKindMicroVM, launcherbackend.BackendKindContainer)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateServing {
		return fmt.Errorf("launcher service cannot update instance backend posture from state %s", s.state)
	}
	s.instanceBackendKind = normalized
	return nil
}

func (s *Service) GetInstanceBackendPosture() InstanceBackendPosture {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return InstanceBackendPosture{
		InstanceID:           s.instanceID,
		BackendKind:          s.instanceBackendKind,
		PreferredBackendKind: s.preferredBackendKind,
	}
}

func (s *Service) ApplyInstanceBackendPosture(targetInstanceID, backendKind string) error {
	normalized := normalizeInstanceBackendKind(backendKind)
	if normalized == launcherbackend.BackendKindUnknown {
		return fmt.Errorf("instance backend kind must be %q or %q", launcherbackend.BackendKindMicroVM, launcherbackend.BackendKindContainer)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateServing {
		return fmt.Errorf("launcher service cannot update instance backend posture from state %s", s.state)
	}
	if targetInstanceID != "" && targetInstanceID != s.instanceID {
		return fmt.Errorf("instance backend posture target instance mismatch")
	}
	s.instanceBackendKind = normalized
	return nil
}
