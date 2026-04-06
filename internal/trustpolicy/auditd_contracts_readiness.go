package trustpolicy

import "fmt"

type AuditStoragePostureEvidence struct {
	EncryptedAtRestDefault     bool   `json:"encrypted_at_rest_default"`
	EncryptedAtRestEffective   bool   `json:"encrypted_at_rest_effective"`
	DevPlaintextOverrideActive bool   `json:"dev_plaintext_override_active"`
	DevPlaintextOverrideReason string `json:"dev_plaintext_override_reason,omitempty"`
	SurfacedToOperator         bool   `json:"surfaced_to_operator"`
}

func ValidateAuditStoragePostureEvidence(evidence AuditStoragePostureEvidence) error {
	if !evidence.EncryptedAtRestDefault {
		return fmt.Errorf("encrypted_at_rest_default must be true")
	}
	if !evidence.SurfacedToOperator {
		return fmt.Errorf("storage posture evidence must be surfaced to operators")
	}
	if evidence.DevPlaintextOverrideActive {
		if evidence.EncryptedAtRestEffective {
			return fmt.Errorf("dev plaintext override cannot report encrypted_at_rest_effective=true")
		}
		if evidence.DevPlaintextOverrideReason == "" {
			return fmt.Errorf("dev plaintext override requires explicit reason")
		}
		return nil
	}
	if !evidence.EncryptedAtRestEffective {
		return fmt.Errorf("plaintext fallback without explicit dev override is forbidden")
	}
	return nil
}

type AuditdReadiness struct {
	LocalOnly                 bool   `json:"local_only"`
	ConsumptionChannel        string `json:"consumption_channel"`
	RecoveryComplete          bool   `json:"recovery_complete"`
	AppendPositionStable      bool   `json:"append_position_stable"`
	CurrentSegmentWritable    bool   `json:"current_segment_writable"`
	VerifierMaterialAvailable bool   `json:"verifier_material_available"`
	DerivedIndexCaughtUp      bool   `json:"derived_index_caught_up"`
	Ready                     bool   `json:"ready"`
}

func ValidateAuditdReadinessContract(readiness AuditdReadiness) error {
	if !readiness.LocalOnly {
		return fmt.Errorf("readiness signal must be local_only")
	}
	if readiness.ConsumptionChannel != "broker_local_api" {
		return fmt.Errorf("unsupported readiness consumption_channel %q", readiness.ConsumptionChannel)
	}
	expectedReady := readiness.RecoveryComplete && readiness.AppendPositionStable && readiness.CurrentSegmentWritable && readiness.VerifierMaterialAvailable && readiness.DerivedIndexCaughtUp
	if readiness.Ready != expectedReady {
		return fmt.Errorf("ready=%t does not match required readiness dimensions=%t", readiness.Ready, expectedReady)
	}
	return nil
}
