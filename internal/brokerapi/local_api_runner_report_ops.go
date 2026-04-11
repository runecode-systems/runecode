package brokerapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
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

func (s *Service) validateRunnerCheckpointReport(current string, found bool, runID string, report RunnerCheckpointReport) (map[string]any, error) {
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	if err := validateRunnerCheckpointTransition(current, found, report.LifecycleState); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateRunnerCheckpointCode(report.CheckpointCode); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateCheckpointFields(report); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateCheckpointPlanBinding(report, planned); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateAttemptRetryPosture(runnerAdvisory, report.GateAttemptID, report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex, planned); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateCheckpointGateAttemptMutation(runnerAdvisory, report); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateRunnerCheckpointPhaseTransition(runnerAdvisory, report); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	details, err := sanitizeRunnerDetails(report.Details)
	if err != nil {
		return nil, runnerValidationError{code: "broker_validation_schema_invalid", msg: err.Error()}
	}
	return details, nil
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

func (s *Service) prepareRunnerResultBindings(current string, found bool, runID string, report RunnerResultReport) (runnerResultBindings, error) {
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	if err := validateRunnerResultTransition(current, found, report.LifecycleState); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateRunnerResultCode(report.ResultCode); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateResultFields(report); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateResultPlanBinding(report, planned); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateAttemptRetryPosture(runnerAdvisory, report.GateAttemptID, report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex, planned); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateResultGateAttemptMutation(runnerAdvisory, report); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateOverrideReferenceAgainstHistory(runnerAdvisory, report); err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	details, err := sanitizeRunnerDetails(report.Details)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_schema_invalid", msg: err.Error()}
	}
	overrideActionHash, overridePolicyRef, err := s.resolveOverrideApprovalBindings(runID, report, details)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	plannedEntry, _ := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	gateEvidenceRef, err := s.resolveGateEvidenceRef(runID, report, plannedEntry)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	gateResultRef, err := canonicalGateResultRef(runID, report, gateEvidenceRef)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	return runnerResultBindings{
		details:            details,
		overrideActionHash: overrideActionHash,
		overridePolicyRef:  overridePolicyRef,
		gateEvidenceRef:    gateEvidenceRef,
		gateResultRef:      gateResultRef,
	}, nil
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
		"gate_attempt_started", "gate_attempt_finished",
		"gate_planned", "gate_started", "gate_passed", "gate_failed", "gate_overridden", "gate_superseded",
		"step_execution_started", "step_execution_finished",
		"step_attest_started", "step_attest_finished", "step_attempt_finished", "run_terminal":
		return nil
	default:
		return fmt.Errorf("unsupported checkpoint code %q", strings.TrimSpace(code))
	}
}

func validateRunnerResultCode(code string) error {
	switch strings.TrimSpace(code) {
	case "run_completed", "run_failed", "run_cancelled", "step_failed", "gate_failed", "gate_passed", "gate_overridden", "gate_superseded":
		return nil
	default:
		return fmt.Errorf("unsupported result code %q", strings.TrimSpace(code))
	}
}

func validateGateCheckpointFields(report RunnerCheckpointReport) error {
	if strings.TrimSpace(report.PlanCheckpointCode) != "" && report.PlanOrderIndex < 0 {
		return fmt.Errorf("plan_order_index must be >= 0 when plan_checkpoint_code is set")
	}
	if strings.TrimSpace(report.GateEvidenceRef) != "" {
		if !isValidDigestIdentity(strings.TrimSpace(report.GateEvidenceRef)) {
			return fmt.Errorf("gate_evidence_ref must be digest identity")
		}
		if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
			return fmt.Errorf("gate_evidence_ref requires gate identity binding")
		}
	}
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
		return nil
	}
	if strings.TrimSpace(report.GateID) == "" || strings.TrimSpace(report.GateKind) == "" || strings.TrimSpace(report.GateVersion) == "" || strings.TrimSpace(report.GateAttemptID) == "" || strings.TrimSpace(report.GateLifecycleState) == "" {
		return fmt.Errorf("gate-scoped checkpoint requires gate_id, gate_kind, gate_version, gate_attempt_id, and gate_lifecycle_state")
	}
	if !isGateKind(report.GateKind) {
		return fmt.Errorf("unsupported gate_kind %q", report.GateKind)
	}
	if !isGateCheckpointState(report.GateLifecycleState) {
		return fmt.Errorf("unsupported gate_lifecycle_state %q for checkpoint", report.GateLifecycleState)
	}
	if expected, ok := gateStateForCheckpointCode(report.CheckpointCode); ok && expected != report.GateLifecycleState {
		return fmt.Errorf("checkpoint_code %q requires gate_lifecycle_state %q, got %q", report.CheckpointCode, expected, report.GateLifecycleState)
	}
	if err := validateNormalizedInputDigests(report.NormalizedInputDigests); err != nil {
		return err
	}
	return nil
}

