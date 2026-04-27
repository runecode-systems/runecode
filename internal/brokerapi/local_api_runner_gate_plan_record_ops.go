package brokerapi

import (
	"fmt"
	"strings"
)

func ensureRunExistsForGatePlan(statuses map[string]string, runID string, foundRun bool) error {
	if foundRun {
		return nil
	}
	status, ok := statuses[runID]
	if !ok || strings.TrimSpace(status) == "" {
		return fmt.Errorf("run %q not found", runID)
	}
	return nil
}
