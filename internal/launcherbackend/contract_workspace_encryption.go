package launcherbackend

import "fmt"

func (p WorkspaceEncryptionPosture) Normalized() WorkspaceEncryptionPosture {
	out := p
	out.AtRestProtection = normalizeWorkspaceAtRestProtection(out.AtRestProtection)
	out.KeyProtectionPosture = normalizeWorkspaceKeyProtectionPosture(out.KeyProtectionPosture)
	out.DegradedReasons = uniqueSortedStrings(out.DegradedReasons)
	out.EvidenceRefs = uniqueSortedStrings(out.EvidenceRefs)
	return out
}

func (p WorkspaceEncryptionPosture) Validate() error {
	normalized := p.Normalized()
	if err := validateWorkspaceEncryptionEnumValues(normalized); err != nil {
		return err
	}
	if err := validateWorkspaceEncryptionLeakage(normalized.DegradedReasons, "degraded_reasons"); err != nil {
		return err
	}
	if err := validateWorkspaceEncryptionLeakage(normalized.EvidenceRefs, "evidence_refs"); err != nil {
		return err
	}
	return validateWorkspaceEncryptionConsistency(normalized)
}

func validateWorkspaceEncryptionEnumValues(posture WorkspaceEncryptionPosture) error {
	if posture.AtRestProtection == WorkspaceAtRestProtectionUnknown {
		return fmt.Errorf("at_rest_protection must be %q", WorkspaceAtRestProtectionHostManagedEncryption)
	}
	if posture.KeyProtectionPosture == WorkspaceKeyProtectionUnknown {
		return fmt.Errorf("key_protection_posture must be one of %q, %q, or %q", WorkspaceKeyProtectionHardwareBacked, WorkspaceKeyProtectionOSKeystore, WorkspaceKeyProtectionExplicitDevOptIn)
	}
	return nil
}

func validateWorkspaceEncryptionLeakage(values []string, fieldName string) error {
	for _, value := range values {
		if looksLikeHostPath(value) {
			return fmt.Errorf("%s must not include host-local path material", fieldName)
		}
		if looksLikeDeviceNumberingMaterial(value) {
			return fmt.Errorf("%s must not include device numbering material", fieldName)
		}
	}
	return nil
}

func validateWorkspaceEncryptionConsistency(posture WorkspaceEncryptionPosture) error {
	if posture.Required {
		if posture.AtRestProtection != WorkspaceAtRestProtectionHostManagedEncryption {
			return fmt.Errorf("required encryption requires at_rest_protection=%q", WorkspaceAtRestProtectionHostManagedEncryption)
		}
		if !posture.Effective {
			return fmt.Errorf("required encryption must be effective (fail-closed)")
		}
	}
	if posture.Effective && posture.AtRestProtection != WorkspaceAtRestProtectionHostManagedEncryption {
		return fmt.Errorf("effective encryption requires at_rest_protection=%q", WorkspaceAtRestProtectionHostManagedEncryption)
	}
	if !posture.Effective && len(posture.DegradedReasons) == 0 {
		return fmt.Errorf("effective=false requires degraded_reasons")
	}
	return nil
}
