package brokerapi

import (
	"io"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) Put(req artifacts.PutRequest) (artifacts.ArtifactReference, error) {
	ref, err := s.store.Put(req)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	if isTrustedRunPlanPutCandidate(req.TrustedSource, req.CreatedByRole, req.ContentType, req.RunID, req.StepID) {
		s.runGatePlanCache.invalidateRun(req.RunID)
	}
	return ref, nil
}

func (s *Service) PutStream(req artifacts.PutStreamRequest) (artifacts.ArtifactReference, error) {
	ref, err := s.store.PutStream(req)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	if isTrustedRunPlanPutCandidate(req.TrustedSource, req.CreatedByRole, req.ContentType, req.RunID, req.StepID) {
		s.runGatePlanCache.invalidateRun(req.RunID)
	}
	return ref, nil
}

func (s *Service) List() []artifacts.ArtifactRecord {
	return s.store.List()
}

func (s *Service) RunStatuses() map[string]string {
	return s.store.RunStatuses()
}

func (s *Service) Head(digest string) (artifacts.ArtifactRecord, error) {
	return s.store.Head(digest)
}

func (s *Service) Get(digest string) (io.ReadCloser, error) {
	return s.store.Get(digest)
}

func (s *Service) GetForFlow(req artifacts.ArtifactReadRequest) (io.ReadCloser, artifacts.ArtifactRecord, error) {
	return s.store.GetForFlow(req)
}

func (s *Service) CheckFlow(req artifacts.FlowCheckRequest) error {
	return s.store.CheckFlow(req)
}

func (s *Service) PromoteApprovedExcerpt(req artifacts.PromotionRequest) (artifacts.ArtifactReference, error) {
	return s.store.PromoteApprovedExcerpt(req)
}

func (s *Service) RevokeApprovedExcerpt(digest, actor string) error {
	return s.store.RevokeApprovedExcerpt(digest, actor)
}

func (s *Service) SetRunStatus(runID, status string) error {
	if err := s.store.SetRunStatus(runID, status); err != nil {
		return err
	}
	if s.gatewayQuota != nil && isTerminalRunStatus(status) {
		s.gatewayQuota.releaseRun(runID)
	}
	return nil
}

func (s *Service) RecordRunnerCheckpoint(runID string, checkpoint artifacts.RunnerCheckpointAdvisory) (bool, error) {
	return s.store.RecordRunnerCheckpoint(runID, checkpoint)
}

func (s *Service) RecordRunnerResult(runID string, result artifacts.RunnerResultAdvisory, overridePolicyRef string) (bool, error) {
	return s.store.RecordRunnerResult(runID, result, overridePolicyRef)
}

func (s *Service) PutGateEvidence(runID string, evidence artifacts.GateEvidenceArtifact) (artifacts.ArtifactReference, error) {
	return s.store.PutGateEvidence(runID, evidence)
}

func (s *Service) RunnerAdvisory(runID string) (artifacts.RunnerAdvisoryState, bool) {
	return s.store.RunnerAdvisory(runID)
}

func (s *Service) RecordRunnerApprovalWait(approval artifacts.RunnerApproval) error {
	return s.store.RecordRunnerApprovalWait(approval)
}

func (s *Service) SyncSessionExecutionFromRunRuntime(runID string, facts launcherbackend.RuntimeFactsSnapshot, advisory artifacts.RunnerAdvisoryState, occurredAt time.Time) error {
	return s.store.SyncSessionExecutionFromRunRuntime(runID, facts, advisory, occurredAt)
}

func (s *Service) GarbageCollect() (artifacts.GCResult, error) {
	result, err := s.store.GarbageCollect()
	if err != nil {
		return artifacts.GCResult{}, err
	}
	s.runGatePlanCache.invalidateAll()
	return result, nil
}

func (s *Service) DeleteDigest(digest string) error {
	if err := s.store.DeleteDigest(digest); err != nil {
		return err
	}
	s.runGatePlanCache.invalidateAll()
	return nil
}

func (s *Service) ExportBackup(path string) error {
	return s.store.ExportBackup(path)
}

func (s *Service) RestoreBackup(path string) error {
	if err := s.store.RestoreBackup(path); err != nil {
		return err
	}
	s.runGatePlanCache.invalidateAll()
	return s.reloadProviderDurableState()
}

