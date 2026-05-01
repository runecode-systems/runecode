package trustpolicy

import "fmt"

func validateAnchorReceiptPayload(receipt auditReceiptPayloadStrict) error {
	payload, err := decodeAnchorReceiptPayload(receipt)
	if err != nil {
		return err
	}
	if err := validateAnchorPayloadCore(payload); err != nil {
		return err
	}
	if err := validateAnchorPayloadApprovalFields(payload); err != nil {
		return err
	}
	if err := validateAnchorPayloadWitness(payload); err != nil {
		return err
	}
	return validateAnchorApprovalLinkAndPosture(payload)
}

func decodeAnchorReceiptPayload(receipt auditReceiptPayloadStrict) (anchorReceiptPayload, error) {
	if receipt.ReceiptPayloadSchema != "runecode.protocol.audit.receipt.anchor.v0" {
		return anchorReceiptPayload{}, fmt.Errorf("anchor receipts require anchor payload schema")
	}
	payload := anchorReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return anchorReceiptPayload{}, fmt.Errorf("decode anchor payload: %w", err)
	}
	return payload, nil
}

func validateAnchorPayloadCore(payload anchorReceiptPayload) error {
	if _, ok := supportedAnchorKinds[payload.AnchorKind]; !ok {
		return fmt.Errorf("unsupported anchor_kind %q", payload.AnchorKind)
	}
	if payload.AnchorKind == mvpAnchorKindLocalUserPresence {
		return validateLocalPresenceAnchorPayload(payload)
	}
	return validateExternalAnchorPayloadEnvelope(payload)
}

func validateLocalPresenceAnchorPayload(payload anchorReceiptPayload) error {
	if payload.ExternalAnchor != nil {
		return fmt.Errorf("anchor_kind %q cannot declare external_anchor", payload.AnchorKind)
	}
	if _, ok := allowedKeyProtectionPostures[payload.KeyProtectionPosture]; !ok {
		return fmt.Errorf("unsupported key_protection_posture %q", payload.KeyProtectionPosture)
	}
	if _, ok := allowedPresenceModes[payload.PresenceMode]; !ok {
		return fmt.Errorf("unsupported presence_mode %q", payload.PresenceMode)
	}
	if payload.PresenceMode == "none" {
		return fmt.Errorf("unsupported presence_mode %q", payload.PresenceMode)
	}
	return nil
}

func validateExternalAnchorPayloadEnvelope(payload anchorReceiptPayload) error {
	if payload.KeyProtectionPosture != "" {
		return fmt.Errorf("anchor_kind %q must not declare key_protection_posture", payload.AnchorKind)
	}
	if payload.PresenceMode != "" {
		return fmt.Errorf("anchor_kind %q must not declare presence_mode", payload.AnchorKind)
	}
	if payload.ExternalAnchor == nil {
		return fmt.Errorf("anchor_kind %q requires external_anchor", payload.AnchorKind)
	}
	return validateExternalAnchorPayload(payload.AnchorKind, *payload.ExternalAnchor)
}

func validateAnchorPayloadApprovalFields(payload anchorReceiptPayload) error {
	if err := validateAnchorPayloadApprovalAssurance(payload.ApprovalAssurance); err != nil {
		return err
	}
	if err := validateAnchorPayloadApprovalDecision(payload.ApprovalDecision); err != nil {
		return err
	}
	return validateAnchorPayloadApprovalCoupling(payload.ApprovalAssurance, payload.ApprovalDecision != nil)
}

func validateAnchorPayloadApprovalAssurance(assurance string) error {
	if assurance == "" {
		return nil
	}
	if _, ok := allowedAssuranceLevels[assurance]; !ok {
		return fmt.Errorf("unsupported approval_assurance_level %q", assurance)
	}
	return nil
}

func validateAnchorPayloadApprovalDecision(decision *Digest) error {
	if decision == nil {
		return nil
	}
	if _, err := decision.Identity(); err != nil {
		return fmt.Errorf("approval_decision_digest: %w", err)
	}
	return nil
}

func validateAnchorPayloadApprovalCoupling(assurance string, hasDecision bool) error {
	if assurance == "none" && hasDecision {
		return fmt.Errorf("approval_assurance_level=none cannot declare approval_decision_digest")
	}
	if assurance == "hardware_backed" && !hasDecision {
		return fmt.Errorf("approval_decision_digest is required when approval_assurance_level=hardware_backed")
	}
	if hasDecision && assurance == "" {
		return fmt.Errorf("approval_assurance_level is required when approval_decision_digest is present")
	}
	return nil
}

func validateAnchorPayloadWitness(payload anchorReceiptPayload) error {
	if payload.AnchorWitness == nil {
		if payload.AnchorKind == mvpAnchorKindLocalUserPresence {
			return fmt.Errorf("anchor_witness is required for anchor_kind %q", payload.AnchorKind)
		}
		return nil
	}
	if payload.AnchorKind != mvpAnchorKindLocalUserPresence {
		return fmt.Errorf("anchor_kind %q must not declare anchor_witness", payload.AnchorKind)
	}
	if payload.AnchorWitness.WitnessKind != mvpAnchorWitnessKindLocalPresenceV0 {
		return fmt.Errorf("unsupported anchor_witness.witness_kind %q", payload.AnchorWitness.WitnessKind)
	}
	if _, err := payload.AnchorWitness.WitnessDigest.Identity(); err != nil {
		return fmt.Errorf("anchor_witness.witness_digest: %w", err)
	}
	return nil
}

func validateAnchorApprovalLinkAndPosture(payload anchorReceiptPayload) error {
	if payload.AnchorKind != mvpAnchorKindLocalUserPresence {
		if payload.ApprovalAssurance == "hardware_backed" {
			return fmt.Errorf("approval_assurance_level=hardware_backed is only valid for anchor_kind=%s", mvpAnchorKindLocalUserPresence)
		}
		return nil
	}
	if payload.ApprovalAssurance != "hardware_backed" {
		return nil
	}
	if payload.PresenceMode != "hardware_touch" {
		return fmt.Errorf("approval_assurance_level=hardware_backed requires presence_mode=hardware_touch")
	}
	if payload.KeyProtectionPosture != "hardware_backed" {
		return fmt.Errorf("approval_assurance_level=hardware_backed requires key_protection_posture=hardware_backed")
	}
	return nil
}

func validateReceiptSignerContract(receipt auditReceiptPayloadStrict, verifier VerifierRecord) error {
	if receipt.AuditReceiptKind != "anchor" {
		return nil
	}
	if verifier.LogicalPurpose != "audit_anchor" {
		return fmt.Errorf("anchor receipts require verifier logical_purpose audit_anchor")
	}
	if verifier.LogicalScope != "node" && verifier.LogicalScope != "deployment" {
		return fmt.Errorf("anchor receipts require verifier logical_scope node or deployment")
	}
	payload := anchorReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode anchor payload for signer posture check: %w", err)
	}
	if payload.AnchorKind != mvpAnchorKindLocalUserPresence {
		return nil
	}
	if payload.KeyProtectionPosture != verifier.KeyProtectionPosture {
		return fmt.Errorf("anchor key_protection_posture %q does not match verifier key_protection_posture %q", payload.KeyProtectionPosture, verifier.KeyProtectionPosture)
	}
	if payload.PresenceMode != verifier.PresenceMode {
		return fmt.Errorf("anchor presence_mode %q does not match verifier presence_mode %q", payload.PresenceMode, verifier.PresenceMode)
	}
	return nil
}
