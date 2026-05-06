package launcherperf

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

const CheckSchemaVersion = "runecode.performance.check.v1"

type HarnessConfig struct{}

func Run(_ HarnessConfig) (perfcontracts.CheckOutput, error) {
	measurements := make([]perfcontracts.MeasurementRecord, 0, 8)

	microVMCold, microVMWarm, err := simulateBackendStartup(launcherbackend.BackendKindMicroVM)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	measurements = append(measurements,
		perfcontracts.MeasurementRecord{MetricID: "metric.launcher.microvm.cold_start.wall_ms", Value: microVMCold, Unit: "ms"},
		perfcontracts.MeasurementRecord{MetricID: "metric.launcher.microvm.warm_start.wall_ms", Value: microVMWarm, Unit: "ms"},
	)

	containerCold, containerWarm, err := simulateBackendStartup(launcherbackend.BackendKindContainer)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	measurements = append(measurements,
		perfcontracts.MeasurementRecord{MetricID: "metric.launcher.container.cold_start.wall_ms", Value: containerCold, Unit: "ms"},
		perfcontracts.MeasurementRecord{MetricID: "metric.launcher.container.warm_start.wall_ms", Value: containerWarm, Unit: "ms"},
	)

	attestCold, attestWarm, err := simulateAttestationPath()
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	measurements = append(measurements,
		perfcontracts.MeasurementRecord{MetricID: "metric.attestation.cold.verify.wall_ms", Value: attestCold, Unit: "ms"},
		perfcontracts.MeasurementRecord{MetricID: "metric.attestation.warm.verify.wall_ms", Value: attestWarm, Unit: "ms"},
	)

	return perfcontracts.CheckOutput{SchemaVersion: CheckSchemaVersion, Measurements: measurements}, nil
}

func simulateBackendStartup(backend string) (float64, float64, error) {
	image, err := deterministicRuntimeImage(backend)
	if err != nil {
		return 0, 0, err
	}
	admissionStart := time.Now()
	record, err := launcherbackend.NewRuntimeAdmissionRecord(image)
	if err != nil {
		return 0, 0, err
	}
	if err := record.Validate(); err != nil {
		return 0, 0, err
	}
	cold := float64(time.Since(admissionStart).Milliseconds())

	warmStart := time.Now()
	if err := record.Validate(); err != nil {
		return 0, 0, err
	}
	warm := float64(time.Since(warmStart).Milliseconds())
	return cold, warm, nil
}

func simulateAttestationPath() (float64, float64, error) {
	image, err := deterministicRuntimeImage(launcherbackend.BackendKindMicroVM)
	if err != nil {
		return 0, 0, err
	}
	receipt := launcherbackend.BackendLaunchReceipt{
		RunID:                              "run-attestation",
		BackendKind:                        launcherbackend.BackendKindMicroVM,
		IsolationAssuranceLevel:            launcherbackend.IsolationAssuranceIsolated,
		ProvisioningPosture:                launcherbackend.ProvisioningPostureAttested,
		RuntimeImageDescriptorDigest:       image.DescriptorDigest,
		RuntimeImageBootProfile:            image.BootContractVersion,
		BootComponentDigestByName:          image.ComponentDigests,
		AttestationEvidenceSourceKind:      launcherbackend.AttestationSourceKindTrustedRuntime,
		AttestationMeasurementProfile:      image.Attestation.MeasurementProfile,
		AttestationEvidenceDigest:          "sha256:" + repeatHex('a'),
		AttestationVerificationResult:      launcherbackend.AttestationVerificationResultValid,
		AttestationReplayVerdict:           launcherbackend.AttestationReplayVerdictOriginal,
		AttestationVerificationDigest:      "sha256:" + repeatHex('b'),
		AttestationVerificationReasonCodes: []string{},
	}

	coldStart := time.Now()
	posture, reasons := launcherbackend.DeriveAttestationPosture(receipt)
	if posture != launcherbackend.AttestationPostureValid || len(reasons) > 0 {
		return 0, 0, fmt.Errorf("unexpected cold attestation posture %q reasons=%v", posture, reasons)
	}
	cold := float64(time.Since(coldStart).Milliseconds())

	warmStart := time.Now()
	posture, reasons = launcherbackend.DeriveAttestationPosture(receipt)
	if posture != launcherbackend.AttestationPostureValid || len(reasons) > 0 {
		return 0, 0, fmt.Errorf("unexpected warm attestation posture %q reasons=%v", posture, reasons)
	}
	warm := float64(time.Since(warmStart).Milliseconds())
	return cold, warm, nil
}

