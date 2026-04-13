package secretsd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) validateSecretRecords(records map[string]secretRecord) error {
	for ref, rec := range records {
		if !validSecretRecord(ref, rec) {
			return fmt.Errorf("%w", ErrStateRecoveryFailed)
		}
		materialPath, err := s.secretMaterialPath(rec)
		if err != nil {
			return err
		}
		if err := validateSecretMaterialDigest(materialPath, rec.MaterialDigest); err != nil {
			return err
		}
	}
	return nil
}

func validSecretRecord(secretRef string, rec secretRecord) bool {
	return strings.TrimSpace(secretRef) != "" && strings.TrimSpace(rec.SecretID) != "" && strings.TrimSpace(rec.MaterialDigest) != ""
}

func validateSecretMaterialDigest(materialPath, expectedDigest string) error {
	material, err := os.ReadFile(materialPath)
	if err != nil {
		return fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	if digestHex(material) != expectedDigest {
		return fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	return nil
}

func validateLeaseRecords(leases map[string]leaseRecord, secrets map[string]secretRecord) error {
	for _, lease := range leases {
		if !validLeaseRecord(lease) {
			return fmt.Errorf("%w", ErrStateRecoveryFailed)
		}
		if _, ok := secrets[lease.SecretRef]; !ok {
			return fmt.Errorf("%w", ErrStateRecoveryFailed)
		}
	}
	return nil
}

func validLeaseRecord(lease leaseRecord) bool {
	if strings.TrimSpace(lease.LeaseID) == "" || strings.TrimSpace(lease.SecretRef) == "" {
		return false
	}
	if lease.Status != leaseStatusActive && lease.Status != leaseStatusRevoked {
		return false
	}
	if lease.Status == leaseStatusRevoked && lease.RevokedAt == nil {
		return false
	}
	return true
}

func (s *Service) secretMaterialPath(rec secretRecord) (string, error) {
	fileName := filepath.Base(rec.MaterialFile)
	if fileName != rec.MaterialFile || fileName == "." || fileName == string(filepath.Separator) {
		return "", fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	return filepath.Join(s.root, secretsDirName, fileName), nil
}
