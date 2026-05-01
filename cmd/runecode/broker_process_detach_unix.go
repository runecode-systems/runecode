//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func prepareDetachedBrokerProcessIO(cmd *exec.Cmd, runtimeDir string) (func(), error) {
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return func() {}, err
	}
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return func() {}, err
	}
	logPath := filepath.Join(runtimeDir, "broker.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = stdin.Close()
		return func() {}, err
	}
	cmd.Stdin = stdin
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return func() {
		_ = stdin.Close()
		_ = logFile.Close()
	}, nil
}
