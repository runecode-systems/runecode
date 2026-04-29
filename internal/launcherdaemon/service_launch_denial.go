package launcherdaemon

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Service) recordLaunchDeniedRuntimeFacts(spec launcherbackend.BackendLaunchSpec, launchErr error) error {
	return s.reporter.RecordRuntimeFacts(spec.RunID, buildLaunchDeniedRuntimeFacts(spec, launchErr))
}

func launchDeniedErrorWithReportingContext(launchErr, reportErr error) error {
	if launchErr == nil {
		return reportErr
	}
	if reportErr == nil {
		return launchErr
	}
	return fmt.Errorf("launch denied: %w (denied-launch runtime-facts reporting failed: %v)", launchErr, reportErr)
}

func buildLaunchDeniedRuntimeFacts(spec launcherbackend.BackendLaunchSpec, launchErr error) launcherbackend.RuntimeFactsSnapshot {
	facts := launcherbackend.DefaultRuntimeFacts(spec.RunID)
	receipt := facts.LaunchReceipt
	receipt.RunID = spec.RunID
	receipt.StageID = spec.StageID
	receipt.RoleInstanceID = spec.RoleInstanceID
	receipt.RoleFamily = spec.RoleFamily
	receipt.RoleKind = spec.RoleKind
	receipt.BackendKind = spec.RequestedBackend
	receipt.RuntimeImageDescriptorDigest = spec.Image.DescriptorDigest
	receipt.RuntimeImageBootProfile = spec.Image.BootContractVersion
	receipt.BootComponentDigestByName = cloneStringMapForService(spec.Image.ComponentDigests)
	receipt.LaunchFailureReasonCode = launchFailureReasonCode(launchErr)
	receipt.Lifecycle = &launcherbackend.BackendLifecycleSnapshot{
		CurrentState:          launcherbackend.BackendLifecycleStateTerminated,
		PreviousState:         launcherbackend.BackendLifecycleStateLaunching,
		TerminateBetweenSteps: spec.LifecyclePolicy.TerminateBetweenSteps,
		TransitionCount:       1,
	}
	applyDeniedLaunchSigningState(&receipt, spec.Image.Signing)
	facts.LaunchReceipt = receipt
	return facts
}

func applyDeniedLaunchSigningState(receipt *launcherbackend.BackendLaunchReceipt, signing *launcherbackend.RuntimeImageSigningHooks) {
	if receipt == nil || signing == nil {
		return
	}
	receipt.RuntimeImageSignerRef = signing.SignerRef
	receipt.RuntimeImageVerifierRef = signing.VerifierSetRef
	receipt.RuntimeImageSignatureDigest = signing.SignatureDigest
	if signing.Toolchain == nil {
		return
	}
	receipt.RuntimeToolchainDescriptorDigest = signing.Toolchain.DescriptorDigest
	receipt.RuntimeToolchainSignerRef = signing.Toolchain.SignerRef
	receipt.RuntimeToolchainVerifierRef = signing.Toolchain.VerifierSetRef
	receipt.RuntimeToolchainSignatureDigest = signing.Toolchain.SignatureDigest
}

func launchFailureReasonCode(err error) string {
	text := strings.TrimSpace(strings.ToLower(errString(err)))
	for _, code := range launchFailureReasonCodes() {
		if strings.Contains(text, code) {
			return code
		}
	}
	return launcherbackend.BackendErrorCodeHypervisorLaunchFailed
}

func launchFailureReasonCodes() []string {
	return []string{
		launcherbackend.BackendErrorCodeAccelerationUnavailable,
		launcherbackend.BackendErrorCodeHypervisorLaunchFailed,
		launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch,
		launcherbackend.BackendErrorCodeAttachmentPlanInvalid,
		launcherbackend.BackendErrorCodeHandshakeFailed,
		launcherbackend.BackendErrorCodeReplayDetected,
		launcherbackend.BackendErrorCodeSessionBindingMismatch,
		launcherbackend.BackendErrorCodeGuestUnresponsive,
		launcherbackend.BackendErrorCodeWatchdogTimeout,
		launcherbackend.BackendErrorCodeRequiredHardeningUnavailable,
		launcherbackend.BackendErrorCodeRequiredDiskEncryptionUnavailable,
		launcherbackend.BackendErrorCodeContainerAutomaticFallbackDisallowed,
		launcherbackend.BackendErrorCodeContainerOptInRequired,
		launcherbackend.BackendErrorCodeTerminalReportInvalid,
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func cloneStringMapForService(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
