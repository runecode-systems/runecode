package brokerapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func ensureDevManualSeedLedgerAllowed(root string) error {
	if filepath.Clean(root) == filepath.Clean(auditd.DefaultLedgerRoot()) {
		return fmt.Errorf("dev manual seeding refuses default audit ledger root")
	}
	if hasDevManualSeedMarker(root) {
		return nil
	}
	populated, err := devManualLedgerHasRecordedData(root)
	if err != nil {
		return err
	}
	if populated {
		return fmt.Errorf("dev manual seeding refuses populated audit ledger root")
	}
	return nil
}

func hasDevManualSeedMarker(root string) bool {
	_, err := os.Stat(devManualLedgerSeedMarkerPath(root))
	return err == nil
}

func devManualLedgerHasRecordedData(root string) (bool, error) {
	if populated, err := devManualLedgerHasNonBootstrapSegments(root); populated || err != nil {
		return populated, err
	}
	if populated, err := devManualLedgerHasJSON(filepath.Join(root, "sidecar", "segment-seals")); populated || err != nil {
		return populated, err
	}
	if populated, err := devManualLedgerHasJSON(filepath.Join(root, "sidecar", "verification-reports")); populated || err != nil {
		return populated, err
	}
	return devManualLedgerHasJSON(filepath.Join(root, "contracts"))
}

func devManualLedgerHasNonBootstrapSegments(root string) (bool, error) {
	entries, err := os.ReadDir(filepath.Join(root, "segments"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	seen := 0
	for _, entry := range entries {
		isSegment, bootstrap, err := devManualSegmentEntryState(root, entry)
		if err != nil {
			return false, err
		}
		if !isSegment {
			continue
		}
		seen++
		if !bootstrap {
			return true, nil
		}
	}
	return seen > 1, nil
}

func devManualSegmentEntryState(root string, entry os.DirEntry) (bool, bool, error) {
	if !isJSONFileEntry(entry) {
		return false, false, nil
	}
	if entry.Name() != "segment-000001.json" {
		return true, false, nil
	}
	bootstrap, err := isBootstrapOpenSegment(filepath.Join(root, "segments", entry.Name()))
	if err != nil {
		return false, false, err
	}
	return true, bootstrap, nil
}

func isBootstrapOpenSegment(path string) (bool, error) {
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := readDevManualJSON(path, &segment); err != nil {
		return false, err
	}
	return segment.Header.SegmentState == trustpolicy.AuditSegmentStateOpen && len(segment.Frames) == 0, nil
}

func devManualLedgerHasJSON(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if isJSONFileEntry(entry) {
			return true, nil
		}
	}
	return false, nil
}

func isJSONFileEntry(entry os.DirEntry) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json")
}

func readDevManualJSON(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
