//go:build !runecode_devseed

package brokerapi

import "fmt"

func seedDevManualAuditLedger(root string) (string, error) {
	_ = root
	return "", fmt.Errorf("dev manual seeding unavailable in this build")
}
