package artifacts

import (
	"fmt"
	"strings"
	"time"
)

func stateFromBackup(manifest BackupManifest, lastAuditSequence int64, ioStore *storeIO) (StoreState, error) {
	if err := validatePolicy(manifest.Policy); err != nil {
		return StoreState{}, err
	}
	next := newStateFromBackup(manifest, lastAuditSequence)
	unapprovedByDigest, err := loadRestoredArtifacts(&next, manifest.Artifacts, ioStore)
	if err != nil {
		return StoreState{}, err
	}
	if err := validateApprovedRestores(next.Artifacts, unapprovedByDigest); err != nil {
		return StoreState{}, err
	}
	if err := loadRestoredApprovals(&next, manifest.Approvals); err != nil {
		return StoreState{}, err
	}
	if err := loadRestoredSessions(&next, manifest.Sessions); err != nil {
		return StoreState{}, err
	}
	if err := loadRestoredPolicyDecisions(&next, manifest.PolicyDecisions); err != nil {
		return StoreState{}, err
	}
	if err := validateRestoredApprovalPolicyDecisionLinks(next.Approvals, next.PolicyDecisions); err != nil {
		return StoreState{}, err
	}
	return next, nil
}

func newStateFromBackup(manifest BackupManifest, lastAuditSequence int64) StoreState {
	return StoreState{
		Artifacts:                map[string]ArtifactRecord{},
		Sessions:                 map[string]SessionDurableState{},
		Approvals:                map[string]ApprovalRecord{},
		RunApprovalRefs:          map[string][]string{},
		PolicyDecisions:          map[string]PolicyDecisionRecord{},
		RunPolicyDecisionRefs:    map[string][]string{},
		Policy:                   manifest.Policy,
		Runs:                     manifest.Runs,
		PromotionEventsByActor:   map[string][]time.Time{},
		LastAuditSequence:        lastAuditSequence,
		StorageProtectionPosture: manifest.StorageProtection,
	}
}

func loadRestoredSessions(next *StoreState, records []SessionDurableState) error {
	for _, rec := range records {
		normalized := normalizeSessionDurableState(rec)
		if normalized.SessionID == "" {
			return fmt.Errorf("session id is required")
		}
		next.Sessions[normalized.SessionID] = normalized
	}
	return nil
}

func loadRestoredArtifacts(next *StoreState, records []ArtifactRecord, ioStore *storeIO) (map[string]ArtifactRecord, error) {
	unapprovedByDigest := map[string]ArtifactRecord{}
	for _, rec := range records {
		validated, err := validateRestoredRecord(rec, ioStore)
		if err != nil {
			return nil, err
		}
		next.Artifacts[validated.Reference.Digest] = validated
		if validated.Reference.DataClass == DataClassUnapprovedFileExcerpts {
			unapprovedByDigest[validated.Reference.Digest] = validated
		}
	}
	return unapprovedByDigest, nil
}

func loadRestoredApprovals(next *StoreState, records []ApprovalRecord) error {
	for _, rec := range records {
		if err := validateApprovalRecord(rec); err != nil {
			return err
		}
		if err := requirePolicyDecisionHashForBoundApproval(rec); err != nil {
			return err
		}
		next.Approvals[rec.ApprovalID] = rec
		if rec.RunID != "" {
			next.RunApprovalRefs[rec.RunID] = uniqueSortedStrings(append(next.RunApprovalRefs[rec.RunID], rec.ApprovalID))
		}
	}
	return nil
}

func loadRestoredPolicyDecisions(next *StoreState, records []PolicyDecisionRecord) error {
	for _, rec := range records {
		if err := validatePolicyDecisionRecord(rec); err != nil {
			return err
		}
		if _, canonicalPayload, err := canonicalizePolicyDecisionRecord(rec); err != nil {
			return err
		} else if err := applyComputedPolicyDecisionDigest(&rec, canonicalPayload); err != nil {
			return err
		}
		next.PolicyDecisions[rec.Digest] = rec
		if rec.RunID != "" {
			next.RunPolicyDecisionRefs[rec.RunID] = uniqueSortedStrings(append(next.RunPolicyDecisionRefs[rec.RunID], rec.Digest))
		}
	}
	return nil
}

func validateRestoredApprovalPolicyDecisionLinks(approvals map[string]ApprovalRecord, decisions map[string]PolicyDecisionRecord) error {
	for approvalID, rec := range approvals {
		if !approvalHasBindingKeys(&rec) {
			continue
		}
		hash := strings.TrimSpace(rec.PolicyDecisionHash)
		decision, ok := decisions[hash]
		if !ok {
			return fmt.Errorf("%w: approval %q policy decision %q not found", ErrApprovalPolicyDecisionRequired, approvalID, hash)
		}
		if decision.ManifestHash != rec.ManifestHash || decision.ActionRequestHash != rec.ActionRequestHash {
			return fmt.Errorf("%w: approval %q policy decision %q binding mismatch", ErrApprovalPolicyDecisionRequired, approvalID, hash)
		}
	}
	return nil
}
