package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteDeterministicZip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	writeZipFixture(t, sourceDir)

	firstZip := filepath.Join(root, "first.zip")
	secondZip := filepath.Join(root, "second.zip")
	writeZipArchive(t, sourceDir, firstZip)
	writeZipArchive(t, sourceDir, secondZip)

	firstBytes := mustReadFile(t, firstZip)
	secondBytes := mustReadFile(t, secondZip)

	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("zip output is not deterministic")
	}

	reader, err := zip.OpenReader(firstZip)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	assertZipEntries(t, reader.File)
}

func TestWriteDeterministicZipRejectsSymlinks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	writeZipFixture(t, sourceDir)

	symlinkPath := filepath.Join(sourceDir, "README-link")
	if err := os.Symlink(filepath.Join(sourceDir, "README.md"), symlinkPath); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	targetPath := filepath.Join(root, "release.zip")
	err := writeDeterministicZip(sourceDir, targetPath)
	if err == nil || !strings.Contains(err.Error(), "must not contain symlinks") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}

	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected failed zip build to remove %s, got %v", targetPath, statErr)
	}
}

func TestValidateZipRelativePathRejectsEscapes(t *testing.T) {
	t.Parallel()

	if _, err := validateZipRelativePath(filepath.Join("..", "escape.txt")); err == nil {
		t.Fatal("expected escape path validation to fail")
	}
	if _, err := validateZipRelativePath("..\\escape.txt"); err == nil {
		t.Fatal("expected backslash escape path validation to fail")
	}
	if _, err := validateZipRelativePath("C:\\escape.txt"); err == nil {
		t.Fatal("expected drive-letter path validation to fail")
	}
	if _, err := validateZipRelativePath("C:relative.txt"); err == nil {
		t.Fatal("expected drive-relative path validation to fail")
	}
	if _, err := validateZipRelativePath("a/../../escape.txt"); err == nil {
		t.Fatal("expected escaping internal traversal to fail")
	}
}

func TestValidateZipRelativePathAcceptsValidPaths(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{name: "nested path", in: "foo/bar.txt", want: "foo/bar.txt"},
		{name: "source binary", in: "source/bin/runecode", want: "source/bin/runecode"},
		{name: "normalized inner dotdot", in: "a/b/../c.txt", want: "a/c.txt"},
		{name: "unix colon filename", in: "man3/std::string.3", want: "man3/std::string.3"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := validateZipRelativePath(tc.in)
			if err != nil {
				t.Fatalf("validateZipRelativePath(%q) error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("validateZipRelativePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func writeZipFixture(t *testing.T, sourceDir string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	mustWriteFile(t, filepath.Join(sourceDir, "README.md"), []byte("readme\n"), 0o644)
	mustWriteFile(t, filepath.Join(sourceDir, "bin", "runecode-launcher"), []byte("launcher\n"), 0o755)
}

func writeZipArchive(t *testing.T, sourceDir, targetPath string) {
	t.Helper()

	if err := writeDeterministicZip(sourceDir, targetPath); err != nil {
		t.Fatalf("write zip %s: %v", targetPath, err)
	}
}

func assertZipEntries(t *testing.T, files []*zip.File) {
	t.Helper()

	if len(files) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(files))
	}

	if files[0].Name != "source/README.md" {
		t.Fatalf("expected first entry source/README.md, got %q", files[0].Name)
	}
	if files[1].Name != "source/bin/runecode-launcher" {
		t.Fatalf("expected second entry source/bin/runecode-launcher, got %q", files[1].Name)
	}

	for _, file := range files {
		if !file.Modified.Equal(deterministicArchiveTime) {
			t.Fatalf("entry %q has modified time %v", file.Name, file.Modified)
		}
	}

	if mode := files[1].Mode().Perm(); mode != expectedExecutableMode() {
		t.Fatalf("expected executable mode %o, got %o", expectedExecutableMode(), mode)
	}
	if mode := files[0].Mode().Perm(); mode != 0o644 {
		t.Fatalf("expected README mode 0644, got %o", mode)
	}
}

func expectedExecutableMode() os.FileMode {
	// Local Windows tests cannot preserve Unix execute bits via os.WriteFile.
	// Official release archives are still built through Nix on Linux, where the
	// canonical package payload records executable binaries as 0755.
	if runtime.GOOS == "windows" {
		return 0o644
	}

	return 0o755
}

func mustWriteFile(t *testing.T, path string, contents []byte, mode os.FileMode) {
	t.Helper()

	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return contents
}
