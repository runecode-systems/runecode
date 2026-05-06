//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/tuiperf"
)

func TestRunModeUnsupportedModeReturnsUsageError(t *testing.T) {
	t.Parallel()

	err := runMode(config{mode: "unknown"})
	if err == nil {
		t.Fatal("runMode error = nil, want usage error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("runMode error = %T, want usageError", err)
	}
}

func TestRunBenchParseModeRequiresBenchOutputAsUsageError(t *testing.T) {
	t.Parallel()

	err := runBenchParseMode(config{})
	if err == nil {
		t.Fatal("runBenchParseMode error = nil, want usage error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("runBenchParseMode error = %T, want usageError", err)
	}
}

func TestCollectLatencySamplesFromFreshSpawnStartsAndStopsPerTrial(t *testing.T) {
	t.Parallel()

	started, stopped, attach, key, err := collectTrialLifecycleSamples(t)
	if err != nil {
		t.Fatalf("collectLatencySamplesFromFreshSpawn error = %v", err)
	}
	assertTrialLifecycle(t, started, stopped, attach, key)
}

func collectTrialLifecycleSamples(t *testing.T) ([]string, []string, []float64, []float64, error) {
	t.Helper()
	var started []string
	var stopped []string
	sampleCalls := 0
	attach, key, err := collectLatencySamplesFromFreshSpawn(
		3,
		func() (runningHarness, error) {
			id := fmt.Sprintf("trial-%d", len(started)+1)
			started = append(started, id)
			return runningHarness{tuiCmd: &exec.Cmd{Path: id}}, nil
		},
		func(h runningHarness) { stopped = append(stopped, h.tuiCmd.Path) },
		func(_ runningHarness, marker string, _ time.Time) (float64, float64, error) {
			sampleCalls++
			if marker != "Runecode TUI α shell" {
				t.Fatalf("marker = %q, want %q", marker, "Runecode TUI α shell")
			}
			return float64(sampleCalls), float64(sampleCalls + 10), nil
		},
	)
	return started, stopped, attach, key, err
}

func assertTrialLifecycle(t *testing.T, started, stopped []string, attach, key []float64) {
	t.Helper()
	assertLifecycleCounts(t, started, stopped, attach, key)
	assertLifecycleOrder(t, started, stopped)
	assertLifecycleSamples(t, attach, key)
}

func assertLifecycleCounts(t *testing.T, started, stopped []string, attach, key []float64) {
	t.Helper()
	if got, want := len(started), 3; got != want {
		t.Fatalf("start calls = %d, want %d", got, want)
	}
	if got, want := len(stopped), 3; got != want {
		t.Fatalf("stop calls = %d, want %d", got, want)
	}
	if got, want := len(attach), 3; got != want {
		t.Fatalf("attach sample count = %d, want %d", got, want)
	}
	if got, want := len(key), 3; got != want {
		t.Fatalf("key sample count = %d, want %d", got, want)
	}
}

func assertLifecycleOrder(t *testing.T, started, stopped []string) {
	t.Helper()
	for i := range started {
		if started[i] != stopped[i] {
			t.Fatalf("stopped[%d] = %q, want %q", i, stopped[i], started[i])
		}
	}
}

func assertLifecycleSamples(t *testing.T, attach, key []float64) {
	t.Helper()
	if attach[0] != 1 || attach[1] != 2 || attach[2] != 3 {
		t.Fatalf("attach samples = %v, want [1 2 3]", attach)
	}
	if key[0] != 11 || key[1] != 12 || key[2] != 13 {
		t.Fatalf("key samples = %v, want [11 12 13]", key)
	}
}

func TestCollectLatencySamplesFromFreshSpawnStopsHarnessOnSampleError(t *testing.T) {
	t.Parallel()

	var stopped []string
	starts := 0

	_, _, err := collectLatencySamplesFromFreshSpawn(
		3,
		func() (runningHarness, error) {
			starts++
			return runningHarness{tuiCmd: &exec.Cmd{Path: fmt.Sprintf("trial-%d", starts)}}, nil
		},
		func(h runningHarness) {
			stopped = append(stopped, h.tuiCmd.Path)
		},
		func(h runningHarness, _ string, _ time.Time) (float64, float64, error) {
			if h.tuiCmd.Path == "trial-2" {
				return 0, 0, errors.New("sample failed")
			}
			return 1, 1, nil
		},
	)
	if err == nil {
		t.Fatal("collectLatencySamplesFromFreshSpawn error = nil, want error")
	}
	if got, want := starts, 2; got != want {
		t.Fatalf("start calls = %d, want %d", got, want)
	}
	if got, want := len(stopped), 2; got != want {
		t.Fatalf("stop calls = %d, want %d", got, want)
	}
	if stopped[1] != "trial-2" {
		t.Fatalf("stopped harness on error = %q, want trial-2", stopped[1])
	}
}

func TestCollectLatencySamplesFromFreshSpawnMeasuresFromPreSpawnStart(t *testing.T) {
	t.Parallel()

	spawnDelay := 15 * time.Millisecond
	_, _, err := collectLatencySamplesFromFreshSpawn(
		1,
		func() (runningHarness, error) {
			time.Sleep(spawnDelay)
			return runningHarness{tuiCmd: &exec.Cmd{Path: "trial-1"}}, nil
		},
		func(runningHarness) {},
		func(_ runningHarness, _ string, start time.Time) (float64, float64, error) {
			if elapsed := time.Since(start); elapsed < spawnDelay {
				t.Fatalf("elapsed since start = %s, want >= %s", elapsed, spawnDelay)
			}
			return 1, 1, nil
		},
	)
	if err != nil {
		t.Fatalf("collectLatencySamplesFromFreshSpawn error = %v", err)
	}
}

func TestWaitForMarkerAfterSkipsStaleEvents(t *testing.T) {
	t.Parallel()
	events := make(chan tuiperf.MarkerEvent, 3)
	start := time.Now()
	events <- tuiperf.MarkerEvent{Marker: "Runecode TUI α shell", At: start.Add(-time.Millisecond)}
	events <- tuiperf.MarkerEvent{Marker: "Runecode TUI α shell", At: start.Add(time.Millisecond)}
	got, err := waitForMarkerAfter(events, "Runecode TUI α shell", start, time.Second)
	if err != nil {
		t.Fatalf("waitForMarkerAfter error = %v", err)
	}
	if got.Before(start) {
		t.Fatalf("got = %s, want >= %s", got, start)
	}
}

func TestRunBenchParseModeStoresBenchOutputBaseName(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	benchPath := filepath.Join(tmp, "bench.txt")
	outputPath := filepath.Join(tmp, "out.json")
	content := strings.Join([]string{
		"BenchmarkShellViewEmpty-8 1000 10 ns/op",
		"BenchmarkShellViewWaitingSession-8 1000 11 ns/op",
		"BenchmarkShellWatchApply-8 1000 12 ns/op",
		"BenchmarkBuildPaletteEntries-8 1000 13 ns/op",
	}, "\n") + "\n"
	if err := os.WriteFile(benchPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile benchPath: %v", err)
	}
	if err := runBenchParseMode(config{benchOutput: benchPath, outputPath: outputPath}); err != nil {
		t.Fatalf("runBenchParseMode error = %v", err)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile output: %v", err)
	}
	if strings.Contains(string(raw), benchPath) {
		t.Fatalf("output leaked full bench path: %s", benchPath)
	}
	if !strings.Contains(string(raw), filepath.Base(benchPath)) {
		t.Fatalf("output missing bench basename: %s", filepath.Base(benchPath))
	}
}
