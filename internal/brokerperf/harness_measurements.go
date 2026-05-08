package brokerperf

import (
	"context"
	"fmt"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func measureUnary(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	ctx := context.Background()
	specs := unaryLatencySpecs(ctx, service)
	return collectLatencyMeasurements(trials, specs)
}

func unaryLatencySpecs(ctx context.Context, service *brokerapi.Service) []latencySpec {
	return []latencySpec{
		{metricID: "metric.broker.unary.session_list.p95_ms", call: func() error {
			_, errResp := service.HandleSessionList(ctx, brokerapi.SessionListRequest{SchemaID: "runecode.protocol.v0.SessionListRequest", SchemaVersion: "0.1.0", RequestID: "perf-session-list", Limit: 20}, brokerapi.RequestContext{})
			return unaryErr(errResp, "session_list")
		}},
		{metricID: "metric.broker.unary.session_get.p95_ms", call: func() error {
			_, errResp := service.HandleSessionGet(ctx, brokerapi.SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: "0.1.0", RequestID: "perf-session-get", SessionID: "sess-broker-1"}, brokerapi.RequestContext{})
			return unaryErr(errResp, "session_get")
		}},
		{metricID: "metric.broker.unary.run_list.p95_ms", call: func() error {
			_, errResp := service.HandleRunList(ctx, brokerapi.RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "perf-run-list", Limit: 20}, brokerapi.RequestContext{})
			return unaryErr(errResp, "run_list")
		}},
		{metricID: "metric.broker.unary.run_get.p95_ms", call: func() error {
			_, errResp := service.HandleRunGet(ctx, brokerapi.RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "perf-run-get", RunID: "run-broker-1"}, brokerapi.RequestContext{})
			return unaryErr(errResp, "run_get")
		}},
		{metricID: "metric.broker.unary.approval_list.p95_ms", call: func() error {
			_, errResp := service.HandleApprovalList(ctx, brokerapi.ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "perf-approval-list", Limit: 20}, brokerapi.RequestContext{})
			return unaryErr(errResp, "approval_list")
		}},
		{metricID: "metric.broker.unary.readiness_get.p95_ms", call: func() error {
			_, errResp := service.HandleReadinessGet(ctx, brokerapi.ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: "perf-readiness"}, brokerapi.RequestContext{})
			return unaryErr(errResp, "readiness_get")
		}},
		{metricID: "metric.broker.unary.version_info_get.p95_ms", call: func() error {
			_, errResp := service.HandleVersionInfoGet(ctx, brokerapi.VersionInfoGetRequest{SchemaID: "runecode.protocol.v0.VersionInfoGetRequest", SchemaVersion: "0.1.0", RequestID: "perf-version"}, brokerapi.RequestContext{})
			return unaryErr(errResp, "version_info_get")
		}},
		{metricID: "metric.broker.unary.project_substrate_posture_get.p95_ms", call: func() error {
			_, errResp := service.HandleProjectSubstratePostureGet(ctx, brokerapi.ProjectSubstratePostureGetRequest{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureGetRequest", SchemaVersion: "0.1.0", RequestID: "perf-project-posture"}, brokerapi.RequestContext{})
			return unaryErr(errResp, "project_substrate_posture_get")
		}},
	}
}