func (s *Service) ReadAuditEvents() ([]artifacts.AuditEvent, error) {
	return s.store.ReadAuditEvents()
}

func (s *Service) AppendTrustedAuditEvent(eventType, actor string, details map[string]interface{}) error {
	return s.store.AppendTrustedAuditEvent(eventType, actor, details)
}

func (s *Service) RecordPolicyDecision(runID string, digest string, decision policyengine.PolicyDecision) error {
	return s.store.RecordPolicyDecision(artifacts.PolicyDecisionRecord{
		Digest:                   digest,
		RunID:                    runID,
		SchemaID:                 decision.SchemaID,
		SchemaVersion:            decision.SchemaVersion,
		DecisionOutcome:          string(decision.DecisionOutcome),
		PolicyReasonCode:         decision.PolicyReasonCode,
		ManifestHash:             decision.ManifestHash,
		ActionRequestHash:        decision.ActionRequestHash,
		PolicyInputHashes:        append([]string{}, decision.PolicyInputHashes...),
		RelevantArtifactHashes:   append([]string{}, decision.RelevantArtifactHashes...),
		DetailsSchemaID:          decision.DetailsSchemaID,
		Details:                  decision.Details,
		RequiredApprovalSchemaID: decision.RequiredApprovalSchemaID,
		RequiredApproval:         decision.RequiredApproval,
	})
}

func (s *Service) PolicyDecisionRefsForRun(runID string) []string {
	return s.store.PolicyDecisionRefsForRun(runID)
}

func (s *Service) PolicyDecisionGet(digest string) (artifacts.PolicyDecisionRecord, bool) {
	return s.store.PolicyDecisionGet(digest)
}

func (s *Service) ApprovalList() []artifacts.ApprovalRecord {
	return s.store.ApprovalList()
}

func (s *Service) ApprovalGet(approvalID string) (artifacts.ApprovalRecord, bool) {
	return s.store.ApprovalGet(approvalID)
}

func (s *Service) SessionState(sessionID string) (artifacts.SessionDurableState, bool) {
	return s.store.SessionState(sessionID)
}

func (s *Service) SessionStates() map[string]artifacts.SessionDurableState {
	return s.store.SessionStates()
}

func (s *Service) UpdateSessionState(sessionID string, mutate func(artifacts.SessionDurableState) artifacts.SessionDurableState) (artifacts.SessionDurableState, error) {
	return s.store.UpdateSessionState(sessionID, mutate)
}

func (s *Service) AppendSessionMessage(req artifacts.SessionMessageAppendRequest) (artifacts.SessionMessageAppendResult, error) {
	return s.store.AppendSessionMessage(req)
}

func (s *Service) AppendSessionExecutionTrigger(req artifacts.SessionExecutionTriggerAppendRequest) (artifacts.SessionExecutionTriggerAppendResult, error) {
	return s.store.AppendSessionExecutionTrigger(req)
}

func (s *Service) UpdateSessionTurnExecution(req artifacts.SessionTurnExecutionUpdateRequest) (artifacts.SessionTurnExecutionDurableState, error) {
	return s.store.UpdateSessionTurnExecution(req)
}

func (s *Service) RecordApproval(record artifacts.ApprovalRecord) error {
	return s.store.RecordApproval(record)
}

func (s *Service) GitRemotePreparedUpsert(record artifacts.GitRemotePreparedMutationRecord) error {
	return s.store.GitRemotePreparedUpsert(record)
}

func (s *Service) GitRemotePreparedGet(preparedMutationID string) (artifacts.GitRemotePreparedMutationRecord, bool) {
	return s.store.GitRemotePreparedGet(preparedMutationID)
}

func (s *Service) GitRemotePreparedTransitionLifecycle(preparedMutationID, expectedLifecycle string, mutate func(artifacts.GitRemotePreparedMutationRecord) artifacts.GitRemotePreparedMutationRecord) (artifacts.GitRemotePreparedMutationRecord, error) {
	return s.store.GitRemotePreparedTransitionLifecycle(preparedMutationID, expectedLifecycle, mutate)
}

