// Command gofmtcheck enforces deterministic Go formatting checks for the repo.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const maxFilesPerInvocation = 200

func main() {
	write := flag.Bool("write", false, "write gofmt changes")
	flag.Parse()

	files, err := collectGoFiles(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to collect go files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		return
	}

	if *write {
		if err := runGofmt("-w", files); err != nil {
			fmt.Fprintf(os.Stderr, "gofmt failed: %v\n", err)
			os.Exit(1)
		}

		return
	}

	output, err := runGofmtList(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt check failed: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(output) != "" {
		fmt.Fprintln(os.Stderr, "Go files need formatting:")
		fmt.Fprint(os.Stderr, output)
		os.Exit(1)
	}
}

func collectGoFiles(root string) ([]string, error) {
	files := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			switch d.Name() {
			case ".git", ".direnv", "node_modules":
				return filepath.SkipDir
			}

			return nil
		}

		if filepath.Ext(path) == ".go" {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func runGofmt(mode string, files []string) error {
	for _, batch := range splitIntoBatches(files, maxFilesPerInvocation) {
		args := append([]string{mode}, batch...)
		cmd := exec.Command("gofmt", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func runGofmtList(files []string) (string, error) {
	var listed bytes.Buffer

	for _, batch := range splitIntoBatches(files, maxFilesPerInvocation) {
		args := append([]string{"-l"}, batch...)
		cmd := exec.Command("gofmt", args...)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			if stderr.Len() > 0 {
				return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
			}

			return "", err
		}

		listed.Write(stdout.Bytes())
	}

	return listed.String(), nil
}

func splitIntoBatches(files []string, maxBatch int) [][]string {
	if len(files) == 0 {
		return nil
	}

	if maxBatch <= 0 || len(files) <= maxBatch {
		return [][]string{files}
	}

	batches := make([][]string, 0, (len(files)+maxBatch-1)/maxBatch)
	for start := 0; start < len(files); start += maxBatch {
		end := start + maxBatch
		if end > len(files) {
			end = len(files)
		}

		batches = append(batches, files[start:end])
	}

	return batches
}
