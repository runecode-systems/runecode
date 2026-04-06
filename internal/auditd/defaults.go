package auditd

import (
	"os"
	"path/filepath"
)

func DefaultLedgerRoot() string {
	cacheDir, err := os.UserCacheDir()
	if err == nil && cacheDir != "" {
		return filepath.Join(cacheDir, "runecode", "audit-ledger")
	}
	configDir, configErr := os.UserConfigDir()
	if configErr == nil && configDir != "" {
		return filepath.Join(configDir, "runecode", "audit-ledger")
	}
	return filepath.Join(os.TempDir(), "runecode", "audit-ledger")
}
