package brokerperf

import (
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func TestRunDeterministicBrokerHarnessProducesPhase3Metrics(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(path.Join(filepath.Dir(file), "..", ".."))
	out, err := Run(HarnessConfig{Trials: 2, RepositoryRoot: repoRoot})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if out.SchemaVersion != CheckSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, CheckSchemaVersion)
	}
	if len(out.Measurements) == 0 {
		t.Fatal("measurements empty")
	}
	assertMetricUnit(t, out.Measurements, "metric.broker.unary.session_list.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.watch.run.snapshot_follow.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.watch.run.snapshot_follow.payload_bytes", "bytes")
	assertMetricUnit(t, out.Measurements, "metric.broker.watch.turn_execution.snapshot_follow.event_count", "count")
	assertMetricUnit(t, out.Measurements, "metric.broker.mutation.session_execution_trigger.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.mutation.session_execution_continue.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.mutation.approval_resolve.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.mutation.backend_posture_change.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.attach.local_control_plane.p95_ms", "ms")
	assertMetricUnit(t, out.Measurements, "metric.broker.resume.local_control_plane.p95_ms", "ms")
}

func TestP95RecordsRejectsEmptySampleSet(t *testing.T) {
	t.Parallel()
	if _, err := p95Records(map[string][]float64{"metric.empty": nil}); err == nil {
		t.Fatal("p95Records error = nil, want empty sample failure")
	}
}

func assertMetricUnit(t *testing.T, measurements []perfcontracts.MeasurementRecord, metricID, unit string) {
	t.Helper()
	for _, m := range measurements {
		if m.MetricID == metricID {
			if m.Unit != unit {
				t.Fatalf("metric %s unit = %q, want %q", metricID, m.Unit, unit)
			}
			return
		}
	}
	t.Fatalf("metric %s missing", metricID)
}
