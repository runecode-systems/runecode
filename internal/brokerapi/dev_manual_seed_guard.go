package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const devManualSeedMarkerMaxBytes = 64

func ensureDevManualSeedLedgerAllowed(root string) (string, error) {
	canonicalRoot, err := canonicalPathWithoutSymlinkComponents(root)
	if err != nil {
		return "", err
	}
	root = canonicalRoot
	if same, err := pathsReferToSameLocation(root, auditd.DefaultLedgerRoot()); err != nil {
		return "", err
	} else if same {
		return "", fmt.Errorf("dev manual seeding refuses default audit ledger root")
	}
	if hasDevManualSeedMarker(root) {
		matches, err := devManualLedgerMatchesSeedFootprint(root)
		if err != nil {
			return "", err
		}
		if matches {
			return root, nil
		}
	}
	populated, err := devManualLedgerHasRecordedData(root)
	if err != nil {
		return "", err
	}
	if populated {
		return "", fmt.Errorf("dev manual seeding refuses populated audit ledger root")
	}
	return root, nil
}

func hasDevManualSeedMarker(root string) bool {
	markerPath := devManualLedgerSeedMarkerPath(root)
	info, err := os.Lstat(markerPath)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false
	}
	f, err := openReadOnlyNoFollow(markerPath)
	if err != nil {
		return false
	}
	defer f.Close()
	openedInfo, err := f.Stat()
	if err != nil {
		return false
	}
	if !os.SameFile(info, openedInfo) {
		return false
	}
	b, err := io.ReadAll(io.LimitReader(f, devManualSeedMarkerMaxBytes+1))
	if err != nil {
		return false
	}
	if len(b) > devManualSeedMarkerMaxBytes {
		return false
	}
	if strings.TrimSpace(string(b)) != devManualSeedProfile {
		return false
	}
	return true
}

func canonicalPathWithoutSymlinkComponents(path string) (string, error) {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cleanPath, nil
		}
		return "", err
	}
	if !sameFilesystemPath(cleanPath, resolvedPath) {
		return "", fmt.Errorf("dev manual seeding refuses ledger root paths containing symlink components")
	}
	return cleanPath, nil
}

func pathsReferToSameLocation(first string, second string) (bool, error) {
	firstAbs, err := filepath.Abs(first)
	if err != nil {
		return false, err
	}
	secondAbs, err := filepath.Abs(second)
	if err != nil {
		return false, err
	}
	firstResolved, err := filepath.EvalSymlinks(firstAbs)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		firstResolved = firstAbs
	}
	secondResolved, err := filepath.EvalSymlinks(secondAbs)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		secondResolved = secondAbs
	}
	firstInfo, firstStatErr := os.Stat(firstResolved)
	secondInfo, secondStatErr := os.Stat(secondResolved)
	if firstStatErr == nil && secondStatErr == nil {
		return os.SameFile(firstInfo, secondInfo), nil
	}
	return sameFilesystemPath(firstResolved, secondResolved), nil
}

func sameFilesystemPath(first string, second string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(first), filepath.Clean(second))
	}
	return filepath.Clean(first) == filepath.Clean(second)
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
	for _, entry := range entries {
		isSegment, bootstrap, err := devManualSegmentEntryState(root, entry)
		if err != nil {
			return false, err
		}
		if !isSegment {
			continue
		}
		if !bootstrap {
			return true, nil
		}
	}
	return false, nil
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
