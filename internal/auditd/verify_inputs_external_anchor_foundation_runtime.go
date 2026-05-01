package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) noteIncrementalVerificationBaselineLocked(segmentID string, sealDigest trustpolicy.Digest, report trustpolicy.AuditVerificationReportPayload, reportDigest trustpolicy.Digest) error {
	sealIdentity, err := sealDigest.Identity()
	if err != nil {
		return fmt.Errorf("seal digest identity: %w", err)
	}
	reportIdentity, err := reportDigest.Identity()
	if err != nil {
		return fmt.Errorf("report digest identity: %w", err)
	}
	foundation, err := l.ensureExternalAnchorIncrementalFoundationLocked()
	if err != nil {
		return err
	}
	entry := foundation.Seals[sealIdentity]
	entry.SegmentID = strings.TrimSpace(segmentID)
	entry.BaselineVerificationReport = reportIdentity
	entry.BaselineVerificationReportedAt = strings.TrimSpace(report.VerifiedAt)
	foundation.Seals[sealIdentity] = entry
	return l.saveExternalAnchorIncrementalFoundationLocked(foundation)
}

func (l *Ledger) notePersistedReceiptInIncrementalFoundationLocked(receiptDigest trustpolicy.Digest, envelope trustpolicy.SignedObjectEnvelope) error {
	sealIdentity, ok, err := receiptSubjectSealDigestIdentity(envelope)
	if err != nil || !ok {
		return err
	}
	receiptIdentity, err := receiptDigest.Identity()
	if err != nil {
		return fmt.Errorf("receipt digest identity: %w", err)
	}
	foundation, err := l.ensureExternalAnchorIncrementalFoundationLocked()
	if err != nil {
		return err
	}
	entry := foundation.Seals[sealIdentity]
	entry.ReceiptDigests = appendUniqueIdentity(entry.ReceiptDigests, receiptIdentity)
	foundation.Seals[sealIdentity] = entry
	return l.saveExternalAnchorIncrementalFoundationLocked(foundation)
}

func (l *Ledger) notePersistedExternalAnchorEvidenceInIncrementalFoundationLocked(payload trustpolicy.ExternalAnchorEvidencePayload, evidenceDigest trustpolicy.Digest) error {
	sealIdentity, err := payload.AnchoringSubjectDigest.Identity()
	if err != nil {
		return fmt.Errorf("external anchor evidence anchoring subject digest: %w", err)
	}
	evidenceIdentity, err := evidenceDigest.Identity()
	if err != nil {
		return fmt.Errorf("external anchor evidence digest identity: %w", err)
	}
	foundation, err := l.ensureExternalAnchorIncrementalFoundationLocked()
	if err != nil {
		return err
	}
	entry, err := appendExternalAnchorEvidenceEntryIdentity(foundation.Seals[sealIdentity], evidenceIdentity, payload)
	if err != nil {
		return err
	}
	foundation.Seals[sealIdentity] = entry
	return l.saveExternalAnchorIncrementalFoundationLocked(foundation)
}

func appendExternalAnchorEvidenceEntryIdentity(entry externalAnchorIncrementalSealSnapshot, evidenceIdentity string, payload trustpolicy.ExternalAnchorEvidencePayload) (externalAnchorIncrementalSealSnapshot, error) {
	entry.ExternalAnchorEvidenceDigests = appendUniqueIdentity(entry.ExternalAnchorEvidenceDigests, evidenceIdentity)
	entry.ExternalAnchorTargets = appendUniqueExternalAnchorTarget(entry.ExternalAnchorTargets, trustpolicy.ExternalAnchorVerificationTarget{
		TargetKind:             strings.TrimSpace(payload.CanonicalTargetKind),
		TargetDescriptorDigest: payload.CanonicalTargetDigest,
		TargetRequirement:      trustpolicy.NormalizeExternalAnchorTargetRequirement(payload.TargetRequirement),
	})
	for i := range payload.SidecarRefs {
		sidecarIdentity, err := payload.SidecarRefs[i].Digest.Identity()
		if err != nil {
			return externalAnchorIncrementalSealSnapshot{}, fmt.Errorf("external anchor sidecar digest identity: %w", err)
		}
		entry.ExternalAnchorSidecarDigests = appendUniqueIdentity(entry.ExternalAnchorSidecarDigests, sidecarIdentity)
	}
	return entry, nil
}

func (l *Ledger) loadSealScopedDurableVerificationInputsLocked(segmentID string, sealDigest trustpolicy.Digest) ([]trustpolicy.SignedObjectEnvelope, []trustpolicy.ExternalAnchorEvidencePayload, []trustpolicy.Digest, []trustpolicy.ExternalAnchorVerificationTarget, error) {
	entry, err := l.requireSealIncrementalFoundationEntryLocked(segmentID, sealDigest, true)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	receipts, err := l.loadReceiptsByIdentitiesLocked(entry.ReceiptDigests)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	evidence, err := l.loadExternalAnchorEvidenceByIdentitiesLocked(entry.ExternalAnchorEvidenceDigests)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sidecars, err := l.loadExternalAnchorSidecarDigestsByIdentitiesLocked(entry.ExternalAnchorSidecarDigests)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	targetSet, err := externalAnchorVerificationTargetsFromSnapshot(entry.ExternalAnchorTargets)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return receipts, evidence, sidecars, targetSet, nil
}

