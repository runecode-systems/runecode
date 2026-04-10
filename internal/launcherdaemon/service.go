package launcherdaemon

import (
	"context"
	"fmt"
	"sync"
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

// Controller is launcher-owned runtime backend control.
//
// It intentionally does not expose ad-hoc runner-facing transport APIs.
// Broker remains the authoritative public local API boundary.
type Controller interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type NoopController struct{}

func (NoopController) Start(context.Context) error { return nil }
func (NoopController) Stop(context.Context) error  { return nil }

type Config struct {
	Controller Controller
}

type Service struct {
	controller Controller

	mu    sync.RWMutex
	state State
}

func New(cfg Config) (*Service, error) {
	controller := cfg.Controller
	if controller == nil {
		controller = NoopController{}
	}
	return &Service{controller: controller, state: StatePlanned}, nil
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

	if err := s.controller.Start(ctx); err != nil {
		s.mu.Lock()
		s.state = StateFailed
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.state = StateServing
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
	s.mu.Unlock()

	if err := s.controller.Stop(ctx); err != nil {
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
