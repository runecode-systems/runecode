package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
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

func (s *Service) HandleRunnerCheckpointReport(ctx context.Context, req RunnerCheckpointReportRequest, meta RequestContext) (RunnerCheckpointReportResponse, *ErrorResponse) {
	prep, release, errResp := s.prepareRunnerCheckpointReport(ctx, req, meta)
	if errResp != nil {
		return RunnerCheckpointReportResponse{}, errResp
	}
	defer release()
	if err := validateRunnerCheckpointTransition(prep.current, prep.found, req.Report.LifecycleState); err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	if err := validateRunnerCheckpointCode(req.Report.CheckpointCode); err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	runnerAdvisory, _ := s.RunnerAdvisory(prep.runID)
	if err := validateRunnerCheckpointPhaseTransition(runnerAdvisory, req.Report); err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	details, err := sanitizeRunnerDetails(req.Report.Details)
	if err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	accepted, err := s.RecordRunnerCheckpoint(prep.runID, artifacts.RunnerCheckpointAdvisory{
		LifecycleState:   req.Report.LifecycleState,
		CheckpointCode:   req.Report.CheckpointCode,
		OccurredAt:       prep.occurred.UTC(),
		IdempotencyKey:   req.Report.IdempotencyKey,
		StageID:          req.Report.StageID,
		StepID:           req.Report.StepID,
		RoleInstanceID:   req.Report.RoleInstanceID,
		StageAttemptID:   req.Report.StageAttemptID,
		StepAttemptID:    req.Report.StepAttemptID,
		GateAttemptID:    req.Report.GateAttemptID,
		PendingApprovals: req.Report.PendingApprovalCount,
		Details:          details,
	})
	if err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	if !accepted {
		canonical, _, lookupErr := s.currentCanonicalLifecycleForRun(prep.runID)
		if lookupErr != nil {
			errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, lookupErr.Error())
			return RunnerCheckpointReportResponse{}, &errOut
		}
		return s.buildRunnerCheckpointReportResponse(prep.requestID, prep.runID, canonical, req.Report.IdempotencyKey, false)
	}
	if err := s.SetRunStatus(prep.runID, mapLifecycleToStoreStatus(req.Report.LifecycleState)); err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	canonical, _, err := s.currentCanonicalLifecycleForRun(prep.runID)
	if err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	return s.buildRunnerCheckpointReportResponse(prep.requestID, prep.runID, canonical, req.Report.IdempotencyKey, true)
}