func measureWatches(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	ctx := context.Background()
	specs := []watchSpec{
		{latencyMetricID: "metric.broker.watch.run.snapshot_follow.p95_ms", payloadMetricID: "metric.broker.watch.run.snapshot_follow.payload_bytes", countMetricID: "metric.broker.watch.run.snapshot_follow.event_count", call: func() (any, error) { return streamRunWatch(ctx, service) }},
		{latencyMetricID: "metric.broker.watch.approval.snapshot_follow.p95_ms", payloadMetricID: "metric.broker.watch.approval.snapshot_follow.payload_bytes", countMetricID: "metric.broker.watch.approval.snapshot_follow.event_count", call: func() (any, error) { return streamApprovalWatch(ctx, service) }},
		{latencyMetricID: "metric.broker.watch.session.snapshot_follow.p95_ms", payloadMetricID: "metric.broker.watch.session.snapshot_follow.payload_bytes", countMetricID: "metric.broker.watch.session.snapshot_follow.event_count", call: func() (any, error) { return streamSessionWatch(ctx, service) }},
		{latencyMetricID: "metric.broker.watch.turn_execution.snapshot_follow.p95_ms", payloadMetricID: "metric.broker.watch.turn_execution.snapshot_follow.payload_bytes", countMetricID: "metric.broker.watch.turn_execution.snapshot_follow.event_count", call: func() (any, error) { return streamTurnExecutionWatch(ctx, service) }},
	}
	return collectWatchMeasurements(trials, specs)
}

func measureMutations(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	ctx := context.Background()
	latency := map[string][]float64{}
	for i := 0; i < trials; i++ {
		triggerDuration, err := measureSessionExecutionTriggerMutation(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		latency["metric.broker.mutation.session_execution_trigger.p95_ms"] = append(latency["metric.broker.mutation.session_execution_trigger.p95_ms"], triggerDuration)

		continueDuration, err := measureSessionExecutionContinueMutation(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		latency["metric.broker.mutation.session_execution_continue.p95_ms"] = append(latency["metric.broker.mutation.session_execution_continue.p95_ms"], continueDuration)

		approvalDuration, err := measureApprovalResolveMutation(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		latency["metric.broker.mutation.approval_resolve.p95_ms"] = append(latency["metric.broker.mutation.approval_resolve.p95_ms"], approvalDuration)

		postureDuration, err := measureBackendPostureChangeMutation(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		latency["metric.broker.mutation.backend_posture_change.p95_ms"] = append(latency["metric.broker.mutation.backend_posture_change.p95_ms"], postureDuration)
	}
	return p95Records(latency)
}

func measureAttachResume(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	attachSamples := make([]float64, 0, trials)
	resumeSamples := make([]float64, 0, trials)
	ctx := context.Background()
	for i := 0; i < trials; i++ {
		attachDuration, resumeDuration, err := measureAttachResumeTrial(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		attachSamples = append(attachSamples, attachDuration)
		resumeSamples = append(resumeSamples, resumeDuration)
	}
	attachP95, err := p95(attachSamples)
	if err != nil {
		return nil, err
	}
	resumeP95, err := p95(resumeSamples)
	if err != nil {
		return nil, err
	}
	return []perfcontracts.MeasurementRecord{
		{MetricID: "metric.broker.attach.local_control_plane.p95_ms", Value: attachP95, Unit: "ms"},
		{MetricID: "metric.broker.resume.local_control_plane.p95_ms", Value: resumeP95, Unit: "ms"},
	}, nil
}

func measureAttachResumeTrial(ctx context.Context, repoRoot string) (float64, float64, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return 0, 0, err
	}
	defer cleanup()
	attachDuration, err := timedProductLifecyclePostureGet(ctx, service, "perf-attach", "attach")
	if err != nil {
		return 0, 0, err
	}
	resumeDuration, err := timedProductLifecyclePostureGet(ctx, service, "perf-resume", "resume")
	if err != nil {
		return 0, 0, err
	}
	return attachDuration, resumeDuration, nil
}

func timedProductLifecyclePostureGet(ctx context.Context, service *brokerapi.Service, requestID, label string) (float64, error) {
	return timedCall(func() error {
		_, errResp := service.HandleProductLifecyclePostureGet(ctx, brokerapi.ProductLifecyclePostureGetRequest{SchemaID: "runecode.protocol.v0.ProductLifecyclePostureGetRequest", SchemaVersion: "0.1.0", RequestID: requestID}, brokerapi.RequestContext{})
		if errResp != nil {
			return fmt.Errorf("product_lifecycle_posture_get %s: %s", label, errResp.Error.Code)
		}
		return nil
	})
}

func measureSessionExecutionTriggerMutation(ctx context.Context, repoRoot string) (float64, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	return timedCall(func() error {
		_, errResp := service.HandleSessionExecutionTrigger(ctx, brokerapi.SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "perf-trigger", SessionID: "sess-broker-1", TriggerSource: "autonomous_background", RequestedOperation: "start", WorkflowRouting: &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "change_draft"}, AutonomyPosture: "operator_guided"}, brokerapi.RequestContext{})
		if errResp != nil {
			return fmt.Errorf("session_execution_trigger: %s", errResp.Error.Code)
		}
		return nil
	})
}

func measureSessionExecutionContinueMutation(ctx context.Context, repoRoot string) (float64, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	startResp, errResp := service.HandleSessionExecutionTrigger(ctx, brokerapi.SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "perf-continue-start", SessionID: "sess-broker-1", TriggerSource: "autonomous_background", RequestedOperation: "start", WorkflowRouting: &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "change_draft"}, AutonomyPosture: "operator_guided"}, brokerapi.RequestContext{})
	if errResp != nil {
		return 0, fmt.Errorf("continue start seed: %s", errResp.Error.Code)
	}
	if _, err := service.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-broker-1", TurnID: startResp.TurnID, ExecutionState: "blocked", WaitKind: "project_blocked", WaitState: "waiting_project_blocked", BlockedReasonCode: "project_substrate_posture_blocked", OccurredAt: time.Now().UTC()}); err != nil {
		return 0, fmt.Errorf("continue blocked seed: %w", err)
	}
	return timedCall(func() error {
		_, errResp = service.HandleSessionExecutionTrigger(ctx, brokerapi.SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "perf-continue", SessionID: "sess-broker-1", TurnID: startResp.TurnID, TriggerSource: "resume_follow_up", RequestedOperation: "continue", WorkflowRouting: &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "change_draft"}}, brokerapi.RequestContext{})
		if errResp != nil {
			return fmt.Errorf("session_execution_continue: %s", errResp.Error.Code)
		}
		return nil
	})
}

