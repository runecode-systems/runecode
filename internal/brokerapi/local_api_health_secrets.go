package brokerapi

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/runecode-ai/runecode/internal/secretsd"
)

func projectSecretsReadinessFromLocalState() (bool, string, *SecretsOperationalMetrics, *SecretStoragePosture) {
	root, rootErr := validatedSecretsStateRoot(defaultSecretsStateRoot())
	if rootErr != nil {
		return false, "degraded", nil, nil
	}
	if _, statErr := os.Stat(root); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return false, "failed", nil, nil
		}
		return false, "degraded", nil, nil
	}
	svc, err := secretsd.Open(root)
	if err != nil {
		if errors.Is(err, secretsd.ErrStateRecoveryFailed) {
			return false, "degraded", nil, nil
		}
		return false, "failed", nil, nil
	}
	snapshot := svc.RuntimeSnapshot()
	metrics := &SecretsOperationalMetrics{
		LeaseIssueCount:  snapshot.LeaseIssueCount,
		LeaseRenewCount:  snapshot.LeaseRenewCount,
		LeaseRevokeCount: snapshot.LeaseRevokeCount,
		LeaseDeniedCount: snapshot.LeaseDenyCount,
		ActiveLeaseCount: snapshot.ActiveLeaseCount,
	}
	return true, "ok", metrics, nil
}

func defaultSecretsStateRoot() string {
	if root := strings.TrimSpace(os.Getenv("RUNE_SECRETS_STATE_ROOT")); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".runecode-secretsd"
	}
	return filepath.Join(home, ".runecode", "secretsd")
}

func validatedSecretsStateRoot(root string) (string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return "", fmt.Errorf("secrets state root is required")
	}
	abs, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", err
	}
	left := filepath.Clean(abs)
	right := filepath.Clean(resolved)
	if rootHasSymlinkComponent(left, right) {
		return "", fmt.Errorf("secrets state root must not contain symlink components")
	}
	return validateSecretsRootType(abs)
}

func rootHasSymlinkComponent(left, right string) bool {
	if runtime.GOOS == "windows" {
		return !strings.EqualFold(left, right)
	}
	return left != right
}

func validateSecretsRootType(abs string) (string, error) {
	info, statErr := os.Lstat(abs)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return abs, nil
		}
		return "", statErr
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("secrets state root must not be a symlink")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("secrets state root must be a directory")
	}
	return abs, nil
}
