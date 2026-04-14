package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type instanceBackendPostureController interface {
	GetInstanceBackendPosture() instanceBackendPostureSnapshot
	ApplyInstanceBackendPosture(targetInstanceID, backendKind string) error
}

type instanceBackendPostureSnapshot struct {
	InstanceID           string
	BackendKind          string
	PreferredBackendKind string
}

type localInstanceBackendPostureController struct {
	mu                   sync.Mutex
	nextID               uint64
	current              instanceBackendPostureSnapshot
	preferredBackendKind string
}

func newLocalInstanceBackendPostureController() instanceBackendPostureController {
	c := &localInstanceBackendPostureController{preferredBackendKind: launcherbackend.BackendKindMicroVM}
	c.current = instanceBackendPostureSnapshot{
		InstanceID:           c.nextInstanceIDLocked(),
		BackendKind:          c.preferredBackendKind,
		PreferredBackendKind: c.preferredBackendKind,
	}
	return c
}

func (c *localInstanceBackendPostureController) nextInstanceIDLocked() string {
	c.nextID++
	return fmt.Sprintf("launcher-instance-%d", c.nextID)
}

func (c *localInstanceBackendPostureController) GetInstanceBackendPosture() instanceBackendPostureSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *localInstanceBackendPostureController) ApplyInstanceBackendPosture(targetInstanceID, backendKind string) error {
	normalized := strings.ToLower(strings.TrimSpace(backendKind))
	if normalized != launcherbackend.BackendKindMicroVM && normalized != launcherbackend.BackendKindContainer {
		return fmt.Errorf("instance backend kind must be %q or %q", launcherbackend.BackendKindMicroVM, launcherbackend.BackendKindContainer)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(targetInstanceID) != "" && targetInstanceID != c.current.InstanceID {
		return fmt.Errorf("instance backend posture target instance mismatch")
	}
	c.current.BackendKind = normalized
	return nil
}

func (s *Service) currentInstanceBackendPosture() instanceBackendPostureSnapshot {
	if s.instancePostureController == nil {
		s.instancePostureController = newLocalInstanceBackendPostureController()
	}
	posture := s.instancePostureController.GetInstanceBackendPosture()
	if strings.TrimSpace(posture.PreferredBackendKind) == "" {
		posture.PreferredBackendKind = launcherbackend.BackendKindMicroVM
	}
	if strings.TrimSpace(posture.BackendKind) == "" {
		posture.BackendKind = posture.PreferredBackendKind
	}
	return posture
}

func (s *Service) applyInstanceBackendPosture(_ context.Context, targetInstanceID, backendKind string) error {
	if s.instancePostureController == nil {
		s.instancePostureController = newLocalInstanceBackendPostureController()
	}
	return s.instancePostureController.ApplyInstanceBackendPosture(targetInstanceID, backendKind)
}
