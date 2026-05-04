package brokerapi

import "path/filepath"

func devManualLedgerSeedMarkerPath(root string) string {
	return filepath.Join(root, "contracts", "dev-manual-seed.marker")
}