func (l *Ledger) loadExternalAnchorVerificationTargetSetForSealLocked(segmentID string, sealDigest trustpolicy.Digest) ([]trustpolicy.ExternalAnchorVerificationTarget, error) {
	entry, err := l.requireSealIncrementalFoundationEntryLocked(segmentID, sealDigest, false)
	if err != nil || entry == nil {
		return nil, err
	}
	return externalAnchorVerificationTargetsFromSnapshot(entry.ExternalAnchorTargets)
}

func (l *Ledger) requireSealIncrementalFoundationEntryLocked(segmentID string, sealDigest trustpolicy.Digest, requireBaseline bool) (*externalAnchorIncrementalSealSnapshot, error) {
	foundation, err := l.ensureExternalAnchorIncrementalFoundationLocked()
	if err != nil {
		return nil, err
	}
	sealIdentity, err := sealDigest.Identity()
	if err != nil {
		return nil, fmt.Errorf("seal digest identity: %w", err)
	}
	entry, ok := foundation.Seals[sealIdentity]
	if !ok {
		if requireBaseline {
			return nil, fmt.Errorf("incremental verification foundation missing for seal %s; run full verification replay", sealIdentity)
		}
		return nil, nil
	}
	if err := requireSealIncrementalFoundationEntry(segmentID, sealIdentity, entry, requireBaseline); err != nil {
		return nil, err
	}
	return &entry, nil
}

func requireSealIncrementalFoundationEntry(segmentID, sealIdentity string, entry externalAnchorIncrementalSealSnapshot, requireBaseline bool) error {
	if strings.TrimSpace(entry.SegmentID) != "" && strings.TrimSpace(entry.SegmentID) != strings.TrimSpace(segmentID) {
		return fmt.Errorf("incremental verification foundation segment mismatch for seal %s", sealIdentity)
	}
	if !requireBaseline {
		return nil
	}
	if strings.TrimSpace(entry.BaselineVerificationReport) == "" {
		return fmt.Errorf("incremental verification baseline missing for seal %s; run full verification replay", sealIdentity)
	}
	return nil
}

func (l *Ledger) requireVerificationReportSidecarLocked(reportIdentity string) error {
	if strings.TrimSpace(reportIdentity) == "" {
		return fmt.Errorf("verification report digest identity is required")
	}
	reportDigest, err := digestFromIdentity(reportIdentity)
	if err != nil {
		return fmt.Errorf("verification report digest identity invalid: %w", err)
	}
	identity, _ := reportDigest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("verification report sidecar missing for baseline %s: %w", identity, err)
	}
	return nil
}

func (l *Ledger) loadReceiptsByIdentitiesLocked(identities []string) ([]trustpolicy.SignedObjectEnvelope, error) {
	receipts := make([]trustpolicy.SignedObjectEnvelope, 0, len(identities))
	for i := range identities {
		envelope, err := l.loadReceiptByIdentityLocked(identities[i])
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, envelope)
	}
	return receipts, nil
}

func (l *Ledger) loadReceiptByIdentityLocked(identity string) (trustpolicy.SignedObjectEnvelope, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("receipt digest identity invalid: %w", err)
	}
	identity, _ = digest.Identity()
	envelope := trustpolicy.SignedObjectEnvelope{}
	path := filepath.Join(l.rootDir, sidecarDirName, receiptsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := readJSONFile(path, &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	computed, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("compute receipt digest %s: %w", identity, err)
	}
	if mustDigestIdentity(computed) != identity {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("receipt digest mismatch for %s", identity)
	}
	return envelope, nil
}

func (l *Ledger) loadExternalAnchorEvidenceByIdentitiesLocked(identities []string) ([]trustpolicy.ExternalAnchorEvidencePayload, error) {
	evidence := make([]trustpolicy.ExternalAnchorEvidencePayload, 0, len(identities))
	for i := range identities {
		rec, err := l.loadExternalAnchorEvidenceByIdentityLocked(identities[i])
		if err != nil {
			return nil, err
		}
		evidence = append(evidence, rec)
	}
	return evidence, nil
}

func (l *Ledger) loadExternalAnchorEvidenceByIdentityLocked(identity string) (trustpolicy.ExternalAnchorEvidencePayload, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, fmt.Errorf("external anchor evidence digest identity invalid: %w", err)
	}
	identity, _ = digest.Identity()
	entryName := strings.TrimPrefix(identity, "sha256:") + ".json"
	rec, loadedDigest, ok, loadErr := l.loadExternalAnchorEvidenceEntry(entryName)
	if loadErr != nil {
		return trustpolicy.ExternalAnchorEvidencePayload{}, loadErr
	}
	if !ok {
		return trustpolicy.ExternalAnchorEvidencePayload{}, fmt.Errorf("external anchor evidence sidecar %s does not decode as external anchor evidence payload", identity)
	}
	if loadedDigest == nil || mustDigestIdentity(*loadedDigest) != identity {
		return trustpolicy.ExternalAnchorEvidencePayload{}, fmt.Errorf("external anchor evidence digest mismatch for %s", identity)
	}
	return rec, nil
}

func (l *Ledger) loadExternalAnchorSidecarDigestsByIdentitiesLocked(identities []string) ([]trustpolicy.Digest, error) {
	sidecars := make([]trustpolicy.Digest, 0, len(identities))
	for i := range identities {
		digest, err := l.loadExternalAnchorSidecarDigestByIdentityLocked(identities[i])
		if err != nil {
			return nil, err
		}
		sidecars = append(sidecars, digest)
	}
	return sidecars, nil
}
