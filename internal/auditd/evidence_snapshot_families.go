package auditd

import (
	"os"
	"path/filepath"
	"reflect"
)

type evidenceSnapshotFamilies struct {
	receiptDigests            []string
	verificationReportDigests []string
	runtimeEvidenceDigests    []string
	verifierRecordDigests     []string
	eventContractDigests      []string
	signerEvidenceDigests     []string
	storagePostureDigests     []string
	typedRequestDigests       []string
	actionRequestDigests      []string
	controlPlaneDigests       []string
	providerInvocationDigests []string
	secretLeaseDigests        []string
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
	verificationDigests, err := l.verificationContractDigestFamiliesLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	receiptApprovalDigests, err := l.approvalDigestIdentitiesFromReceiptsLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	policyDigests, typedRequestDigests, actionRequestDigests, controlPlaneDigests, approvalDigests, requiredApprovalIDs, attestationDigests, instanceIdentityDigests, providerInvocationDigests, secretLeaseDigests, err := l.externalAnchorDerivedEvidenceIdentitiesLocked()
	if err != nil {
		return evidenceSnapshotFamilies{}, err
	}
	runtimeEvidenceDigests := append([]string{}, verificationDigests.signerEvidenceDigests...)
	return evidenceSnapshotFamilies{
		receiptDigests:            sidecars.receiptDigests,
		verificationReportDigests: sidecars.verificationReportDigests,
		runtimeEvidenceDigests:    runtimeEvidenceDigests,
		verifierRecordDigests:     verificationDigests.verifierRecordDigests,
		eventContractDigests:      verificationDigests.eventContractDigests,
		signerEvidenceDigests:     verificationDigests.signerEvidenceDigests,
		storagePostureDigests:     verificationDigests.storagePostureDigests,
		typedRequestDigests:       typedRequestDigests,
		actionRequestDigests:      actionRequestDigests,
		controlPlaneDigests:       controlPlaneDigests,
		providerInvocationDigests: providerInvocationDigests,
		secretLeaseDigests:        secretLeaseDigests,
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

type verificationContractDigestFamilies struct {
	verifierRecordDigests []string
	eventContractDigests  []string
	signerEvidenceDigests []string
	storagePostureDigests []string
}

func (l *Ledger) verificationContractDigestFamiliesLocked() (verificationContractDigestFamilies, error) {
	inputs, err := l.loadVerificationContractInputsOnlyLocked()
	if err != nil {
		return verificationContractDigestFamilies{}, err
	}
	verifierRecordDigests, err := canonicalIdentityFromAny(inputs.verifierRecords)
	if err != nil {
		return verificationContractDigestFamilies{}, err
	}
	eventContractDigests, err := canonicalIdentityFromAny(inputs.catalog)
	if err != nil {
		return verificationContractDigestFamilies{}, err
	}
	storagePostureDigests, err := canonicalIdentityFromPointerAny(inputs.storagePosture)
	if err != nil {
		return verificationContractDigestFamilies{}, err
	}
	signerEvidenceDigests, err := canonicalIdentityFromAny(inputs.signerEvidence)
	if err != nil {
		return verificationContractDigestFamilies{}, err
	}
	return verificationContractDigestFamilies{
		verifierRecordDigests: verifierRecordDigests,
		eventContractDigests:  eventContractDigests,
		signerEvidenceDigests: signerEvidenceDigests,
		storagePostureDigests: storagePostureDigests,
	}, nil
}

func canonicalIdentityFromAny(value any) ([]string, error) {
	digest, err := canonicalDigest(value)
	if err != nil {
		return nil, err
	}
	identity, err := digest.Identity()
	if err != nil {
		return nil, err
	}
	return []string{identity}, nil
}

func canonicalIdentityFromPointerAny(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, nil
	}
	return canonicalIdentityFromAny(value)
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
