package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const releaseBuilderID = "nix build .#release-artifacts"

type releaseManifest struct {
	PackageName   string            `json:"package_name"`
	Version       string            `json:"version"`
	Tag           string            `json:"tag"`
	Binaries      []string          `json:"binaries"`
	Archives      []archiveManifest `json:"archives"`
	ChecksumsFile string            `json:"checksums_file"`
	Builder       string            `json:"builder"`
}

type archiveManifest struct {
	Archive          string `json:"archive"`
	File             string `json:"file"`
	GoArch           string `json:"goarch"`
	GoOS             string `json:"goos"`
	PayloadDirectory string `json:"payload_directory"`
	SHA256           string `json:"sha256"`
}

func runManifest(args []string) error {
	fs := flag.NewFlagSet("releasebuilder manifest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	packageName := fs.String("package-name", "", "package name for the release")
	version := fs.String("version", "", "semantic version without the v prefix")
	tag := fs.String("tag", "", "release tag")
	binariesFile := fs.String("binaries-file", "", "newline-delimited binary name list")
	targetsFile := fs.String("targets-file", "", "newline-delimited target matrix")
	checksumsFile := fs.String("checksums-file", "", "sha256sum manifest path")
	output := fs.String("output", "", "output json path")
	if err := fs.Parse(args); err != nil {
		return usageError{err: err}
	}

	if *packageName == "" || *version == "" || *tag == "" || *binariesFile == "" || *targetsFile == "" || *checksumsFile == "" || *output == "" {
		return usageError{err: fmt.Errorf("manifest requires --package-name, --version, --tag, --binaries-file, --targets-file, --checksums-file, and --output")}
	}

	return writeManifest(*packageName, *version, *tag, *binariesFile, *targetsFile, *checksumsFile, *output)
}

func writeManifest(packageName, version, tag, binariesFile, targetsFile, checksumsFile, outputPath string) error {
	binaries, targets, checksums, err := loadManifestInputs(binariesFile, targetsFile, checksumsFile)
	if err != nil {
		return err
	}

	archives, err := buildArchiveManifestList(packageName, tag, targets, checksums)
	if err != nil {
		return err
	}

	manifest := releaseManifest{
		PackageName:   packageName,
		Version:       version,
		Tag:           tag,
		Binaries:      binaries,
		Archives:      archives,
		ChecksumsFile: "SHA256SUMS",
		Builder:       releaseBuilderID,
	}

	return writeManifestFile(outputPath, manifest)
}

func loadManifestInputs(binariesFile, targetsFile, checksumsFile string) ([]string, []string, map[string]string, error) {
	binaries, err := readNonEmptyLines(binariesFile)
	if err != nil {
		return nil, nil, nil, err
	}

	targets, err := readNonEmptyLines(targetsFile)
	if err != nil {
		return nil, nil, nil, err
	}

	checksums, err := readChecksums(checksumsFile)
	if err != nil {
		return nil, nil, nil, err
	}

	return binaries, targets, checksums, nil
}

func buildArchiveManifestList(packageName, tag string, targets []string, checksums map[string]string) ([]archiveManifest, error) {
	archives := make([]archiveManifest, 0, len(targets))
	for _, target := range targets {
		archive, err := buildArchiveManifest(packageName, tag, target, checksums)
		if err != nil {
			return nil, err
		}

		archives = append(archives, archive)
	}

	return archives, nil
}

func buildArchiveManifest(packageName, tag, target string, checksums map[string]string) (archiveManifest, error) {
	parts := strings.Fields(target)
	if len(parts) != 3 {
		return archiveManifest{}, fmt.Errorf("invalid target entry %q", target)
	}

	goos, goarch, archiveFormat := parts[0], parts[1], parts[2]
	payloadDirectory := fmt.Sprintf("%s_%s_%s_%s", packageName, tag, goos, goarch)
	fileName := fmt.Sprintf("%s.%s", payloadDirectory, archiveFormat)
	sha256, ok := checksums[fileName]
	if !ok {
		return archiveManifest{}, fmt.Errorf("missing checksum for %s", fileName)
	}

	return archiveManifest{
		Archive:          archiveFormat,
		File:             fileName,
		GoArch:           goarch,
		GoOS:             goos,
		PayloadDirectory: payloadDirectory,
		SHA256:           sha256,
	}, nil
}

func writeManifestFile(outputPath string, manifest releaseManifest) error {
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal release manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, append(encoded, '\n'), 0o644); err != nil {
		return fmt.Errorf("write release manifest: %w", err)
	}

	return nil
}

func readNonEmptyLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return lines, nil
}

func readChecksums(path string) (map[string]string, error) {
	lines, err := readNonEmptyLines(path)
	if err != nil {
		return nil, err
	}

	checksums := make(map[string]string, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid checksum entry %q", line)
		}
		if !isSHA256Hex(parts[0]) {
			return nil, fmt.Errorf("invalid sha256 checksum in entry %q", line)
		}
		if _, exists := checksums[parts[1]]; exists {
			return nil, fmt.Errorf("duplicate checksum entry for %q", parts[1])
		}

		checksums[parts[1]] = parts[0]
	}

	return checksums, nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}

	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}

	return true
}
