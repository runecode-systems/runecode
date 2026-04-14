package launcherdaemon

import (
	"context"
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
	// Controller is a legacy single-controller override used by tests.
	// When set, it is used for all backend kinds.
	Controller Controller

	// Optional backend-specific controller overrides.
	MicroVMController   Controller
	ContainerController Controller
	Reporter            RuntimeReporter
}

type Service struct {
	microVMController   Controller
	containerController Controller
	reporter            RuntimeReporter

	mu                   sync.RWMutex
	state                State
	instanceID           string
	nextInstanceID       uint64
	preferredBackendKind string
	instanceBackendKind  string
	nextUpdateID         uint64
	updates              map[string]updateRegistration
	instanceBackendByKey map[string]string
}

type InstanceBackendPosture struct {
	InstanceID           string
	BackendKind          string
	PreferredBackendKind string
}

type updateRegistration struct {
	id     uint64
	cancel context.CancelFunc
}