func validateGateResultFields(report RunnerResultReport) error {
	if strings.TrimSpace(report.PlanCheckpointCode) != "" && report.PlanOrderIndex < 0 {
		return fmt.Errorf("plan_order_index must be >= 0 when plan_checkpoint_code is set")
	}
	if err := validateGateEvidenceBindingFields(report); err != nil {
		return err
	}
	if !shouldValidateGateScopedResult(report) {
		return nil
	}
	if err := validateGateScopedResultCode(report.ResultCode); err != nil {
		return err
	}
	if err := validateGateResultIdentity(report); err != nil {
		return err
	}
	if err := validateGateResultStateAndInputs(report); err != nil {
		return err
	}
	return validateGateResultOverrideFields(report)
}

func validateGateCheckpointPlanBinding(report RunnerCheckpointReport, planned compiledRunGatePlan) error {
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
		if strings.TrimSpace(report.PlanCheckpointCode) != "" {
			return fmt.Errorf("plan_checkpoint_code requires gate identity binding")
		}
		if report.PlanOrderIndex != 0 {
			return fmt.Errorf("plan_order_index requires gate identity binding")
		}
		return nil
	}
	if !planned.hasEntries() {
		if strings.TrimSpace(report.PlanCheckpointCode) != "" || report.PlanOrderIndex != 0 {
			return fmt.Errorf("gate-scoped checkpoint requires trusted run plan gate entries")
		}
		return nil
	}
	if strings.TrimSpace(report.PlanCheckpointCode) == "" {
		return fmt.Errorf("gate-scoped checkpoint requires plan_checkpoint_code")
	}
	entry, ok := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	if !ok {
		return fmt.Errorf("gate-scoped checkpoint does not match trusted run plan placement")
	}
	if err := validatePlannedInputDigestHooks(entry.ExpectedInputDigests, report.NormalizedInputDigests); err != nil {
		return err
	}
	return nil
}

func validateGateResultPlanBinding(report RunnerResultReport, planned compiledRunGatePlan) error {
	if !shouldValidateGateScopedResult(report) {
		if strings.TrimSpace(report.PlanCheckpointCode) != "" {
			return fmt.Errorf("plan_checkpoint_code requires gate identity binding")
		}
		if report.PlanOrderIndex != 0 {
			return fmt.Errorf("plan_order_index requires gate identity binding")
		}
		return nil
	}
	if !planned.hasEntries() {
		if strings.TrimSpace(report.PlanCheckpointCode) != "" || report.PlanOrderIndex != 0 {
			return fmt.Errorf("gate-scoped result requires trusted run plan gate entries")
		}
		return nil
	}
	if strings.TrimSpace(report.PlanCheckpointCode) == "" {
		return fmt.Errorf("gate-scoped result requires plan_checkpoint_code")
	}
	entry, ok := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	if !ok {
		return fmt.Errorf("gate-scoped result does not match trusted run plan placement")
	}
	if err := validatePlannedInputDigestHooks(entry.ExpectedInputDigests, report.NormalizedInputDigests); err != nil {
		return err
	}
	return nil
}

func validateGateAttemptRetryPosture(advisory artifacts.RunnerAdvisoryState, gateAttemptID, gateID, gateKind, gateVersion, planCheckpointCode string, planOrderIndex int, planned compiledRunGatePlan) error {
	if !planned.hasEntries() {
		return nil
	}
	entry, ok := planned.entryFor(gateID, gateKind, gateVersion, planCheckpointCode, planOrderIndex)
	if !ok {
		return nil
	}
	trimmedAttemptID := strings.TrimSpace(gateAttemptID)
	if trimmedAttemptID == "" {
		return nil
	}
	if _, exists := advisory.GateAttempts[trimmedAttemptID]; exists {
		return nil
	}
	seen := map[string]struct{}{}
	for _, existing := range advisory.GateAttempts {
		if strings.TrimSpace(existing.GateID) != entry.GateID || strings.TrimSpace(existing.GateKind) != entry.GateKind || strings.TrimSpace(existing.GateVersion) != entry.GateVersion {
			continue
		}
		if strings.TrimSpace(existing.PlanCheckpoint) != entry.PlanCheckpointCode || existing.PlanOrderIndex != entry.PlanOrderIndex {
			continue
		}
		seen[strings.TrimSpace(existing.GateAttemptID)] = struct{}{}
	}
	if entry.MaxAttempts > 0 && len(seen) >= entry.MaxAttempts {
		return fmt.Errorf("gate plan retry posture exceeded: max_attempts=%d for gate %q at %s[%d]", entry.MaxAttempts, entry.GateID, entry.PlanCheckpointCode, entry.PlanOrderIndex)
	}
	return nil
}

