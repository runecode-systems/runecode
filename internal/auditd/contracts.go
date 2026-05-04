package auditd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type VerificationConfiguration struct {
	VerifierRecords      []trustpolicy.VerifierRecord
	EventContractCatalog trustpolicy.AuditEventContractCatalog
	SignerEvidence       []trustpolicy.AuditSignerEvidenceReference
	StoragePosture       *trustpolicy.AuditStoragePostureEvidence
}

func (l *Ledger) ConfigureVerificationInputs(config VerificationConfiguration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	contractsDir := filepath.Join(l.rootDir, "contracts")
	if err := ensureDir(contractsDir); err != nil {
		return err
	}
	if err := writeConfiguredInputs(contractsDir, config); err != nil {
		return err
	}
	return l.persistMetaAuditReceiptsForVerificationContractsLocked(config)
}

func writeConfiguredInputs(contractsDir string, config VerificationConfiguration) error {
	if err := writeOptionalConfig(filepath.Join(contractsDir, "verifier-records.json"), len(config.VerifierRecords) > 0, config.VerifierRecords); err != nil {
		return err
	}
	if err := writeOptionalConfig(filepath.Join(contractsDir, "event-contract-catalog.json"), config.EventContractCatalog.SchemaID != "", config.EventContractCatalog); err != nil {
		return err
	}
	if err := writeOptionalConfig(filepath.Join(contractsDir, "signer-evidence.json"), len(config.SignerEvidence) > 0, config.SignerEvidence); err != nil {
		return err
	}
	return writeOptionalConfig(filepath.Join(contractsDir, "storage-posture.json"), config.StoragePosture != nil, config.StoragePosture)
}

func writeOptionalConfig(path string, enabled bool, value any) error {
	if !enabled {
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeCanonicalJSONFile(path, value)
}
