package launcherdaemon

import (
	"context"
	"fmt"
)

func (s *Service) consumeRuntimeUpdates(ctx context.Context, ref InstanceRef, updateID uint64, updates <-chan RuntimeUpdate) {
	defer func() {
		s.unregisterUpdate(ref, updateID)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if err := s.handleRuntimeUpdate(ref, update); err != nil {
				s.failRuntimeUpdates(ref, updateID)
				return
			}
		}
	}
}

func (s *Service) unregisterUpdate(ref InstanceRef, updateID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	registration, ok := s.updates[instanceKey(ref)]
	if !ok || registration.id != updateID {
		return
	}
	registration.cancel()
	delete(s.updates, instanceKey(ref))
}

func (s *Service) failRuntimeUpdates(ref InstanceRef, updateID uint64) {
	s.mu.Lock()
	current := s.updates[instanceKey(ref)]
	if s.state == StateServing {
		s.state = StateFailed
	}
	s.mu.Unlock()
	if current.id == updateID {
		controller, err := s.controllerForRef(ref)
		if err == nil {
			if terminateErr := controller.Terminate(context.Background(), ref); terminateErr != nil {
				return
			}
		}
	}
}

func (s *Service) handleRuntimeUpdate(ref InstanceRef, update RuntimeUpdate) error {
	if update.RunID == "" {
		update.RunID = ref.RunID
	}
	if update.Facts != nil {
		if err := s.reporter.RecordRuntimeFacts(update.RunID, *update.Facts); err != nil {
			return err
		}
	}
	if update.Lifecycle != nil {
		if err := s.reporter.RecordRuntimeLifecycleState(update.RunID, *update.Lifecycle); err != nil {
			return err
		}
	}
	return nil
}

func instanceKey(ref InstanceRef) string {
	return fmt.Sprintf("%d:%s|%d:%s|%d:%s", len(ref.RunID), ref.RunID, len(ref.StageID), ref.StageID, len(ref.RoleInstanceID), ref.RoleInstanceID)
}
