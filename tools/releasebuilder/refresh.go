package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const releaseArtifactsVendorHashPrefix = "vendorHash = \""

var nixHashMismatchRegexp = regexp.MustCompile(`got:\s+(sha256-[A-Za-z0-9+/=]+)`)

func runRefreshVendorHash(args []string) error {
	fs := flag.NewFlagSet("releasebuilder refresh-vendor-hash", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	packageFile := fs.String("package-file", filepath.Join("nix", "packages", "release-artifacts.nix"), "path to the Nix package file containing vendorHash")
	if err := fs.Parse(args); err != nil {
		return usageError{err: err}
	}
	if fs.NArg() != 0 {
		return usageError{err: fmt.Errorf("refresh-vendor-hash does not accept positional arguments")}
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determine working directory: %w", err)
	}
	relPackageFile := filepath.Clean(*packageFile)
	absPackageFile := filepath.Join(repoRoot, relPackageFile)
	nextHash, err := discoverReleaseArtifactsVendorHash(repoRoot)
	if err != nil {
		return err
	}
	if err := rewriteVendorHash(absPackageFile, nextHash); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "updated %s vendorHash to %s\n", relPackageFile, nextHash)
	return nil
}

func discoverReleaseArtifactsVendorHash(repoRoot string) (string, error) {
	cmd := exec.Command("nix", "build", ".#release-artifacts", "--no-link")
	cmd.Dir = repoRoot
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		return "", fmt.Errorf("nix build .#release-artifacts succeeded; vendorHash already up to date")
	}
	match := nixHashMismatchRegexp.FindStringSubmatch(stderr.String())
	if len(match) != 2 {
		return "", fmt.Errorf("discover release-artifacts vendorHash: nix output did not report replacement hash")
	}
	return strings.TrimSpace(match[1]), nil
}

func rewriteVendorHash(packageFile, nextHash string) error {
	content, err := os.ReadFile(packageFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", packageFile, err)
	}
	updated, changed, err := replaceVendorHashLine(string(content), nextHash)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("rewrite vendorHash in %s: replacement hash already present", packageFile)
	}
	if err := os.WriteFile(packageFile, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", packageFile, err)
	}
	return nil
}

func replaceVendorHashLine(content, nextHash string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, releaseArtifactsVendorHashPrefix) {
			continue
		}
		replacement := strings.Replace(line, trimmed, releaseArtifactsVendorHashPrefix+nextHash+"\";", 1)
		if replacement == line {
			return content, false, nil
		}
		lines[i] = replacement
		return strings.Join(lines, "\n"), true, nil
	}
	return "", false, fmt.Errorf("replace vendorHash: no vendorHash line found")
}