func validateGateScopedResultCode(code string) error {
	switch strings.TrimSpace(code) {
	case "gate_failed", "gate_passed", "gate_overridden", "gate_superseded":
		return nil
	default:
		return fmt.Errorf("gate-scoped result requires gate_* result_code, got %q", strings.TrimSpace(code))
	}
}

func validateGateEvidenceBindingFields(report RunnerResultReport) error {
	if err := validateGateEvidencePayloadShape(report.GateEvidence); err != nil {
		return err
	}
	if strings.TrimSpace(report.GateEvidenceRef) != "" && !isValidDigestIdentity(strings.TrimSpace(report.GateEvidenceRef)) {
		return fmt.Errorf("gate_evidence_ref must be digest identity")
	}
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) && strings.TrimSpace(report.OverriddenFailedResultRef) == "" {
		if report.GateEvidence != nil || strings.TrimSpace(report.GateEvidenceRef) != "" {
			return fmt.Errorf("gate evidence requires gate identity binding")
		}
	}
	return nil
}

func shouldValidateGateScopedResult(report RunnerResultReport) bool {
	return hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) || strings.TrimSpace(report.OverriddenFailedResultRef) != ""
}

func validateGateResultIdentity(report RunnerResultReport) error {
	if strings.TrimSpace(report.GateID) == "" || strings.TrimSpace(report.GateKind) == "" || strings.TrimSpace(report.GateVersion) == "" || strings.TrimSpace(report.GateAttemptID) == "" || strings.TrimSpace(report.GateLifecycleState) == "" {
		return fmt.Errorf("gate-scoped result requires gate_id, gate_kind, gate_version, gate_attempt_id, and gate_lifecycle_state")
	}
	if !isGateKind(report.GateKind) {
		return fmt.Errorf("unsupported gate_kind %q", report.GateKind)
	}
	return nil
}

func validateGateResultStateAndInputs(report RunnerResultReport) error {
	if !isGateResultState(report.GateLifecycleState) {
		return fmt.Errorf("unsupported gate_lifecycle_state %q for result", report.GateLifecycleState)
	}
	if expected, ok := gateStateForResultCode(report.ResultCode); ok && expected != report.GateLifecycleState {
		return fmt.Errorf("result_code %q requires gate_lifecycle_state %q, got %q", report.ResultCode, expected, report.GateLifecycleState)
	}
	if err := validateNormalizedInputDigests(report.NormalizedInputDigests); err != nil {
		return err
	}
	if strings.TrimSpace(report.ResultCode) == "gate_failed" && strings.TrimSpace(report.LifecycleState) != "failed" {
		return fmt.Errorf("gate_failed result requires lifecycle_state failed")
	}
	return nil
}

func validateGateResultOverrideFields(report RunnerResultReport) error {
	if report.GateLifecycleState == "overridden" && strings.TrimSpace(report.OverriddenFailedResultRef) == "" {
		return fmt.Errorf("overridden gate result requires overridden_failed_result_ref")
	}
	if strings.TrimSpace(report.OverriddenFailedResultRef) != "" && !isValidDigestIdentity(report.OverriddenFailedResultRef) {
		return fmt.Errorf("overridden_failed_result_ref must be digest identity")
	}
	return nil
}

func validateGateEvidencePayloadShape(evidence *GateEvidence) error {
	if evidence == nil {
		return nil
	}
	if err := validateGateEvidenceCoreFields(evidence); err != nil {
		return err
	}
	if err := validateGateEvidenceDigestFields(evidence); err != nil {
		return err
	}
	return validateGateEvidenceOverrideRefs(evidence)
}

func validateGateEvidenceCoreFields(evidence *GateEvidence) error {
	if evidence.SchemaID != "runecode.protocol.v0.GateEvidence" {
		return fmt.Errorf("gate_evidence.schema_id must be runecode.protocol.v0.GateEvidence")
	}
	if evidence.SchemaVersion != "0.1.0" {
		return fmt.Errorf("gate_evidence.schema_version must be 0.1.0")
	}
	if strings.TrimSpace(evidence.GateID) == "" || strings.TrimSpace(evidence.GateKind) == "" || strings.TrimSpace(evidence.GateVersion) == "" || strings.TrimSpace(evidence.GateAttemptID) == "" {
		return fmt.Errorf("gate_evidence requires gate_id, gate_kind, gate_version, and gate_attempt_id")
	}
	if strings.TrimSpace(evidence.PlanCheckpointCode) != "" && evidence.PlanOrderIndex < 0 {
		return fmt.Errorf("gate_evidence.plan_order_index must be >= 0 when plan_checkpoint_code is set")
	}
	if !isGateKind(evidence.GateKind) {
		return fmt.Errorf("gate_evidence has unsupported gate_kind %q", evidence.GateKind)
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(evidence.StartedAt)); err != nil {
		return fmt.Errorf("gate_evidence.started_at must be RFC3339")
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(evidence.FinishedAt)); err != nil {
		return fmt.Errorf("gate_evidence.finished_at must be RFC3339")
	}
	if len(evidence.Runtime) == 0 {
		return fmt.Errorf("gate_evidence.runtime is required")
	}
	if len(evidence.Outcome) == 0 {
		return fmt.Errorf("gate_evidence.outcome is required")
	}
	return nil
}

