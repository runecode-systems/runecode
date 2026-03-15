package main

import (
	"archive/zip"
	"compress/flate"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var deterministicArchiveTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

type zipInput struct {
	absolutePath string
	relativePath string
	mode         fs.FileMode
}

func runZip(args []string) error {
	fs := flag.NewFlagSet("releasebuilder zip", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	source := fs.String("source", "", "source directory to archive")
	target := fs.String("target", "", "zip file path to create")
	if err := fs.Parse(args); err != nil {
		return usageError{err: err}
	}

	if *source == "" || *target == "" {
		return usageError{err: fmt.Errorf("zip requires --source and --target")}
	}

	return writeDeterministicZip(*source, *target)
}

func writeDeterministicZip(sourceDir, targetPath string) error {
	inputs, err := collectZipInputs(sourceDir)
	if err != nil {
		return err
	}

	targetFile, archive, err := createZipTarget(targetPath)
	if err != nil {
		return err
	}
	cleanupFailedArchive := true
	defer func() {
		if cleanupFailedArchive {
			_ = os.Remove(targetPath)
		}
	}()

	if err := writeZipEntries(archive, inputs); err != nil {
		_ = archive.Close()
		_ = targetFile.Close()
		return err
	}

	if err := archive.Close(); err != nil {
		_ = targetFile.Close()
		return fmt.Errorf("close zip archive: %w", err)
	}

	if err := targetFile.Close(); err != nil {
		return fmt.Errorf("close zip target: %w", err)
	}

	cleanupFailedArchive = false

	return nil
}

func createZipTarget(targetPath string) (*os.File, *zip.Writer, error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create target directory: %w", err)
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create zip target: %w", err)
	}

	archive := zip.NewWriter(targetFile)
	archive.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})

	return targetFile, archive, nil
}

func writeZipEntries(archive *zip.Writer, inputs []zipInput) error {
	for _, input := range inputs {
		if err := writeZipEntry(archive, input); err != nil {
			return err
		}
	}

	return nil
}

func writeZipEntry(archive *zip.Writer, input zipInput) error {
	header := &zip.FileHeader{
		Name:     input.relativePath,
		Method:   zip.Deflate,
		Modified: deterministicArchiveTime,
	}
	header.SetMode(normalizeZipFileMode(input.mode))

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip entry %s: %w", input.relativePath, err)
	}

	sourceFile, err := os.Open(input.absolutePath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", input.absolutePath, err)
	}
	defer sourceFile.Close()

	if _, err := io.Copy(writer, sourceFile); err != nil {
		return fmt.Errorf("write zip entry %s: %w", input.relativePath, err)
	}

	return nil
}

func collectZipInputs(sourceDir string) ([]zipInput, error) {
	absSourceDir, err := validateSourceDir(sourceDir)
	if err != nil {
		return nil, err
	}

	inputs, err := walkZipInputs(absSourceDir)
	if err != nil {
		return nil, err
	}

	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].relativePath < inputs[j].relativePath
	})

	return inputs, nil
}

func validateSourceDir(sourceDir string) (string, error) {
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", fmt.Errorf("resolve source directory: %w", err)
	}

	info, err := os.Stat(absSourceDir)
	if err != nil {
		return "", fmt.Errorf("stat source directory: %w", err)
	}
	if !info.IsDir() {
		return "", usageError{err: fmt.Errorf("source must be a directory")}
	}

	return absSourceDir, nil
}

func walkZipInputs(sourceDir string) ([]zipInput, error) {
	inputs := make([]zipInput, 0)
	err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := validateArchiveEntryType(path, d); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		input, err := newZipInput(sourceDir, path, d)
		if err != nil {
			return err
		}

		inputs = append(inputs, input)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect zip inputs: %w", err)
	}

	return inputs, nil
}

func validateArchiveEntryType(path string, d fs.DirEntry) error {
	if d.Type()&fs.ModeSymlink != 0 {
		return fmt.Errorf("release archive inputs must not contain symlinks: %s", path)
	}
	if d.IsDir() {
		return nil
	}
	if !d.Type().IsRegular() {
		return fmt.Errorf("release archive inputs must be regular files: %s", path)
	}

	return nil
}

func newZipInput(sourceDir, path string, d fs.DirEntry) (zipInput, error) {
	archiveRoot := filepath.Dir(sourceDir)
	rel, err := filepath.Rel(archiveRoot, path)
	if err != nil {
		return zipInput{}, err
	}

	cleanRel, err := validateZipRelativePath(rel)
	if err != nil {
		return zipInput{}, err
	}

	info, err := d.Info()
	if err != nil {
		return zipInput{}, err
	}

	return zipInput{
		absolutePath: path,
		relativePath: cleanRel,
		mode:         info.Mode(),
	}, nil
}

func validateZipRelativePath(rel string) (string, error) {
	// Normalize backslashes to forward slashes before validation so archive entry
	// checks use ZIP path semantics on every host OS. This intentionally treats
	// literal backslashes in source filenames as path separators for portability.
	slashPath := strings.ReplaceAll(rel, "\\", "/")
	cleanRel := path.Clean(slashPath)
	if cleanRel == "." {
		return "", fmt.Errorf("zip entry path must not be empty")
	}
	if path.IsAbs(cleanRel) {
		return "", fmt.Errorf("zip entry path must be relative: %s", rel)
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, "../") {
		return "", fmt.Errorf("zip entry path escapes source directory: %s", rel)
	}
	if hasWindowsDrivePrefix(cleanRel) {
		return "", fmt.Errorf("zip entry path must not contain Windows drive-letter forms: %s", rel)
	}

	return cleanRel, nil
}

func hasWindowsDrivePrefix(pathValue string) bool {
	if len(pathValue) < 2 || pathValue[1] != ':' {
		return false
	}

	first := pathValue[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}

func normalizeZipFileMode(mode fs.FileMode) fs.FileMode {
	if mode&0o111 != 0 {
		return 0o755
	}

	return 0o644
}
