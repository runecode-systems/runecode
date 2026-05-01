package launcherbackend

import (
	"fmt"
	"strings"
)

const (
	MeasurementProfileUnknown          = "unknown"
	MeasurementProfileMicroVMBootV1    = "microvm-boot-v1"
	MeasurementProfileContainerImageV1 = "container-image-v1"
)

type normalizedMeasurementIdentity struct {
	MeasurementProfile string            `json:"measurement_profile"`
	BootProfile        string            `json:"boot_profile"`
	ComponentDigests   map[string]string `json:"component_digests"`
}

type measurementProfileSpec struct {
	acceptedSourceKinds map[string]struct{}
}

var trustedMeasurementProfiles = map[string]measurementProfileSpec{
	MeasurementProfileMicroVMBootV1: {
		acceptedSourceKinds: map[string]struct{}{
			AttestationSourceKindTrustedRuntime: {},
			AttestationSourceKindTPMQuote:       {},
			AttestationSourceKindSEVSNPReport:   {},
			AttestationSourceKindTDXQuote:       {},
		},
	},
	MeasurementProfileContainerImageV1: {
		acceptedSourceKinds: map[string]struct{}{
			AttestationSourceKindTrustedRuntime: {},
			AttestationSourceKindContainerImage: {},
		},
	},
}

func normalizeMeasurementProfile(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case MeasurementProfileMicroVMBootV1:
		return MeasurementProfileMicroVMBootV1
	case MeasurementProfileContainerImageV1:
		return MeasurementProfileContainerImageV1
	default:
		return MeasurementProfileUnknown
	}
}

func measurementProfileKnown(value string) bool {
	_, ok := trustedMeasurementProfiles[normalizeMeasurementProfile(value)]
	return ok
}

func normalizeExpectedMeasurementDigests(profile string, digests []string) []string {
	if !measurementProfileKnown(profile) {
		return nil
	}
	return uniqueSortedStrings(digests)
}

func validateMeasurementProfileExpectedDigests(profile string, digests []string) error {
	normalizedProfile := normalizeMeasurementProfile(profile)
	if normalizedProfile == MeasurementProfileUnknown {
		return fmt.Errorf("measurement_profile %q is invalid", profile)
	}
	if len(digests) == 0 {
		return fmt.Errorf("expected_measurement_digests must be present for measurement_profile %q", normalizedProfile)
	}
	for i, digest := range digests {
		if !looksLikeDigest(strings.TrimSpace(digest)) {
			return fmt.Errorf("expected_measurement_digests[%d] must be sha256:<64 lowercase hex>", i)
		}
	}
	return nil
}

func measurementProfileAcceptsSourceKind(profile string, sourceKind string) bool {
	spec, ok := trustedMeasurementProfiles[normalizeMeasurementProfile(profile)]
	if !ok {
		return false
	}
	_, ok = spec.acceptedSourceKinds[normalizeAttestationSourceKind(sourceKind)]
	return ok
}

func DeriveExpectedMeasurementDigests(profile string, bootProfile string, componentDigests map[string]string) ([]string, error) {
	normalizedProfile := normalizeMeasurementProfile(profile)
	if normalizedProfile == MeasurementProfileUnknown {
		return nil, fmt.Errorf("measurement_profile %q is invalid", profile)
	}
	digest, err := canonicalSHA256Digest(normalizedMeasurementIdentity{
		MeasurementProfile: normalizedProfile,
		BootProfile:        normalizeBootProfile(bootProfile),
		ComponentDigests:   cloneStringMap(componentDigests),
	}, "expected measurement identity")
	if err != nil {
		return nil, err
	}
	return []string{digest}, nil
}
