package auditd

import (
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type evidenceSnapshotFamilies struct {
	receiptDigests            []string
	verificationReportDigests []string
	runtimeEvidenceDigests    []string
	policyDigests             []string
	requiredApprovalIDs       []string
	approvalDigests           []string
	attestationDigests        []string
	instanceIdentityDigests   []string
	anchorEvidenceDigests     []string
}

func (l *Ledger) collectEvidenceSnapshotFamiliesLocked() (evidenceSnapshotFamilies, error) {
	sidecars, err := l.collectEvidenceSnapshotSidecarsLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	runtimeEvidenceDigests, err := l.runtimeEvidenceDigestIdentitiesLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	receiptApprovalDigests, err := l.approvalDigestIdentitiesFromReceiptsLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	policyDigests, approvalDigests, requiredApprovalIDs, attestationDigests, instanceIdentityDigests, err := l.externalAnchorDerivedEvidenceIdentitiesLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	return evidenceSnapshotFamilies{
		receiptDigests:            sidecars.receiptDigests,
		verificationReportDigests: sidecars.verificationReportDigests,
		runtimeEvidenceDigests:    runtimeEvidenceDigests,
		policyDigests:             policyDigests,
		requiredApprovalIDs:       requiredApprovalIDs,
		approvalDigests:           append(approvalDigests, receiptApprovalDigests...),
		attestationDigests:        attestationDigests,
		instanceIdentityDigests:   instanceIdentityDigests,
		anchorEvidenceDigests:     append(sidecars.anchorEvidenceDigests, sidecars.anchorSidecarDigests...),
	}, nil
}

type evidenceSnapshotSidecars struct {
	receiptDigests            []string
	verificationReportDigests []string
	anchorEvidenceDigests     []string
	anchorSidecarDigests      []string
}

func (l *Ledger) collectEvidenceSnapshotSidecarsLocked() (evidenceSnapshotSidecars, error) {
	receiptDigests, err := l.sidecarDigestIdentitiesLocked(receiptsDirName)
	if err != nil {
		return evidenceSnapshotSidecars{}, err
	}
	verificationReportDigests, err := l.sidecarDigestIdentitiesLocked(verificationReportsDirName)
	if err != nil {
		return evidenceSnapshotSidecars{}, err
	}
	anchorEvidenceDigests, err := l.sidecarDigestIdentitiesLocked(externalAnchorEvidenceDir)
	if err != nil {
		return evidenceSnapshotSidecars{}, err
	}
	anchorSidecarDigests, err := l.sidecarDigestIdentitiesLocked(externalAnchorSidecarsDir)
	if err != nil {
		return evidenceSnapshotSidecars{}, err
	}
	return evidenceSnapshotSidecars{
		receiptDigests:            receiptDigests,
		verificationReportDigests: verificationReportDigests,
		anchorEvidenceDigests:     anchorEvidenceDigests,
		anchorSidecarDigests:      anchorSidecarDigests,
	}, nil
}

func (l *Ledger) sidecarDigestIdentitiesLocked(dirName string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, dirName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		identity, ok, err := digestIdentityFromSidecarName(entry.Name())
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, identity)
		}
	}
	return out, nil
}

func (l *Ledger) runtimeEvidenceDigestIdentitiesLocked() ([]string, error) {
	path := filepath.Join(l.rootDir, "contracts", "signer-evidence.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	refs := []trustpolicy.AuditSignerEvidenceReference{}
	if err := readJSONFile(path, &refs); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(refs))
	for i := range refs {
		identity, err := refs[i].Digest.Identity()
		if err != nil {
			return nil, err
		}
		out = append(out, identity)
	}
	return out, nil
}

func (l *Ledger) approvalDigestIdentitiesFromReceiptsLocked() ([]string, error) {
	receipts, err := l.loadAllReceiptsLocked()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(receipts))
	for i := range receipts {
		identity, ok, err := approvalDecisionDigestFromReceipt(receipts[i])
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, identity)
		}
	}
	return out, nil
}
