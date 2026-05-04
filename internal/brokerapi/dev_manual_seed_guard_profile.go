package brokerapi

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func devManualLedgerMatchesSeedFootprint(root string) (bool, error) {
	checks := []func(string) (bool, error){
		devManualSeedSegmentsMatch,
		devManualSeedSegmentSealsMatch,
		devManualSeedReceiptsMatch,
		devManualSeedExternalAnchorEvidenceMatch,
		devManualSeedExternalAnchorSidecarsMatch,
		devManualSeedVerificationReportsMatch,
	}
	for _, check := range checks {
		ok, err := check(root)
		if err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

func devManualSeedSegmentsMatch(root string) (bool, error) {
	ok, err := devManualLedgerHasExactJSONNames(filepath.Join(root, "segments"), "segment-000001.json", "segment-000002.json")
	if err != nil || !ok {
		return ok, err
	}
	segmentOne := trustpolicy.AuditSegmentFilePayload{}
	if err := readDevManualJSON(filepath.Join(root, "segments", "segment-000001.json"), &segmentOne); err != nil {
		return false, err
	}
	if segmentOne.Header.SegmentState != trustpolicy.AuditSegmentStateSealed || len(segmentOne.Frames) == 0 {
		return false, nil
	}
	segmentTwo := trustpolicy.AuditSegmentFilePayload{}
	if err := readDevManualJSON(filepath.Join(root, "segments", "segment-000002.json"), &segmentTwo); err != nil {
		return false, err
	}
	if segmentTwo.Header.SegmentState != trustpolicy.AuditSegmentStateOpen || len(segmentTwo.Frames) != 0 {
		return false, nil
	}
	return true, nil
}

func devManualSeedSegmentSealsMatch(root string) (bool, error) {
	names, err := devManualJSONFileNames(filepath.Join(root, "sidecar", "segment-seals"))
	if err != nil {
		return false, err
	}
	return len(names) == 1, nil
}

func devManualSeedVerificationReportsMatch(root string) (bool, error) {
	names, err := devManualJSONFileNames(filepath.Join(root, "sidecar", "verification-reports"))
	if err != nil {
		return false, err
	}
	return len(names) == 1, nil
}

func devManualSeedReceiptsMatch(root string) (bool, error) {
	names, err := devManualJSONFileNames(filepath.Join(root, "sidecar", "receipts"))
	if err != nil {
		return false, err
	}
	return len(names) == 0, nil
}

func devManualSeedExternalAnchorEvidenceMatch(root string) (bool, error) {
	names, err := devManualJSONFileNames(filepath.Join(root, "sidecar", "external-anchor-evidence"))
	if err != nil {
		return false, err
	}
	return len(names) == 0, nil
}

func devManualSeedExternalAnchorSidecarsMatch(root string) (bool, error) {
	names, err := devManualJSONFileNames(filepath.Join(root, "sidecar", "external-anchor-sidecars"))
	if err != nil {
		return false, err
	}
	return len(names) == 0, nil
}

func devManualLedgerHasExactJSONNames(path string, expected ...string) (bool, error) {
	names, err := devManualJSONFileNames(path)
	if err != nil {
		return false, err
	}
	if len(names) != len(expected) {
		return false, nil
	}
	for index := range expected {
		if names[index] != expected[index] {
			return false, nil
		}
	}
	return true, nil
}

func devManualJSONFileNames(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if !isJSONFileEntry(entry) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
