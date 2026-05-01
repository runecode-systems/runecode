package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	runnerDetailsMaxEntries  = 64
	runnerDetailsMaxDepth    = 4
	runnerDetailsMaxStrLen   = 1024
	runnerDetailsMaxArrayLen = 64
)

type runnerReportPreparation struct {
	requestID string
	runID     string
	occurred  time.Time
	current   string
	found     bool
}

type runnerResultBindings struct {
	details            map[string]any
	overrideActionHash string
	overridePolicyRef  string
	gateEvidenceRef    string
	gateResultRef      string
}

type runnerValidationError struct {
	code string
	msg  string
}

func (e runnerValidationError) Error() string { return e.msg }

func (s *Service) HandleRunnerCheckpointReport(ctx context.Context, req RunnerCheckpointReportRequest, meta RequestContext) (RunnerCheckpointReportResponse, *ErrorResponse) {
	prep, release, errResp := s.prepareRunnerCheckpointReport(ctx, req, meta)
	if errResp != nil {
		return RunnerCheckpointReportResponse{}, errResp
	}
	defer release()
	details, err := s.validateRunnerCheckpointReport(prep.current, prep.found, prep.runID, req.Report)
	if err != nil {
		return s.runnerCheckpointValidationError(prep.requestID, err)
	}
	accepted, err := s.RecordRunnerCheckpoint(prep.runID, buildRunnerCheckpointAdvisory(req.Report, prep.occurred, details))
	if err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	if !accepted {
		return s.buildRunnerCheckpointResponseFromCanonical(prep.requestID, prep.runID, req.Report.IdempotencyKey, false)
	}
	if errResp := s.persistRunnerCheckpointSideEffects(prep, req.Report); errResp != nil {
		return RunnerCheckpointReportResponse{}, errResp
	}
	return s.buildRunnerCheckpointResponseFromCanonical(prep.requestID, prep.runID, req.Report.IdempotencyKey, true)
}

func (s *Service) HandleRunnerResultReport(ctx context.Context, req RunnerResultReportRequest, meta RequestContext) (RunnerResultReportResponse, *ErrorResponse) {
	prep, release, errResp := s.prepareRunnerResultReport(ctx, req, meta)
	if errResp != nil {
		return RunnerResultReportResponse{}, errResp
	}
	defer release()
	bindings, err := s.prepareRunnerResultBindings(prep.current, prep.found, prep.runID, req.Report)
	if err != nil {
		return s.runnerResultValidationError(prep.requestID, err)
	}
	accepted, err := s.RecordRunnerResult(prep.runID, buildRunnerResultAdvisory(req.Report, prep.occurred, bindings.details, bindings.gateEvidenceRef, bindings.gateResultRef, bindings.overrideActionHash, bindings.overridePolicyRef), bindings.overridePolicyRef)
	if err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	if !accepted {
		return s.buildRunnerResultResponseFromCanonical(prep.requestID, prep.runID, req.Report.IdempotencyKey, false)
	}
	if errResp := s.persistRunnerResultSideEffects(prep, req.Report); errResp != nil {
		return RunnerResultReportResponse{}, errResp
	}
	return s.buildRunnerResultResponseFromCanonical(prep.requestID, prep.runID, req.Report.IdempotencyKey, true)
}

