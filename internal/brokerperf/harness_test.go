package brokerperf

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/runecode-systems/runecode/internal/brokerapi"
	"github.com/runecode-systems/runecode/internal/perfcontracts"
)

func TestRunDeterministicBrokerHarnessProducesPhase3Metrics(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
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
	assertMetricValue(t, out.Measurements, "metric.broker.watch.run.snapshot_follow.event_count", 3)
	assertMetricValue(t, out.Measurements, "metric.broker.watch.turn_execution.snapshot_follow.event_count", 3)
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

func TestMeasureMutationHarnessUsesContractBoundaryScenarios(t *testing.T) {
	t.Parallel()

	repoRoot := repositoryRootForHarnessTests(t)
	ctx := context.Background()
	triggerDuration, triggerErr := measureSessionExecutionTriggerMutation(ctx, repoRoot)
	assertNonNegativeMutationDuration(t, "trigger", triggerDuration, triggerErr)
	continueDuration, continueErr := measureSessionExecutionContinueMutation(ctx, repoRoot)
	assertNonNegativeMutationDuration(t, "continue", continueDuration, continueErr)
	assertBackendPostureMutationFixture(t, repoRoot, ctx)
}

func assertNonNegativeMutationDuration(t *testing.T, label string, duration float64, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s mutation returned error: %v", label, err)
	}
	if duration < 0 {
		t.Fatalf("%s duration = %v, want non-negative", label, duration)
	}
}

func assertBackendPostureMutationFixture(t *testing.T, repoRoot string, ctx context.Context) {
	t.Helper()
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		t.Fatalf("newSeededService returned error: %v", err)
	}
	defer cleanup()
	if err := seedBackendPosturePolicyContext(service); err != nil {
		t.Fatalf("seedBackendPosturePolicyContext returned error: %v", err)
	}
	instanceID := serviceCurrentInstanceID(service)
	if instanceID == "" {
		t.Fatal("instanceID empty")
	}
	changeResp, errResp := service.HandleBackendPostureChange(ctx, brokerapi.BackendPostureChangeRequest{
		SchemaID:                     "runecode.protocol.v0.BackendPostureChangeRequest",
		SchemaVersion:                "0.1.0",
		RequestID:                    "req-harness-backend-posture-change",
		TargetInstanceID:             instanceID,
		TargetBackendKind:            "container",
		SelectionMode:                "explicit_selection",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
		Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
	}, brokerapi.RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleBackendPostureChange returned error: %+v", errResp)
	}
	if changeResp.Outcome.Outcome != "approval_required" {
		t.Fatalf("backend posture outcome = %q, want approval_required", changeResp.Outcome.Outcome)
	}
	assertBackendPostureResolveFixture(t, service)
}

func assertBackendPostureResolveFixture(t *testing.T, service *brokerapi.Service) {
	t.Helper()
	resolveReq, err := seedBackendPostureApprovalForResolveWithRunID(service, "run-backend")
	if err != nil {
		t.Fatalf("seedBackendPostureApprovalForResolveWithRunID returned error: %v", err)
	}
	if resolveReq.BoundScope.RunID != "run-backend" {
		t.Fatalf("resolve bound_scope.run_id = %q, want run-backend", resolveReq.BoundScope.RunID)
	}
}

func repositoryRootForHarnessTests(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
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

func assertMetricValue(t *testing.T, measurements []perfcontracts.MeasurementRecord, metricID string, value float64) {
	t.Helper()
	for _, m := range measurements {
		if m.MetricID == metricID {
			if m.Value != value {
				t.Fatalf("metric %s value = %v, want %v", metricID, m.Value, value)
			}
			return
		}
	}
	t.Fatalf("metric %s missing", metricID)
}
