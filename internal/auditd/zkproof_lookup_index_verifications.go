package auditd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type verificationIdentityLookupKey struct {
	proofDigest            string
	statementFamily        string
	statementVersion       string
	schemeID               string
	curveID                string
	circuitID              string
	constraintSystemDigest string
	verifierKeyDigest      string
	setupProvenanceDigest  string
	normalizationProfileID string
	schemeAdapterID        string
	publicInputsDigest     string
	verifierImplID         string
	verificationOutcome    string
	reasonCodes            []string
}

type verificationPayloadDigestIdentitySet struct {
	proofDigest            string
	constraintSystemDigest string
	verifierKeyDigest      string
	setupProvenanceDigest  string
	publicInputsDigest     string
}

func verificationIdentityKey(payload trustpolicy.ZKProofVerificationRecordPayload) (string, error) {
	reasonCodes, err := normalizeReasonCodes(payload.ReasonCodes)
	if err != nil {
		return "", err
	}
	identities, err := verificationPayloadDigestIdentities(payload)
	if err != nil {
		return "", err
	}
	key := verificationIdentityLookupKey{
		proofDigest:            identities.proofDigest,
		statementFamily:        strings.TrimSpace(payload.StatementFamily),
		statementVersion:       strings.TrimSpace(payload.StatementVersion),
		schemeID:               strings.TrimSpace(payload.SchemeID),
		curveID:                strings.TrimSpace(payload.CurveID),
		circuitID:              strings.TrimSpace(payload.CircuitID),
		constraintSystemDigest: identities.constraintSystemDigest,
		verifierKeyDigest:      identities.verifierKeyDigest,
		setupProvenanceDigest:  identities.setupProvenanceDigest,
		normalizationProfileID: strings.TrimSpace(payload.NormalizationProfileID),
		schemeAdapterID:        strings.TrimSpace(payload.SchemeAdapterID),
		publicInputsDigest:     identities.publicInputsDigest,
		verifierImplID:         strings.TrimSpace(payload.VerifierImplementationID),
		verificationOutcome:    strings.TrimSpace(payload.VerificationOutcome),
		reasonCodes:            reasonCodes,
	}
	b, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func verificationPayloadDigestIdentities(payload trustpolicy.ZKProofVerificationRecordPayload) (verificationPayloadDigestIdentitySet, error) {
	proofDigest, err := payload.ProofDigest.Identity()
	if err != nil {
		return verificationPayloadDigestIdentitySet{}, err
	}
	constraintDigest, err := payload.ConstraintSystemDigest.Identity()
	if err != nil {
		return verificationPayloadDigestIdentitySet{}, err
	}
	verifierKeyDigest, err := payload.VerifierKeyDigest.Identity()
	if err != nil {
		return verificationPayloadDigestIdentitySet{}, err
	}
	setupDigest, err := payload.SetupProvenanceDigest.Identity()
	if err != nil {
		return verificationPayloadDigestIdentitySet{}, err
	}
	publicInputsDigest, err := payload.PublicInputsDigest.Identity()
	if err != nil {
		return verificationPayloadDigestIdentitySet{}, err
	}
	return verificationPayloadDigestIdentitySet{proofDigest: proofDigest, constraintSystemDigest: constraintDigest, verifierKeyDigest: verifierKeyDigest, setupProvenanceDigest: setupDigest, publicInputsDigest: publicInputsDigest}, nil
}

func normalizeReasonCodes(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for i := range values {
		trimmed := strings.TrimSpace(values[i])
		if trimmed == "" {
			return nil, fmt.Errorf("reason code must not be empty")
		}
		if _, ok := seen[trimmed]; ok {
			return nil, fmt.Errorf("reason codes must not contain duplicates")
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func chooseVerificationLookup(current verificationLookup, nextDigestIdentity, nextVerifiedAt string) bool {
	if strings.TrimSpace(current.DigestIdentity) == "" {
		return true
	}
	if decision, ok := chooseVerificationLookupByTime(current.VerifiedAt, nextVerifiedAt); ok {
		return decision
	}
	return strings.TrimSpace(nextDigestIdentity) > strings.TrimSpace(current.DigestIdentity)
}

func chooseVerificationLookupByTime(currentVerifiedAt, nextVerifiedAt string) (bool, bool) {
	nextTime, nextOK := parseRFC3339(nextVerifiedAt)
	curTime, curOK := parseRFC3339(currentVerifiedAt)
	switch {
	case nextOK && curOK && nextTime.After(curTime):
		return true, true
	case nextOK && curOK && nextTime.Before(curTime):
		return false, true
	case nextOK && !curOK:
		return true, true
	case !nextOK && curOK:
		return false, true
	default:
		return false, false
	}
}

func parseRFC3339(value string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func (idx *proofLookupIndex) notePersistedZKProofVerificationRecord(digest trustpolicy.Digest, payload trustpolicy.ZKProofVerificationRecordPayload) error {
	if idx == nil {
		return fmt.Errorf("proof lookup index is required")
	}
	normalizeProofLookupIndex(idx)
	key, err := verificationIdentityKey(payload)
	if err != nil {
		return err
	}
	digestIdentity, err := digest.Identity()
	if err != nil {
		return err
	}
	current := idx.VerificationByKey[key]
	if chooseVerificationLookup(current, digestIdentity, payload.VerifiedAt) {
		idx.VerificationByKey[key] = verificationLookup{DigestIdentity: digestIdentity, VerifiedAt: strings.TrimSpace(payload.VerifiedAt)}
	}
	return nil
}
