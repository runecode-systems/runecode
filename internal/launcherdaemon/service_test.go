package launcherdaemon

import (
	"context"
	"errors"
	"testing"
)

type fakeController struct {
	startErr error
	stopErr  error

	starts int
	stops  int
}

func (f *fakeController) Start(context.Context) error {
	f.starts++
	return f.startErr
}

func (f *fakeController) Stop(context.Context) error {
	f.stops++
	return f.stopErr
}

func TestServiceStartStopLifecycle(t *testing.T) {
	controller := &fakeController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if got := svc.State(); got != StatePlanned {
		t.Fatalf("initial state = %s, want %s", got, StatePlanned)
	}

	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if got := svc.State(); got != StateServing {
		t.Fatalf("state after start = %s, want %s", got, StateServing)
	}
	if controller.starts != 1 {
		t.Fatalf("controller starts = %d, want 1", controller.starts)
	}

	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if got := svc.State(); got != StateStopped {
		t.Fatalf("state after stop = %s, want %s", got, StateStopped)
	}
	if controller.stops != 1 {
		t.Fatalf("controller stops = %d, want 1", controller.stops)
	}
}

func TestServiceStartFailureMarksFailed(t *testing.T) {
	svc, err := New(Config{Controller: &fakeController{startErr: errors.New("boom")}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err == nil {
		t.Fatal("Start expected error")
	}
	if got := svc.State(); got != StateFailed {
		t.Fatalf("state after failed start = %s, want %s", got, StateFailed)
	}
}

func TestServiceStopFromInvalidStateFails(t *testing.T) {
	svc, err := New(Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Stop(context.Background()); err == nil {
		t.Fatal("Stop expected invalid-state error")
	}
}
