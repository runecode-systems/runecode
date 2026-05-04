//go:build linux

package launcherdaemon

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestResolveControllersDefaultWiresRuntimePostHandshakeProviderForBothBackends(t *testing.T) {
	micro, container := resolveControllers(Config{WorkRoot: t.TempDir()})

	microController, ok := micro.(*qemuController)
	if !ok {
		t.Fatalf("microvm controller type = %T, want *qemuController", micro)
	}
	if microController.cfg.RuntimePostHandshakeMaterialProvider == nil {
		t.Fatal("microvm default runtime post-handshake material provider must be configured")
	}

	containerController, ok := container.(*containerController)
	if !ok {
		t.Fatalf("container controller type = %T, want *containerController", container)
	}
	if containerController.runtimePostHandshakeMaterialProvider == nil {
		t.Fatal("container default runtime post-handshake material provider must be configured")
	}
}

func TestResolveControllersUsesConfiguredRuntimePostHandshakeProviderForBothBackends(t *testing.T) {
	called := 0
	provider := func(launcherbackend.BackendLaunchSpec, launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
		called += 1
		return &launcherbackend.RuntimePostHandshakeMaterial{}, nil
	}
	micro, container := resolveControllers(Config{WorkRoot: t.TempDir(), RuntimePostHandshakeMaterialProvider: provider})

	microController := micro.(*qemuController)
	if _, err := microController.cfg.RuntimePostHandshakeMaterialProvider(launcherbackend.BackendLaunchSpec{}, launcherbackend.BackendLaunchReceipt{}); err != nil {
		t.Fatalf("microvm runtime material provider returned error: %v", err)
	}

	containerController := container.(*containerController)
	if _, err := containerController.runtimePostHandshakeMaterialProvider(launcherbackend.BackendLaunchSpec{}, launcherbackend.BackendLaunchReceipt{}); err != nil {
		t.Fatalf("container runtime material provider returned error: %v", err)
	}

	if got, want := called, 2; got != want {
		t.Fatalf("provider call count = %d, want %d", got, want)
	}
}
