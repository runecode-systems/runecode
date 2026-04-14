package brokerapi

import (
	"fmt"
	"io"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var (
	BrokerProductVersion = "0.0.0-dev"
	BrokerBuildRevision  = "dev"
	BrokerBuildTime      = "1970-01-01T00:00:00Z"
)

const (
	brokerProtocolBundleVersion      = "0.9.0"
	brokerProtocolBundleManifestHash = "sha256:47427e96642a0f2cb7fb4e66aed61817f72f4233f0273744baa8469a2d13f170"
)

type Service struct {
	store                     *artifacts.Store
	auditLedger               *auditd.Ledger
	auditor                   *brokerAuditEmitter
	auditRoot                 string
	gatewayQuota              *gatewayQuotaBackend
	gatewayRuntime            *modelGatewayRuntime
	instancePostureController instanceBackendPostureController
	apiConfig                 APIConfig
	apiInflight               *inFlightGate
	versionInfo               BrokerVersionInfo
	now                       func() time.Time
}

func NewService(storeRoot string, ledgerRoot string) (*Service, error) {
	return NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{})
}

func NewServiceWithConfig(storeRoot string, ledgerRoot string, cfg APIConfig) (*Service, error) {
	store, err := artifacts.NewStore(storeRoot)
	if err != nil {
		return nil, err
	}
	ledger, err := auditd.Open(ledgerRoot)
	if err != nil {
		return nil, err
	}
	resolved := cfg.withDefaults()
	auditor, err := newBrokerAuditEmitter()
	if err != nil {
		return nil, err
	}
	quotaBackend := newGatewayQuotaBackend()
	quotaBackend.setLimits(resolved.GatewayQuota)
	runtime := newModelGatewayRuntime(quotaBackend)
	svc := &Service{
		store:                     store,
		auditLedger:               ledger,
		auditor:                   auditor,
		auditRoot:                 ledgerRoot,
		gatewayQuota:              quotaBackend,
		gatewayRuntime:            runtime,
		instancePostureController: newLocalInstanceBackendPostureController(),
		apiConfig:                 resolved,
		apiInflight:               newInFlightGate(resolved.Limits),
		now:                       time.Now,
		versionInfo:               defaultBrokerVersionInfo(),
	}
	runtime.auditFn = svc.AppendTrustedAuditEvent
	return svc, nil
}

func defaultBrokerVersionInfo() BrokerVersionInfo {
	return BrokerVersionInfo{
		SchemaID:                    "runecode.protocol.v0.BrokerVersionInfo",
		SchemaVersion:               "0.1.0",
		ProductVersion:              BrokerProductVersion,
		BuildRevision:               BrokerBuildRevision,
		BuildTime:                   BrokerBuildTime,
		ProtocolBundleVersion:       brokerProtocolBundleVersion,
		ProtocolBundleManifestHash:  brokerProtocolBundleManifestHash,
		APIFamily:                   "broker_local_api",
		APIVersion:                  "v0",
		SupportedTransportEncodings: []string{"json"},
	}
}

func (s *Service) SetVersionInfo(info BrokerVersionInfo) {
	if info.SchemaID == "" {
		info.SchemaID = "runecode.protocol.v0.BrokerVersionInfo"
	}
	if info.SchemaVersion == "" {
		info.SchemaVersion = "0.1.0"
	}
	if info.APIFamily == "" {
		info.APIFamily = "broker_local_api"
	}
	if info.APIVersion == "" {
		info.APIVersion = "v0"
	}
	if len(info.SupportedTransportEncodings) == 0 {
		info.SupportedTransportEncodings = []string{"json"}
	}
	s.versionInfo = info
}

func (s *Service) SetNowFuncForTests(nowFn func() time.Time) {
	if nowFn == nil {
		s.now = time.Now
		s.store.SetNowFuncForTests(nil)
		return
	}
	s.now = nowFn
	s.store.SetNowFuncForTests(nowFn)
}

func (s *Service) Put(req artifacts.PutRequest) (artifacts.ArtifactReference, error) {
	return s.store.Put(req)
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

func (s *Service) GarbageCollect() (artifacts.GCResult, error) {
	return s.store.GarbageCollect()
}

func (s *Service) DeleteDigest(digest string) error {
	return s.store.DeleteDigest(digest)
}

func (s *Service) ExportBackup(path string) error {
	return s.store.ExportBackup(path)
}

func (s *Service) RestoreBackup(path string) error {
	return s.store.RestoreBackup(path)
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

func (s *Service) AppendSessionMessage(req artifacts.SessionMessageAppendRequest) (artifacts.SessionMessageAppendResult, error) {
	return s.store.AppendSessionMessage(req)
}

func (s *Service) RecordApproval(record artifacts.ApprovalRecord) error {
	return s.store.RecordApproval(record)
}

func (s *Service) SetPolicy(policy artifacts.Policy) error {
	return s.store.SetPolicy(policy)
}

func (s *Service) Policy() artifacts.Policy {
	return s.store.Policy()
}

func (s *Service) AuditReadiness() (trustpolicy.AuditdReadiness, error) {
	return s.auditLedger.Readiness()
}

type AuditVerificationSurface struct {
	Summary trustpolicy.DerivedRunAuditVerificationSummary `json:"summary"`
	Report  trustpolicy.AuditVerificationReportPayload     `json:"report"`
	Views   []trustpolicy.AuditOperationalView             `json:"views"`
}

func (s *Service) LatestAuditVerificationSurface(limit int) (AuditVerificationSurface, error) {
	if s.auditLedger == nil {
		return AuditVerificationSurface{}, fmt.Errorf("audit ledger unavailable")
	}
	summary, views, report, err := s.auditLedger.LatestVerificationSummaryAndViews(limit)
	if err != nil {
		return AuditVerificationSurface{}, err
	}
	return AuditVerificationSurface{Summary: summary, Report: report, Views: views}, nil
}

func (s *Service) APILimits() Limits { return s.apiConfig.Limits }
