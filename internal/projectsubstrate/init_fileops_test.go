package projectsubstrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceConfigFileReturnsErrorWhenBackupCleanupFails(t *testing.T) {
	src, dst := writeReplaceConfigFixtures(t)
	backupRemoveErr := errors.New("simulated backup remove failure")
	installReplaceConfigSeams(t, backupRemoveErr)

	err := replaceConfigFile(src, dst)
	if err == nil {
		t.Fatal("replaceConfigFile error = nil, want backup remove failure")
	}
	if !errors.Is(err, backupRemoveErr) {
		t.Fatalf("replaceConfigFile error = %v, want wrapped backup remove failure", err)
	}
	if !strings.Contains(err.Error(), "replacement applied") {
		t.Fatalf("replaceConfigFile error = %q, want post-apply cleanup context", err.Error())
	}
	if !strings.Contains(err.Error(), "remove backup") {
		t.Fatalf("replaceConfigFile error = %q, want backup cleanup context", err.Error())
	}
	assertReplacedConfig(t, dst)
}

func TestReplaceConfigFileIgnoresNotExistFromBackupCleanup(t *testing.T) {
	src, dst := writeReplaceConfigFixtures(t)
	installReplaceConfigSeams(t, os.ErrNotExist)

	err := replaceConfigFile(src, dst)
	if err != nil {
		t.Fatalf("replaceConfigFile returned error: %v", err)
	}
	assertReplacedConfig(t, dst)
}

func writeReplaceConfigFixtures(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "runecontext.yaml.tmp")
	dst := filepath.Join(root, "runecontext.yaml")
	if err := os.WriteFile(src, []byte("next config"), 0o644); err != nil {
		t.Fatalf("WriteFile(src) returned error: %v", err)
	}
	if err := os.WriteFile(dst, []byte("prior config"), 0o644); err != nil {
		t.Fatalf("WriteFile(dst) returned error: %v", err)
	}
	return src, dst
}

func installReplaceConfigSeams(t *testing.T, backupCleanupErr error) {
	t.Helper()
	originalRename := renameConfigFile
	originalRemoveBackup := removeConfigBackup
	t.Cleanup(func() {
		renameConfigFile = originalRename
		removeConfigBackup = originalRemoveBackup
	})
	renameCalls := 0
	renameConfigFile = func(oldpath, newpath string) error {
		renameCalls++
		if renameCalls == 1 {
			return errors.New("simulated first rename failure")
		}
		return os.Rename(oldpath, newpath)
	}
	removeConfigBackup = func(_ string) error {
		return backupCleanupErr
	}
}

func assertReplacedConfig(t *testing.T, dst string) {
	t.Helper()
	gotConfig, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("ReadFile(dst) returned error: %v", readErr)
	}
	if string(gotConfig) != "next config" {
		t.Fatalf("dst content = %q, want replacement content", string(gotConfig))
	}
}
