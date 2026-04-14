// Command runecode-launcher provides trusted isolate-session validation helpers.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/launcherdaemon"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var u *usageError
		if errors.As(err, &u) {
			fmt.Fprintln(os.Stderr, u.Error())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

type commandHandler func([]string, io.Writer) error

type serveConfig struct {
	once       bool
	helloWorld bool
	storeRoot  string
	ledgerRoot string
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		return writeHelp(stdout)
	}
	handler, ok := commandHandlers()[args[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
	return handler(args[1:], stdout)
}

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"serve":                    handleServe,
		"validate-isolate-binding": handleValidateIsolateBinding,
	}
}

func handleServe(args []string, stdout io.Writer) error {
	cfg, err := parseServeConfig(args)
	if err != nil {
		return err
	}
	svcCfg, brokerSvc, cleanup, err := buildServeServiceConfig(cfg)
	if err != nil {
		return err
	}
	defer cleanup()
	svc, err := launcherdaemon.New(svcCfg)
	if err != nil {
		return err
	}
	if err := svc.Start(context.Background()); err != nil {
		return err
	}
	stopped := false
	defer func() {
		if !stopped {
			_ = svc.Stop(context.Background())
		}
	}()
	if err := maybeRunHelloWorldSlice(svc, brokerSvc, cfg, stdout); err != nil {
		return err
	}
	if cfg.once {
		if err := svc.Stop(context.Background()); err != nil {
			return err
		}
		stopped = true
		_, err = fmt.Fprintln(stdout, "launcher service started and stopped")
		return err
	}
	if _, err := fmt.Fprintln(stdout, "launcher service running"); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	err = svc.Stop(context.Background())
	stopped = true
	return err
}

func parseServeConfig(args []string) (serveConfig, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	once := fs.Bool("once", false, "start and stop service immediately (test/validation)")
	helloWorld := fs.Bool("hello-world", false, "run deterministic qemu hello-world vertical slice")
	storeRoot := fs.String("store-root", "", "broker runtime store root (optional)")
	ledgerRoot := fs.String("ledger-root", "", "broker audit ledger root (optional)")
	if err := fs.Parse(args); err != nil {
		return serveConfig{}, &usageError{message: "serve usage: runecode-launcher serve [--once] [--hello-world] [--store-root path] [--ledger-root path]"}
	}
	return serveConfig{once: *once, helloWorld: *helloWorld, storeRoot: *storeRoot, ledgerRoot: *ledgerRoot}, nil
}

func buildServeServiceConfig(cfg serveConfig) (launcherdaemon.Config, *brokerapi.Service, func(), error) {
	if !cfg.helloWorld {
		return launcherdaemon.Config{}, nil, func() {}, nil
	}
	storeRoot, ledgerRoot, cleanup, err := resolveServeRoots(cfg)
	if err != nil {
		return launcherdaemon.Config{}, nil, func() {}, err
	}
	brokerSvc, err := brokerapi.NewService(storeRoot, ledgerRoot)
	if err != nil {
		cleanup()
		return launcherdaemon.Config{}, nil, func() {}, err
	}
	return launcherdaemon.Config{Reporter: brokerSvc}, brokerSvc, cleanup, nil
}

func resolveServeRoots(cfg serveConfig) (string, string, func(), error) {
	if cfg.storeRoot != "" && cfg.ledgerRoot != "" {
		return cfg.storeRoot, cfg.ledgerRoot, func() {}, nil
	}
	base, err := os.MkdirTemp("", "runecode-launcher-cli-")
	if err != nil {
		return "", "", func() {}, err
	}
	cleanup := func() {
		_ = os.RemoveAll(base)
	}
	storeRoot := cfg.storeRoot
	ledgerRoot := cfg.ledgerRoot
	if storeRoot == "" {
		storeRoot = filepath.Join(base, "store")
	}
	if ledgerRoot == "" {
		ledgerRoot = filepath.Join(base, "ledger")
	}
	return storeRoot, ledgerRoot, cleanup, nil
}

func maybeRunHelloWorldSlice(svc *launcherdaemon.Service, brokerSvc *brokerapi.Service, cfg serveConfig, stdout io.Writer) error {
	if !cfg.helloWorld {
		return nil
	}
	if brokerSvc == nil {
		return fmt.Errorf("hello-world reporter unavailable")
	}
	runID := fmt.Sprintf("launcher-cli-hello-%d", time.Now().Unix())
	ref, err := svc.Launch(context.Background(), helloWorldLaunchSpec(runID))
	if err != nil {
		return err
	}
	if err := waitForTerminalReport(brokerSvc, ref.RunID, 45*time.Second); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "launcher hello-world completed")
	return err
}