func (s *Service) persistRunnerCheckpointSideEffects(prep runnerReportPreparation, report RunnerCheckpointReport) *ErrorResponse {
	if err := s.SetRunStatus(prep.runID, mapLifecycleToStoreStatus(report.LifecycleState)); err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	if err := s.syncSessionExecutionForRun(prep.runID, prep.occurred); err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	if err := s.appendRunnerCheckpointExecutionCheckpoint(prep.runID, report.CheckpointCode); err != nil {
		errOut := s.makeError(prep.requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) persistRunnerResultSideEffects(prep runnerReportPreparation, report RunnerResultReport) *ErrorResponse {
	if err := s.SetRunStatus(prep.runID, mapLifecycleToStoreStatus(report.LifecycleState)); err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	if err := s.syncSessionExecutionForRun(prep.runID, prep.occurred); err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	if err := s.appendRunnerResultExecutionCheckpoint(prep.runID, report.ResultCode, report.LifecycleState); err != nil {
		errOut := s.makeError(prep.requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) prepareRunnerCheckpointReport(ctx context.Context, req RunnerCheckpointReportRequest, meta RequestContext) (runnerReportPreparation, func(), *ErrorResponse) {
	return s.prepareRunnerReport(ctx, req.RequestID, runnerCheckpointRequestSchemaPath, req.RunID, req.Report.OccurredAt, req, meta)
}

func (s *Service) runnerCheckpointValidationError(requestID string, err error) (RunnerCheckpointReportResponse, *ErrorResponse) {
	errOut := s.makeError(requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
	var validationErr runnerValidationError
	if errors.As(err, &validationErr) {
		errOut = s.makeError(requestID, validationErr.code, "validation", false, validationErr.msg)
	}
	return RunnerCheckpointReportResponse{}, &errOut
}

func (s *Service) prepareRunnerResultReport(ctx context.Context, req RunnerResultReportRequest, meta RequestContext) (runnerReportPreparation, func(), *ErrorResponse) {
	return s.prepareRunnerReport(ctx, req.RequestID, runnerResultRequestSchemaPath, req.RunID, req.Report.OccurredAt, req, meta)
}

func (s *Service) runnerResultValidationError(requestID string, err error) (RunnerResultReportResponse, *ErrorResponse) {
	errOut := s.makeError(requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
	var validationErr runnerValidationError
	if errors.As(err, &validationErr) {
		errOut = s.makeError(requestID, validationErr.code, "validation", false, validationErr.msg)
	}
	return RunnerResultReportResponse{}, &errOut
}

func (s *Service) prepareRunnerReport(ctx context.Context, requestIDInput, requestSchemaPath, runIDInput, occurredAtInput string, requestPayload any, meta RequestContext) (runnerReportPreparation, func(), *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(requestIDInput, meta.RequestID, meta.AdmissionErr, requestPayload, requestSchemaPath)
	if errResp != nil {
		return runnerReportPreparation{}, noOpRelease, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return runnerReportPreparation{}, noOpRelease, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		release()
		errOut := s.errorFromContext(requestID, err)
		return runnerReportPreparation{}, noOpRelease, &errOut
	}
	runID := strings.TrimSpace(runIDInput)
	if runID == "" {
		release()
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
		return runnerReportPreparation{}, noOpRelease, &errOut
	}
	occurredAt, err := time.Parse(time.RFC3339, occurredAtInput)
	if err != nil {
		release()
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "report.occurred_at must be RFC3339")
		return runnerReportPreparation{}, noOpRelease, &errOut
	}
	current, found, err := s.currentCanonicalLifecycleForRun(runID)
	if err != nil {
		release()
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return runnerReportPreparation{}, noOpRelease, &errOut
	}
	return runnerReportPreparation{requestID: requestID, runID: runID, occurred: occurredAt, current: current, found: found}, release, nil
}

func (s *Service) buildRunnerCheckpointReportResponse(requestID, runID, canonical, idempotencyKey string, accepted bool) (RunnerCheckpointReportResponse, *ErrorResponse) {
	resp := RunnerCheckpointReportResponse{
		SchemaID:                "runecode.protocol.v0.RunnerCheckpointReportResponse",
		SchemaVersion:           "0.1.0",
		RequestID:               requestID,
		RunID:                   runID,
		Accepted:                accepted,
		CanonicalLifecycleState: canonical,
		AcceptedAt:              s.now().UTC().Format(time.RFC3339),
		IdempotencyKey:          idempotencyKey,
	}
	if err := s.validateResponse(resp, runnerCheckpointRespSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return RunnerCheckpointReportResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) buildRunnerResultReportResponse(requestID, runID, canonical, idempotencyKey string, accepted bool) (RunnerResultReportResponse, *ErrorResponse) {
	resp := RunnerResultReportResponse{
		SchemaID:                "runecode.protocol.v0.RunnerResultReportResponse",
		SchemaVersion:           "0.1.0",
		RequestID:               requestID,
		RunID:                   runID,
		Accepted:                accepted,
		CanonicalLifecycleState: canonical,
		AcceptedAt:              s.now().UTC().Format(time.RFC3339),
		IdempotencyKey:          idempotencyKey,
	}
	if err := s.validateResponse(resp, runnerResultRespSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return RunnerResultReportResponse{}, &errOut
	}
	return resp, nil
}

func noOpRelease() {}

func (s *Service) syncSessionExecutionForRun(runID string, occurredAt time.Time) error {
	facts := s.RuntimeFacts(runID)
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	return s.SyncSessionExecutionFromRunRuntime(runID, facts, runnerAdvisory, occurredAt)
}

func (s *Service) currentCanonicalLifecycleForRun(runID string) (string, bool, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return "", false, fmt.Errorf("run id is required")
	}
	statusByRun := s.RunStatuses()
	status, hasStatus := statusByRun[trimmedRunID]
	hasArtifacts := false
	for _, rec := range s.List() {
		if rec.RunID == trimmedRunID {
			hasArtifacts = true
			break
		}
	}
	pending := 0
	for _, approval := range s.listApprovals() {
		if approval.Status == "pending" && approval.BoundScope.RunID == trimmedRunID {
			pending++
		}
	}
	hasRun := hasStatus || hasArtifacts || pending > 0
	if !hasRun {
		return "", false, nil
	}
	runnerAdvisory, _ := s.RunnerAdvisory(trimmedRunID)
	return runLifecycleFromStore(status, pending, hasArtifacts, runnerAdvisory, s.RuntimeFacts(trimmedRunID)), true, nil
}