func (s *Service) GitRemotePreparedRefsForRun(runID string) []string {
	return s.store.GitRemotePreparedRefsForRun(runID)
}

func (s *Service) ExternalAnchorPreparedUpsert(record artifacts.ExternalAnchorPreparedMutationRecord) error {
	return s.store.ExternalAnchorPreparedUpsert(record)
}

func (s *Service) ExternalAnchorPreparedGet(preparedMutationID string) (artifacts.ExternalAnchorPreparedMutationRecord, bool) {
	return s.store.ExternalAnchorPreparedGet(preparedMutationID)
}

func (s *Service) ExternalAnchorPreparedTransitionLifecycle(preparedMutationID, expectedLifecycle string, mutate func(artifacts.ExternalAnchorPreparedMutationRecord) artifacts.ExternalAnchorPreparedMutationRecord) (artifacts.ExternalAnchorPreparedMutationRecord, error) {
	return s.store.ExternalAnchorPreparedTransitionLifecycle(preparedMutationID, expectedLifecycle, mutate)
}

func (s *Service) ExternalAnchorPreparedRefsForRun(runID string) []string {
	return s.store.ExternalAnchorPreparedRefsForRun(runID)
}

func (s *Service) ExternalAnchorPreparedIDs() []string {
	return s.store.ExternalAnchorPreparedIDs()
}

func (s *Service) ExternalAnchorPreparedClaimDeferredExecution(preparedMutationID, expectedAttemptID, claimID string, staleAfter time.Duration, claimedAt time.Time) (artifacts.ExternalAnchorPreparedMutationRecord, bool, error) {
	return s.store.ExternalAnchorPreparedClaimDeferredExecution(preparedMutationID, expectedAttemptID, claimID, staleAfter, claimedAt)
}

func (s *Service) SetPolicy(policy artifacts.Policy) error {
	return s.store.SetPolicy(policy)
}

func (s *Service) Policy() artifacts.Policy {
	return s.store.Policy()
}

func (s *Service) SyncExternalStoreState() error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.SyncExternalState()
}

func (s *Service) DependencyCacheLookup(req artifacts.DependencyCacheHitRequest) (artifacts.DependencyCacheBatchRecord, artifacts.DependencyCacheResolvedUnitRecord, bool, error) {
	return s.store.DependencyCacheLookup(req)
}

func (s *Service) RecordDependencyCacheBatch(batch artifacts.DependencyCacheBatchRecord, units []artifacts.DependencyCacheResolvedUnitRecord) error {
	return s.store.RecordDependencyCacheBatch(batch, units)
}

func (s *Service) DependencyCacheResolvedUnitByRequest(requestDigest string) (artifacts.DependencyCacheResolvedUnitRecord, bool, error) {
	return s.store.DependencyCacheResolvedUnitByRequest(requestDigest)
}

func (s *Service) RecordDependencyCacheResolvedUnit(unit artifacts.DependencyCacheResolvedUnitRecord) error {
	return s.store.RecordDependencyCacheResolvedUnit(unit)
}

func (s *Service) DependencyCacheHandoffByRequest(req artifacts.DependencyCacheHandoffRequest) (artifacts.DependencyCacheHandoff, bool, error) {
	return s.store.DependencyCacheHandoffByRequest(req)
}

func (s *Service) RecordRunPlanAuthority(authority artifacts.RunPlanAuthorityRecord, compilation artifacts.RunPlanCompilationRecord) error {
	if err := s.store.RecordRunPlanAuthority(authority, compilation); err != nil {
		return err
	}
	s.runGatePlanCache.invalidateRun(authority.RunID)
	return nil
}

func (s *Service) ActiveRunPlanAuthority(runID string) (artifacts.RunPlanAuthorityRecord, bool, error) {
	return s.store.ActiveRunPlanAuthority(runID)
}

func (s *Service) RunPlanCompilationRecord(runID, planID string) (artifacts.RunPlanCompilationRecord, bool) {
	return s.store.RunPlanCompilationRecord(runID, planID)
}

func (s *Service) RunPlanCompilationRecordByCacheKey(cacheKey string) (artifacts.RunPlanCompilationRecord, bool) {
	return s.store.RunPlanCompilationRecordByCacheKey(cacheKey)
}
