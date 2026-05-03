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
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.evidenceSnapshotLocked()
}

func (l *Ledger) evidenceSnapshotLocked() (AuditEvidenceSnapshot, error) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	collected, err := l.collectEvidenceSnapshotFamiliesLocked()
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	segmentIDs, segmentSealDigests := evidenceSnapshotSegmentsFromIndex(index)

	return AuditEvidenceSnapshot{
		SchemaID:                    auditEvidenceSnapshotSchemaID,
		SchemaVersion:               auditEvidenceSnapshotSchemaVersion,
		CreatedAt:                   l.nowFn().UTC().Format(time.RFC3339),
		SegmentIDs:                  normalizeIdentityList(segmentIDs),
		SegmentSealDigests:          normalizeIdentityList(segmentSealDigests),
		AuditReceiptDigests:         normalizeIdentityList(collected.receiptDigests),
		VerificationReportDigests:   normalizeIdentityList(collected.verificationReportDigests),
		RuntimeEvidenceDigests:      normalizeIdentityList(collected.runtimeEvidenceDigests),
		VerifierRecordDigests:       normalizeIdentityList(collected.verifierRecordDigests),
		EventContractCatalogDigests: normalizeIdentityList(collected.eventContractDigests),
		SignerEvidenceDigests:       normalizeIdentityList(collected.signerEvidenceDigests),
		StoragePostureDigests:       normalizeIdentityList(collected.storagePostureDigests),
		TypedRequestDigests:         normalizeIdentityList(collected.typedRequestDigests),
		ActionRequestDigests:        normalizeIdentityList(collected.actionRequestDigests),
		ControlPlaneDigests:         normalizeIdentityList(collected.controlPlaneDigests),
		AttestationEvidenceDigests:  normalizeIdentityList(collected.attestationDigests),
		InstanceIdentityDigests:     normalizeIdentityList(collected.instanceIdentityDigests),
		PolicyEvidenceDigests:       normalizeIdentityList(collected.policyDigests),
		RequiredApprovalIDs:         normalizeStringList(collected.requiredApprovalIDs),
		ApprovalEvidenceDigests:     normalizeIdentityList(collected.approvalDigests),
		AnchorEvidenceDigests:       normalizeIdentityList(collected.anchorEvidenceDigests),
		ProviderInvocationDigests:   normalizeIdentityList(collected.providerInvocationDigests),
		SecretLeaseDigests:          normalizeIdentityList(collected.secretLeaseDigests),
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
