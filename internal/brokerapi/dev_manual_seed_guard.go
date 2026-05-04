package brokerapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const devManualSeedMarkerMaxBytes = 64

func ensureDevManualSeedLedgerAllowed(root string, profile string) (string, error) {
	canonicalRoot, err := normalizeDevManualSeedRoot(root)
	if err != nil {
		return "", err
	}
	markerState, markerProfile, err := devManualSeedMarkerState(canonicalRoot)
	if err != nil {
		return "", err
	}
	if markerState == devManualSeedMarkerInvalid {
		return "", fmt.Errorf("dev manual seeding refuses tampered seed marker")
	}
	if markerState == devManualSeedMarkerValid {
		_ = markerProfile
		return canonicalRoot, ensureValidSeedMarkerLedger(canonicalRoot)
	}
	populated, err := devManualLedgerHasRecordedData(canonicalRoot)
	if err != nil {
		return "", err
	}
	if populated {
		return "", fmt.Errorf("dev manual seeding refuses populated audit ledger root")
	}
	return canonicalRoot, nil
}

func normalizeDevManualSeedRoot(root string) (string, error) {
	canonicalRoot, err := canonicalPathWithoutSymlinkComponents(root)
	if err != nil {
		return "", err
	}
	if same, err := pathsReferToSameLocation(canonicalRoot, auditd.DefaultLedgerRoot()); err != nil {
		return "", err
	} else if same {
		return "", fmt.Errorf("dev manual seeding refuses default audit ledger root")
	}
	return canonicalRoot, nil
}

func ensureValidSeedMarkerLedger(root string) error {
	matches, err := devManualLedgerMatchesSeedFootprint(root)
	if err != nil {
		return err
	}
	if !matches {
		return fmt.Errorf("dev manual seeding refuses populated audit ledger root")
	}
	return nil
}

type devManualSeedMarkerStatus int

const (
	devManualSeedMarkerAbsent devManualSeedMarkerStatus = iota
	devManualSeedMarkerValid
	devManualSeedMarkerInvalid
)

func hasDevManualSeedMarker(root string) bool {
	state, _, _ := devManualSeedMarkerState(root)
	return state == devManualSeedMarkerValid
}

func devManualSeedMarkerState(root string) (devManualSeedMarkerStatus, string, error) {
	markerPath := devManualLedgerSeedMarkerPath(root)
	info, status, err := loadSeedMarkerInfo(markerPath)
	if status != devManualSeedMarkerValid {
		return status, "", err
	}
	status, profile := evaluateSeedMarkerFile(markerPath, info)
	return status, profile, nil
}

func loadSeedMarkerInfo(markerPath string) (os.FileInfo, devManualSeedMarkerStatus, error) {
	info, err := os.Lstat(markerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, devManualSeedMarkerAbsent, nil
		}
		return nil, devManualSeedMarkerInvalid, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, devManualSeedMarkerInvalid, nil
	}
	return info, devManualSeedMarkerValid, nil
}

func evaluateSeedMarkerFile(markerPath string, info os.FileInfo) (devManualSeedMarkerStatus, string) {
	f, err := openReadOnlyNoFollow(markerPath)
	if err != nil {
		return devManualSeedMarkerInvalid, ""
	}
	defer f.Close()
	openedInfo, err := f.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return devManualSeedMarkerInvalid, ""
	}
	b, err := io.ReadAll(io.LimitReader(f, devManualSeedMarkerMaxBytes+1))
	if err != nil || len(b) > devManualSeedMarkerMaxBytes {
		return devManualSeedMarkerInvalid, ""
	}
	profile := strings.TrimSpace(string(b))
	for _, supported := range SupportedDevManualSeedProfiles() {
		if profile == supported {
			return devManualSeedMarkerValid, profile
		}
	}
	return devManualSeedMarkerInvalid, ""
}

func canonicalPathWithoutSymlinkComponents(path string) (string, error) {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if err := rejectLinkedPathComponents(filepath.Dir(cleanPath)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return "", fmt.Errorf("dev manual seeding refuses ledger root paths containing symlink components")
		}
		return "", err
	}
	info, err := os.Lstat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cleanPath, nil
		}
		return "", err
	}
	linked, err := pathEntryIsLinkOrReparse(cleanPath, info)
	if err != nil {
		return "", err
	}
	if linked {
		return "", fmt.Errorf("dev manual seeding refuses ledger root paths containing symlink components")
	}
	return cleanPath, nil
}

func devManualLedgerHasRecordedData(root string) (bool, error) {
	if populated, err := devManualLedgerHasNonBootstrapSegments(root); populated || err != nil {
		return populated, err
	}
	if populated, err := devManualLedgerHasRecordedSidecars(root); populated || err != nil {
		return populated, err
	}
	return devManualLedgerHasJSON(filepath.Join(root, "contracts"))
}

func devManualLedgerHasRecordedSidecars(root string) (bool, error) {
	for _, rel := range []string{
		filepath.Join("sidecar", "segment-seals"),
		filepath.Join("sidecar", "receipts"),
		filepath.Join("sidecar", "external-anchor-evidence"),
		filepath.Join("sidecar", "external-anchor-sidecars"),
		filepath.Join("sidecar", "verification-reports"),
	} {
		if populated, err := devManualLedgerHasJSON(filepath.Join(root, rel)); populated || err != nil {
			return populated, err
		}
	}
	return false, nil
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