func validateGateEvidenceDigestFields(evidence *GateEvidence) error {
	if err := validateNormalizedInputDigests(evidence.NormalizedInputDigests); err != nil {
		return fmt.Errorf("gate_evidence.%w", err)
	}
	if err := validateNormalizedInputDigests(evidence.OutputArtifactDigests); err != nil {
		return fmt.Errorf("gate_evidence output_artifact_digests: %w", err)
	}
	if err := validateNormalizedInputDigests(evidence.PolicyDecisionRefs); err != nil {
		return fmt.Errorf("gate_evidence policy_decision_refs: %w", err)
	}
	return nil
}

func validateGateEvidenceOverrideRefs(evidence *GateEvidence) error {
	if strings.TrimSpace(evidence.OverrideActionRequestHash) != "" && !isValidDigestIdentity(strings.TrimSpace(evidence.OverrideActionRequestHash)) {
		return fmt.Errorf("gate_evidence.override_action_request_hash must be digest identity")
	}
	if strings.TrimSpace(evidence.OverridePolicyDecisionRef) != "" && !isValidDigestIdentity(strings.TrimSpace(evidence.OverridePolicyDecisionRef)) {
		return fmt.Errorf("gate_evidence.override_policy_decision_ref must be digest identity")
	}
	if strings.TrimSpace(evidence.OverriddenFailedResultRef) != "" && !isValidDigestIdentity(strings.TrimSpace(evidence.OverriddenFailedResultRef)) {
		return fmt.Errorf("gate_evidence.overridden_failed_result_ref must be digest identity")
	}
	return nil
}

func hasGateBinding(gateID, gateKind, gateVersion, gateAttemptID, gateState string, normalized []string) bool {
	return strings.TrimSpace(gateID) != "" || strings.TrimSpace(gateKind) != "" || strings.TrimSpace(gateVersion) != "" || strings.TrimSpace(gateAttemptID) != "" || strings.TrimSpace(gateState) != "" || len(normalized) > 0
}

func isGateKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "build", "test", "lint", "format", "secret_scan", "policy":
		return true
	default:
		return false
	}
}

func isGateCheckpointState(state string) bool {
	switch strings.TrimSpace(state) {
	case "planned", "running", "passed", "failed", "overridden", "superseded":
		return true
	default:
		return false
	}
}

func isGateResultState(state string) bool {
	switch strings.TrimSpace(state) {
	case "passed", "failed", "overridden", "superseded":
		return true
	default:
		return false
	}
}

func gateStateForCheckpointCode(code string) (string, bool) {
	switch strings.TrimSpace(code) {
	case "gate_planned":
		return "planned", true
	case "gate_started":
		return "running", true
	case "gate_passed":
		return "passed", true
	case "gate_failed":
		return "failed", true
	case "gate_overridden":
		return "overridden", true
	case "gate_superseded":
		return "superseded", true
	default:
		return "", false
	}
}

func gateStateForResultCode(code string) (string, bool) {
	switch strings.TrimSpace(code) {
	case "gate_passed":
		return "passed", true
	case "gate_failed":
		return "failed", true
	case "gate_overridden":
		return "overridden", true
	case "gate_superseded":
		return "superseded", true
	default:
		return "", false
	}
}

func validateNormalizedInputDigests(digests []string) error {
	if len(digests) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, digest := range digests {
		d := strings.TrimSpace(digest)
		if !isValidDigestIdentity(d) {
			return fmt.Errorf("normalized_input_digests contains invalid digest %q", digest)
		}
		if _, ok := seen[d]; ok {
			return fmt.Errorf("normalized_input_digests contains duplicate digest %q", d)
		}
		seen[d] = struct{}{}
	}
	return nil
}

