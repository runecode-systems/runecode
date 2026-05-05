package brokerperf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func newSeededService(repoRoot string) (*brokerapi.Service, func(), error) {
	root, err := os.MkdirTemp("", "runecode-brokerperf-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	service, err := brokerapi.NewServiceWithConfig(root, filepath.Join(root, "audit-ledger"), brokerapi.APIConfig{RepositoryRoot: repoRoot})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if err := seedServiceData(service); err != nil {
		cleanup()
		return nil, nil, err
	}
	return service, cleanup, nil
}

func seedServiceData(service *brokerapi.Service) error {
	if err := service.SetRunStatus("run-broker-1", "active"); err != nil {
		return err
	}
	if err := service.RecordRuntimeFacts("run-broker-1", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-broker-1", SessionID: "sess-broker-1"}}); err != nil {
		return err
	}
	if err := seedBlockedTurn(service); err != nil {
		return err
	}
	return service.RecordPolicyDecision("run-broker-1", "", seedPolicyDecision())
}

func seedBlockedTurn(service *brokerapi.Service) error {
	triggerResp, errResp := service.HandleSessionExecutionTrigger(context.Background(), brokerapi.SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "seed-trigger", SessionID: "sess-broker-1", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "change_draft"}, UserMessageContentText: "seed"}, brokerapi.RequestContext{})
	if errResp != nil {
		return fmt.Errorf("seed session_execution_trigger: %s", errResp.Error.Code)
	}
	_, _ = service.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-broker-1", TurnID: triggerResp.TurnID, ExecutionState: "blocked", WaitKind: "project_blocked", WaitState: "waiting_project_blocked", BlockedReasonCode: "project_substrate_posture_blocked", OccurredAt: time.Now().UTC()})
	return nil
}

func seedPolicyDecision() policyengine.PolicyDecision {
	return policyengine.PolicyDecision{
		SchemaID:                 "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:            "0.3.0",
		DecisionOutcome:          policyengine.DecisionRequireHumanApproval,
		PolicyReasonCode:         "approval_required",
		ManifestHash:             "sha256:" + strings.Repeat("1", 64),
		ActionRequestHash:        "sha256:" + strings.Repeat("2", 64),
		PolicyInputHashes:        []string{"sha256:" + strings.Repeat("3", 64)},
		DetailsSchemaID:          "runecode.protocol.details.policy.evaluation.v0",
		Details:                  map[string]any{"precedence": "approval_profile_moderate"},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "excerpt_promotion",
			"approval_assurance_level": "session_authenticated",
			"presence_mode":            "os_confirmation",
			"scope": map[string]any{
				"schema_id":      "runecode.protocol.v0.ApprovalBoundScope",
				"schema_version": "0.1.0",
				"workspace_id":   "workspace-local",
				"run_id":         "run-broker-1",
				"stage_id":       "artifact_flow",
				"step_id":        "step-1",
				"action_kind":    "promotion",
			},
			"changes_if_approved":  "Promote reviewed file excerpts for downstream use.",
			"approval_ttl_seconds": 1800,
		},
	}
}

func runWatchLatencyAndPayload(call func() (any, error)) (float64, float64, float64, error) {
	started := time.Now()
	result, err := call()
	if err != nil {
		return 0, 0, 0, err
	}
	bytes, count, err := payloadStats(result)
	if err != nil {
		return 0, 0, 0, err
	}
	return float64(time.Since(started).Milliseconds()), float64(bytes), float64(count), nil
}

func collectLatencyMeasurements(trials int, specs []latencySpec) ([]perfcontracts.MeasurementRecord, error) {
	latency := map[string][]float64{}
	for i := 0; i < trials; i++ {
		for _, spec := range specs {
			duration, err := timedCall(spec.call)
			if err != nil {
				return nil, err
			}
			latency[spec.metricID] = append(latency[spec.metricID], duration)
		}
	}
	return p95Records(latency)
}

func collectWatchMeasurements(trials int, specs []watchSpec) ([]perfcontracts.MeasurementRecord, error) {
	latency := map[string][]float64{}
	watchPayload := map[string]float64{}
	watchCounts := map[string]float64{}
	for i := 0; i < trials; i++ {
		for _, spec := range specs {
			duration, bytes, count, err := runWatchLatencyAndPayload(spec.call)
			if err != nil {
				return nil, err
			}
			latency[spec.latencyMetricID] = append(latency[spec.latencyMetricID], duration)
			watchPayload[spec.payloadMetricID] = maxFloat64(watchPayload[spec.payloadMetricID], bytes)
			watchCounts[spec.countMetricID] = maxFloat64(watchCounts[spec.countMetricID], count)
		}
	}
	measurements, err := p95Records(latency)
	if err != nil {
		return nil, err
	}
	return appendWatchStats(measurements, watchPayload, watchCounts), nil
}

