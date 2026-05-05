package brokerapi

import (
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func TestRunPhase5PerformanceHarnessProducesExpectedMetrics(t *testing.T) {
	out, err := RunPhase5PerformanceHarness(testPhase5HarnessConfig())
	if err != nil {
		t.Fatalf("RunPhase5PerformanceHarness returned error: %v", err)
	}
	if out.SchemaVersion != phase5PerfCheckSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, phase5PerfCheckSchemaVersion)
	}
	required := requiredPhase5Metrics()
	for metricID, unit := range required {
		if !hasPhase5Metric(out.Measurements, metricID, unit) {
			t.Fatalf("missing metric %s (%s)", metricID, unit)
		}
	}
}

func testPhase5HarnessConfig() Phase5PerformanceHarnessConfig {
	return Phase5PerformanceHarnessConfig{
		Trials: 2,
		CommandRunner: func(_ string, _ time.Duration, command ...string) (float64, error) {
			return phase5CommandLatency(command), nil
		},
	}
}

func phase5CommandLatency(command []string) float64 {
	if len(command) >= 3 && command[0] == "go" && command[1] == "test" {
		return 120
	}
	if len(command) >= 3 && command[0] == "node" && command[1] == "--test" {
		return 90
	}
	return 10
}

func requiredPhase5Metrics() map[string]string {
	return map[string]string{
		"metric.gateway.model_invoke.overhead.p95_ms":           "ms",
		"metric.secrets.lease_issue.p95_ms":                     "ms",
		"metric.secrets.ingress.prepare_submit.p95_ms":          "ms",
		"metric.deps.cache_miss.small.wall_ms":                  "ms",
		"metric.deps.cache_hit.small.wall_ms":                   "ms",
		"metric.deps.cache_coalesced.upstream_fetch_count":      "count",
		"metric.deps.cache_coalesced.cas_write_count":           "count",
		"metric.deps.materialization.workspace_handoff.wall_ms": "ms",
		"metric.deps.stream_to_cas.max_read_buffer_bytes":       "bytes",
		"metric.deps.stream_to_cas.read_calls":                  "count",
		"metric.deps.cache_fill.peak_alloc_mb":                  "mb",
		"metric.audit.verify_current_segment.wall_ms":           "ms",
		"metric.audit.finalize_verify.wall_ms":                  "ms",
		"metric.protocol.schema_validation.wall_ms":             "ms",
		"metric.protocol.fixture_parity.wall_ms":                "ms",
		"metric.anchor.prepare.latency.p95_ms":                  "ms",
		"metric.anchor.execute.completed.p95_ms":                "ms",
		"metric.anchor.execute.deferred.handoff.p95_ms":         "ms",
		"metric.anchor.deferred.visibility.p95_ms":              "ms",
		"metric.anchor.receipt_admission.unchanged_seal.p95_ms": "ms",
		"metric.anchor.network_io_under_ledger_lock.count":      "count",
		"metric.anchor.verifier_bypass.count":                   "count",
	}
}

func hasPhase5Metric(measurements []perfcontracts.MeasurementRecord, metricID, unit string) bool {
	for _, m := range measurements {
		if m.MetricID == metricID && m.Unit == unit {
			return true
		}
	}
	return false
}