func isValidDigestIdentity(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, ch := range value[len("sha256:"):] {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func buildRunnerCheckpointAdvisory(report RunnerCheckpointReport, occurred time.Time, details map[string]any) artifacts.RunnerCheckpointAdvisory {
	return artifacts.RunnerCheckpointAdvisory{
		LifecycleState:   report.LifecycleState,
		CheckpointCode:   report.CheckpointCode,
		OccurredAt:       occurred.UTC(),
		IdempotencyKey:   report.IdempotencyKey,
		PlanCheckpoint:   report.PlanCheckpointCode,
		PlanOrderIndex:   report.PlanOrderIndex,
		GateID:           report.GateID,
		GateKind:         report.GateKind,
		GateVersion:      report.GateVersion,
		GateState:        report.GateLifecycleState,
		StageID:          report.StageID,
		StepID:           report.StepID,
		RoleInstanceID:   report.RoleInstanceID,
		StageAttemptID:   report.StageAttemptID,
		StepAttemptID:    report.StepAttemptID,
		GateAttemptID:    report.GateAttemptID,
		GateEvidenceRef:  strings.TrimSpace(report.GateEvidenceRef),
		NormalizedInputs: append([]string{}, report.NormalizedInputDigests...),
		PendingApprovals: report.PendingApprovalCount,
		Details:          details,
	}
}

func buildRunnerResultAdvisory(report RunnerResultReport, occurred time.Time, details map[string]any, gateEvidenceRef string, gateResultRef string, overrideActionHash string, overridePolicyRef string) artifacts.RunnerResultAdvisory {
	return artifacts.RunnerResultAdvisory{
		LifecycleState:     report.LifecycleState,
		ResultCode:         report.ResultCode,
		OccurredAt:         occurred.UTC(),
		IdempotencyKey:     report.IdempotencyKey,
		PlanCheckpoint:     report.PlanCheckpointCode,
		PlanOrderIndex:     report.PlanOrderIndex,
		GateID:             report.GateID,
		GateKind:           report.GateKind,
		GateVersion:        report.GateVersion,
		GateState:          report.GateLifecycleState,
		StageID:            report.StageID,
		StepID:             report.StepID,
		RoleInstanceID:     report.RoleInstanceID,
		StageAttemptID:     report.StageAttemptID,
		StepAttemptID:      report.StepAttemptID,
		GateAttemptID:      report.GateAttemptID,
		NormalizedInputs:   append([]string{}, report.NormalizedInputDigests...),
		FailureReasonCode:  report.FailureReasonCode,
		OverrideFailedRef:  report.OverriddenFailedResultRef,
		OverrideActionHash: overrideActionHash,
		OverridePolicyRef:  overridePolicyRef,
		ResultRef:          gateResultRef,
		GateEvidenceRef:    strings.TrimSpace(gateEvidenceRef),
		Details:            details,
	}
}

func (s *Service) resolveOverrideApprovalBindings(runID string, report RunnerResultReport, sanitizedDetails map[string]any) (string, string, error) {
	if strings.TrimSpace(report.GateLifecycleState) != "overridden" {
		return "", "", nil
	}
	action, err := overrideActionForResult(report, sanitizedDetails)
	if err != nil {
		return "", "", err
	}
	actionHash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		return "", "", fmt.Errorf("canonical override action hash: %w", err)
	}
	latestRef, ok := s.latestGateOverridePolicyDecisionRef(runID, actionHash)
	if !ok {
		return "", "", fmt.Errorf("gate override requires prior policy decision approval for exact override action")
	}
	if err := s.requireValidGateOverrideApproval(runID, latestRef); err != nil {
		return "", "", err
	}
	if !hasPolicyContextDigest(sanitizedDetails, report.NormalizedInputDigests) {
		return "", "", fmt.Errorf("gate override result requires details.policy_context_hash present in normalized_input_digests")
	}
	return actionHash, latestRef, nil
}

func hasPolicyContextDigest(details map[string]any, normalizedInputDigests []string) bool {
	value, _ := details["policy_context_hash"].(string)
	policyContextHash := strings.TrimSpace(value)
	if !isValidDigestIdentity(policyContextHash) {
		return false
	}
	for _, digest := range normalizedInputDigests {
		if strings.TrimSpace(digest) == policyContextHash {
			return true
		}
	}
	return false
}

func (s *Service) latestGateOverridePolicyDecisionRef(runID, actionHash string) (string, bool) {
	latest := ""
	for _, ref := range s.PolicyDecisionRefsForRun(runID) {
		rec, ok := s.PolicyDecisionGet(ref)
		if !ok || !matchesGateOverridePolicyDecision(rec, actionHash) {
			continue
		}
		latest = ref
	}
	if latest == "" {
		return "", false
	}
	return latest, true
}

func matchesGateOverridePolicyDecision(rec artifacts.PolicyDecisionRecord, actionHash string) bool {
	if rec.DecisionOutcome != string(policyengine.DecisionRequireHumanApproval) {
		return false
	}
	if strings.TrimSpace(rec.ActionRequestHash) != actionHash {
		return false
	}
	if strings.TrimSpace(rec.PolicyReasonCode) != "approval_required" {
		return false
	}
	trigger, _ := rec.RequiredApproval["approval_trigger_code"].(string)
	return strings.TrimSpace(trigger) == "gate_override"
}

func (s *Service) requireValidGateOverrideApproval(runID, policyDecisionRef string) error {
	for _, approval := range s.listApprovals() {
		if !isMatchingApprovedGateOverrideApproval(approval, runID, policyDecisionRef) {
			continue
		}
		if err := validateGateOverrideApprovalExpiry(approval.ExpiresAt, s.now().UTC()); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("gate override requires explicit approved approval")
}

func isMatchingApprovedGateOverrideApproval(approval ApprovalSummary, runID, policyDecisionRef string) bool {
	if approval.BoundScope.RunID != runID {
		return false
	}
	if approval.BoundScope.ActionKind != policyengine.ActionKindGateOverride {
		return false
	}
	if strings.TrimSpace(approval.PolicyDecisionHash) != policyDecisionRef {
		return false
	}
	return approval.Status == "approved"
}

func validateGateOverrideApprovalExpiry(expiresAtRaw string, now time.Time) error {
	if expiresAtRaw == "" {
		return fmt.Errorf("gate override approval missing expires_at")
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtRaw)
	if err != nil {
		return fmt.Errorf("gate override approval has invalid expires_at")
	}
	if now.After(expiresAt.UTC()) {
		return fmt.Errorf("gate override approval expired")
	}
	return nil
}

func overrideActionForResult(report RunnerResultReport, details map[string]any) (policyengine.ActionRequest, error) {
	if strings.TrimSpace(report.GateID) == "" || strings.TrimSpace(report.GateKind) == "" || strings.TrimSpace(report.GateVersion) == "" || strings.TrimSpace(report.GateAttemptID) == "" || strings.TrimSpace(report.OverriddenFailedResultRef) == "" {
		return policyengine.ActionRequest{}, fmt.Errorf("gate override action requires gate identity, gate_attempt_id, and overridden_failed_result_ref")
	}
	policyContextHash, _ := details["policy_context_hash"].(string)
	policyContextHash = strings.TrimSpace(policyContextHash)
	if !isValidDigestIdentity(policyContextHash) {
		return policyengine.ActionRequest{}, fmt.Errorf("gate override result requires details.policy_context_hash digest")
	}
	if !hasPolicyContextDigest(details, report.NormalizedInputDigests) {
		return policyengine.ActionRequest{}, fmt.Errorf("details.policy_context_hash must be present in normalized_input_digests")
	}
	expiresAt := ""
	if value, ok := details["override_expires_at"].(string); ok {
		expiresAt = strings.TrimSpace(value)
	}
	if expiresAt == "" {
		return policyengine.ActionRequest{}, fmt.Errorf("gate override result requires details.override_expires_at")
	}
	if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
		return policyengine.ActionRequest{}, fmt.Errorf("details.override_expires_at must be RFC3339")
	}
	ticketRef := ""
	if value, ok := details["ticket_ref"].(string); ok {
		ticketRef = strings.TrimSpace(value)
	}
	overrideMode := "break_glass"
	if value, ok := details["override_mode"].(string); ok && strings.TrimSpace(value) != "" {
		overrideMode = strings.TrimSpace(value)
	}
	if overrideMode != "break_glass" && overrideMode != "temporary_allow" {
		return policyengine.ActionRequest{}, fmt.Errorf("details.override_mode must be one of: break_glass, temporary_allow")
	}
	justification := "gate override continuation"
	if value, ok := details["override_reason"].(string); ok && strings.TrimSpace(value) != "" {
		justification = strings.TrimSpace(value)
	}
	if len(justification) > 512 {
		return policyengine.ActionRequest{}, fmt.Errorf("details.override_reason exceeds max length 512")
	}
	if len(ticketRef) > 256 {
		return policyengine.ActionRequest{}, fmt.Errorf("details.ticket_ref exceeds max length 256")
	}
	return policyengine.NewGateOverrideAction(policyengine.GateOverrideActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: "cap_gate_override",
			Actor: policyengine.ActionActor{
				ActorKind:  "daemon",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		GateID:                    strings.TrimSpace(report.GateID),
		GateKind:                  strings.TrimSpace(report.GateKind),
		GateVersion:               strings.TrimSpace(report.GateVersion),
		GateAttemptID:             strings.TrimSpace(report.GateAttemptID),
		OverriddenFailedResultRef: strings.TrimSpace(report.OverriddenFailedResultRef),
		PolicyContextHash:         policyContextHash,
		OverrideMode:              overrideMode,
		Justification:             justification,
		ExpiresAt:                 expiresAt,
		TicketRef:                 ticketRef,
	}), nil
}

func validateCheckpointGateAttemptMutation(advisory artifacts.RunnerAdvisoryState, report RunnerCheckpointReport) error {
	gateAttemptID := strings.TrimSpace(report.GateAttemptID)
	if gateAttemptID == "" {
		return nil
	}
	existing, ok := advisory.GateAttempts[gateAttemptID]
	if !ok {
		return nil
	}
	if existing.Terminal {
		return fmt.Errorf("gate_attempt_id %q already terminal; retries must mint a new gate_attempt_id", gateAttemptID)
	}
	if strings.TrimSpace(existing.GateID) != "" && strings.TrimSpace(existing.GateID) != strings.TrimSpace(report.GateID) {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_id", gateAttemptID)
	}
	if strings.TrimSpace(existing.GateKind) != "" && strings.TrimSpace(existing.GateKind) != strings.TrimSpace(report.GateKind) {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_kind", gateAttemptID)
	}
	if strings.TrimSpace(existing.GateVersion) != "" && strings.TrimSpace(existing.GateVersion) != strings.TrimSpace(report.GateVersion) {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_version", gateAttemptID)
	}
	return nil
}

func validateResultGateAttemptMutation(advisory artifacts.RunnerAdvisoryState, report RunnerResultReport) error {
	gateAttemptID := strings.TrimSpace(report.GateAttemptID)
	if gateAttemptID == "" {
		return nil
	}
	existing, ok := advisory.GateAttempts[gateAttemptID]
	if !ok {
		return nil
	}
	if existing.Terminal {
		return fmt.Errorf("gate_attempt_id %q already has terminal result; retries must mint a new gate_attempt_id", gateAttemptID)
	}
	if strings.TrimSpace(existing.GateID) != "" && strings.TrimSpace(existing.GateID) != strings.TrimSpace(report.GateID) {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_id", gateAttemptID)
	}
	if strings.TrimSpace(existing.GateKind) != "" && strings.TrimSpace(existing.GateKind) != strings.TrimSpace(report.GateKind) {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_kind", gateAttemptID)
	}
	if strings.TrimSpace(existing.GateVersion) != "" && strings.TrimSpace(existing.GateVersion) != strings.TrimSpace(report.GateVersion) {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_version", gateAttemptID)
	}
	return nil
}

func validateOverrideReferenceAgainstHistory(advisory artifacts.RunnerAdvisoryState, report RunnerResultReport) error {
	if strings.TrimSpace(report.GateLifecycleState) != "overridden" {
		return nil
	}
	ref := strings.TrimSpace(report.OverriddenFailedResultRef)
	if ref == "" {
		return nil
	}
	for _, attempt := range advisory.GateAttempts {
		if strings.TrimSpace(attempt.ResultRef) != ref {
			continue
		}
		if strings.TrimSpace(attempt.GateState) != "failed" {
			return fmt.Errorf("overridden_failed_result_ref must reference a failed gate result")
		}
		if strings.TrimSpace(attempt.GateID) != strings.TrimSpace(report.GateID) || strings.TrimSpace(attempt.GateKind) != strings.TrimSpace(report.GateKind) || strings.TrimSpace(attempt.GateVersion) != strings.TrimSpace(report.GateVersion) {
			return fmt.Errorf("overridden_failed_result_ref must reference matching gate identity")
		}
		if strings.TrimSpace(attempt.GateAttemptID) == strings.TrimSpace(report.GateAttemptID) {
			return fmt.Errorf("overridden_failed_result_ref must reference a prior failed gate attempt")
		}
		return nil
	}
	return fmt.Errorf("overridden_failed_result_ref does not reference known failed gate result")
}

func canonicalGateResultRef(runID string, report RunnerResultReport, gateEvidenceRef string) (string, error) {
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
		return "", nil
	}
	payload := map[string]any{
		"schema_id":                    "runecode.protocol.v0.GateResultReport",
		"schema_version":               "0.1.0",
		"run_id":                       strings.TrimSpace(runID),
		"gate_id":                      strings.TrimSpace(report.GateID),
		"gate_kind":                    strings.TrimSpace(report.GateKind),
		"gate_version":                 strings.TrimSpace(report.GateVersion),
		"gate_attempt_id":              strings.TrimSpace(report.GateAttemptID),
		"lifecycle_state":              strings.TrimSpace(report.GateLifecycleState),
		"result_code":                  strings.TrimSpace(report.ResultCode),
		"occurred_at":                  strings.TrimSpace(report.OccurredAt),
		"idempotency_key":              strings.TrimSpace(report.IdempotencyKey),
		"stage_id":                     strings.TrimSpace(report.StageID),
		"step_id":                      strings.TrimSpace(report.StepID),
		"role_instance_id":             strings.TrimSpace(report.RoleInstanceID),
		"stage_attempt_id":             strings.TrimSpace(report.StageAttemptID),
		"step_attempt_id":              strings.TrimSpace(report.StepAttemptID),
		"failure_reason_code":          strings.TrimSpace(report.FailureReasonCode),
		"overridden_failed_result_ref": strings.TrimSpace(report.OverriddenFailedResultRef),
		"gate_evidence_ref":            strings.TrimSpace(gateEvidenceRef),
	}
	if strings.TrimSpace(report.PlanCheckpointCode) != "" {
		payload["plan_checkpoint_code"] = strings.TrimSpace(report.PlanCheckpointCode)
		payload["plan_order_index"] = report.PlanOrderIndex
	}
	if len(report.NormalizedInputDigests) > 0 {
		payload["normalized_input_digests"] = append([]string{}, report.NormalizedInputDigests...)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal gate result ref payload: %w", err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", fmt.Errorf("canonicalize gate result ref payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s *Service) resolveGateEvidenceRef(runID string, report RunnerResultReport, planned runPlannedGateEntry) (string, error) {
	providedRef := strings.TrimSpace(report.GateEvidenceRef)
	if report.GateEvidence == nil {
		return providedRef, nil
	}
	evidence := report.GateEvidence
	runtimeSummary, err := sanitizeRunnerDetails(evidence.Runtime)
	if err != nil {
		return "", fmt.Errorf("gate_evidence.runtime: %w", err)
	}
	outcomeSummary, err := sanitizeRunnerDetails(evidence.Outcome)
	if err != nil {
		return "", fmt.Errorf("gate_evidence.outcome: %w", err)
	}
	if strings.TrimSpace(evidence.RunID) != strings.TrimSpace(runID) {
		return "", fmt.Errorf("gate_evidence.run_id must match run_id")
	}
	if strings.TrimSpace(evidence.GateID) != strings.TrimSpace(report.GateID) || strings.TrimSpace(evidence.GateKind) != strings.TrimSpace(report.GateKind) || strings.TrimSpace(evidence.GateVersion) != strings.TrimSpace(report.GateVersion) || strings.TrimSpace(evidence.GateAttemptID) != strings.TrimSpace(report.GateAttemptID) {
		return "", fmt.Errorf("gate_evidence identity must match gate report binding")
	}
	if strings.TrimSpace(evidence.PlanCheckpointCode) != "" && strings.TrimSpace(evidence.PlanCheckpointCode) != strings.TrimSpace(report.PlanCheckpointCode) {
		return "", fmt.Errorf("gate_evidence.plan_checkpoint_code must match gate report binding")
	}
	if strings.TrimSpace(evidence.PlanCheckpointCode) != "" && evidence.PlanOrderIndex != report.PlanOrderIndex {
		return "", fmt.Errorf("gate_evidence.plan_order_index must match gate report binding")
	}
	evidenceRecord := artifacts.GateEvidenceArtifact{
		SchemaID:               evidence.SchemaID,
		SchemaVersion:          evidence.SchemaVersion,
		GateID:                 evidence.GateID,
		GateKind:               evidence.GateKind,
		GateVersion:            evidence.GateVersion,
		PlanCheckpointCode:     evidence.PlanCheckpointCode,
		PlanOrderIndex:         evidence.PlanOrderIndex,
		RunID:                  evidence.RunID,
		StageID:                evidence.StageID,
		StepID:                 evidence.StepID,
		RoleInstanceID:         evidence.RoleInstanceID,
		GateAttemptID:          evidence.GateAttemptID,
		StartedAt:              evidence.StartedAt,
		FinishedAt:             evidence.FinishedAt,
		NormalizedInputDigests: append([]string{}, evidence.NormalizedInputDigests...),
		Runtime:                runtimeSummary,
		Outcome:                outcomeSummary,
		OutputArtifactDigests:  append([]string{}, evidence.OutputArtifactDigests...),
		PolicyDecisionRefs:     append([]string{}, evidence.PolicyDecisionRefs...),
		OverrideActionHash:     evidence.OverrideActionRequestHash,
		OverridePolicyRef:      evidence.OverridePolicyDecisionRef,
		OverriddenFailedRef:    evidence.OverriddenFailedResultRef,
		FailureReasonCode:      evidence.FailureReasonCode,
	}
	if strings.TrimSpace(report.PlanCheckpointCode) != "" {
		evidenceRecord.PlanCheckpointCode = strings.TrimSpace(report.PlanCheckpointCode)
		evidenceRecord.PlanOrderIndex = report.PlanOrderIndex
	}
	if planned.MaxAttempts > 0 {
		evidenceRecord.Runtime["planned_retry_max_attempts"] = planned.MaxAttempts
	}
	canonicalEvidence, err := canonicalGateEvidenceDigest(evidenceRecord)
	if err != nil {
		return "", err
	}
	if providedRef != "" && providedRef != canonicalEvidence {
		return "", fmt.Errorf("gate_evidence_ref does not match canonical evidence digest")
	}
	ref, err := s.PutGateEvidence(runID, evidenceRecord)
	if err != nil {
		return "", err
	}
	return ref.Digest, nil
}

func canonicalGateEvidenceDigest(evidence artifacts.GateEvidenceArtifact) (string, error) {
	payload, err := json.Marshal(evidence)
	if err != nil {
		return "", fmt.Errorf("marshal gate evidence: %w", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize gate evidence: %w", err)
	}
	return artifacts.DigestBytes(canonical), nil
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
		out[key] = cloneRunnerDetailsValue(value)
	}
	return out
}

func cloneRunnerDetailsValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneRunnerDetailsMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneRunnerDetailsValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