func handleValidateIsolateBinding(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-isolate-binding", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to isolate session binding JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-isolate-binding usage: runecode-launcher validate-isolate-binding --file binding.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-isolate-binding requires --file"}
	}
	binding, err := loadIsolateSessionBinding(*filePath)
	if err != nil {
		return err
	}
	if err := trustpolicy.ValidateIsolateSessionBinding(binding); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "valid")
	return err
}

func waitForTerminalReport(svc *brokerapi.Service, runID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		facts := svc.RuntimeFacts(runID)
		if facts.TerminalReport != nil {
			if facts.TerminalReport.TerminationKind == launcherbackend.BackendTerminationKindCompleted {
				return nil
			}
			return fmt.Errorf("hello-world run failed: %s", facts.TerminalReport.FailureReasonCode)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("hello-world run timed out waiting for terminal report")
}

func helloWorldLaunchSpec(runID string) launcherbackend.BackendLaunchSpec {
	return launcherbackend.BackendLaunchSpec{
		RunID:                     runID,
		StageID:                   "stage-1",
		RoleInstanceID:            "role-hello-world",
		RoleFamily:                "system",
		RoleKind:                  "hello_world",
		RequestedBackend:          launcherbackend.BackendKindMicroVM,
		RequestedAccelerationKind: launcherbackend.AccelerationKindKVM,
		ControlTransportKind:      launcherbackend.TransportKindVirtioSerial,
		Image: launcherbackend.RuntimeImageDescriptor{
			DescriptorDigest:    "sha256:" + repeatHex('a'),
			BackendKind:         launcherbackend.BackendKindMicroVM,
			BootContractVersion: "v1",
			PlatformCompatibility: launcherbackend.RuntimeImagePlatformCompat{
				OS: "linux", Architecture: "amd64", AccelerationKind: launcherbackend.AccelerationKindKVM,
			},
			ComponentDigests: map[string]string{"kernel": "sha256:" + repeatHex('b'), "rootfs": "sha256:" + repeatHex('c')},
			Signing:          &launcherbackend.RuntimeImageSigningHooks{SignerRef: "launcher-cli", SignatureDigest: "sha256:" + repeatHex('d')},
		},
		Attachments: launcherbackend.AttachmentPlan{
			ByRole: map[string]launcherbackend.AttachmentBinding{
				launcherbackend.AttachmentRoleLaunchContext:  {ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelReadOnlyVolume, RequiredDigests: []string{"sha256:" + repeatHex('e')}},
				launcherbackend.AttachmentRoleWorkspace:      {ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelWritableVolume},
				launcherbackend.AttachmentRoleInputArtifacts: {ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelArtifactImage, RequiredDigests: []string{"sha256:" + repeatHex('f')}},
				launcherbackend.AttachmentRoleScratch:        {ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelEphemeralVolume},
			},
			Constraints: launcherbackend.AttachmentRealizationConstraints{NoHostFilesystemMounts: true},
			WorkspaceEncryption: &launcherbackend.WorkspaceEncryptionPosture{
				Required:             true,
				AtRestProtection:     launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption,
				KeyProtectionPosture: launcherbackend.WorkspaceKeyProtectionOSKeystore,
				Effective:            true,
			},
		},
		ResourceLimits:  launcherbackend.BackendResourceLimits{VCPUCount: 1, MemoryMiB: 256, DiskMiB: 128, LaunchTimeoutSeconds: 30, BindTimeoutSeconds: 10, ActiveTimeoutSeconds: 25, TerminationGraceSeconds: 2},
		WatchdogPolicy:  launcherbackend.BackendWatchdogPolicy{Enabled: true, TerminateOnMisbehavior: true, HeartbeatTimeoutSeconds: 5, NoProgressTimeoutSeconds: 20},
		LifecyclePolicy: launcherbackend.BackendLifecyclePolicy{TerminateBetweenSteps: true},
		CachePosture:    launcherbackend.BackendCachePosture{ResetOrDestroyBeforeReuse: true, DigestPinned: true, SignaturePinned: true},
	}
}

func repeatHex(ch byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func loadIsolateSessionBinding(filePath string) (trustpolicy.IsolateSessionBinding, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return trustpolicy.IsolateSessionBinding{}, err
	}
	binding := trustpolicy.IsolateSessionBinding{}
	if err := json.Unmarshal(b, &binding); err != nil {
		return trustpolicy.IsolateSessionBinding{}, err
	}
	return binding, nil
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-launcher <command> [flags]

Commands:
  serve [--once] [--hello-world] [--store-root path] [--ledger-root path]
  validate-isolate-binding --file binding.json`)
	return err
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
