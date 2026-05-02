package auditd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) indexStatusLocked() (indexed int, total int, err error) {
	index, idxErr := l.ensureDerivedIndexLocked()
	if idxErr == nil {
		indexed = index.TotalRecords
	} else {
		state, readErr := l.loadState()
		if readErr == nil {
			indexed = state.LastIndexedRecordCount
		}
	}
	segments, err := l.listSegments()
	if err != nil {
		return 0, 0, err
	}
	for _, segment := range segments {
		total += len(segment.Frames)
	}
	if indexed > total {
		indexed = 0
	}
	return indexed, total, nil
}

func hasVerificationInputs(l *Ledger) bool {
	return validateVerificationInputs(l) == nil
}

func validateVerificationInputs(l *Ledger) error {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	eventCatalogPath := filepath.Join(contractsDir, "event-contract-catalog.json")
	verifierRecordsPath := filepath.Join(contractsDir, "verifier-records.json")
	if !fileExists(eventCatalogPath) {
		return fmt.Errorf("missing event contract catalog")
	}
	if !fileExists(verifierRecordsPath) {
		return fmt.Errorf("missing verifier records")
	}
	catalog := trustpolicy.AuditEventContractCatalog{}
	if err := readJSONFile(eventCatalogPath, &catalog); err != nil {
		return err
	}
	if err := trustpolicy.ValidateAuditEventContractCatalogForRuntime(catalog); err != nil {
		return err
	}
	verifierRecords := []trustpolicy.VerifierRecord{}
	if err := readJSONFile(verifierRecordsPath, &verifierRecords); err != nil {
		return err
	}
	if _, err := trustpolicy.NewVerifierRegistry(verifierRecords); err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
