package trustpolicy

import "fmt"

func validateSignerEvidenceRefs(event AuditEventPayload, envelopeSignature SignatureBlock, entry AuditEventContractCatalogEntry, provided []AuditSignerEvidenceReference) error {
	signerIdentity, err := signatureVerifierIdentity(envelopeSignature)
	if err != nil {
		return err
	}
	providedByDigest, providedByIdentity, err := buildSignerEvidenceIndexes(provided)
	if err != nil {
		return err
	}
	referencedEvidence, err := collectReferencedSignerEvidence(event.SignerEvidenceRefs, providedByDigest)
	if err != nil {
		return err
	}
	if len(referencedEvidence) == 0 {
		return nil
	}
	return validateEnvelopeSignerEvidence(signerIdentity, entry, providedByIdentity)
}

func buildSignerEvidenceIndexes(provided []AuditSignerEvidenceReference) (map[string]AuditSignerEvidence, map[string]AuditSignerEvidence, error) {
	providedByDigest := map[string]AuditSignerEvidence{}
	providedByIdentity := map[string]AuditSignerEvidence{}
	for index := range provided {
		digestIdentity, err := provided[index].Digest.Identity()
		if err != nil {
			return nil, nil, fmt.Errorf("signer_evidence[%d].digest: %w", index, err)
		}
		if _, exists := providedByDigest[digestIdentity]; exists {
			return nil, nil, fmt.Errorf("duplicate signer evidence digest %q", digestIdentity)
		}
		if err := ValidateAuditSignerEvidence(provided[index].Evidence); err != nil {
			return nil, nil, fmt.Errorf("signer_evidence[%d].evidence: %w", index, err)
		}
		evidenceIdentity, err := signatureVerifierIdentity(provided[index].Evidence.SignerKey)
		if err != nil {
			return nil, nil, fmt.Errorf("signer_evidence[%d].evidence.signer_key: %w", index, err)
		}
		providedByDigest[digestIdentity] = provided[index].Evidence
		providedByIdentity[evidenceIdentity] = provided[index].Evidence
	}
	return providedByDigest, providedByIdentity, nil
}

func collectReferencedSignerEvidence(refs []AuditTypedReference, providedByDigest map[string]AuditSignerEvidence) ([]AuditSignerEvidence, error) {
	referencedEvidence := make([]AuditSignerEvidence, 0, len(refs))
	for index := range refs {
		digestIdentity, err := refs[index].Digest.Identity()
		if err != nil {
			return nil, fmt.Errorf("signer_evidence_refs[%d].digest: %w", index, err)
		}
		evidence, ok := providedByDigest[digestIdentity]
		if !ok {
			return nil, fmt.Errorf("missing signer evidence payload for digest %q", digestIdentity)
		}
		referencedEvidence = append(referencedEvidence, evidence)
	}
	return referencedEvidence, nil
}

func validateEnvelopeSignerEvidence(signerIdentity string, entry AuditEventContractCatalogEntry, providedByIdentity map[string]AuditSignerEvidence) error {
	signerEvidence, ok := providedByIdentity[signerIdentity]
	if !ok {
		return fmt.Errorf("missing signer evidence for envelope signer %q", signerIdentity)
	}
	if !containsString(entry.AllowedSignerPurposes, signerEvidence.SignerPurpose) {
		return fmt.Errorf("signer purpose %q is not allowed", signerEvidence.SignerPurpose)
	}
	if !containsString(entry.AllowedSignerScopes, signerEvidence.SignerScope) {
		return fmt.Errorf("signer scope %q is not allowed", signerEvidence.SignerScope)
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
