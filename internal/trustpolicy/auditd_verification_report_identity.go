package trustpolicy

import (
	"fmt"
	"strings"
)

func validateVerificationIdentityFootprint(verifierIdentity string, trustRootIdentities []string) error {
	if err := validateVerifierIdentityValue(verifierIdentity); err != nil {
		return err
	}
	if len(trustRootIdentities) == 0 {
		return fmt.Errorf("trust_root_identities is required")
	}
	seenTrustRoots := map[string]struct{}{}
	for i := range trustRootIdentities {
		if err := validateTrustRootIdentityEntry(i, trustRootIdentities[i], seenTrustRoots); err != nil {
			return err
		}
	}
	return nil
}

func validateVerifierIdentityValue(verifierIdentity string) error {
	trimmed := strings.TrimSpace(verifierIdentity)
	if trimmed == "" {
		return fmt.Errorf("verifier_identity is required")
	}
	if trimmed != "unknown" && !strings.HasPrefix(trimmed, KeyIDProfile+":") {
		return fmt.Errorf("verifier_identity must use %s identity profile", KeyIDProfile)
	}
	return nil
}

func validateTrustRootIdentityEntry(index int, raw string, seenTrustRoots map[string]struct{}) error {
	identity := strings.TrimSpace(raw)
	if identity == "" {
		return fmt.Errorf("trust_root_identities[%d] is required", index)
	}
	if identity != "unknown" {
		if _, err := parseAuditVerificationDigestIdentity(identity); err != nil {
			return fmt.Errorf("trust_root_identities[%d]: %w", index, err)
		}
	}
	if _, exists := seenTrustRoots[identity]; exists {
		return fmt.Errorf("trust_root_identities[%d] duplicates identity %q", index, identity)
	}
	seenTrustRoots[identity] = struct{}{}
	return nil
}

func parseAuditVerificationDigestIdentity(identity string) (Digest, error) {
	parts := strings.SplitN(strings.TrimSpace(identity), ":", 2)
	if len(parts) != 2 {
		return Digest{}, fmt.Errorf("digest identity must be hash_alg:hash")
	}
	d := Digest{HashAlg: parts[0], Hash: parts[1]}
	if _, err := d.Identity(); err != nil {
		return Digest{}, err
	}
	return d, nil
}

func validateAuditVerificationAnchoringPosture(posture string) error {
	switch posture {
	case AuditVerificationAnchoringPostureLocalAnchorReceiptOnly,
		AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound,
		AuditVerificationAnchoringPostureExternalAnchorValidated,
		AuditVerificationAnchoringPostureExternalAnchorDeferredOrUnknown,
		AuditVerificationAnchoringPostureExternalAnchorInvalid:
		return nil
	default:
		return fmt.Errorf("unsupported anchoring posture %q", posture)
	}
}
