//go:build !linux

package launcherdaemon

import (
	"context"
	"fmt"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type QEMUControllerConfig struct {
	QEMUBinary string
	KernelPath string
	WorkRoot   string
	Now        func() time.Time
}

type unsupportedController struct{}

func NewQEMUController(QEMUControllerConfig) Controller { return unsupportedController{} }

func (unsupportedController) Launch(context.Context, launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	return nil, fmt.Errorf("qemu launcher backend is linux-only")
}

func (unsupportedController) Terminate(context.Context, InstanceRef) error { return nil }

func (unsupportedController) GetState(context.Context, InstanceRef) (InstanceState, error) {
	return InstanceState{}, fmt.Errorf("qemu launcher backend is linux-only")
}

func (unsupportedController) Shutdown(context.Context) error { return nil }
