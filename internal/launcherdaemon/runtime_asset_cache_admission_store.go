package launcherdaemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func cacheResultForAdmission(found bool) string {
	if found {
		return launcherbackend.CacheResultHit
	}
	return launcherbackend.CacheResultMiss
}

func loadRuntimeAdmissionRecord(cacheRoot string, descriptorDigest string) (launcherbackend.RuntimeAdmissionRecord, bool, error) {
	path, err := runtimeAdmissionRecordPath(cacheRoot, descriptorDigest)
	if err != nil {
		return launcherbackend.RuntimeAdmissionRecord{}, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return launcherbackend.RuntimeAdmissionRecord{}, false, nil
		}
		return launcherbackend.RuntimeAdmissionRecord{}, false, fmt.Errorf("read persisted runtime admission record: %w", err)
	}
	record := launcherbackend.RuntimeAdmissionRecord{}
	if err := json.Unmarshal(data, &record); err != nil {
		return launcherbackend.RuntimeAdmissionRecord{}, false, fmt.Errorf("decode persisted runtime admission record: %w", err)
	}
	if err := record.Validate(); err != nil {
		return launcherbackend.RuntimeAdmissionRecord{}, false, fmt.Errorf("persisted runtime admission record invalid: %w", err)
	}
	return record, true, nil
}

func persistRuntimeAdmissionRecord(cacheRoot string, record launcherbackend.RuntimeAdmissionRecord) error {
	if err := record.Validate(); err != nil {
		return fmt.Errorf("persisted runtime admission record invalid: %w", err)
	}
	path, err := runtimeAdmissionRecordPath(cacheRoot, record.DescriptorDigest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create runtime admission cache directory: %w", err)
	}
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode persisted runtime admission record: %w", err)
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write persisted runtime admission record: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("finalize persisted runtime admission record: %w", err)
	}
	return nil
}

func runtimeAdmissionRecordPath(cacheRoot string, descriptorDigest string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(descriptorDigest), ":", 2)
	if len(parts) != 2 || parts[0] != "sha256" || parts[1] == "" {
		return "", fmt.Errorf("invalid descriptor digest")
	}
	return filepath.Join(cacheRoot, verifiedRuntimeAdmissionDir, parts[0], parts[1]+".json"), nil
}
