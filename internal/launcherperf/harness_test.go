package launcherperf

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func TestRunProducesPhase4LauncherAndAttestationMetrics(t *testing.T) {
	out, err := Run(HarnessConfig{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if out.SchemaVersion != CheckSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, CheckSchemaVersion)
	}
	required := map[string]string{
		"metric.launcher.microvm.cold_start.wall_ms":   "ms",
		"metric.launcher.microvm.warm_start.wall_ms":   "ms",
		"metric.launcher.container.cold_start.wall_ms": "ms",
		"metric.launcher.container.warm_start.wall_ms": "ms",
		"metric.attestation.cold.verify.wall_ms":       "ms",
		"metric.attestation.warm.verify.wall_ms":       "ms",
	}
	for metricID, unit := range required {
		if !hasMetric(out.Measurements, metricID, unit) {
			t.Fatalf("missing metric %s (%s)", metricID, unit)
		}
	}
}

func hasMetric(measurements []perfcontracts.MeasurementRecord, metricID, unit string) bool {
	for _, m := range measurements {
		if m.MetricID == metricID && m.Unit == unit {
			return true
		}
	}
	return false
}
