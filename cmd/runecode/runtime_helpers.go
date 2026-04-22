package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

func queryRepoProjectSubstratePosture(ctx context.Context, cfg brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	client, err := brokerapi.DialLocalRPC(ctx, cfg)
	if err != nil {
		return brokerapi.ProjectSubstratePostureGetResponse{}, errNoLiveBroker
	}
	defer client.Close()
	resp := brokerapi.ProjectSubstratePostureGetResponse{}
	errResp := client.Invoke(ctx, "project_substrate_posture_get", brokerapi.ProjectSubstratePostureGetRequest{
		SchemaID:      "runecode.protocol.v0.ProjectSubstratePostureGetRequest",
		SchemaVersion: localAPISchemaVersion,
		RequestID:     "runecode-cli-project-substrate-posture",
	}, &resp)
	if errResp != nil {
		return brokerapi.ProjectSubstratePostureGetResponse{}, fmt.Errorf("%s: %s", errResp.Error.Code, strings.TrimSpace(errResp.Error.Message))
	}
	return resp, nil
}

func recoverStaleRepoRuntimeArtifacts(scope localbootstrap.RepoScope, cfg brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, bool, error) {
	posture, err := currentRepoLifecycle(scope, cfg)
	if err == nil {
		return posture, true, nil
	}
	if !errors.Is(err, errNoLiveBroker) {
		return brokerapi.BrokerProductLifecyclePosture{}, false, err
	}

	pidAlive, err := cleanupStaleBrokerPIDFile(scope)
	if err != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, false, err
	}
	if pidAlive {
		return brokerapi.BrokerProductLifecyclePosture{}, false, errBrokerProcessUnreachable
	}

	posture, err = currentRepoLifecycle(scope, cfg)
	if err == nil {
		return posture, true, nil
	}
	if !errors.Is(err, errNoLiveBroker) {
		return brokerapi.BrokerProductLifecyclePosture{}, false, err
	}

	if err := removeStaleBrokerSocket(scope); err != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, false, err
	}
	return brokerapi.BrokerProductLifecyclePosture{}, false, nil
}

func cleanupStaleBrokerPIDFile(scope localbootstrap.RepoScope) (bool, error) {
	path := pidFilePath(scope)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(content)))
	if parseErr != nil || pid <= 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		return false, nil
	}
	if processIsAlive(pid) {
		return true, nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return false, nil
}

func removeStaleBrokerSocket(scope localbootstrap.RepoScope) error {
	path := filepath.Join(scope.LocalRuntimeDir, scope.LocalSocketName)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func processIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := process.Signal(syscall.Signal(0)); err == nil {
		return true
	} else if errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}

func validateProjectSubstrateIdentity(scope localbootstrap.RepoScope, posture brokerapi.ProjectSubstratePostureGetResponse) error {
	if comparableRepoRoot(posture.RepositoryRoot) != comparableRepoRoot(scope.RepositoryRoot) {
		return fmt.Errorf("reachable broker repository root does not match authoritative repository root")
	}
	return nil
}

func comparableRepoRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func sanitizeCLIField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	b := strings.Builder{}
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] == 0x1b {
			i = skipANSIEscape(value, i)
			continue
		}
		r := rune(value[i])
		if unicode.IsControl(r) {
			continue
		}
		b.WriteByte(value[i])
	}
	return strings.TrimSpace(b.String())
}

func skipANSIEscape(value string, start int) int {
	if start+1 >= len(value) || value[start+1] != '[' {
		return start
	}
	idx := start + 2
	for idx < len(value) {
		ch := value[idx]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			return idx
		}
		idx++
	}
	return len(value) - 1
}
