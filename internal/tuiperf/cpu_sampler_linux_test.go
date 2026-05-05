//go:build linux

package tuiperf

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestSampleProcessCPUDeterministicProcFixture(t *testing.T) {
	root := t.TempDir()
	pid := 4321
	procDir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatalf("MkdirAll procDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu  1 2 3\ncpu0 1 1 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile stat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte("4321 (runecode-tui) S 1 2 3 4 5 6 7 8 9 10 11 100 50 0 0 20 0 1 0 999 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile stat first: %v", err)
	}
	resultCh := make(chan CPUSampleResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := SampleProcessCPU(pid, CPUSampleConfig{ProcRoot: root, Warmup: 0, Window: 5, Windows: 1, TicksPerSecond: 100, CPUCount: 1})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte("4321 (runecode-tui) S 1 2 3 4 5 6 7 8 9 10 11 200 50 0 0 20 0 1 0 999 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile stat second: %v", err)
	}
	select {
	case err := <-errCh:
		t.Fatalf("SampleProcessCPU returned error: %v", err)
	case result := <-resultCh:
		if result.AverageCPUPercent < 0 {
			t.Fatalf("average cpu = %.2f, want non-negative", result.AverageCPUPercent)
		}
	}
}
