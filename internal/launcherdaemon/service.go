package launcherdaemon

import (
	"context"
	"fmt"
	"sync"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type State string

const (
	StatePlanned  State = "planned"
	StateStarting State = "starting"
	StateServing  State = "serving"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StateFailed   State = "failed"
)

// Controller is the private trusted broker->launcher control contract.
//
// It intentionally excludes ad-hoc runner-facing transport APIs.
// Broker remains the only public/untrusted API boundary.
type Controller interface {
	Launch(context.Context, launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error)
	Terminate(context.Context, InstanceRef) error
	GetState(context.Context, InstanceRef) (InstanceState, error)
	Shutdown(context.Context) error
}

type RuntimeReporter interface {
	RecordRuntimeFacts(runID string, facts launcherbackend.RuntimeFactsSnapshot) error
	RecordRuntimeLifecycleState(runID string, lifecycle launcherbackend.RuntimeLifecycleState) error
}

type RuntimeUpdate struct {
	RunID     string
	Facts     *launcherbackend.RuntimeFactsSnapshot
	Lifecycle *launcherbackend.RuntimeLifecycleState
}

type InstanceRef struct {
	RunID          string
	StageID        string
	RoleInstanceID string
}

type InstanceState struct {
	Ref            InstanceRef
	LifecycleState launcherbackend.RuntimeLifecycleState
	Active         bool
	HelloWorldSeen bool
	LastError      string
}

type discardReporter struct{}

func (discardReporter) RecordRuntimeFacts(string, launcherbackend.RuntimeFactsSnapshot) error {
	return nil
}

func (discardReporter) RecordRuntimeLifecycleState(string, launcherbackend.RuntimeLifecycleState) error {
	return nil
}

type Config struct {
	Controller Controller
	Reporter   RuntimeReporter
}

type Service struct {
	controller Controller
	reporter   RuntimeReporter

	mu                   sync.RWMutex
	state                State
	preferredBackendKind string
	instanceBackendKind  string
	nextUpdateID         uint64
	updates              map[string]updateRegistration
}

type updateRegistration struct {
	id     uint64
	cancel context.CancelFunc
}

func New(cfg Config) (*Service, error) {
	controller := cfg.Controller
	if controller == nil {
		controller = NewQEMUController(QEMUControllerConfig{})
	}
	reporter := cfg.Reporter
	if reporter == nil {
		reporter = discardReporter{}
	}
	return &Service{
		controller:           controller,
		reporter:             reporter,
		state:                StatePlanned,
		preferredBackendKind: launcherbackend.BackendKindMicroVM,
		instanceBackendKind:  launcherbackend.BackendKindMicroVM,
		updates:              map[string]updateRegistration{},
	}, nil
}

func (s *Service) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.state != StatePlanned && s.state != StateStopped {
		state := s.state
		s.mu.Unlock()
		return fmt.Errorf("launcher service cannot start from state %s", state)
	}
	s.state = StateStarting
	s.mu.Unlock()

	s.mu.Lock()
	s.state = StateServing
	s.instanceBackendKind = s.preferredBackendKind
	s.mu.Unlock()
	return nil
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

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.state != StateServing && s.state != StateFailed {
		state := s.state
		s.mu.Unlock()
		return fmt.Errorf("launcher service cannot stop from state %s", state)
	}
	s.state = StateStopping
	cancels := make([]context.CancelFunc, 0, len(s.updates))
	for key, registration := range s.updates {
		cancels = append(cancels, registration.cancel)
		delete(s.updates, key)
	}
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}

	if err := s.controller.Shutdown(ctx); err != nil {
		s.mu.Lock()
		s.state = StateFailed
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.state = StateStopped
	s.mu.Unlock()
	return nil
}

func (s *Service) Launch(ctx context.Context, spec launcherbackend.BackendLaunchSpec) (InstanceRef, error) {
	spec = s.specWithInstanceBackendPosture(spec)
	if err := validateContainerRoleScope(spec); err != nil {
		return InstanceRef{}, err
	}
	if err := spec.Validate(); err != nil {
		return InstanceRef{}, err
	}
	s.mu.RLock()
	if s.state != StateServing {
		state := s.state
		s.mu.RUnlock()
		return InstanceRef{}, fmt.Errorf("launcher service cannot launch from state %s", state)
	}
	s.mu.RUnlock()

	updates, err := s.controller.Launch(ctx, spec)
	if err != nil {
		return InstanceRef{}, err
	}
	ref := InstanceRef{RunID: spec.RunID, StageID: spec.StageID, RoleInstanceID: spec.RoleInstanceID}
	updateCtx, cancel := context.WithCancel(context.Background())

	key := instanceKey(ref)
	s.mu.Lock()
	s.nextUpdateID++
	registration := updateRegistration{id: s.nextUpdateID, cancel: cancel}
	if existing, ok := s.updates[key]; ok {
		existing.cancel()
	}
	s.updates[key] = registration
	s.mu.Unlock()

	go s.consumeRuntimeUpdates(updateCtx, ref, registration.id, updates)
	return ref, nil
}

func (s *Service) Terminate(ctx context.Context, ref InstanceRef) error {
	if err := s.controller.Terminate(ctx, ref); err != nil {
		return err
	}
	s.mu.Lock()
	if registration, ok := s.updates[instanceKey(ref)]; ok {
		registration.cancel()
		delete(s.updates, instanceKey(ref))
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) GetState(ctx context.Context, ref InstanceRef) (InstanceState, error) {
	return s.controller.GetState(ctx, ref)
}
