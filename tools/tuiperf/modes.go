//go:build linux

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/tuiperf"
)

const latencyMarkerTimeout = 20 * time.Second

func runCPUMode(cfg config) error {
	_, cancel, harness, err := startTUIHarness(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer stopHarness(harness)
	targetPID, err := tuiperf.WaitForChildByComm("/proc", harness.tuiCmd.Process.Pid, "runecode-tui", 3*time.Second, 20*time.Millisecond)
	if err != nil {
		return err
	}
	result, err := tuiperf.SampleProcessCPU(targetPID, tuiperf.CPUSampleConfig{Warmup: cfg.warmup, Window: cfg.window, Windows: cfg.windows})
	if err != nil {
		return err
	}
	measurements := cpuMeasurementsForFixture(cfg.fixtureID, result)
	return writeEnvelope(cfg.outputPath, checkEnvelope{SchemaVersion: checkSchemaVersion, Metadata: map[string]any{"mode": "cpu", "fixture_id": cfg.fixtureID, "target_pid": result.TargetPID, "target_comm": result.TargetComm, "sampling": result}, Measurements: measurements})
}

func runLatencyMode(cfg config) error {
	_, cancel, harness, err := startTUIHarness(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer stopHarness(harness)
	attachDurations, keyDurations, err := collectLatencySamples(harness, cfg.trials)
	if err != nil {
		return err
	}
	attachP95, keyP95, err := latencyP95(attachDurations, keyDurations)
	if err != nil {
		return err
	}
	measurements := latencyMeasurementsForFixture(cfg.fixtureID, attachP95, keyP95)
	return writeEnvelope(cfg.outputPath, checkEnvelope{SchemaVersion: checkSchemaVersion, Metadata: map[string]any{"mode": "latency", "fixture_id": cfg.fixtureID, "trials": cfg.trials, "attach_samples_ms": attachDurations, "key_samples_ms": keyDurations}, Measurements: measurements})
}

func cpuMeasurementsForFixture(fixtureID string, result tuiperf.CPUSampleResult) []perfcontracts.MeasurementRecord {
	if fixtureID == "tui.empty.v1" {
		return []perfcontracts.MeasurementRecord{{MetricID: "metric.tui.idle_cpu.empty.avg_pct", Value: result.AverageCPUPercent, Unit: "percent"}, {MetricID: "metric.tui.idle_cpu.empty.max_pct", Value: result.MaxCPUPercent, Unit: "percent"}}
	}
	return []perfcontracts.MeasurementRecord{{MetricID: "metric.tui.idle_cpu.waiting.avg_pct", Value: result.AverageCPUPercent, Unit: "percent"}, {MetricID: "metric.tui.idle_cpu.waiting.max_pct", Value: result.MaxCPUPercent, Unit: "percent"}}
}

func collectLatencySamples(h runningHarness, trials int) ([]float64, []float64, error) {
	marker := "Runecode TUI α shell"
	attachDurations := make([]float64, 0, trials)
	keyDurations := make([]float64, 0, trials)
	events := make(chan tuiperf.MarkerEvent, 64)
	go tuiperf.WatchMarkers(h.tuiOut, []string{marker}, events)
	for i := 0; i < trials; i++ {
		attachMS, keyMS, err := collectLatencySample(events, h.tuiIn, marker)
		if err != nil {
			return nil, nil, err
		}
		attachDurations = append(attachDurations, attachMS)
		keyDurations = append(keyDurations, keyMS)
	}
	return attachDurations, keyDurations, nil
}

func collectLatencySample(events <-chan tuiperf.MarkerEvent, tuiIn io.Writer, marker string) (float64, float64, error) {
	start := time.Now()
	attachAt, err := waitForMarker(events, marker, latencyMarkerTimeout)
	if err != nil {
		return 0, 0, err
	}
	keyStart := time.Now()
	if _, err := io.WriteString(tuiIn, "\t"); err != nil {
		return 0, 0, err
	}
	keyAt, err := waitForMarker(events, marker, latencyMarkerTimeout)
	if err != nil {
		return 0, 0, err
	}
	return float64(attachAt.Sub(start).Milliseconds()), float64(keyAt.Sub(keyStart).Milliseconds()), nil
}

func latencyP95(attachDurations, keyDurations []float64) (float64, float64, error) {
	attachP95, err := tuiperf.P95Millis(attachDurations)
	if err != nil {
		return 0, 0, err
	}
	keyP95, err := tuiperf.P95Millis(keyDurations)
	if err != nil {
		return 0, 0, err
	}
	return attachP95, keyP95, nil
}

func latencyMeasurementsForFixture(fixtureID string, attachP95, keyP95 float64) []perfcontracts.MeasurementRecord {
	if fixtureID == "tui.empty.v1" {
		return []perfcontracts.MeasurementRecord{{MetricID: "metric.tui.attach.quiet.p95_ms", Value: attachP95, Unit: "ms"}, {MetricID: "metric.tui.key_response.quiet.p95_ms", Value: keyP95, Unit: "ms"}}
	}
	return []perfcontracts.MeasurementRecord{{MetricID: "metric.tui.attach.waiting.p95_ms", Value: attachP95, Unit: "ms"}, {MetricID: "metric.tui.key_response.waiting.p95_ms", Value: keyP95, Unit: "ms"}}
}

func runBenchParseMode(cfg config) error {
	if strings.TrimSpace(cfg.benchOutput) == "" {
		return fmt.Errorf("bench-parse mode requires --bench-output")
	}
	file, err := os.Open(cfg.benchOutput)
	if err != nil {
		return err
	}
	defer file.Close()
	measurements, err := tuiperf.ParseGoTestBenchOutput(file, []tuiperf.BenchmarkMetricMap{
		{Benchmark: "BenchmarkShellViewEmpty", Field: "ns/op", MetricID: "metric.tui.render.shell_view_empty.ns_op", Unit: "ns/op"},
		{Benchmark: "BenchmarkShellViewWaitingSession", Field: "ns/op", MetricID: "metric.tui.render.shell_view_waiting.ns_op", Unit: "ns/op"},
		{Benchmark: "BenchmarkShellWatchApply", Field: "ns/op", MetricID: "metric.tui.update.shell_watch_apply.ns_op", Unit: "ns/op"},
		{Benchmark: "BenchmarkBuildPaletteEntries", Field: "ns/op", MetricID: "metric.tui.update.build_palette_entries.ns_op", Unit: "ns/op"},
	})
	if err != nil {
		return err
	}
	return writeEnvelope(cfg.outputPath, checkEnvelope{SchemaVersion: checkSchemaVersion, Metadata: map[string]any{"mode": "bench-parse", "bench_output": cfg.benchOutput}, Measurements: measurements})
}
