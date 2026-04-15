package auditd

import (
	"fmt"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) ensureVerifierRecordDurableLocked(record trustpolicy.VerifierRecord) error {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	if err := requireVerificationContractFiles(contractsDir); err != nil {
		return err
	}
	path := filepath.Join(contractsDir, "verifier-records.json")
	records := []trustpolicy.VerifierRecord{}
	if err := readJSONFile(path, &records); err != nil {
		return fmt.Errorf("read verifier records: %w", err)
	}
	updated, changed, err := addVerifierRecordIfMissing(records, record)
	if err != nil {
		return err
	}
	if _, err := trustpolicy.NewVerifierRegistry(updated); err != nil {
		return fmt.Errorf("verify durable verifier records: %w", err)
	}
	if !changed {
		return nil
	}
	if err := writeCanonicalJSONFile(path, updated); err != nil {
		return fmt.Errorf("persist verifier records: %w", err)
	}
	return nil
}

func addVerifierRecordIfMissing(records []trustpolicy.VerifierRecord, record trustpolicy.VerifierRecord) ([]trustpolicy.VerifierRecord, bool, error) {
	keyID := record.KeyIDValue
	for index := range records {
		if records[index].KeyIDValue == keyID {
			if !sameAnchorVerifierIdentity(records[index], record) {
				return nil, false, fmt.Errorf("existing verifier record conflicts with anchor signer key_id_value %q", keyID)
			}
			return records, false, nil
		}
	}
	updated := make([]trustpolicy.VerifierRecord, 0, len(records)+1)
	updated = append(updated, records...)
	updated = append(updated, record)
	return updated, true, nil
}

func sameAnchorVerifierIdentity(existing trustpolicy.VerifierRecord, current trustpolicy.VerifierRecord) bool {
	if existing.KeyIDValue != current.KeyIDValue {
		return false
	}
	if existing.Alg != current.Alg {
		return false
	}
	if existing.PublicKey.Encoding != current.PublicKey.Encoding || existing.PublicKey.Value != current.PublicKey.Value {
		return false
	}
	if existing.LogicalPurpose != current.LogicalPurpose {
		return false
	}
	if existing.LogicalScope != current.LogicalScope {
		return false
	}
	return true
}
