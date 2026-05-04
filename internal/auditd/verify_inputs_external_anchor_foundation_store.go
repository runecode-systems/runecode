package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) ensureExternalAnchorIncrementalFoundationLocked() (externalAnchorIncrementalFoundation, error) {
	foundation, exists, err := l.loadExternalAnchorIncrementalFoundationLocked()
	if err != nil {
		return externalAnchorIncrementalFoundation{}, err
	}
	if exists {
		return foundation, nil
	}
	rebuilt, rebuildErr := l.rebuildExternalAnchorIncrementalFoundationLocked()
	if rebuildErr != nil {
		return externalAnchorIncrementalFoundation{}, rebuildErr
	}
	if saveErr := l.saveExternalAnchorIncrementalFoundationLocked(rebuilt); saveErr != nil {
		return externalAnchorIncrementalFoundation{}, saveErr
	}
	return rebuilt, nil
}

func (l *Ledger) loadExternalAnchorIncrementalFoundationLocked() (externalAnchorIncrementalFoundation, bool, error) {
	path := filepath.Join(l.rootDir, indexDirName, externalAnchorIncrementalFoundationFileName)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return externalAnchorIncrementalFoundation{}, false, nil
		}
		return externalAnchorIncrementalFoundation{}, false, err
	}
	foundation := externalAnchorIncrementalFoundation{}
	if err := readJSONFile(path, &foundation); err != nil {
		return externalAnchorIncrementalFoundation{}, false, err
	}
	if foundation.SchemaVersion != externalAnchorIncrementalFoundationSchemaVersion {
		return externalAnchorIncrementalFoundation{}, false, fmt.Errorf("external anchor incremental foundation schema_version %d unsupported", foundation.SchemaVersion)
	}
	if foundation.Seals == nil {
		foundation.Seals = map[string]externalAnchorIncrementalSealSnapshot{}
	}
	normalized, err := normalizeExternalAnchorIncrementalFoundation(foundation)
	if err != nil {
		return externalAnchorIncrementalFoundation{}, false, err
	}
	return normalized, true, nil
}

func (l *Ledger) saveExternalAnchorIncrementalFoundationLocked(foundation externalAnchorIncrementalFoundation) error {
	if foundation.Seals == nil {
		foundation.Seals = map[string]externalAnchorIncrementalSealSnapshot{}
	}
	foundation.SchemaVersion = externalAnchorIncrementalFoundationSchemaVersion
	normalized, err := normalizeExternalAnchorIncrementalFoundation(foundation)
	if err != nil {
		return err
	}
	foundation = normalized
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, externalAnchorIncrementalFoundationFileName), foundation)
}

func normalizeExternalAnchorIncrementalFoundation(foundation externalAnchorIncrementalFoundation) (externalAnchorIncrementalFoundation, error) {
	for sealIdentity, entry := range foundation.Seals {
		entry.ReceiptDigests = normalizeIdentityList(entry.ReceiptDigests)
		entry.ExternalAnchorEvidenceDigests = normalizeIdentityList(entry.ExternalAnchorEvidenceDigests)
		entry.ExternalAnchorSidecarDigests = normalizeIdentityList(entry.ExternalAnchorSidecarDigests)
		normalizedTargets, err := normalizeExternalAnchorTargetSnapshots(entry.ExternalAnchorTargets)
		if err != nil {
			return externalAnchorIncrementalFoundation{}, fmt.Errorf("normalize external anchor target snapshots for seal %q: %w", sealIdentity, err)
		}
		entry.ExternalAnchorTargets = normalizedTargets
		foundation.Seals[sealIdentity] = entry
	}
	return foundation, nil
}

func normalizeIdentityList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for i := range values {
		v := strings.TrimSpace(values[i])
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		normalized = append(normalized, v)
	}
	sort.Strings(normalized)
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func appendUniqueExternalAnchorTarget(existing []externalAnchorVerificationTargetSnapshot, target trustpolicy.ExternalAnchorVerificationTarget) []externalAnchorVerificationTargetSnapshot {
	snapshot, ok := externalAnchorVerificationTargetSnapshotFromTarget(target)
	if !ok {
		return existing
	}
	for i := range existing {
		if existing[i].TargetDescriptorDigest != snapshot.TargetDescriptorDigest {
			continue
		}
		existing[i] = mergeExternalAnchorTargetSnapshot(existing[i], snapshot)
		return existing
	}
	return append(existing, snapshot)
}

func externalAnchorVerificationTargetSnapshotFromTarget(target trustpolicy.ExternalAnchorVerificationTarget) (externalAnchorVerificationTargetSnapshot, bool) {
	id, err := target.TargetDescriptorDigest.Identity()
	if err != nil {
		return externalAnchorVerificationTargetSnapshot{}, false
	}
	return externalAnchorVerificationTargetSnapshot{
		TargetKind:             strings.TrimSpace(target.TargetKind),
		TargetDescriptorDigest: id,
		TargetRequirement:      trustpolicy.NormalizeExternalAnchorTargetRequirement(target.TargetRequirement),
	}, true
}

func mergeExternalAnchorTargetSnapshot(current, incoming externalAnchorVerificationTargetSnapshot) externalAnchorVerificationTargetSnapshot {
	if current.TargetRequirement != trustpolicy.ExternalAnchorTargetRequirementRequired && incoming.TargetRequirement == trustpolicy.ExternalAnchorTargetRequirementRequired {
		current.TargetRequirement = trustpolicy.ExternalAnchorTargetRequirementRequired
	}
	if strings.TrimSpace(current.TargetKind) == "" {
		current.TargetKind = incoming.TargetKind
	}
	return current
}

func normalizeExternalAnchorTargetSnapshots(values []externalAnchorVerificationTargetSnapshot) ([]externalAnchorVerificationTargetSnapshot, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make([]externalAnchorVerificationTargetSnapshot, 0, len(values))
	for i := range values {
		target, err := verificationTargetFromSnapshot(values[i])
		if err != nil {
			return nil, err
		}
		normalized = appendUniqueExternalAnchorTarget(normalized, target)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].TargetDescriptorDigest < normalized[j].TargetDescriptorDigest
	})
	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}

func externalAnchorVerificationTargetsFromSnapshot(values []externalAnchorVerificationTargetSnapshot) ([]trustpolicy.ExternalAnchorVerificationTarget, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]trustpolicy.ExternalAnchorVerificationTarget, 0, len(values))
	for i := range values {
		target, err := verificationTargetFromSnapshot(values[i])
		if err != nil {
			return nil, fmt.Errorf("external anchor target digest identity invalid: %w", err)
		}
		result = append(result, target)
	}
	return result, nil
}

func verificationTargetFromSnapshot(value externalAnchorVerificationTargetSnapshot) (trustpolicy.ExternalAnchorVerificationTarget, error) {
	digest, err := digestFromIdentity(value.TargetDescriptorDigest)
	if err != nil {
		return trustpolicy.ExternalAnchorVerificationTarget{}, err
	}
	requirement := trustpolicy.NormalizeExternalAnchorTargetRequirement(value.TargetRequirement)
	if err := trustpolicy.ValidateExternalAnchorTargetRequirement(requirement); err != nil {
		return trustpolicy.ExternalAnchorVerificationTarget{}, err
	}
	return trustpolicy.ExternalAnchorVerificationTarget{
		TargetKind:             strings.TrimSpace(value.TargetKind),
		TargetDescriptorDigest: digest,
		TargetRequirement:      requirement,
	}, nil
}
