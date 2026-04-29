package launcherdaemon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func digestFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func backendError(code, msg string) error {
	if strings.TrimSpace(code) == "" {
		code = launcherbackend.BackendErrorCodeHypervisorLaunchFailed
	}
	if strings.TrimSpace(msg) == "" {
		msg = "backend launch failed"
	}
	return fmt.Errorf("backend_error_code=%s: %s", code, msg)
}
