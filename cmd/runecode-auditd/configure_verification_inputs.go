package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func handleConfigureVerificationInputs(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("configure-verification-inputs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ledgerRoot := fs.String("ledger-root", auditd.DefaultLedgerRoot(), "path to audit ledger root")
	verifierRecordsPath := fs.String("verifier-records", "", "path to verifier records JSON")
	eventContractCatalogPath := fs.String("event-contract-catalog", "", "path to audit event contract catalog JSON")
	signerEvidencePath := fs.String("signer-evidence", "", "path to signer evidence references JSON")
	storagePosturePath := fs.String("storage-posture", "", "path to storage posture JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "configure-verification-inputs usage: runecode-auditd configure-verification-inputs --verifier-records records.json --event-contract-catalog catalog.json [--signer-evidence evidence.json] [--storage-posture posture.json] [--ledger-root path]"}
	}
	if *verifierRecordsPath == "" || *eventContractCatalogPath == "" {
		return &usageError{message: "configure-verification-inputs requires --verifier-records and --event-contract-catalog"}
	}
	config, err := loadVerificationConfiguration(*verifierRecordsPath, *eventContractCatalogPath, *signerEvidencePath, *storagePosturePath)
	if err != nil {
		return err
	}
	ledger, err := auditd.Open(*ledgerRoot)
	if err != nil {
		return err
	}
	if err := ledger.ConfigureVerificationInputs(config); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "configured")
	return err
}

func loadVerificationConfiguration(verifierRecordsPath, eventContractCatalogPath, signerEvidencePath, storagePosturePath string) (auditd.VerificationConfiguration, error) {
	config := auditd.VerificationConfiguration{}
	verifiers, err := loadVerifierRecordsForConfiguration(verifierRecordsPath)
	if err != nil {
		return config, err
	}
	config.VerifierRecords = verifiers
	catalog, err := loadEventContractCatalogForConfiguration(eventContractCatalogPath)
	if err != nil {
		return config, err
	}
	config.EventContractCatalog = catalog
	signerEvidence, err := loadOptionalSignerEvidenceForConfiguration(signerEvidencePath)
	if err != nil {
		return config, err
	}
	config.SignerEvidence = signerEvidence
	storagePosture, err := loadOptionalStoragePostureForConfiguration(storagePosturePath)
	if err != nil {
		return config, err
	}
	config.StoragePosture = storagePosture
	return config, nil
}

func loadVerifierRecordsForConfiguration(filePath string) ([]trustpolicy.VerifierRecord, error) {
	verifiers := []trustpolicy.VerifierRecord{}
	if err := loadJSONFile(filePath, &verifiers); err != nil {
		return nil, fmt.Errorf("invalid verifier records: %w", err)
	}
	if _, err := trustpolicy.NewVerifierRegistry(verifiers); err != nil {
		return nil, fmt.Errorf("invalid verifier records: %w", err)
	}
	return verifiers, nil
}

func loadEventContractCatalogForConfiguration(filePath string) (trustpolicy.AuditEventContractCatalog, error) {
	catalog := trustpolicy.AuditEventContractCatalog{}
	if err := loadJSONFile(filePath, &catalog); err != nil {
		return catalog, fmt.Errorf("invalid event contract catalog: %w", err)
	}
	if err := trustpolicy.ValidateAuditEventContractCatalogForRuntime(catalog); err != nil {
		return catalog, fmt.Errorf("invalid event contract catalog: %w", err)
	}
	return catalog, nil
}

func loadOptionalSignerEvidenceForConfiguration(filePath string) ([]trustpolicy.AuditSignerEvidenceReference, error) {
	if filePath == "" {
		return nil, nil
	}
	signerEvidence := []trustpolicy.AuditSignerEvidenceReference{}
	if err := loadJSONFile(filePath, &signerEvidence); err != nil {
		return nil, fmt.Errorf("invalid signer evidence: %w", err)
	}
	return signerEvidence, nil
}

func loadOptionalStoragePostureForConfiguration(filePath string) (*trustpolicy.AuditStoragePostureEvidence, error) {
	if filePath == "" {
		return nil, nil
	}
	storagePosture := &trustpolicy.AuditStoragePostureEvidence{}
	if err := loadJSONFile(filePath, storagePosture); err != nil {
		return nil, fmt.Errorf("invalid storage posture: %w", err)
	}
	if err := trustpolicy.ValidateAuditStoragePostureEvidence(*storagePosture); err != nil {
		return nil, fmt.Errorf("invalid storage posture: %w", err)
	}
	return storagePosture, nil
}
