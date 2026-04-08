package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
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
	if err := validateSignerEvidenceReferencesForConfiguration(signerEvidence); err != nil {
		return nil, err
	}
	return signerEvidence, nil
}

func validateSignerEvidenceReferencesForConfiguration(references []trustpolicy.AuditSignerEvidenceReference) error {
	seen := map[string]struct{}{}
	for index := range references {
		digestIdentity, err := references[index].Digest.Identity()
		if err != nil {
			return fmt.Errorf("invalid signer evidence: signer_evidence[%d].digest: %w", index, err)
		}
		if _, exists := seen[digestIdentity]; exists {
			return fmt.Errorf("invalid signer evidence: duplicate signer_evidence digest %q", digestIdentity)
		}
		seen[digestIdentity] = struct{}{}
		if err := trustpolicy.ValidateAuditSignerEvidence(references[index].Evidence); err != nil {
			return fmt.Errorf("invalid signer evidence: signer_evidence[%d].evidence: %w", index, err)
		}
		computedDigest, err := signerEvidenceReferenceDigestIdentity(references[index].Evidence)
		if err != nil {
			return fmt.Errorf("invalid signer evidence: signer_evidence[%d].evidence digest: %w", index, err)
		}
		if digestIdentity != computedDigest {
			return fmt.Errorf("invalid signer evidence: signer_evidence[%d].digest %q does not match evidence digest %q", index, digestIdentity, computedDigest)
		}
	}
	return nil
}

func signerEvidenceReferenceDigestIdentity(evidence trustpolicy.AuditSignerEvidence) (string, error) {
	b, err := json.Marshal(evidence)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
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
