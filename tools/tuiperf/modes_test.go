//go:build linux

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

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
