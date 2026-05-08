package launcherdaemon

import (
	"context"
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func New(cfg Config) (*Service, error) {
	microVMController, containerController := resolveControllers(cfg)
	reporter := cfg.Reporter
	if reporter == nil {
		reporter = discardReporter{}
	}
	return &Service{
		microVMController:    microVMController,
		containerController:  containerController,
		reporter:             reporter,
		state:                StatePlanned,
		preferredBackendKind: launcherbackend.BackendKindMicroVM,
		instanceBackendKind:  launcherbackend.BackendKindMicroVM,
		updates:              map[string]updateRegistration{},
		instanceBackendByKey: map[string]string{},
	}, nil
}

func resolveControllers(cfg Config) (Controller, Controller) {
	if cfg.Controller != nil {
		return cfg.Controller, cfg.Controller
	}
	runtimeMaterialProvider := cfg.RuntimePostHandshakeMaterialProvider
	if runtimeMaterialProvider == nil {
		runtimeMaterialProvider = defaultRuntimePostHandshakeMaterialProvider
	}
	microVMController := cfg.MicroVMController
	if microVMController == nil {
		microVMController = NewQEMUController(QEMUControllerConfig{
			WorkRoot:                             cfg.WorkRoot,
			RuntimePostHandshakeMaterialProvider: runtimeMaterialProvider,
		})
	}
	containerController := cfg.ContainerController
	if containerController == nil {
		containerController = NewContainerController(ContainerControllerConfig{
			WorkRoot:                             cfg.WorkRoot,
			RuntimePostHandshakeMaterialProvider: runtimeMaterialProvider,
		})
	}
	return microVMController, containerController
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
	s.instanceID = s.mintInstanceIDLocked()
	s.instanceBackendKind = s.preferredBackendKind
	s.mu.Unlock()
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

	if err := s.shutdownControllers(ctx); err != nil {
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
	effectiveSpec, err := s.prepareLaunchSpec(spec)
	if err != nil {
		return InstanceRef{}, err
	}
	controller, err := s.controllerForBackend(effectiveSpec.RequestedBackend)
	if err != nil {
		return InstanceRef{}, err
	}
	updates, err := controller.Launch(ctx, effectiveSpec)
	if err != nil {
		if reportErr := s.recordLaunchDeniedRuntimeFacts(effectiveSpec, err); reportErr != nil {
			return InstanceRef{}, launchDeniedErrorWithReportingContext(err, reportErr)
		}
		return InstanceRef{}, err
	}
	ref := InstanceRef{RunID: effectiveSpec.RunID, StageID: effectiveSpec.StageID, RoleInstanceID: effectiveSpec.RoleInstanceID}
	updateCtx, cancel := context.WithCancel(context.Background())
	registration := s.registerLaunchUpdate(ref, cancel, effectiveSpec.RequestedBackend)

	go s.consumeRuntimeUpdates(updateCtx, ref, registration.id, updates)
	return ref, nil
}

func (s *Service) Terminate(ctx context.Context, ref InstanceRef) error {
	controller, err := s.controllerForRef(ref)
	if err != nil {
		return err
	}
	if err := controller.Terminate(ctx, ref); err != nil {
		return err
	}
	s.mu.Lock()
	if registration, ok := s.updates[instanceKey(ref)]; ok {
		registration.cancel()
		delete(s.updates, instanceKey(ref))
	}
	delete(s.instanceBackendByKey, instanceKey(ref))
	s.mu.Unlock()
	return nil
}

func (s *Service) GetState(ctx context.Context, ref InstanceRef) (InstanceState, error) {
	controller, err := s.controllerForRef(ref)
	if err != nil {
		return InstanceState{}, err
	}
	return controller.GetState(ctx, ref)
}

func (s *Service) prepareLaunchSpec(spec launcherbackend.BackendLaunchSpec) (launcherbackend.BackendLaunchSpec, error) {
	effectiveSpec, err := s.effectiveLaunchSpec(spec)
	if err != nil {
		return launcherbackend.BackendLaunchSpec{}, err
	}
	if err := validateContainerRoleScope(effectiveSpec); err != nil {
		return launcherbackend.BackendLaunchSpec{}, err
	}
	if err := effectiveSpec.Validate(); err != nil {
		return launcherbackend.BackendLaunchSpec{}, err
	}
	if err := s.ensureServingState(); err != nil {
		return launcherbackend.BackendLaunchSpec{}, err
	}
	return effectiveSpec, nil
}

func (s *Service) ensureServingState() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state == StateServing {
		return nil
	}
	return fmt.Errorf("launcher service cannot launch from state %s", s.state)
}

func (s *Service) registerLaunchUpdate(ref InstanceRef, cancel context.CancelFunc, backendKind string) updateRegistration {
	key := instanceKey(ref)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextUpdateID++
	registration := updateRegistration{id: s.nextUpdateID, cancel: cancel}
	if existing, ok := s.updates[key]; ok {
		existing.cancel()
	}
	s.updates[key] = registration
	s.instanceBackendByKey[key] = backendKind
	return registration
}

func (s *Service) shutdownControllers(ctx context.Context) error {
	seen := map[Controller]struct{}{}
	for _, controller := range []Controller{s.microVMController, s.containerController} {
		if controller == nil {
			continue
		}
		if _, ok := seen[controller]; ok {
			continue
		}
		seen[controller] = struct{}{}
		if err := controller.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) controllerForBackend(backendKind string) (Controller, error) {
	switch normalizeInstanceBackendKind(backendKind) {
	case launcherbackend.BackendKindMicroVM:
		if s.microVMController == nil {
			return nil, fmt.Errorf("microvm controller unavailable")
		}
		return s.microVMController, nil
	case launcherbackend.BackendKindContainer:
		if s.containerController == nil {
			return nil, fmt.Errorf("container controller unavailable")
		}
		return s.containerController, nil
	default:
		return nil, fmt.Errorf("unsupported backend kind %q", backendKind)
	}
}

func (s *Service) controllerForRef(ref InstanceRef) (Controller, error) {
	key := instanceKey(ref)
	s.mu.RLock()
	backendKind := s.instanceBackendByKey[key]
	s.mu.RUnlock()
	if backendKind == "" {
		return nil, fmt.Errorf("instance backend kind unknown for %s", key)
	}
	return s.controllerForBackend(backendKind)
}
