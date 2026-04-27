package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func runScopedPlanRecords(records []artifacts.ArtifactRecord, runID string) ([]artifacts.ArtifactRecord, bool) {
	out := make([]artifacts.ArtifactRecord, 0, len(records))
	foundRun := false
	for _, record := range records {
		if strings.TrimSpace(record.RunID) != runID {
			continue
		}
		foundRun = true
		if !isTrustedGatePlanSourceRecord(record) {
			continue
		}
		out = append(out, record)
	}
	return out, foundRun
}

func isTrustedGatePlanSourceRecord(record artifacts.ArtifactRecord) bool {
	if !isTrustedGatePlanSourceRole(record.CreatedByRole) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(record.Reference.ContentType), "application/json")
}

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
