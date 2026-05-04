package auditd

import (
	"sort"
	"strings"
	"time"
)

const (
	auditEvidenceSnapshotSchemaID      = "runecode.protocol.v0.AuditEvidenceSnapshot"
	auditEvidenceSnapshotSchemaVersion = "0.1.0"
)

// EvidenceSnapshot returns a cheap preservation-manifest view of canonical evidence identities.
func (l *Ledger) EvidenceSnapshot() (AuditEvidenceSnapshot, error) {
	return l.EvidenceSnapshotWithIdentity(AuditEvidenceIdentityContext{})
}

func (l *Ledger) EvidenceSnapshotWithIdentity(identityContext AuditEvidenceIdentityContext) (AuditEvidenceSnapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.evidenceSnapshotLocked(identityContext)
}

func (l *Ledger) evidenceSnapshotLocked(identityContext AuditEvidenceIdentityContext) (AuditEvidenceSnapshot, error) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	collected, err := l.collectEvidenceSnapshotFamiliesLocked()
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	manifestIdentity, err := l.evidenceIdentityManifestLocked()
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	segmentIDs, segmentSealDigests := evidenceSnapshotSegmentsFromIndex(index)
	return buildEvidenceSnapshot(identityContext, manifestIdentity, collected, segmentIDs, segmentSealDigests, l.nowFn), nil
}

func buildEvidenceSnapshot(identityContext, manifestIdentity AuditEvidenceIdentityContext, collected evidenceSnapshotFamilies, segmentIDs, segmentSealDigests []string, now func() time.Time) AuditEvidenceSnapshot {
	return AuditEvidenceSnapshot{
		SchemaID:                      auditEvidenceSnapshotSchemaID,
		SchemaVersion:                 auditEvidenceSnapshotSchemaVersion,
		CreatedAt:                     now().UTC().Format(time.RFC3339),
		RepositoryIdentityDigest:      strings.TrimSpace(identityContext.RepositoryIdentityDigest),
		ProductInstanceID:             strings.TrimSpace(identityContext.ProductInstanceID),
		LedgerIdentity:                strings.TrimSpace(manifestIdentity.LedgerIdentity),
		SegmentIDs:                    normalizeIdentityList(segmentIDs),
		SegmentSealDigests:            normalizeIdentityList(segmentSealDigests),
		AuditReceiptDigests:           normalizeIdentityList(collected.receiptDigests),
		VerificationReportDigests:     normalizeIdentityList(collected.verificationReportDigests),
		RuntimeEvidenceDigests:        normalizeIdentityList(collected.runtimeEvidenceDigests),
		VerifierRecordDigests:         normalizeIdentityList(collected.verifierRecordDigests),
		EventContractCatalogDigests:   normalizeIdentityList(collected.eventContractDigests),
		SignerEvidenceDigests:         normalizeIdentityList(collected.signerEvidenceDigests),
		StoragePostureDigests:         normalizeIdentityList(collected.storagePostureDigests),
		TypedRequestDigests:           normalizeIdentityList(collected.typedRequestDigests),
		ActionRequestDigests:          normalizeIdentityList(collected.actionRequestDigests),
		ControlPlaneDigests:           normalizeIdentityList(collected.controlPlaneDigests),
		AttestationEvidenceDigests:    normalizeIdentityList(collected.attestationDigests),
		ProjectContextIdentityDigests: normalizeIdentityList(collected.projectContextDigests),
		PolicyEvidenceDigests:         normalizeIdentityList(collected.policyDigests),
		RequiredApprovalIDs:           normalizeStringList(collected.requiredApprovalIDs),
		ApprovalEvidenceDigests:       normalizeIdentityList(collected.approvalDigests),
		AnchorEvidenceDigests:         normalizeIdentityList(collected.anchorEvidenceDigests),
		ProviderInvocationDigests:     normalizeIdentityList(collected.providerInvocationDigests),
		SecretLeaseDigests:            normalizeIdentityList(collected.secretLeaseDigests),
	}
}

func (l *Ledger) evidenceIdentityManifestLocked() (AuditEvidenceIdentityContext, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return AuditEvidenceIdentityContext{}, err
	}
	return AuditEvidenceIdentityContext{
		LedgerIdentity: strings.TrimSpace(state.LedgerIdentity),
	}, nil
}

func evidenceSnapshotSegmentsFromIndex(index derivedIndex) ([]string, []string) {
	segmentIDs := make([]string, 0, len(index.SegmentSealLookup))
	segmentSealDigests := make([]string, 0, len(index.SegmentSealLookup))
	for segmentID, lookup := range index.SegmentSealLookup {
		segmentIDs = append(segmentIDs, strings.TrimSpace(segmentID))
		if strings.TrimSpace(lookup.SealDigest) != "" {
			segmentSealDigests = append(segmentSealDigests, lookup.SealDigest)
		}
	}
	sort.Strings(segmentIDs)
	return segmentIDs, segmentSealDigests
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for i := range values {
		trimmed := strings.TrimSpace(values[i])
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