func deterministicRuntimeImage(backend string) (launcherbackend.RuntimeImageDescriptor, error) {
	boot, measurementProfile, accel, componentDigests := runtimeImageBackendParams(backend)
	image := runtimeImageDescriptorBase(backend, boot, accel, componentDigests, measurementProfile)
	if digests, err := launcherbackend.DeriveExpectedMeasurementDigests(measurementProfile, boot, componentDigests); err == nil {
		image.Attestation.ExpectedMeasurementDigests = digests
	}
	digest, err := image.ExpectedDescriptorDigest()
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	image.DescriptorDigest = digest
	image.Signing.PayloadDigest = digest
	return image, nil
}

func runtimeImageBackendParams(backend string) (string, string, string, map[string]string) {
	componentDigests := map[string]string{}
	boot := launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1
	measurementProfile := launcherbackend.MeasurementProfileMicroVMBootV1
	accel := launcherbackend.AccelerationKindKVM
	if backend == launcherbackend.BackendKindContainer {
		boot = launcherbackend.BootProfileContainerOCIImageV1
		measurementProfile = launcherbackend.MeasurementProfileContainerImageV1
		accel = launcherbackend.AccelerationKindNotApplicable
		componentDigests["image"] = "sha256:" + repeatHex('3')
		return boot, measurementProfile, accel, componentDigests
	}
	componentDigests["kernel"] = "sha256:" + repeatHex('1')
	componentDigests["initrd"] = "sha256:" + repeatHex('2')
	return boot, measurementProfile, accel, componentDigests
}

func runtimeImageDescriptorBase(backend, boot, accel string, componentDigests map[string]string, measurementProfile string) launcherbackend.RuntimeImageDescriptor {
	return launcherbackend.RuntimeImageDescriptor{
		BackendKind:         backend,
		BootContractVersion: boot,
		PlatformCompatibility: launcherbackend.RuntimeImagePlatformCompat{
			OS:               "linux",
			Architecture:     "amd64",
			AccelerationKind: accel,
		},
		ComponentDigests: componentDigests,
		Signing: &launcherbackend.RuntimeImageSigningHooks{
			PayloadSchemaID:      launcherbackend.RuntimeImageSignedPayloadSchemaID,
			PayloadSchemaVersion: launcherbackend.RuntimeImageSignedPayloadSchemaVersion,
			PayloadDigest:        "sha256:" + repeatHex('4'),
			SignerRef:            "verifier:runtime-image:v1",
			SignatureDigest:      "sha256:" + repeatHex('5'),
			VerifierSetRef:       "sha256:" + repeatHex('6'),
			Toolchain:            runtimeToolchainSigningHooks(),
		},
		Attestation: &launcherbackend.RuntimeImageAttestationHook{MeasurementProfile: measurementProfile},
	}
}

func runtimeToolchainSigningHooks() *launcherbackend.RuntimeToolchainSigningHooks {
	return &launcherbackend.RuntimeToolchainSigningHooks{
		DescriptorSchemaID:      launcherbackend.RuntimeToolchainDescriptorSchemaID,
		DescriptorSchemaVersion: launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		DescriptorDigest:        "sha256:" + repeatHex('7'),
		SignerRef:               "verifier:runtime-toolchain:v1",
		SignatureDigest:         "sha256:" + repeatHex('8'),
		VerifierSetRef:          "sha256:" + repeatHex('9'),
	}
}

func repeatHex(ch rune) string {
	return strings.Repeat(string(ch), 64)
}
