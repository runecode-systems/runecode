//go:build !linux

package launcherdaemon

import (
	"context"
	"fmt"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type ContainerControllerConfig struct {
	WorkRoot string
	Now      func() time.Time
}

type unsupportedContainerController struct{}

func NewContainerController(ContainerControllerConfig) Controller {
	return unsupportedContainerController{}
}

func (unsupportedContainerController) Launch(context.Context, launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	return nil, fmt.Errorf("container launcher backend is linux-only")
}

func (unsupportedContainerController) Terminate(context.Context, InstanceRef) error { return nil }

func (unsupportedContainerController) GetState(context.Context, InstanceRef) (InstanceState, error) {
	return InstanceState{}, fmt.Errorf("container launcher backend is linux-only")
}

func (unsupportedContainerController) Shutdown(context.Context) error { return nil }
