package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) FindMatchingZKProofVerificationRecord(match trustpolicy.ZKProofVerificationRecordPayload) (trustpolicy.Digest, trustpolicy.ZKProofVerificationRecordPayload, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := l.readOptionalSidecarDirEntries(proofVerificationsDirName)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	for _, entry := range entries {
		identity, candidate, ok, err := l.loadVerifiedZKProofVerificationRecordCandidate(entry)
		if err != nil {
			return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
		}
		if !ok || !sameZKProofVerificationIdentity(candidate, match) {
			continue
		}
		d, err := digestFromIdentity(identity)
		if err != nil {
			return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, err
		}
		return d, candidate, true, nil
	}
	return trustpolicy.Digest{}, trustpolicy.ZKProofVerificationRecordPayload{}, false, nil
}

func (l *Ledger) loadVerifiedZKProofVerificationRecordCandidate(entry os.DirEntry) (string, trustpolicy.ZKProofVerificationRecordPayload, bool, error) {
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, ok, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, proofVerificationsDirName, entry.Name())
	candidate := trustpolicy.ZKProofVerificationRecordPayload{}
	if err := readJSONFile(path, &candidate); err != nil {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if err := trustpolicy.ValidateZKProofVerificationRecordPayload(candidate); err != nil {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	computed, err := canonicalDigest(candidate)
	if err != nil {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, err
	}
	if mustDigestIdentityString(computed) != identity {
		return "", trustpolicy.ZKProofVerificationRecordPayload{}, false, fmt.Errorf("zk proof verification record content digest mismatch for %s", identity)
	}
	return identity, candidate, true, nil
}

func sameZKProofVerificationIdentity(left, right trustpolicy.ZKProofVerificationRecordPayload) bool {
	if !sameVerificationDigest(left.ProofDigest, right.ProofDigest) ||
		!sameVerificationDigest(left.ConstraintSystemDigest, right.ConstraintSystemDigest) ||
		!sameVerificationDigest(left.VerifierKeyDigest, right.VerifierKeyDigest) ||
		!sameVerificationDigest(left.SetupProvenanceDigest, right.SetupProvenanceDigest) ||
		!sameVerificationDigest(left.PublicInputsDigest, right.PublicInputsDigest) {
		return false
	}
	return sameVerificationMetadata(left, right) && sameStringSet(left.ReasonCodes, right.ReasonCodes)
}

func sameVerificationDigest(left, right trustpolicy.Digest) bool {
	leftIdentity, leftErr := left.Identity()
	rightIdentity, rightErr := right.Identity()
	return leftErr == nil && rightErr == nil && leftIdentity == rightIdentity
}

func sameVerificationMetadata(left, right trustpolicy.ZKProofVerificationRecordPayload) bool {
	return strings.TrimSpace(left.StatementFamily) == strings.TrimSpace(right.StatementFamily) &&
		strings.TrimSpace(left.StatementVersion) == strings.TrimSpace(right.StatementVersion) &&
		strings.TrimSpace(left.SchemeID) == strings.TrimSpace(right.SchemeID) &&
		strings.TrimSpace(left.CurveID) == strings.TrimSpace(right.CurveID) &&
		strings.TrimSpace(left.CircuitID) == strings.TrimSpace(right.CircuitID) &&
		strings.TrimSpace(left.NormalizationProfileID) == strings.TrimSpace(right.NormalizationProfileID) &&
		strings.TrimSpace(left.SchemeAdapterID) == strings.TrimSpace(right.SchemeAdapterID) &&
		strings.TrimSpace(left.VerifierImplementationID) == strings.TrimSpace(right.VerifierImplementationID) &&
		strings.TrimSpace(left.VerificationOutcome) == strings.TrimSpace(right.VerificationOutcome)
}

func sameStringSet(left, right []string) bool {
	seen, ok := buildStringSet(left)
	if !ok || len(seen) != len(right) {
		return false
	}
	for _, value := range right {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return false
		}
		if _, ok := seen[trimmed]; !ok {
			return false
		}
	}
	return true
}

func buildStringSet(values []string) (map[string]struct{}, bool) {
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, false
		}
		if _, ok := seen[trimmed]; ok {
			return nil, false
		}
		seen[trimmed] = struct{}{}
	}
	return seen, true
}
