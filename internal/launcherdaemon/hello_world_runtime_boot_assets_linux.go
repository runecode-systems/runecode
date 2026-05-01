//go:build linux

package launcherdaemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	helloWorldBuildTimeout = 30 * time.Second
)

var helloWorldKernelPathResolver = resolveHelloWorldKernelPath

func prepareHelloWorldBootAssets(workRoot string, cacheRoot string) (string, string, error) {
	kernelPath, err := helloWorldKernelPathResolver()
	if err != nil {
		return "", "", fmt.Errorf("prepare hello-world boot assets: %w", err)
	}
	stagingRoot := filepath.Join(workRoot, "hello-world-boot-assets")
	if err := os.MkdirAll(stagingRoot, 0o700); err != nil {
		return "", "", fmt.Errorf("prepare hello-world boot assets: create staging root: %w", err)
	}
	stagingDir, err := os.MkdirTemp(stagingRoot, "build-")
	if err != nil {
		return "", "", fmt.Errorf("prepare hello-world boot assets: create staging dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()
	ctx, cancel := context.WithTimeout(context.Background(), helloWorldBuildTimeout)
	defer cancel()
	initrdPath, err := buildHelloInitramfs(ctx, stagingDir)
	if err != nil {
		return "", "", fmt.Errorf("prepare hello-world boot assets: build initramfs: %w", err)
	}
	kernelDigest, err := seedHelloWorldRuntimeAssetFile(cacheRoot, kernelPath)
	if err != nil {
		return "", "", fmt.Errorf("prepare hello-world boot assets: seed kernel: %w", err)
	}
	initrdDigest, err := seedHelloWorldRuntimeAssetFile(cacheRoot, initrdPath)
	if err != nil {
		return "", "", fmt.Errorf("prepare hello-world boot assets: seed initrd: %w", err)
	}
	return kernelDigest, initrdDigest, nil
}

func resolveHelloWorldKernelPath() (string, error) {
	for _, candidate := range helloWorldKernelCandidates(currentLinuxKernelRelease()) {
		if err := validateHelloWorldKernelPath(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no readable host kernel image found")
}

func currentLinuxKernelRelease() string {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func helloWorldKernelCandidates(release string) []string {
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}
	if release != "" {
		add(filepath.Join("/boot", "vmlinuz-"+release))
		add(filepath.Join("/boot", "bzImage-"+release))
		add(filepath.Join("/boot", "kernel-"+release))
		add(filepath.Join("/lib/modules", release, "vmlinuz"))
		add(filepath.Join("/lib/modules", release, "bzImage"))
	}
	add(filepath.Join("/boot", "vmlinuz"))
	add(filepath.Join("/boot", "bzImage"))
	globbed := make([]string, 0, 8)
	for _, pattern := range []string{"/boot/vmlinuz-*", "/boot/bzImage-*", "/boot/kernel-*", "/lib/modules/*/vmlinuz", "/lib/modules/*/bzImage"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		globbed = append(globbed, matches...)
	}
	sort.Strings(globbed)
	for _, candidate := range globbed {
		add(candidate)
	}
	return candidates
}

func validateHelloWorldKernelPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("kernel path is a directory")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}