func measureApprovalResolveMutation(ctx context.Context, repoRoot string) (float64, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	resolveReq, err := seedBackendPostureApprovalForResolveWithRunID(service, "")
	if err != nil {
		return 0, err
	}
	return timedCall(func() error {
		_, errResp := service.HandleApprovalResolve(ctx, resolveReq, brokerapi.RequestContext{})
		if errResp != nil {
			return fmt.Errorf("approval_resolve: %s: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return nil
	})
}

func measureBackendPostureChangeMutation(ctx context.Context, repoRoot string) (float64, error) {
	service, cleanup, err := newSeededService(repoRoot)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	if err := seedBackendPosturePolicyContext(service); err != nil {
		return 0, fmt.Errorf("backend_posture_change context: %w", err)
	}
	targetInstanceID := serviceCurrentInstanceID(service)
	if targetInstanceID == "" {
		return 0, fmt.Errorf("backend_posture_change target instance missing")
	}
	return timedCall(func() error {
		_, errResp := service.HandleBackendPostureChange(ctx, brokerapi.BackendPostureChangeRequest{
			SchemaID:                     "runecode.protocol.v0.BackendPostureChangeRequest",
			SchemaVersion:                "0.1.0",
			RequestID:                    "perf-backend-posture-change",
			TargetInstanceID:             targetInstanceID,
			TargetBackendKind:            "container",
			SelectionMode:                "explicit_selection",
			ChangeKind:                   "select_backend",
			AssuranceChangeKind:          "reduce_assurance",
			OptInKind:                    "exact_action_approval",
			ReducedAssuranceAcknowledged: true,
			Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
		}, brokerapi.RequestContext{})
		if errResp != nil {
			return fmt.Errorf("backend_posture_change: %s: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return nil
	})
}
