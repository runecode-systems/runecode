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
	if err := validateReferencedEvidenceBinding(signerIdentity, referencedEvidence); err != nil {
		return err
	}
	return validateEnvelopeSignerEvidence(signerIdentity, entry, providedByIdentity)
}

func buildSignerEvidenceIndexes(provided []AuditSignerEvidenceReference) (map[string]AuditSignerEvidence, map[string]AuditSignerEvidence, error) {
	providedByDigest := map[string]AuditSignerEvidence{}
	providedByIdentity := map[string]AuditSignerEvidence{}
	for index := range provided {
		digestIdentity, evidenceIdentity, evidence, err := parseSignerEvidenceIndexEntry(index, provided[index])
		if err != nil {
			return nil, nil, err
		}
		if err := checkSignerEvidenceIndexConflicts(digestIdentity, evidenceIdentity, providedByDigest, providedByIdentity); err != nil {
			return nil, nil, err
		}
		providedByDigest[digestIdentity] = evidence
		providedByIdentity[evidenceIdentity] = evidence
	}
	return providedByDigest, providedByIdentity, nil
}

func parseSignerEvidenceIndexEntry(index int, reference AuditSignerEvidenceReference) (string, string, AuditSignerEvidence, error) {
	digestIdentity, err := reference.Digest.Identity()
	if err != nil {
		return "", "", AuditSignerEvidence{}, fmt.Errorf("signer_evidence[%d].digest: %w", index, err)
	}
	if err := ValidateAuditSignerEvidence(reference.Evidence); err != nil {
		return "", "", AuditSignerEvidence{}, fmt.Errorf("signer_evidence[%d].evidence: %w", index, err)
	}
	evidenceIdentity, err := signatureVerifierIdentity(reference.Evidence.SignerKey)
	if err != nil {
		return "", "", AuditSignerEvidence{}, fmt.Errorf("signer_evidence[%d].evidence.signer_key: %w", index, err)
	}
	return digestIdentity, evidenceIdentity, reference.Evidence, nil
}

func checkSignerEvidenceIndexConflicts(digestIdentity string, evidenceIdentity string, providedByDigest map[string]AuditSignerEvidence, providedByIdentity map[string]AuditSignerEvidence) error {
	if _, exists := providedByDigest[digestIdentity]; exists {
		return fmt.Errorf("duplicate signer evidence digest %q", digestIdentity)
	}
	if _, exists := providedByIdentity[evidenceIdentity]; exists {
		return fmt.Errorf("duplicate signer evidence identity %q", evidenceIdentity)
	}
	return nil
}

func validateReferencedEvidenceBinding(signerIdentity string, referencedEvidence []AuditSignerEvidence) error {
	matchedSignerIdentity := false
	for index := range referencedEvidence {
		referencedIdentity, err := signatureVerifierIdentity(referencedEvidence[index].SignerKey)
		if err != nil {
			return fmt.Errorf("signer_evidence_refs[%d].signer_key: %w", index, err)
		}
		if referencedIdentity != signerIdentity {
			return fmt.Errorf("signer_evidence_refs[%d] is bound to %q, expected envelope signer %q", index, referencedIdentity, signerIdentity)
		}
		matchedSignerIdentity = true
	}
	if !matchedSignerIdentity {
		return fmt.Errorf("missing referenced signer evidence for envelope signer %q", signerIdentity)
	}
	return nil
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
