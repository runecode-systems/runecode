package launcherbackend

func attestationMeasurementIdentityMatchesEvidence(attestation *IsolateAttestationEvidence) bool {
	if attestation == nil || attestation.EvidenceClaimsDigest == "" {
		return false
	}
	expectedDigests, err := DeriveExpectedMeasurementDigests(attestation.MeasurementProfile, attestation.RuntimeImageBootProfile, bootComponentDigestByNameForEvidence(attestation))
	if err != nil {
		return false
	}
	for _, digest := range expectedDigests {
		if digest == attestation.EvidenceClaimsDigest {
			return true
		}
	}
	return false
}

func bootComponentDigestByNameForEvidence(attestation *IsolateAttestationEvidence) map[string]string {
	if attestation == nil {
		return nil
	}
	componentNames := requiredBootProfileComponentNames(attestation.RuntimeImageBootProfile)
	if len(componentNames) == 0 || len(componentNames) != len(attestation.BootComponentDigests) {
		return nil
	}
	components := map[string]string{}
	for i, name := range componentNames {
		components[name] = attestation.BootComponentDigests[i]
	}
	return components
}

func requiredBootProfileComponentNames(bootProfile string) []string {
	switch normalizeBootProfile(bootProfile) {
	case BootProfileMicroVMLinuxKernelInitrdV1:
		return []string{"kernel", "initrd"}
	case BootProfileContainerOCIImageV1:
		return []string{"image"}
	default:
		return nil
	}
}

func containsAnyReasonCode(values []string, expected ...string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		for _, match := range expected {
			if value == match {
				return true
			}
		}
	}
	return false
}