func (s *Service) HandleRunnerResultReport(ctx context.Context, req RunnerResultReportRequest, meta RequestContext) (RunnerResultReportResponse, *ErrorResponse) {
	prep, release, errResp := s.prepareRunnerResultReport(ctx, req, meta)
	if errResp != nil {
		return RunnerResultReportResponse{}, errResp
	}
	defer release()
	if err := validateRunnerResultTransition(prep.current, prep.found, req.Report.LifecycleState); err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	if err := validateRunnerResultCode(req.Report.ResultCode); err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_runner_transition_invalid", "validation", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	details, err := sanitizeRunnerDetails(req.Report.Details)
	if err != nil {
		errOut := s.makeError(prep.requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	accepted, err := s.RecordRunnerResult(prep.runID, artifacts.RunnerResultAdvisory{
		LifecycleState:    req.Report.LifecycleState,
		ResultCode:        req.Report.ResultCode,
		OccurredAt:        prep.occurred.UTC(),
		IdempotencyKey:    req.Report.IdempotencyKey,
		StageID:           req.Report.StageID,
		StepID:            req.Report.StepID,
		RoleInstanceID:    req.Report.RoleInstanceID,
		StageAttemptID:    req.Report.StageAttemptID,
		StepAttemptID:     req.Report.StepAttemptID,
		GateAttemptID:     req.Report.GateAttemptID,
		FailureReasonCode: req.Report.FailureReasonCode,
		Details:           details,
	})
	if err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	if !accepted {
		canonical, _, lookupErr := s.currentCanonicalLifecycleForRun(prep.runID)
		if lookupErr != nil {
			errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, lookupErr.Error())
			return RunnerResultReportResponse{}, &errOut
		}
		return s.buildRunnerResultReportResponse(prep.requestID, prep.runID, canonical, req.Report.IdempotencyKey, false)
	}
	if err := s.SetRunStatus(prep.runID, mapLifecycleToStoreStatus(req.Report.LifecycleState)); err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	canonical, _, err := s.currentCanonicalLifecycleForRun(prep.runID)
	if err != nil {
		errOut := s.makeError(prep.requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	return s.buildRunnerResultReportResponse(prep.requestID, prep.runID, canonical, req.Report.IdempotencyKey, true)
}

func (s *Service) prepareRunnerCheckpointReport(ctx context.Context, req RunnerCheckpointReportRequest, meta RequestContext) (runnerReportPreparation, func(), *ErrorResponse) {
	return s.prepareRunnerReport(ctx, req.RequestID, runnerCheckpointRequestSchemaPath, req.RunID, req.Report.OccurredAt, req, meta)
}

func (s *Service) prepareRunnerResultReport(ctx context.Context, req RunnerResultReportRequest, meta RequestContext) (runnerReportPreparation, func(), *ErrorResponse) {
	return s.prepareRunnerReport(ctx, req.RequestID, runnerResultRequestSchemaPath, req.RunID, req.Report.OccurredAt, req, meta)
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
	return runLifecycleFromStore(status, pending, hasArtifacts, runnerAdvisory), true, nil
}

func validateRunnerCheckpointTransition(current string, found bool, next string) error {
	if !isCheckpointLifecycle(next) {
		return fmt.Errorf("checkpoint lifecycle %q is invalid", next)
	}
	if !found {
		if next != "pending" {
			return fmt.Errorf("checkpoint transition for unknown run is invalid: %q", next)
		}
		return nil
	}
	if isTerminalLifecycle(current) {
		return fmt.Errorf("checkpoint transition %q -> %q is invalid", current, next)
	}
	if !isAllowedLifecycleTransition(current, next) {
		return fmt.Errorf("checkpoint transition %q -> %q is invalid", current, next)
	}
	return nil
}

func validateRunnerResultTransition(current string, found bool, next string) error {
	if !isTerminalLifecycle(next) {
		return fmt.Errorf("result lifecycle %q is invalid", next)
	}
	if !found {
		return fmt.Errorf("result transition for unknown run is invalid: %q", next)
	}
	if isTerminalLifecycle(current) {
		if current != next {
			return fmt.Errorf("result transition %q -> %q is invalid", current, next)
		}
		return nil
	}
	if !isAllowedLifecycleTransition(current, next) {
		return fmt.Errorf("result transition %q -> %q is invalid", current, next)
	}
	return nil
}

func isCheckpointLifecycle(state string) bool {
	switch state {
	case "pending", "starting", "active", "blocked", "recovering":
		return true
	default:
		return false
	}
}

func isTerminalLifecycle(state string) bool {
	switch state {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func isAllowedLifecycleTransition(current, next string) bool {
	if current == next {
		return true
	}
	switch current {
	case "pending":
		return next == "starting" || next == "active" || next == "blocked" || next == "recovering" || isTerminalLifecycle(next)
	case "starting":
		return next == "active" || next == "blocked" || next == "recovering" || isTerminalLifecycle(next)
	case "active":
		return next == "blocked" || next == "recovering" || isTerminalLifecycle(next)
	case "blocked":
		return next == "active" || next == "recovering" || isTerminalLifecycle(next)
	case "recovering":
		return next == "starting" || next == "active" || next == "blocked" || isTerminalLifecycle(next)
	case "completed", "failed", "cancelled":
		return false
	default:
		return false
	}
}

func mapLifecycleToStoreStatus(state string) string {
	if state == "completed" {
		return "closed"
	}
	return state
}

func validateRunnerCheckpointCode(code string) error {
	switch strings.TrimSpace(code) {
	case "run_started", "stage_entered", "step_attempt_started", "action_request_issued",
		"step_validation_started", "step_validation_finished", "approval_wait_entered", "approval_wait_cleared",
		"gate_attempt_started", "gate_attempt_finished", "step_execution_started", "step_execution_finished",
		"step_attest_started", "step_attest_finished", "step_attempt_finished", "run_terminal":
		return nil
	default:
		return fmt.Errorf("unsupported checkpoint code %q", strings.TrimSpace(code))
	}
}

func validateRunnerResultCode(code string) error {
	switch strings.TrimSpace(code) {
	case "run_completed", "run_failed", "run_cancelled", "step_failed", "gate_failed":
		return nil
	default:
		return fmt.Errorf("unsupported result code %q", strings.TrimSpace(code))
	}
}

func validateRunnerCheckpointPhaseTransition(advisory artifacts.RunnerAdvisoryState, report RunnerCheckpointReport) error {
	stepAttemptID := strings.TrimSpace(report.StepAttemptID)
	if stepAttemptID == "" {
		return nil
	}
	nextPhase, ok := phaseForCheckpointCode(report.CheckpointCode)
	if !ok {
		return nil
	}
	current, hasCurrent := advisory.StepAttempts[stepAttemptID]
	if !hasCurrent || strings.TrimSpace(current.CurrentPhase) == "" {
		if nextPhase != "propose" && nextPhase != "validate" && nextPhase != "authorize" {
			return fmt.Errorf("step_attempt %q phase transition <none> -> %s is invalid", stepAttemptID, nextPhase)
		}
		return nil
	}
	if !isAllowedExecutionPhaseTransition(strings.TrimSpace(current.CurrentPhase), nextPhase) {
		return fmt.Errorf("step_attempt %q phase transition %s -> %s is invalid", stepAttemptID, strings.TrimSpace(current.CurrentPhase), nextPhase)
	}
	return nil
}

func phaseForCheckpointCode(code string) (string, bool) {
	switch strings.TrimSpace(code) {
	case "step_attempt_started", "action_request_issued":
		return "propose", true
	case "step_validation_started", "step_validation_finished", "gate_attempt_started", "gate_attempt_finished":
		return "validate", true
	case "approval_wait_entered", "approval_wait_cleared":
		return "authorize", true
	case "step_execution_started", "step_execution_finished":
		return "execute", true
	case "step_attest_started", "step_attest_finished", "step_attempt_finished":
		return "attest", true
	default:
		return "", false
	}
}

func isAllowedExecutionPhaseTransition(current, next string) bool {
	if current == next {
		return true
	}
	order := map[string]int{"propose": 0, "validate": 1, "authorize": 2, "execute": 3, "attest": 4}
	currentOrder, okCurrent := order[current]
	nextOrder, okNext := order[next]
	if !okCurrent || !okNext {
		return false
	}
	if nextOrder == currentOrder+1 {
		return true
	}
	// authorize can be omitted when no human-approval gate is required.
	if current == "validate" && next == "execute" {
		return true
	}
	return false
}

func sanitizeRunnerDetails(details map[string]any) (map[string]any, error) {
	if len(details) == 0 {
		return nil, nil
	}
	state := &runnerDetailsValidationState{}
	if err := validateRunnerDetailsObject(details, 0, state); err != nil {
		return nil, err
	}
	return cloneRunnerDetailsMap(details), nil
}

type runnerDetailsValidationState struct {
	seen int
}

func validateRunnerDetailsObject(details map[string]any, depth int, state *runnerDetailsValidationState) error {
	if depth > runnerDetailsMaxDepth {
		return fmt.Errorf("report.details exceeds max nesting depth")
	}
	if len(details) > runnerDetailsMaxEntries {
		return fmt.Errorf("report.details object exceeds max keys")
	}
	for key, value := range details {
		if err := validateRunnerDetailsKey(key, state); err != nil {
			return err
		}
		if err := validateRunnerDetailsValue(value, depth+1, state); err != nil {
			return err
		}
	}
	return nil
}

func validateRunnerDetailsValue(value any, depth int, state *runnerDetailsValidationState) error {
	if depth > runnerDetailsMaxDepth {
		return fmt.Errorf("report.details exceeds max nesting depth")
	}
	switch typed := value.(type) {
	case nil, bool, float64, int, int64:
		return nil
	case string:
		if len(typed) > runnerDetailsMaxStrLen {
			return fmt.Errorf("report.details string value exceeds max length")
		}
		return nil
	case map[string]any:
		return validateRunnerDetailsObject(typed, depth, state)
	case []any:
		return validateRunnerDetailsArray(typed, depth, state)
	default:
		return fmt.Errorf("report.details contains unsupported value type %T", value)
	}
}

func validateRunnerDetailsArray(items []any, depth int, state *runnerDetailsValidationState) error {
	if len(items) > runnerDetailsMaxArrayLen {
		return fmt.Errorf("report.details array exceeds max length")
	}
	for _, item := range items {
		if err := validateRunnerDetailsValue(item, depth+1, state); err != nil {
			return err
		}
	}
	return nil
}

func validateRunnerDetailsKey(key string, state *runnerDetailsValidationState) error {
	state.seen++
	if state.seen > runnerDetailsMaxEntries {
		return fmt.Errorf("report.details exceeds max total entries")
	}
	if len(strings.TrimSpace(key)) == 0 {
		return fmt.Errorf("report.details contains empty key")
	}
	if len(key) > 128 {
		return fmt.Errorf("report.details key exceeds max length")
	}
	return nil
}

func cloneRunnerDetailsMap(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	out := make(map[string]any, len(details))
	for key, value := range details {
		out[key] = value
	}
	return out
}
