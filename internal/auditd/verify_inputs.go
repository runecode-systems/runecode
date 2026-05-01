package auditd

import (
	"fmt"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type verificationInputs struct {
	verifierRecords        []trustpolicy.VerifierRecord
	catalog                trustpolicy.AuditEventContractCatalog
	signerEvidence         []trustpolicy.AuditSignerEvidenceReference
	storagePosture         *trustpolicy.AuditStoragePostureEvidence
	knownSealDigests       []trustpolicy.Digest
	receipts               []trustpolicy.SignedObjectEnvelope
	externalAnchorEvidence []trustpolicy.ExternalAnchorEvidencePayload
	externalAnchorSidecars []trustpolicy.Digest
}

func (l *Ledger) loadVerificationInputsLocked() (verificationInputs, error) {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	if err := requireVerificationContractFiles(contractsDir); err != nil {
		return verificationInputs{}, err
	}
	inputs, err := loadVerificationContractInputs(contractsDir)
	if err != nil {
		return verificationInputs{}, err
	}
	if err := l.loadVerificationDurableInputsLocked(&inputs); err != nil {
		return verificationInputs{}, err
	}
	return inputs, nil
}

func loadVerificationContractInputs(contractsDir string) (verificationInputs, error) {
	inputs := verificationInputs{}
	if err := readJSONFile(filepath.Join(contractsDir, "event-contract-catalog.json"), &inputs.catalog); err != nil {
		return verificationInputs{}, err
	}
	if err := readJSONFile(filepath.Join(contractsDir, "verifier-records.json"), &inputs.verifierRecords); err != nil {
		return verificationInputs{}, err
	}
	if err := loadOptionalContractFiles(contractsDir, &inputs); err != nil {
		return verificationInputs{}, err
	}
	return inputs, nil
}

func (l *Ledger) loadVerificationDurableInputsLocked(inputs *verificationInputs) error {
	sealDigests, err := l.loadAllSealDigestsLocked()
	if err != nil {
		return err
	}
	inputs.knownSealDigests = sealDigests
	receipts, err := l.loadAllReceiptsLocked()
	if err != nil {
		return err
	}
	inputs.receipts = receipts
	externalEvidence, externalSidecars, err := l.loadExternalAnchorEvidenceLocked()
	if err != nil {
		return err
	}
	inputs.externalAnchorEvidence = externalEvidence
	inputs.externalAnchorSidecars = externalSidecars
	return nil
}

func requireVerificationContractFiles(contractsDir string) error {
	if !fileExists(filepath.Join(contractsDir, "event-contract-catalog.json")) {
		return fmt.Errorf("missing event contract catalog")
	}
	if !fileExists(filepath.Join(contractsDir, "verifier-records.json")) {
		return fmt.Errorf("missing verifier records")
	}
	return nil
}

func loadOptionalContractFiles(contractsDir string, inputs *verificationInputs) error {
	if fileExists(filepath.Join(contractsDir, "signer-evidence.json")) {
		if err := readJSONFile(filepath.Join(contractsDir, "signer-evidence.json"), &inputs.signerEvidence); err != nil {
			return err
		}
	}
	if !fileExists(filepath.Join(contractsDir, "storage-posture.json")) {
		return nil
	}
	var posture trustpolicy.AuditStoragePostureEvidence
	if err := readJSONFile(filepath.Join(contractsDir, "storage-posture.json"), &posture); err != nil {
		return err
	}
	inputs.storagePosture = &posture
	return nil
}
