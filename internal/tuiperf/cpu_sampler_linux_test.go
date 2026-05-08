//go:build linux

package tuiperf

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestSampleProcessCPUDeterministicProcFixture(t *testing.T) {
	root := t.TempDir()
	pid := 4321
	procDir := filepath.Join(root, strconv.Itoa(pid))
	writeDeterministicCPUFixture(t, root, procDir)
	resultCh := make(chan CPUSampleResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := SampleProcessCPU(pid, CPUSampleConfig{ProcRoot: root, Warmup: 0, Window: 5 * time.Millisecond, Windows: 1, TicksPerSecond: 100, CPUCount: 1})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()
	if err := writeFileAtomically(filepath.Join(procDir, "stat"), []byte("4321 (runecode-tui) S 1 2 3 4 5 6 7 8 9 10 11 200 50 0 0 20 0 1 0 999 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("writeFileAtomically stat second: %v", err)
	}
	select {
	case err := <-errCh:
		t.Fatalf("SampleProcessCPU returned error: %v", err)
	case result := <-resultCh:
		if result.WindowMillis != 5 {
			t.Fatalf("window millis = %d, want 5", result.WindowMillis)
		}
		if result.AverageCPUPercent < 0 {
			t.Fatalf("average cpu = %.2f, want non-negative", result.AverageCPUPercent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SampleProcessCPU timed out waiting for deterministic fixture result")
	}
}

func writeDeterministicCPUFixture(t *testing.T, root, procDir string) {
	t.Helper()
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatalf("MkdirAll procDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu  1 2 3\ncpu0 1 1 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile stat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte("4321 (runecode-tui) S 1 2 3 4 5 6 7 8 9 10 11 100 50 0 0 20 0 1 0 999 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile stat first: %v", err)
	}
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}
