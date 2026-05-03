//go:build !runecode_devseed

package brokerapi

import "fmt"

func seedDevManualAuditLedger(root string, profile string) (string, error) {
	_, _ = root, profile
	return "", fmt.Errorf("dev manual seeding unavailable in this build")
}
