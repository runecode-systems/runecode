package tuiperf

import (
	"strings"
	"testing"
)

func TestParseGoTestBenchOutput(t *testing.T) {
	input := strings.NewReader("BenchmarkShellViewEmpty-8  12345  11000 ns/op  1500 B/op  20 allocs/op\nBenchmarkShellWatchApply-8 23456 12000 ns/op 1700 B/op 21 allocs/op\n")
	measurements, err := ParseGoTestBenchOutput(input, []BenchmarkMetricMap{
		{Benchmark: "BenchmarkShellViewEmpty", Field: "ns/op", MetricID: "metric.tui.render.shell_view_empty.ns_op", Unit: "ns/op"},
		{Benchmark: "BenchmarkShellWatchApply", Field: "ns/op", MetricID: "metric.tui.update.shell_watch_apply.ns_op", Unit: "ns/op"},
	})
	if err != nil {
		t.Fatalf("ParseGoTestBenchOutput returned error: %v", err)
	}
	if len(measurements) != 2 {
		t.Fatalf("measurements len = %d, want 2", len(measurements))
	}
}
