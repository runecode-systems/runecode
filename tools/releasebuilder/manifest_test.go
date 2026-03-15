package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outputFile := filepath.Join(root, "release-manifest.json")

	binariesFile, targetsFile, checksumsFile := writeManifestFixtures(t, root)
	if err := writeManifest("runecode", "1.2.3", "v1.2.3", binariesFile, targetsFile, checksumsFile, outputFile); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	assertManifest(t, outputFile, wantReleaseManifest())
}

func TestReadChecksumsRejectsInvalidHashes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	checksumsFile := filepath.Join(root, "SHA256SUMS")
	mustWriteFile(t, checksumsFile, []byte("not-a-sha256  runecode_v1.2.3_linux_amd64.tar.gz\n"), 0o644)

	if _, err := readChecksums(checksumsFile); err == nil {
		t.Fatal("expected invalid sha256 checksum to be rejected")
	}
}

func TestReadChecksumsRejectsDuplicateEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	checksumsFile := filepath.Join(root, "SHA256SUMS")
	mustWriteFile(t, checksumsFile, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  runecode_v1.2.3_linux_amd64.tar.gz\nbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  runecode_v1.2.3_linux_amd64.tar.gz\n"), 0o644)

	if _, err := readChecksums(checksumsFile); err == nil {
		t.Fatal("expected duplicate checksum entry to be rejected")
	}
}

func assertManifest(t *testing.T, outputFile string, want releaseManifest) {
	t.Helper()

	var got releaseManifest
	if err := json.Unmarshal(mustReadFile(t, outputFile), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func wantReleaseManifest() releaseManifest {
	return releaseManifest{
		PackageName:   "runecode",
		Version:       "1.2.3",
		Tag:           "v1.2.3",
		Binaries:      []string{"runecode-launcher", "runecode-tui"},
		ChecksumsFile: "SHA256SUMS",
		Builder:       releaseBuilderID,
		Archives: []archiveManifest{
			{
				Archive:          "tar.gz",
				File:             "runecode_v1.2.3_linux_amd64.tar.gz",
				GoArch:           "amd64",
				GoOS:             "linux",
				PayloadDirectory: "runecode_v1.2.3_linux_amd64",
				SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			{
				Archive:          "zip",
				File:             "runecode_v1.2.3_windows_arm64.zip",
				GoArch:           "arm64",
				GoOS:             "windows",
				PayloadDirectory: "runecode_v1.2.3_windows_arm64",
				SHA256:           "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}
}

func writeManifestFixtures(t *testing.T, root string) (string, string, string) {
	t.Helper()

	binariesFile := filepath.Join(root, "binaries.txt")
	targetsFile := filepath.Join(root, "targets.txt")
	checksumsFile := filepath.Join(root, "SHA256SUMS")

	mustWriteFile(t, binariesFile, []byte("runecode-launcher\nrunecode-tui\n"), 0o644)
	mustWriteFile(t, targetsFile, []byte("linux amd64 tar.gz\nwindows arm64 zip\n"), 0o644)
	mustWriteFile(t, checksumsFile, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  runecode_v1.2.3_linux_amd64.tar.gz\nbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  runecode_v1.2.3_windows_arm64.zip\n"), 0o644)

	return binariesFile, targetsFile, checksumsFile
}