func appendWatchStats(measurements []perfcontracts.MeasurementRecord, watchPayload, watchCounts map[string]float64) []perfcontracts.MeasurementRecord {
	for metricID, value := range watchPayload {
		measurements = append(measurements, perfcontracts.MeasurementRecord{MetricID: metricID, Value: value, Unit: "bytes"})
	}
	for metricID, value := range watchCounts {
		measurements = append(measurements, perfcontracts.MeasurementRecord{MetricID: metricID, Value: value, Unit: "count"})
	}
	return measurements
}

func streamRunWatch(ctx context.Context, service *brokerapi.Service) (any, error) {
	ack, errResp := service.HandleRunWatchRequest(ctx, brokerapi.RunWatchRequest{SchemaID: "runecode.protocol.v0.RunWatchRequest", SchemaVersion: "0.1.0", RequestID: "perf-run-watch", Follow: true, IncludeSnapshot: true}, brokerapi.RequestContext{})
	if errResp != nil {
		return nil, fmt.Errorf("run_watch ack: %s", errResp.Error.Code)
	}
	return service.StreamRunWatchEvents(ack)
}

func streamApprovalWatch(ctx context.Context, service *brokerapi.Service) (any, error) {
	ack, errResp := service.HandleApprovalWatchRequest(ctx, brokerapi.ApprovalWatchRequest{SchemaID: "runecode.protocol.v0.ApprovalWatchRequest", SchemaVersion: "0.1.0", RequestID: "perf-approval-watch", Follow: true, IncludeSnapshot: true}, brokerapi.RequestContext{})
	if errResp != nil {
		return nil, fmt.Errorf("approval_watch ack: %s", errResp.Error.Code)
	}
	return service.StreamApprovalWatchEvents(ack)
}

func streamSessionWatch(ctx context.Context, service *brokerapi.Service) (any, error) {
	ack, errResp := service.HandleSessionWatchRequest(ctx, brokerapi.SessionWatchRequest{SchemaID: "runecode.protocol.v0.SessionWatchRequest", SchemaVersion: "0.1.0", RequestID: "perf-session-watch", Follow: true, IncludeSnapshot: true}, brokerapi.RequestContext{})
	if errResp != nil {
		return nil, fmt.Errorf("session_watch ack: %s", errResp.Error.Code)
	}
	return service.StreamSessionWatchEvents(ack)
}

func streamTurnExecutionWatch(ctx context.Context, service *brokerapi.Service) (any, error) {
	ack, errResp := service.HandleSessionTurnExecutionWatchRequest(ctx, brokerapi.SessionTurnExecutionWatchRequest{SchemaID: "runecode.protocol.v0.SessionTurnExecutionWatchRequest", SchemaVersion: "0.1.0", RequestID: "perf-turn-watch", Follow: true, IncludeSnapshot: true}, brokerapi.RequestContext{})
	if errResp != nil {
		return nil, fmt.Errorf("session_turn_execution_watch ack: %s", errResp.Error.Code)
	}
	return service.StreamSessionTurnExecutionWatchEvents(ack)
}

func timedCall(call func() error) (float64, error) {
	started := time.Now()
	if err := call(); err != nil {
		return 0, err
	}
	return float64(time.Since(started).Milliseconds()), nil
}

func payloadStats(value any) (int, int, error) {
	blob, err := json.Marshal(value)
	if err != nil {
		return 0, 0, err
	}
	count := 1
	if values, ok := value.([]brokerapi.RunWatchEvent); ok {
		count = len(values)
	} else if values, ok := value.([]brokerapi.ApprovalWatchEvent); ok {
		count = len(values)
	} else if values, ok := value.([]brokerapi.SessionWatchEvent); ok {
		count = len(values)
	} else if values, ok := value.([]brokerapi.SessionTurnExecutionWatchEvent); ok {
		count = len(values)
	}
	return len(blob), count, nil
}

func p95Records(samplesByMetric map[string][]float64) ([]perfcontracts.MeasurementRecord, error) {
	keys := make([]string, 0, len(samplesByMetric))
	for metricID := range samplesByMetric {
		keys = append(keys, metricID)
	}
	sort.Strings(keys)
	records := make([]perfcontracts.MeasurementRecord, 0, len(keys))
	for _, metricID := range keys {
		value, err := p95(samplesByMetric[metricID])
		if err != nil {
			return nil, fmt.Errorf("compute p95 for %s: %w", metricID, err)
		}
		records = append(records, perfcontracts.MeasurementRecord{MetricID: metricID, Value: value, Unit: "ms"})
	}
	return records, nil
}

func maxFloat64(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func p95(samples []float64) (float64, error) {
	if len(samples) == 0 {
		return 0, fmt.Errorf("samples required")
	}
	vals := append([]float64(nil), samples...)
	sort.Float64s(vals)
	idx := int(float64(len(vals)-1) * 0.95)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(vals) {
		idx = len(vals) - 1
	}
	return vals[idx], nil
}
